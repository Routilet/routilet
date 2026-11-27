package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Routilet/accoding-companion/launcher/internal/managed"
)

// Real npm installs from a local registry with intentionally tiny synthetic
// Companion packages. No package is uploaded to the public registry.
func managedFixture(t *testing.T, version string, compatible bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	meta, err := json.Marshal(map[string]string{"name": managed.App, "version": version})
	if err != nil {
		t.Fatal(err)
	}
	info, err := json.Marshal(managed.Info{SchemaVersion: 1, Version: version, Ready: true})
	if err != nil {
		t.Fatal(err)
	}
	cli := fmt.Sprintf("if(process.argv[2]==='--version') console.log(%q); else ", version)
	if compatible {
		cli += fmt.Sprintf("if(process.argv[2]==='--installation-info') console.log(%q); else ", string(info))
	}
	cli += "{ console.error('错误：不支持的命令。'); process.exitCode=1; }"
	for _, file := range []struct {
		name string
		data []byte
	}{
		{"package/package.json", meta},
		{"package/dist/accoding.mjs", []byte(cli)},
	} {
		if err := tw.WriteHeader(&tar.Header{Name: file.name, Mode: 0600, Size: int64(len(file.data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(file.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func exerciseLocalRegistry(t *testing.T, runtime nodeRuntime, root string) {
	t.Helper()
	packages := map[string][]byte{
		"0.2.7": managedFixture(t, "0.2.7", false),
		"0.2.8": managedFixture(t, "0.2.8", true),
		"0.2.9": managedFixture(t, "0.2.9", true),
	}
	var latest atomic.Value
	latest.Store("0.2.7")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/"+managed.App {
			versions := map[string]any{}
			for v, data := range packages {
				sum := sha1.Sum(data)
				versions[v] = map[string]any{"name": managed.App, "version": v, "dist": map[string]string{
					"tarball": "http://" + r.Host + "/pkg-" + v + ".tgz",
					"shasum": hex.EncodeToString(sum[:]),
				}}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": managed.App, "dist-tags": map[string]string{"latest": latest.Load().(string)}, "versions": versions})
			return
		}
		version := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/pkg-"), ".tgz")
		if data, ok := packages[version]; ok {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(data)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(root, "empty.npmrc")
	if err := os.WriteFile(config, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("npm_config_registry", server.URL+"/")
	t.Setenv("npm_config_userconfig", config)
	t.Setenv("NO_PROXY", "127.0.0.1,localhost")
	t.Setenv("no_proxy", "127.0.0.1,localhost")
	manager := v2Manager(root, "latest", runtime)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	install := func(version string) {
		t.Helper()
		got, err := manager.Resolve(ctx)
		if err != nil || got != version {
			t.Fatalf("resolve: %q %v", got, err)
		}
		candidate, err := manager.Plan(ctx, version)
		if err != nil {
			t.Fatal(err)
		}
		defer candidate.Close()
		info, err := candidate.Commit(ctx)
		if err != nil || info.Version != version || !info.Ready {
			t.Fatalf("commit: %+v %v", info, err)
		}
	}
	v, err := manager.Resolve(ctx)
	if err != nil || v != "0.2.7" {
		t.Fatalf("resolve legacy: %q %v", v, err)
	}
	candidate, err := manager.Plan(ctx, v)
	if candidate != nil {
		defer candidate.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "0.2.7 --installation-info") {
		t.Fatalf("old CLI must be rejected with the actual failing command: %v", err)
	}
	state, err := manager.State()
	if err != nil || state.Current != "" {
		t.Fatalf("failed installation activated: %+v %v", state, err)
	}
	latest.Store("0.2.8")
	install("0.2.8")
	latest.Store("0.2.9")
	install("0.2.9")
	info, err := manager.Rollback(ctx)
	if err != nil || info.Version != "0.2.8" {
		t.Fatalf("rollback: %+v %v", info, err)
	}
	dir, _, free, err := manager.AcquireCurrent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer free()
	output, err := managedRunner(root, runtime).CLI(ctx, dir, "--version")
	if err != nil || strings.TrimSpace(string(output)) != "0.2.8" {
		t.Fatalf("start entrypoint: %q %v", output, err)
	}
}

// Run only after the stable npm package exists, before releasing the ZIPs.
// This is the same manager and downloaded Node/npm used by the new launcher.
func exercisePublishedRegistry(t *testing.T, runtime nodeRuntime, root, version, observerVersion string) {
	t.Helper()
	manager := v2Manager(root, "latest", runtime)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	got, err := manager.Resolve(ctx)
	if err != nil || got != version {
		t.Fatalf("public registry: %q %v", got, err)
	}
	candidate, err := manager.Plan(ctx, version)
	if err != nil {
		t.Fatal(err)
	}
	defer candidate.Close()
	info, err := candidate.Commit(ctx)
	if err != nil || info.Version != version || !info.Ready {
		t.Fatalf("public package: %+v %v", info, err)
	}
	found := false
	for _, c := range info.Components {
		if c.Package == "@routilet/c-observer" {
			found = c.Version == observerVersion && c.Status == "ready"
		}
	}
	if !found {
		t.Fatalf("Observer not ready: %+v", info.Components)
	}
	dir, _, free, err := manager.AcquireCurrent(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer free()
	output, err := managedRunner(root, runtime).CLI(ctx, dir, "--version")
	if err != nil || strings.TrimSpace(string(output)) != version {
		t.Fatalf("installed CLI: %q %v", output, err)
	}
}
