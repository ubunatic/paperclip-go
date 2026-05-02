// Package config loads and validates the paperclip-go YAML configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all runtime settings for paperclip-go.
type Config struct {
	DataDir        string `yaml:"data_dir"`
	ListenAddr     string `yaml:"listen_addr"`
	LogLevel       string `yaml:"log_level"`
	LogFormat      string `yaml:"log_format"`
	SkillsDir      string `yaml:"skills_dir"`
	UIDir          string `yaml:"ui_dir"`
	DeploymentMode string `yaml:"deployment_mode"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		DataDir:        filepath.Join(home, ".paperclip-go"),
		ListenAddr:     "127.0.0.1:3200",
		LogLevel:       "info",
		LogFormat:      "text",
		SkillsDir:      "./skills",
		UIDir:          "./ui/dist",
		DeploymentMode: "local_trusted",
	}
}

// DefaultPath returns the default config file path (~/.paperclip-go/config.yaml).
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".paperclip-go", "config.yaml")
}

// Load reads a Config from path, falling back to defaults for any missing fields.
// If the PAPERCLIP_GO_CONFIG env var is set it overrides path.
// A missing file is not an error — defaults are returned instead.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if envPath := os.Getenv("PAPERCLIP_GO_CONFIG"); envPath != "" {
		path = envPath
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}

// Write marshals the Config to YAML and writes it to path, creating parent
// directories as needed.
func (c *Config) Write(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// DBPath returns the SQLite database file path derived from DataDir.
func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "paperclip.db")
}

// BackupsDir returns the backups directory path derived from DataDir.
func (c *Config) BackupsDir() string {
	return filepath.Join(c.DataDir, "backups")
}
