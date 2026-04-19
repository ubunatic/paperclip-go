package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/config"
	"gopkg.in/yaml.v3"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Show the current paperclip-go configuration",
	Long:  "Display the configuration path and the current YAML configuration.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return configureRun()
	},
}

func configureRun() error {
	// Resolve effective config path (respecting env var override)
	path := os.Getenv("PAPERCLIP_GO_CONFIG")
	if path == "" {
		path = config.DefaultPath()
	}

	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Configuration path: %s\n\n", path)

	// Marshal and print the YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Current configuration:\n")
	fmt.Fprintf(os.Stdout, "%s", string(data))

	return nil
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
