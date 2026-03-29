package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	m, err := Load(nil)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if m.BaseURL != "https://api.tally.so" {
		t.Errorf("BaseURL = %q", m.BaseURL)
	}
}

func TestLoadFrontmatterOverride(t *testing.T) {
	fm := map[string]any{
		"workspace": "fm-ws",
	}
	m, err := Load(fm)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	// Frontmatter workspace should override global (which may be empty in test)
	if m.Workspace != "fm-ws" {
		t.Errorf("Workspace = %q, want fm-ws", m.Workspace)
	}
}

func TestAPITokenFromEnv(t *testing.T) {
	t.Setenv("TALLY_API_TOKEN", "env-token-123")
	m, err := Load(nil)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if m.Token != "env-token-123" {
		t.Errorf("Token = %q, want env-token-123", m.Token)
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath returned empty string")
	}
}
