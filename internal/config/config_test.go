package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "tally-config.yaml")
	err := os.WriteFile(cfgPath, []byte(`
defaults:
  language: "de"
  hasProgressBar: true

logo: "https://example.com/logo.svg"

styles:
  theme: "CUSTOM"
  color:
    background: "#ffffff"
    accent: "#A219B1"
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	m, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if m.Settings["language"] != "de" {
		t.Errorf("language = %v", m.Settings["language"])
	}
	if m.Settings["hasProgressBar"] != true {
		t.Errorf("hasProgressBar = %v", m.Settings["hasProgressBar"])
	}
	if m.Logo != "https://example.com/logo.svg" {
		t.Errorf("Logo = %q", m.Logo)
	}
	if m.Styles == "" {
		t.Error("Styles is empty")
	}
}

func TestMergeOrder(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "tally-config.yaml")
	err := os.WriteFile(cfgPath, []byte(`
workspace: "project-ws"
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Frontmatter should override project config
	fm := map[string]any{
		"workspace": "frontmatter-ws",
	}

	m, err := Load(cfgPath, fm)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if m.Workspace != "frontmatter-ws" {
		t.Errorf("Workspace = %q, want frontmatter-ws", m.Workspace)
	}
}

func TestMissingConfigFiles(t *testing.T) {
	m, err := Load("", nil)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if m.BaseURL != "https://api.tally.so" {
		t.Errorf("BaseURL = %q", m.BaseURL)
	}
}

func TestAPITokenFromEnv(t *testing.T) {
	t.Setenv("TALLY_API_TOKEN", "env-token-123")
	m, err := Load("", nil)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if m.Token != "env-token-123" {
		t.Errorf("Token = %q, want env-token-123", m.Token)
	}
}
