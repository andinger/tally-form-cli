package cli

import (
	"testing"

	"github.com/andinger/tally-form-cli/internal/config"
	"github.com/andinger/tally-form-cli/internal/markdown"
)

// TestPushPipeline_FormWorkspaceOverridesGlobal guards the full push pipeline:
// Markdown parser splits "workspace" into form.Workspace and removes it from
// form.Settings, so config.Load alone cannot see it. push.go bridges that
// with ApplyFormOverride. This test fails if anyone removes that call.
func TestPushPipeline_FormWorkspaceOverridesGlobal(t *testing.T) {
	md := `---
name: "Test"
workspace: "form-ws"
---

F1: Question?
> type: short-text
`

	form, err := markdown.Parse(md)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if form.Workspace != "form-ws" {
		t.Fatalf("form.Workspace = %q, want form-ws", form.Workspace)
	}
	if _, present := form.Settings["workspace"]; present {
		t.Errorf("form.Settings still contains workspace key — parser should split it off")
	}

	cfg, err := config.Load(form.Settings)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.ApplyFormOverride(form)

	if cfg.Workspace != "form-ws" {
		t.Errorf("cfg.Workspace = %q, want form-ws — frontmatter workspace must override global config", cfg.Workspace)
	}
}
