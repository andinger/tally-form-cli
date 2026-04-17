package cli

import (
	"strings"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd("1.0.0", "abc123", "2026-01-01")

	if cmd.Use != "tally" {
		t.Errorf("Use = %q, want tally", cmd.Use)
	}

	// Check subcommands are registered
	wantCmds := []string{"push", "pull", "delete", "diff", "submissions", "prepare", "config", "reference"}
	for _, name := range wantCmds {
		found := false
		for _, sub := range cmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing subcommand %q", name)
		}
	}

	// Check global flags
	if cmd.PersistentFlags().Lookup("token") == nil {
		t.Error("Missing --token flag")
	}

	// --config should NOT exist (removed)
	if cmd.PersistentFlags().Lookup("config") != nil {
		t.Error("--config flag should be removed")
	}
}

func TestVersionOutput(t *testing.T) {
	cmd := NewRootCmd("1.2.3", "deadbeef", "2026-03-29")
	if !strings.Contains(cmd.Version, "1.2.3") {
		t.Errorf("Version = %q, should contain 1.2.3", cmd.Version)
	}
}
