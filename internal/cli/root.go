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

// Execute runs the root command. Call this from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
