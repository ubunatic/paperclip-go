package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ListenAddr != "127.0.0.1:3200" {
		t.Errorf("expected ListenAddr 127.0.0.1:3200, got %s", cfg.ListenAddr)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel info, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("expected LogFormat text, got %s", cfg.LogFormat)
	}
	if cfg.SkillsDir != "./skills" {
		t.Errorf("expected SkillsDir ./skills, got %s", cfg.SkillsDir)
	}
	if cfg.UIDir != "./ui/dist" {
		t.Errorf("expected UIDir ./ui/dist, got %s", cfg.UIDir)
	}
	if cfg.DeploymentMode != "local_trusted" {
		t.Errorf("expected DeploymentMode local_trusted, got %s", cfg.DeploymentMode)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to determine user home directory: %v", err)
	}
	if !filepath.HasPrefix(cfg.DataDir, home) {
		t.Errorf("expected DataDir under home directory %s, got %s", home, cfg.DataDir)
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".paperclip-go", "config.yaml")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Errorf("expected no error for missing file, got %v", err)
	}
	if cfg == nil {
		t.Errorf("expected default config for missing file")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel, got %s", cfg.LogLevel)
	}
}

func TestLoadValidFile(t *testing.T) {
	// Create a temporary directory and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a test config
	testCfg := &Config{
		DataDir:        filepath.Join(tmpDir, "data"),
		ListenAddr:     "0.0.0.0:8080",
		LogLevel:       "debug",
		LogFormat:      "json",
		SkillsDir:      "/custom/skills",
		UIDir:          "/custom/ui",
		DeploymentMode: "production",
	}

	if err := testCfg.Write(configPath); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load the config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.ListenAddr != "0.0.0.0:8080" {
		t.Errorf("expected ListenAddr 0.0.0.0:8080, got %s", cfg.ListenAddr)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel debug, got %s", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("expected LogFormat json, got %s", cfg.LogFormat)
	}
	if cfg.SkillsDir != "/custom/skills" {
		t.Errorf("expected SkillsDir /custom/skills, got %s", cfg.SkillsDir)
	}
	if cfg.UIDir != "/custom/ui" {
		t.Errorf("expected UIDir /custom/ui, got %s", cfg.UIDir)
	}
	if cfg.DeploymentMode != "production" {
		t.Errorf("expected DeploymentMode production, got %s", cfg.DeploymentMode)
	}
}

func TestLoadEnvironmentOverride(t *testing.T) {
	// Create two temporary config files
	tmpDir := t.TempDir()
	defaultPath := filepath.Join(tmpDir, "default.yaml")
	overridePath := filepath.Join(tmpDir, "override.yaml")

	// Write default config
	defaultCfg := &Config{LogLevel: "info"}
	if err := defaultCfg.Write(defaultPath); err != nil {
		t.Fatalf("failed to write default config: %v", err)
	}

	// Write override config
	overrideCfg := &Config{LogLevel: "debug"}
	if err := overrideCfg.Write(overridePath); err != nil {
		t.Fatalf("failed to write override config: %v", err)
	}

	// Set environment variable
	oldEnv := os.Getenv("PAPERCLIP_GO_CONFIG")
	defer os.Setenv("PAPERCLIP_GO_CONFIG", oldEnv)
	os.Setenv("PAPERCLIP_GO_CONFIG", overridePath)

	// Load should use override path from env var
	cfg, err := Load(defaultPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel debug from env override, got %s", cfg.LogLevel)
	}
}

func TestWrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yaml")

	cfg := &Config{
		DataDir:        filepath.Join(tmpDir, "data"),
		ListenAddr:     "0.0.0.0:9000",
		LogLevel:       "warn",
		LogFormat:      "json",
		SkillsDir:      "/skills",
		UIDir:          "/ui",
		DeploymentMode: "staging",
	}

	if err := cfg.Write(configPath); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(configPath)); err != nil {
		t.Errorf("config directory not created: %v", err)
	}

	// Load and verify content
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load written config: %v", err)
	}

	if loaded.ListenAddr != "0.0.0.0:9000" {
		t.Errorf("expected ListenAddr 0.0.0.0:9000, got %s", loaded.ListenAddr)
	}
	if loaded.LogLevel != "warn" {
		t.Errorf("expected LogLevel warn, got %s", loaded.LogLevel)
	}
}

func TestDBPath(t *testing.T) {
	cfg := &Config{
		DataDir: "/var/data",
	}
	expected := filepath.Join("/var/data", "paperclip.db")
	if cfg.DBPath() != expected {
		t.Errorf("expected %s, got %s", expected, cfg.DBPath())
	}
}
