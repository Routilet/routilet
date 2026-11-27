package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const cliPackage = "@routilet/accoding-companion"
const presetPackage = "@routilet/c-exam-kit"

func legacyPackage(name string) string {
	if name == presetPackage {
		return "@annouimo/c-exam-kit-preset"
	}
	return strings.Replace(name, "@routilet/", "@annouimo/", 1)
}

var stableVersion = regexp.MustCompile("^[0-9]+\\.[0-9]+\\.[0-9]+$")

func packagePath(dir, name string) string {
	return filepath.Join(dir, "node_modules", filepath.FromSlash(name))
}

func versionOf(dir, name string) string {
	if dir == "" {
		return ""
	}
	metadata, err := os.ReadFile(filepath.Join(packagePath(dir, name), "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct{ Name, Version string }
	if json.Unmarshal(metadata, &pkg) != nil || pkg.Name != name || !stableVersion.MatchString(pkg.Version) {
		return ""
	}
	return pkg.Version
}

func validScope(dir string, legacy bool) bool {
	for name, file := range map[string]string{cliPackage: "dist/accoding.mjs", presetPackage: "preset/manifest.json"} {
		if legacy {
			name = legacyPackage(name)
		}
		if versionOf(dir, name) == "" || !isFile(filepath.Join(packagePath(dir, name), filepath.FromSlash(file))) {
			return false
		}
	}
	return true
}

func validInstallation(dir string) bool {
	return validScope(dir, false) || validScope(dir, true)
}

func activePackage(dir, name string) string {
	if !validScope(dir, false) && validScope(dir, true) {
		return legacyPackage(name)
	}
	return name
}

func installedVersion(dir, name string) string {
	return versionOf(dir, activePackage(dir, name))
}

func legacyInstallation(dir string) bool {
	return dir != "" && !validScope(dir, false) && validScope(dir, true)
}

func canMigrate(dir string, available updateResult) bool {
	return legacyInstallation(dir) && available.err == nil &&
		stableVersion.MatchString(available.cli) && stableVersion.MatchString(available.preset) &&
		!newerVersion(installedVersion(dir, cliPackage), available.cli) &&
		!newerVersion(installedVersion(dir, presetPackage), available.preset)
}

func readInstallation(root string) string {
	value, err := os.ReadFile(filepath.Join(root, "current"))
	name := strings.TrimSpace(string(value))
	if err != nil || !strings.HasPrefix(name, "release-") || filepath.Base(name) != name {
		return ""
	}
	dir := filepath.Join(root, "packages", name)
	if validInstallation(dir) {
		return dir
	}
	return ""
}

func registryVersion(name string) (string, error) {
	client := http.Client{Timeout: 5 * time.Second}
	response, err := client.Get("https://registry.npmjs.org/" + name + "/latest")
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	var pack struct{ Version string }
	if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&pack) != nil ||
		!stableVersion.MatchString(pack.Version) {
		return "", fmt.Errorf("无法读取 %s 的最新版本", name)
	}
	return pack.Version, nil
}

func newerVersion(latest, current string) bool {
	if !stableVersion.MatchString(latest) || !stableVersion.MatchString(current) {
		return false
	}
	a, b := strings.Split(latest, "."), strings.Split(current, ".")
	for index := range a {
		x, _ := strconv.Atoi(a[index])
		y, _ := strconv.Atoi(b[index])
		if x != y {
			return x > y
		}
	}
	return false
}

func progress(step int, label string) {
	fmt.Printf("[%s%s] %d/5 %s\n", strings.Repeat("█", step), strings.Repeat("░", 5-step), step, label)
}

func install(root, installed string, available updateResult) (string, error) {
	progress(1, "检查运行环境")
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	r, err := findNode(root)
	if err != nil {
		fmt.Println("正在安装本应用专用的 Node.js…")
		r, err = installNode(root)
		if err != nil {
			return "", fmt.Errorf("安装 Node.js：%w", err)
		}
	}
	progress(2, "确认安装版本")
	cli, preset := available.cli, available.preset
	migrating := legacyInstallation(installed)
	if migrating && !canMigrate(installed, available) {
		return installed, fmt.Errorf("Routilet 新软件包暂不可用或版本较旧；原安装保留")
	}
	if installed == "" {
		cli, err = r.latest(cliPackage)
		if err != nil {
			return "", err
		}
		preset, err = r.latest(presetPackage)
		if err != nil {
			return "", err
		}
	} else {
		installedCLI, installedPreset := installedVersion(installed, cliPackage), installedVersion(installed, presetPackage)
		if !newerVersion(cli, installedCLI) {
			cli = installedCLI
		}
		if !newerVersion(preset, installedPreset) {
			preset = installedPreset
		}
	}
	if installed != "" {
		fmt.Printf("Companion %s → %s\n资源包 %s → %s\n",
			installedVersion(installed, cliPackage), cli, installedVersion(installed, presetPackage), preset)
		if !migrating && installedVersion(installed, cliPackage) == cli && installedVersion(installed, presetPackage) == preset {
			pause("已是当前发布版本。")
			return installed, nil
		}
		message := "安装上述更新？"
		if migrating {
			message = "迁移到 Routilet 新发行包？原安装与课程文件保留。"
		}
		if !confirm(message) {
			return installed, nil
		}
	}
	parent := filepath.Join(root, "packages")
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(parent, "release-")
	if err != nil {
		return "", err
	}
	saved := false
	defer func() {
		if !saved {
			_ = os.RemoveAll(stage)
		}
	}()
	progress(3, fmt.Sprintf("安装 Companion %s 和 C Exam Kit %s", cli, preset))
	cmd := exec.Command(r.node, r.npm, "install", "--prefix", stage, "--no-save",
		"--ignore-scripts", "--no-audit", "--no-fund",
		cliPackage+"@"+cli, presetPackage+"@"+preset)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("安装 npm 包：%w", err)
	}
	progress(4, "验证安装文件")
	if !validScope(stage, false) || versionOf(stage, cliPackage) != cli || versionOf(stage, presetPackage) != preset {
		return "", fmt.Errorf("下载的包缺少文件或版本不匹配")
	}
	check := exec.Command(r.node, filepath.Join(packagePath(stage, cliPackage), "dist", "accoding.mjs"), "--version")
	output, err := check.Output()
	if err != nil || strings.TrimSpace(string(output)) != cli {
		return "", fmt.Errorf("无法启动新下载的 Companion %s", cli)
	}
	pointer, err := os.CreateTemp(root, ".current-")
	if err != nil {
		return "", err
	}
	defer os.Remove(pointer.Name())
	_, err = fmt.Fprintln(pointer, filepath.Base(stage))
	closeErr := pointer.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err = os.Rename(pointer.Name(), filepath.Join(root, "current")); err != nil {
		return "", err
	}
	saved = true
	progress(5, "安装完成")
	return stage, nil
}
