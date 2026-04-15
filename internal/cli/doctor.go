package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/config"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check paperclip-go installation status",
	Long:  "Verify that paperclip-go is properly initialized and configured.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return doctorRun()
	},
}

func doctorRun() error {
	fmt.Println("Running paperclip-go doctor...")
	fmt.Println()

	// Check config file
	path := config.DefaultPath()
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Printf("❌ Config error: %v\n", err)
		return err
	}

	// Check if config file exists (Load returns defaults if missing)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("⚠️  No config file at %s (using defaults)\n", path)
	} else if err == nil {
		fmt.Printf("✓ Config file exists: %s\n", path)
	} else {
		fmt.Printf("⚠️  Could not stat config file: %v\n", err)
	}

	// Check data directory
	if _, err := os.Stat(cfg.DataDir); os.IsNotExist(err) {
		fmt.Printf("⚠️  Data directory does not exist: %s\n", cfg.DataDir)
	} else if err == nil {
		fmt.Printf("✓ Data directory exists: %s\n", cfg.DataDir)
	} else {
		fmt.Printf("❌ Could not stat data directory: %v\n", err)
		return err
	}

	// Check database file (optional, may not exist yet)
	dbPath := cfg.DBPath()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("ℹ️  Database not yet initialized: %s\n", dbPath)
	} else if err == nil {
		fmt.Printf("✓ Database file exists: %s\n", dbPath)
	} else {
		fmt.Printf("⚠️  Could not stat database file: %v\n", err)
	}

	// Report configuration values
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  ListenAddr: %s\n", cfg.ListenAddr)
	fmt.Printf("  LogLevel: %s\n", cfg.LogLevel)
	fmt.Printf("  LogFormat: %s\n", cfg.LogFormat)
	fmt.Printf("  SkillsDir: %s\n", cfg.SkillsDir)
	fmt.Printf("  UIDir: %s\n", cfg.UIDir)
	fmt.Printf("  DeploymentMode: %s\n", cfg.DeploymentMode)

	// Check if skills directory exists
	skillsDir := cfg.SkillsDir
	if !filepath.IsAbs(skillsDir) {
		// Relative path is relative to CWD
		fmt.Printf("ℹ️  SkillsDir is relative: %s\n", skillsDir)
	}
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		fmt.Printf("⚠️  SkillsDir does not exist: %s\n", skillsDir)
	} else if err == nil {
		fmt.Printf("✓ SkillsDir exists: %s\n", skillsDir)
	}

	// Check if UI directory exists
	uiDir := cfg.UIDir
	if !filepath.IsAbs(uiDir) {
		// Relative path is relative to CWD
		fmt.Printf("ℹ️  UIDir is relative: %s\n", uiDir)
	}
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		fmt.Printf("⚠️  UIDir does not exist: %s\n", uiDir)
	} else if err == nil {
		fmt.Printf("✓ UIDir exists: %s\n", uiDir)
	}

	fmt.Println()
	fmt.Println("✓ paperclip-go doctor completed")
	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
