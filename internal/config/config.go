package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for a tally-form-cli operation.
type Config struct {
	API       APIConfig      `yaml:"api"`
	Workspace string         `yaml:"workspace"`
	Defaults  map[string]any `yaml:"defaults"`
	Logo      string         `yaml:"logo"`
	Styles    map[string]any `yaml:"styles"`
}

// APIConfig holds API credentials.
type APIConfig struct {
	Token   string `yaml:"token"`
	BaseURL string `yaml:"base_url"`
}

// Merged holds the final merged configuration ready for use.
type Merged struct {
	Token     string
	BaseURL   string
	Workspace string
	Settings  map[string]any
	Logo      string
	Styles    string // JSON-encoded styles
}

// Load reads and merges configuration from user config, project config, and frontmatter.
func Load(projectConfigPath string, frontmatter map[string]any) (*Merged, error) {
	// Layer 1: User config
	userCfg, err := loadUserConfig()
	if err != nil {
		return nil, fmt.Errorf("user config: %w", err)
	}

	// Layer 2: Project config
	var projCfg Config
	if projectConfigPath != "" {
		data, err := os.ReadFile(projectConfigPath)
		if err != nil {
			return nil, fmt.Errorf("project config %s: %w", projectConfigPath, err)
		}
		if err := yaml.Unmarshal(data, &projCfg); err != nil {
			return nil, fmt.Errorf("parse project config: %w", err)
		}
	}

	// Build merged result
	m := &Merged{
		Token:    userCfg.API.Token,
		BaseURL:  "https://api.tally.so",
		Settings: make(map[string]any),
	}

	if userCfg.API.BaseURL != "" {
		m.BaseURL = userCfg.API.BaseURL
	}
	if userCfg.Workspace != "" {
		m.Workspace = userCfg.Workspace
	}

	// Apply project config
	if projCfg.Workspace != "" {
		m.Workspace = projCfg.Workspace
	}
	for k, v := range projCfg.Defaults {
		m.Settings[k] = v
	}
	if projCfg.Logo != "" {
		m.Logo = projCfg.Logo
	}
	if projCfg.Styles != nil {
		stylesJSON, err := json.Marshal(projCfg.Styles)
		if err == nil {
			m.Styles = string(stylesJSON)
		}
	}

	// Layer 3: Frontmatter overrides
	if frontmatter != nil {
		if ws, ok := frontmatter["workspace"].(string); ok && ws != "" {
			m.Workspace = ws
		}
	}

	// Env var override for token
	if envToken := os.Getenv("TALLY_API_TOKEN"); envToken != "" {
		m.Token = envToken
	}

	return m, nil
}

func loadUserConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &Config{}, nil
	}

	path := filepath.Join(home, ".config", "tally-form-cli", "config.yaml")
	data, err := os.ReadFile(path)
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
