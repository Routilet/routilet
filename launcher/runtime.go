package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const nodeVersion = "v22.23.5"

var releaseDigests = map[string]string{
	"darwin-arm64.tar.gz": "e7131995759b61a0170b646010d2ad35a5d37b4cb0d471f3e8317eac6f742f3d",
	"darwin-x64.tar.gz":   "c53383238206b9ae4db23c5d8a63645e2e753bec8ab51a49cc81939aa64e9422",
	"linux-arm64.tar.gz":  "8bd3dc69eee8f3f226315a80d7d1d4faccf217a4c8fae500168ac2e38f3bdba3",
	"linux-x64.tar.gz":    "c54118104acc524633b5abe106bbf44a53126ec8b4e5fa105cb92b5d7de8a1a1",
	"win-arm64.zip":       "c1856817f611738c77534efbff8a5a17990593eab24046bc8d2c23a77a160d13",
	"win-x64.zip":         "4a191082e9706866830379d577d2cc8a50c776fcd9436a8d061b063ed0d9a0e3",
}

type nodeRuntime struct{ node, npm string }

func isFile(file string) bool {
	info, err := os.Stat(file)
	return err == nil && info.Mode().IsRegular()
}

func (r nodeRuntime) valid() bool {
	if !isFile(r.node) || !isFile(r.npm) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, r.node, "-p", "process.versions.node").Output()
	if err != nil {
		return false
	}
	parts := strings.Split(strings.TrimSpace(string(output)), ".")
	if len(parts) != 3 {
		return false
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	if !(major > 22 || major == 22 && minor >= 12) {
		return false
	}
	output, err = exec.CommandContext(ctx, r.node, r.npm, "--version").Output()
	return err == nil && strings.TrimSpace(string(output)) != ""
}

func (r nodeRuntime) latest(name string) (string, error) {
	cmd := exec.Command(r.node, r.npm, "view", name+"@latest", "version", "--prefer-online",
		"--fetch-retries=0", "--fetch-timeout=15000")
	data, err := cmd.Output()
	version := strings.TrimSpace(string(data))
	if err != nil || !stableVersion.MatchString(version) {
		return "", fmt.Errorf("无法从 npm 读取 %s 的正式版本，请检查网络后重试", name)
	}
	return version, nil
}

func runtimeAt(root string) nodeRuntime {
	if runtime.GOOS == "windows" {
		return nodeRuntime{filepath.Join(root, "node.exe"),
			filepath.Join(root, "node_modules", "npm", "bin", "npm-cli.js")}
	}
	return nodeRuntime{filepath.Join(root, "bin", "node"),
		filepath.Join(root, "lib", "node_modules", "npm", "bin", "npm-cli.js")}
}

func findNode(root string) (nodeRuntime, error) {
	managed := runtimeAt(filepath.Join(root, "runtime", nodeVersion))
	if managed.valid() {
		return managed, nil
	}
	entries, _ := os.ReadDir(filepath.Join(root, "runtime"))
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "v") {
			previous := runtimeAt(filepath.Join(root, "runtime", entry.Name()))
			if previous.valid() {
				return previous, nil
			}
		}
	}
	node, err := exec.LookPath("node")
	if err == nil {
		actual, resolveErr := filepath.EvalSymlinks(node)
		if resolveErr != nil {
			actual = node
		}
		bin := filepath.Dir(actual)
		dir := filepath.Dir(bin)
		if runtime.GOOS == "windows" {
			dir = bin
		}
		system := runtimeAt(dir)
		system.node = actual
		if system.valid() {
			return system, nil
		}
		if runtime.GOOS != "windows" {
			if npm, lookupErr := exec.LookPath("npm"); lookupErr == nil {
				if resolved, linkErr := filepath.EvalSymlinks(npm); linkErr == nil && strings.HasSuffix(resolved, "npm-cli.js") {
					system.npm = resolved
					if system.valid() {
						return system, nil
					}
				}
			}
		}
	}
	return nodeRuntime{}, fmt.Errorf("未找到可用的 Node.js 和 npm")
}

func installNode(root string) (nodeRuntime, error) {
	target := runtime.GOOS + "-" + nodeArch() + ".tar.gz"
	if runtime.GOOS == "windows" {
		target = "win-" + nodeArch() + ".zip"
	}
	digest, ok := releaseDigests[target]
	if !ok {
		return nodeRuntime{}, fmt.Errorf("当前平台没有可下载的 Node.js")
	}
	runtimes := filepath.Join(root, "runtime")
	if err := os.MkdirAll(runtimes, 0700); err != nil {
		return nodeRuntime{}, err
	}
	stage, err := os.MkdirTemp(runtimes, ".node-")
	if err != nil {
		return nodeRuntime{}, err
	}
	defer os.RemoveAll(stage)
	name := "node-" + nodeVersion + "-" + target
	archive := filepath.Join(stage, name)
	if err := downloadNode(archive, name, digest); err != nil {
		return nodeRuntime{}, err
	}
	bin := filepath.Join(stage, "extracted")
	if err := extractRuntime(archive, bin); err != nil {
		return nodeRuntime{}, err
	}
	if !runtimeAt(bin).valid() {
		return nodeRuntime{}, fmt.Errorf("下载的 Node.js 未通过运行检查")
	}
	destination := filepath.Join(runtimes, nodeVersion)
	if ready := runtimeAt(destination); ready.valid() {
		return ready, nil
	}
	if err := os.RemoveAll(destination); err != nil {
		return nodeRuntime{}, err
	}
	if err := retryRuntimeRename(func() error { return os.Rename(bin, destination) }, time.Sleep, runtime.GOOS == "windows"); err != nil {
		// Another installer may have completed while our directory was staged.
		if ready := runtimeAt(destination); ready.valid() {
			return ready, nil
		}
		return nodeRuntime{}, fmt.Errorf("无法启用 Node.js 运行环境（请关闭占用安装目录的程序后重试）：%w", err)
	}
	return runtimeAt(destination), nil
}

// Windows may temporarily refuse to move newly extracted executables (for
// example while an antivirus scanner holds a handle). Retry only the specific
// filesystem errors, with a finite delay; never overwrite the destination.
func retryRuntimeRename(rename func() error, sleep func(time.Duration), windows bool) error {
	const attempts = 8
	for i := 0; ; i++ {
		err := rename()
		if err == nil || !windows || i == attempts-1 ||
			!(errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.Errno(32)) || errors.Is(err, syscall.Errno(33))) {
			return err
		}
		sleep(time.Duration(min(100*(1<<i), 1000)) * time.Millisecond)
	}
}

func downloadNode(file, name, digest string) error {
	return downloadArchive("https://nodejs.org/dist/"+nodeVersion+"/"+name, file, digest, 200<<20)
}

func extractRuntime(archive, root string) error {
	write := func(name string, r io.Reader) error {
		_, file, _ := strings.Cut(name, "/")
		if file == "" || path.Clean(file) != file || strings.Contains(file, "\\") || !filepath.IsLocal(filepath.FromSlash(file)) {
			return nil
		}
		if file != "node.exe" && file != "bin/node" &&
			!strings.HasPrefix(file, "node_modules/npm/") && !strings.HasPrefix(file, "lib/node_modules/npm/") {
			return nil
		}
		dest := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return err
		}
		mode := os.FileMode(0600)
		if file == "bin/node" {
			mode = 0700
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, r)
		closeErr := out.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
	if strings.HasSuffix(archive, ".zip") {
		files, err := zip.OpenReader(archive)
		if err != nil {
			return err
		}
		defer files.Close()
		for _, file := range files.File {
			if !file.Mode().IsRegular() {
				continue
			}
			r, err := file.Open()
			if err != nil {
				return err
			}
			err = write(file.Name, r)
			r.Close()
			if err != nil {
				return err
			}
		}
		return nil
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	stream, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer stream.Close()
	files := tar.NewReader(stream)
	for {
		entry, err := files.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.Typeflag == tar.TypeReg {
			if err := write(entry.Name, files); err != nil {
				return err
			}
		}
	}
}

func downloadArchive(url, destination, digest string, maxSize int64) error {
	client := http.Client{Timeout: 8 * time.Minute}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("下载返回 %s", response.Status)
	}
	output, err := os.Create(destination)
	if err != nil {
		return err
	}
	hash := sha256.New()
	total := response.ContentLength
	var writer io.Writer = io.MultiWriter(output, hash)
	if total > 0 {
		writer = io.MultiWriter(writer, &downloadProgress{total: total})
	}
	size, copyErr := io.Copy(writer, io.LimitReader(response.Body, maxSize+1))
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if size > maxSize || hex.EncodeToString(hash.Sum(nil)) != digest {
		return fmt.Errorf("下载文件校验失败")
	}
	return nil
}

type downloadProgress struct {
	total, bytes int64
	step         int
}

func (p *downloadProgress) Write(data []byte) (int, error) {
	p.bytes += int64(len(data))
	step := int(min(10*p.bytes/p.total, 10))
	if step > p.step {
		p.step = step
		fmt.Printf("\r[%s%s] %d%%", strings.Repeat("█", step), strings.Repeat("░", 10-step), step*10)
		if step == 10 {
			fmt.Println()
		}
	}
	return len(data), nil
}
