package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuKeepsInstallSeparateFromStart(t *testing.T) {
	m := menu{options: menuOptions(false, nil), action: -1}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(menu).action != installAction {
		t.Fatal("first-time Enter must select Install")
	}
	m.options = menuOptions(true, nil)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(menu).action != startAction {
		t.Fatal("installed Enter must select Start")
	}
}

func TestManagedRuntimeFromChinesePath(t *testing.T) {
	if os.Getenv("ACCODING_TEST_NODE_DOWNLOAD") != "1" {
		t.Skip("requires downloading the pinned Node.js distribution")
	}
	root := filepath.Join(t.TempDir(), "中文 测试目录")
	r, err := installNode(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.node, root) {
		t.Fatalf("expected managed Node at %s, got %s", root, r.node)
	}
	out, err := exec.Command(r.node, r.npm, "--version").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		t.Fatalf("npm failed after Node installation: %v %s", err, out)
	}
	next, err := findNode(root)
	if err != nil || next != r {
		t.Fatalf("could not rediscover Node.js without restarting: %v", err)
	}
	t.Run("v2-real-npm-local-registry", func(t *testing.T) {
		exerciseLocalRegistry(t, r, filepath.Join(root, "npm integration"))
	})
	if version := os.Getenv("ACCODING_TEST_PUBLISHED_VERSION"); version != "" {
		t.Run("v2-published-registry", func(t *testing.T) {
			observer := os.Getenv("ACCODING_TEST_PUBLISHED_OBSERVER")
			if observer == "" {
				t.Fatal("missing expected published Observer version")
			}
			exercisePublishedRegistry(t, r, filepath.Join(root, "published installation"), version, observer)
		})
	}
}

func TestInstalledMenuOnlyShowsAvailableActions(t *testing.T) {
	options := menuOptions(true, []string{"C Exam Kit: 0.6.6 → 0.6.7"})
	if len(options) != 4 || options[1].label != "更新 C Exam Kit" {
		t.Fatalf("unexpected options: %v", options)
	}
	if len(menuOptions(true, nil)) != 3 || len(menuOptions(false, nil)) != 2 {
		t.Fatal("unexpected menu for an installation without updates")
	}
	if !newerVersion("0.6.10", "0.6.9") || newerVersion("0.6.9", "0.6.10") {
		t.Fatal("versions must be compared numerically")
	}
}


func TestLegacyScopeMigration(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "packages", "release-old")
	for name, file := range map[string]string{cliPackage: "dist/accoding.mjs", presetPackage: "preset/manifest.json"} {
		old := legacyPackage(name)
		base := packagePath(dir, old)
		if err := os.MkdirAll(filepath.Dir(filepath.Join(base, filepath.FromSlash(file))), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "package.json"), []byte(`{"name":"`+old+`","version":"1.2.3"}`), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, filepath.FromSlash(file)), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "current"), []byte("release-old\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if readInstallation(root) != dir || !legacyInstallation(dir) || installedVersion(dir, cliPackage) != "1.2.3" || activePackage(dir, presetPackage) != legacyPackage(presetPackage) {
		t.Fatal("legacy pair must remain usable offline")
	}
	for _, release := range []updateResult{{}, {cli: "1.2.4", preset: "1.2.2"}} {
		if len(availableUpdates(dir, release)) != 0 {
			t.Fatal("must not offer incomplete or downgrading migration")
		}
	}
	if len(availableUpdates(dir, updateResult{cli: "1.2.3", preset: "1.2.3"})) != 1 {
		t.Fatal("same-version namespace migration must be offered")
	}
}

func TestRuntimeRenameRetry(t *testing.T) {
	for _, tc := range []struct {
		name string
		err error
		windows bool
		failures int
		wantCalls int
		wantError bool
	}{
		{"access denied then succeeds", os.ErrPermission, true, 2, 3, false},
		{"sharing violation", syscall.Errno(32), true, 1, 2, false},
		{"lock violation", syscall.Errno(33), true, 1, 2, false},
		{"permanent access denied", os.ErrPermission, true, 99, 8, true},
		{"unexpected error", errors.New("disk error"), true, 1, 1, true},
		{"non-Windows", os.ErrPermission, false, 1, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, sleeps := 0, 0
			err := retryRuntimeRename(func() error {
				calls++
				if calls <= tc.failures {
					return &os.LinkError{Op: "rename", Old: "stage", New: "runtime", Err: tc.err}
				}
				return nil
			}, func(d time.Duration) {
				if d <= 0 || d > time.Second {
					t.Fatalf("invalid retry delay: %v", d)
				}
				sleeps++
			}, tc.windows)
			if calls != tc.wantCalls || sleeps != calls-1 || (err != nil) != tc.wantError {
				t.Fatalf("calls=%d sleeps=%d err=%v", calls, sleeps, err)
			}
			if tc.wantError && !errors.Is(err, tc.err) {
				t.Fatalf("original error lost: %v", err)
			}
		})
	}
}
