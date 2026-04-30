package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/andinger/tally-form-cli/internal/model"
)

// Config holds the global configuration from ~/.config/tally/config.yaml.
type Config struct {
	API                   APIConfig `yaml:"api"`
	Workspace             string    `yaml:"workspace"`
	Logo                  string    `yaml:"logo"`
	Cover                 string    `yaml:"cover"`
	PrimaryColor          string    `yaml:"primary_color"`
	Password              string    `yaml:"password"`
	Language              string    `yaml:"language"`
	HasProgressBar        *bool     `yaml:"has_progress_bar"`
	HasPartialSubmissions *bool     `yaml:"has_partial_submissions"`
	SaveForLater          *bool     `yaml:"save_for_later"`
}

// APIConfig holds API credentials.
type APIConfig struct {
	Token   string `yaml:"token"`
	BaseURL string `yaml:"base_url"`
}

// Merged holds the final merged configuration ready for use.
type Merged struct {
	Token        string
	BaseURL      string
	Workspace    string
	Logo         string
	Cover        string
	PrimaryColor string
	Password     string
	Language     string
	Settings     map[string]any
}

// Load reads and merges configuration from global config and inline frontmatter
// settings (the part of the YAML frontmatter that is not split off into the
// dedicated Form fields like workspace, password, name, form_id).
//
// To apply form-level overrides like Workspace, call ApplyFormOverride on the
// returned *Merged.
//
// Merge order: Global config < Frontmatter inline settings < Environment variables
func Load(frontmatter map[string]any) (*Merged, error) {
	userCfg, err := loadUserConfig()
	if err != nil {
		return nil, fmt.Errorf("user config: %w", err)
	}

	m := &Merged{
		Token:        userCfg.API.Token,
		BaseURL:      "https://api.tally.so",
		Workspace:    userCfg.Workspace,
		Logo:         userCfg.Logo,
		Cover:        userCfg.Cover,
		PrimaryColor: userCfg.PrimaryColor,
		Password:     userCfg.Password,
		Language:     userCfg.Language,
		Settings:     make(map[string]any),
	}

	if userCfg.API.BaseURL != "" {
		m.BaseURL = userCfg.API.BaseURL
	}

	// Apply form settings from global config
	if userCfg.Language != "" {
		m.Settings["language"] = userCfg.Language
	}
	if userCfg.HasProgressBar != nil {
		m.Settings["hasProgressBar"] = *userCfg.HasProgressBar
	}
	if userCfg.HasPartialSubmissions != nil {
		m.Settings["hasPartialSubmissions"] = *userCfg.HasPartialSubmissions
	}
	if userCfg.SaveForLater != nil {
		m.Settings["saveForLater"] = *userCfg.SaveForLater
	}

	// Env var override for token
	if envToken := os.Getenv("TALLY_API_TOKEN"); envToken != "" {
		m.Token = envToken
	}

	return m, nil
}

// ApplyFormOverride applies form-level overrides from the parsed Form to the
// merged configuration. The markdown parser splits dedicated frontmatter
// fields (workspace, name, form_id, password) into typed Form fields and
// removes them from Form.Settings — so Load alone cannot see them. This
// method bridges that gap.
//
// Currently only Workspace is overridable per form; extend here when more
// form-level overrides are added.
func (m *Merged) ApplyFormOverride(form *model.Form) {
	if form == nil {
		return
	}
	if form.Workspace != "" {
		m.Workspace = form.Workspace
	}
}

// ConfigPath returns the path to the global config file.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.config/tally/config.yaml"
	}
	return filepath.Join(home, ".config", "tally", "config.yaml")
}

func loadUserConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &Config{}, nil
	}

	path := filepath.Join(home, ".config", "tally", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil && os.IsNotExist(err) {
		// Fallback to legacy path
		legacyPath := filepath.Join(home, ".config", "tally-form-cli", "config.yaml")
		data, err = os.ReadFile(legacyPath)
		if err == nil {
			fmt.Fprintf(os.Stderr, "warning: config at %s is deprecated, move to %s\n", legacyPath, path)
			path = legacyPath
		}
	}
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}
