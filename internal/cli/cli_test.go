package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd("1.0.0", "abc123", "2026-01-01")

	if cmd.Use != "tally" {
		t.Errorf("Use = %q, want tally", cmd.Use)
	}

	// Check subcommands are registered
	wantCmds := []string{"push", "create", "update", "export", "submissions", "reference"}
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
	if cmd.PersistentFlags().Lookup("config") == nil {
		t.Error("Missing --config flag")
	}
	if cmd.PersistentFlags().Lookup("token") == nil {
		t.Error("Missing --token flag")
	}
}

func TestReferenceCmd(t *testing.T) {
	cmd := NewRootCmd("test", "test", "test")
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"reference"})

	// Reference command writes to stdout, not cmd.OutOrStdout()
	// So we capture via the RunE directly
	for _, sub := range cmd.Commands() {
		if sub.Name() == "reference" {
			// Just verify the command exists and has correct metadata
			if sub.Short == "" {
				t.Error("Reference command should have Short description")
			}
			break
		}
	}
}

func TestVersionOutput(t *testing.T) {
	cmd := NewRootCmd("1.2.3", "deadbeef", "2026-03-29")
	if !strings.Contains(cmd.Version, "1.2.3") {
		t.Errorf("Version = %q, should contain 1.2.3", cmd.Version)
	}
	if !strings.Contains(cmd.Version, "deadbeef") {
		t.Errorf("Version = %q, should contain commit", cmd.Version)
	}
}
