// Package cli defines the root cobra command and Execute entry point.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "paperclip-go",
	Short: "Paperclip control-plane — Go edition",
	Long: `paperclip-go is a Go reimplementation of the Paperclip AI agent
control plane. Run 'paperclip-go help' for available commands.`,
	// No-op run so the binary exits 0 when called without a sub-command.
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), "paperclip-go — nothing to do yet. Try --help.")
	},
}

// Execute runs the root command with the given version. Call this from main.
func Execute(version string) {
	// Store version globally for access by handlers
	globalVersion = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// globalVersion holds the build version.
var globalVersion = "dev"

// GetVersion returns the current build version.
func GetVersion() string {
	return globalVersion
}
