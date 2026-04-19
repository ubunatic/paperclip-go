package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/config"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactively onboard a new company",
	Long:  "Create a new company via interactive prompts, storing it in the configured database.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return onboardRun()
	},
}

func onboardRun() error {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	s, err := openStoreFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close()

	ctx := context.Background()
	svc := companies.New(s)

	// Use bufio.Scanner for prompts
	scanner := bufio.NewScanner(os.Stdin)

	// Prompt for name
	fmt.Fprint(os.Stdout, "Company name: ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read company name")
	}
	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		return fmt.Errorf("company name is required")
	}

	// Prompt for shortname
	fmt.Fprint(os.Stdout, "Company shortname: ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read company shortname")
	}
	shortname := strings.TrimSpace(scanner.Text())
	if shortname == "" {
		return fmt.Errorf("company shortname is required")
	}

	// Prompt for description (optional)
	fmt.Fprint(os.Stdout, "Company description (optional): ")
	if !scanner.Scan() {
		return fmt.Errorf("failed to read company description")
	}
	description := strings.TrimSpace(scanner.Text())

	// Create the company
	company, err := svc.Create(ctx, name, shortname, description)
	if err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Company created with ID: %s\n", company.ID)

	return nil
}

func init() {
	rootCmd.AddCommand(onboardCmd)
}
