package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize paperclip-go configuration and data directory",
	Long:  "Create ~/.paperclip-go/config.yaml with sensible defaults and create the data directory.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return initRun()
	},
}

func initRun() error {
	cfg := config.DefaultConfig()
	path := config.DefaultPath()

	// Write config file
	if err := cfg.Write(path); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Wrote config to %s\n", path)

	// Create data directory
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Created data directory: %s\n", cfg.DataDir)

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
