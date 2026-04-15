package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/config"
	"github.com/ubunatic/paperclip-go/internal/store"
)

var companyCmd = &cobra.Command{
	Use:   "company",
	Short: "Manage companies",
}

var companyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new company",
	RunE:  runCompanyCreate,
}

var companyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all companies",
	RunE:  runCompanyList,
}

var (
	flagCompanyName        string
	flagCompanyShortname   string
	flagCompanyDescription string
)

func init() {
	companyCreateCmd.Flags().StringVar(&flagCompanyName, "name", "", "Company name (required)")
	companyCreateCmd.Flags().StringVar(&flagCompanyShortname, "shortname", "", "Company shortname (required)")
	companyCreateCmd.Flags().StringVar(&flagCompanyDescription, "description", "", "Company description")

	companyCmd.AddCommand(companyCreateCmd, companyListCmd)
	rootCmd.AddCommand(companyCmd)
}

// openStore loads config and opens the SQLite store, creating the data dir if needed.
func openStore() (*store.Store, error) {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	return store.Open(cfg.DBPath())
}

func runCompanyCreate(cmd *cobra.Command, args []string) error {
	if flagCompanyName == "" || flagCompanyShortname == "" {
		return fmt.Errorf("--name and --shortname are required")
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := companies.New(s)
	c, err := svc.Create(cmd.Context(), flagCompanyName, flagCompanyShortname, flagCompanyDescription)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}

func runCompanyList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := companies.New(s)
	list, err := svc.List(cmd.Context())
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"items": list})
}
