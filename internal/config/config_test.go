package config

import (
	"testing"

	"github.com/andinger/tally-form-cli/internal/model"
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

func TestApplyFormOverride_Workspace(t *testing.T) {
	m := &Merged{Workspace: "global-ws"}
	m.ApplyFormOverride(&model.Form{Workspace: "fm-ws"})
	if m.Workspace != "fm-ws" {
		t.Errorf("Workspace = %q, want fm-ws (form override should win over global)", m.Workspace)
	}
}

func TestApplyFormOverride_EmptyWorkspaceKeepsGlobal(t *testing.T) {
	m := &Merged{Workspace: "global-ws"}
	m.ApplyFormOverride(&model.Form{Workspace: ""})
	if m.Workspace != "global-ws" {
		t.Errorf("Workspace = %q, want global-ws (empty form workspace must not clobber global)", m.Workspace)
	}
}

func TestApplyFormOverride_NilFormIsSafe(t *testing.T) {
	m := &Merged{Workspace: "global-ws"}
	m.ApplyFormOverride(nil)
	if m.Workspace != "global-ws" {
		t.Errorf("Workspace = %q, want global-ws (nil form must be a no-op)", m.Workspace)
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
