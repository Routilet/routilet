package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestV2SettingsAndMenu(t *testing.T) {
	root := t.TempDir()
	s, err := readV2Settings(root)
	if err != nil || s.Channel != "latest" {
		t.Fatal(s, err)
	}
	if err = writeV2Settings(root, v2Settings{1, "next"}); err != nil {
		t.Fatal(err)
	}
	s, err = readV2Settings(root)
	if err != nil || s.Channel != "next" {
		t.Fatal(s, err)
	}
	if err = writeV2Settings(root, v2Settings{1, "latest"}); err != nil {
		t.Fatal(err)
	}
	s, err = readV2Settings(root)
	if err != nil || s.Channel != "latest" {
		t.Fatal(s, err)
	}
	file := filepath.Join(root, "v2", "settings.json")
	if err := os.WriteFile(file, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readV2Settings(root); err == nil {
		t.Fatal("invalid settings accepted")
	}
	raw, _ := os.ReadFile(file)
	if string(raw) != "invalid" {
		t.Fatal("settings modified on read")
	}
	m := v2Menu{title: "更新 Accoding", options: []v2Option{{"cancel", "取消"}, {"apply", "确认更新"}}}
	if !strings.Contains(m.View(), "› 取消") {
		t.Fatal("confirmation should default to cancel")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, _ = next.(v2Menu).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(v2Menu).choice != "apply" {
		t.Fatal("confirmation navigation failed")
	}
}
