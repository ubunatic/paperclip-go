package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/approvals"
)

var approvalCmd = &cobra.Command{
	Use:   "approval",
	Short: "Manage approvals",
}

var approvalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List approvals for a company",
	RunE:  runApprovalList,
}

var approvalGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get approval details",
	Args:  cobra.ExactArgs(1),
	RunE:  runApprovalGet,
}

var (
	flagApprovalCompany string
)

func init() {
	approvalListCmd.Flags().StringVar(&flagApprovalCompany, "company", "", "Company ID (required)")
	_ = approvalListCmd.MarkFlagRequired("company")

	approvalCmd.AddCommand(approvalListCmd, approvalGetCmd)
	rootCmd.AddCommand(approvalCmd)
}

func runApprovalList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := approvals.New(s)
	list, err := svc.ListByCompany(cmd.Context(), flagApprovalCompany)
	if err != nil {
		return err
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tKIND\tSTATUS\tCREATED AT")

	for _, a := range list {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.Kind, a.Status, a.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	w.Flush()

	return nil
}

func runApprovalGet(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := approvals.New(s)
	approval, err := svc.GetByID(cmd.Context(), args[0])
	if err != nil {
		if err == approvals.ErrNotFound {
			fmt.Fprintf(os.Stderr, "approval not found: %s\n", args[0])
			return err
		}
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(approval)
}
