package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/domain"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents",
}

var agentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent",
	RunE:  runAgentCreate,
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents",
	RunE:  runAgentList,
}

var (
	flagAgentCompany     string
	flagAgentShortname   string
	flagAgentDisplayName string
	flagAgentRole        string
)

// resolveCompanyID looks up a company by shortname and returns its ID.
func resolveCompanyID(ctx context.Context, svc *companies.Service, shortname string) (string, error) {
	company, err := svc.GetByShortname(ctx, shortname)
	if err != nil {
		if err == companies.ErrNotFound {
			return "", fmt.Errorf("company %q not found", shortname)
		}
		return "", err
	}
	return company.ID, nil
}

func init() {
	agentCreateCmd.Flags().StringVar(&flagAgentCompany, "company", "", "Company shortname (required)")
	agentCreateCmd.Flags().StringVar(&flagAgentShortname, "shortname", "", "Agent shortname (required)")
	agentCreateCmd.Flags().StringVar(&flagAgentDisplayName, "display-name", "", "Agent display name (required)")
	agentCreateCmd.Flags().StringVar(&flagAgentRole, "role", "", "Agent role")
	_ = agentCreateCmd.MarkFlagRequired("company")
	_ = agentCreateCmd.MarkFlagRequired("shortname")
	_ = agentCreateCmd.MarkFlagRequired("display-name")

	agentListCmd.Flags().StringVar(&flagAgentCompany, "company", "", "Filter by company shortname")

	agentCmd.AddCommand(agentCreateCmd, agentListCmd)
	rootCmd.AddCommand(agentCmd)
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := cmd.Context()

	// Resolve company ID by shortname
	companySvc := companies.New(s)
	companyID, err := resolveCompanyID(ctx, companySvc, flagAgentCompany)
	if err != nil {
		return err
	}

	svc := agents.New(s, activity.New(s))
	a, err := svc.Create(ctx, companyID, flagAgentShortname, flagAgentDisplayName, flagAgentRole, nil, "stub")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(a)
}

func runAgentList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := cmd.Context()
	svc := agents.New(s, activity.New(s))

	var list []*domain.Agent
	if flagAgentCompany != "" {
		// Resolve company ID by shortname
		companySvc := companies.New(s)
		companyID, err := resolveCompanyID(ctx, companySvc, flagAgentCompany)
		if err != nil {
			return err
		}
		list, err = svc.ListByCompany(ctx, companyID)
		if err != nil {
			return err
		}
	} else {
		var err error
		list, err = svc.List(ctx)
		if err != nil {
			return err
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"items": list})
}
