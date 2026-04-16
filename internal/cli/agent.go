package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/companies"
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

	ctx := context.Background()

	// Resolve company ID by shortname
	companySvc := companies.New(s)
	company, err := companySvc.List(ctx)
	if err != nil {
		return err
	}
	var companyID string
	for _, c := range company {
		if c.Shortname == flagAgentCompany {
			companyID = c.ID
			break
		}
	}
	if companyID == "" {
		return fmt.Errorf("company %q not found", flagAgentCompany)
	}

	svc := agents.New(s)
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

	ctx := context.Background()
	svc := agents.New(s)

	var list interface{}
	if flagAgentCompany != "" {
		// Resolve company ID by shortname
		companySvc := companies.New(s)
		companies, err := companySvc.List(ctx)
		if err != nil {
			return err
		}
		var companyID string
		for _, c := range companies {
			if c.Shortname == flagAgentCompany {
				companyID = c.ID
				break
			}
		}
		if companyID == "" {
			return fmt.Errorf("company %q not found", flagAgentCompany)
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
