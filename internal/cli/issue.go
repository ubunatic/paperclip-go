package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/issues"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues",
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	RunE:  runIssueCreate,
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	RunE:  runIssueList,
}

var issueGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an issue by ID",
	RunE:  runIssueGet,
}

var (
	flagIssueCompany      string
	flagIssueTitle        string
	flagIssueBody         string
	flagIssueAssignee     string
	flagIssueStatus       string
	flagIssueID           string
	flagIssueListAssignee string
)

func init() {
	issueCreateCmd.Flags().StringVar(&flagIssueCompany, "company", "", "Company shortname (required)")
	issueCreateCmd.Flags().StringVar(&flagIssueTitle, "title", "", "Issue title (required)")
	issueCreateCmd.Flags().StringVar(&flagIssueBody, "body", "", "Issue body")
	issueCreateCmd.Flags().StringVar(&flagIssueAssignee, "assignee", "", "Assignee agent ID")
	_ = issueCreateCmd.MarkFlagRequired("company")
	_ = issueCreateCmd.MarkFlagRequired("title")

	issueListCmd.Flags().StringVar(&flagIssueCompany, "company", "", "Company shortname (required)")
	issueListCmd.Flags().StringVar(&flagIssueStatus, "status", "", "Filter by status (open, in_progress, blocked, done, cancelled)")
	issueListCmd.Flags().StringVar(&flagIssueListAssignee, "assignee", "", "Filter by assignee agent ID")
	_ = issueListCmd.MarkFlagRequired("company")

	issueGetCmd.Flags().StringVar(&flagIssueID, "id", "", "Issue ID (required)")
	_ = issueGetCmd.MarkFlagRequired("id")

	issueCmd.AddCommand(issueCreateCmd, issueListCmd, issueGetCmd)
	rootCmd.AddCommand(issueCmd)
}

func runIssueCreate(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := cmd.Context()

	// Resolve company ID by shortname
	companySvc := companies.New(s)
	companyID, err := resolveCompanyID(ctx, companySvc, flagIssueCompany)
	if err != nil {
		return err
	}

	svc := issues.New(s)
	var assigneeID *string
	if flagIssueAssignee != "" {
		assigneeID = &flagIssueAssignee
	}

	issue, err := svc.Create(ctx, companyID, flagIssueTitle, flagIssueBody, "", "", assigneeID)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(issue)
}

func runIssueList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := cmd.Context()

	// Resolve company ID by shortname
	companySvc := companies.New(s)
	companyID, err := resolveCompanyID(ctx, companySvc, flagIssueCompany)
	if err != nil {
		return err
	}

	svc := issues.New(s)
	var assigneeID *string
	if flagIssueListAssignee != "" {
		assigneeID = &flagIssueListAssignee
	}

	list, err := svc.ListWithFilters(ctx, companyID, flagIssueStatus, assigneeID, false)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"items": list})
}

func runIssueGet(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := cmd.Context()
	svc := issues.New(s)

	issue, err := svc.Get(ctx, flagIssueID)
	if err != nil {
		if errors.Is(err, issues.ErrNotFound) {
			return fmt.Errorf("issue %q not found", flagIssueID)
		}
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(issue)
}
