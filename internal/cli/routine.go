package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/routines"
)

var routineCmd = &cobra.Command{
	Use:   "routine",
	Short: "Manage routines",
}

var routineCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a routine",
	RunE:  runRoutineCreate,
}

var routineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List routines",
	RunE:  runRoutineList,
}

var (
	flagRoutineCompany string
	flagRoutineAgent   string
	flagRoutineName    string
	flagRoutineCron    string
)

func init() {
	// create flags
	routineCreateCmd.Flags().StringVar(&flagRoutineCompany, "company", "", "Company ID (required)")
	routineCreateCmd.Flags().StringVar(&flagRoutineAgent, "agent", "", "Agent ID (required)")
	routineCreateCmd.Flags().StringVar(&flagRoutineName, "name", "", "Routine name (required)")
	routineCreateCmd.Flags().StringVar(&flagRoutineCron, "cron", "", "Cron expression (required)")
	_ = routineCreateCmd.MarkFlagRequired("company")
	_ = routineCreateCmd.MarkFlagRequired("agent")
	_ = routineCreateCmd.MarkFlagRequired("name")
	_ = routineCreateCmd.MarkFlagRequired("cron")

	// list flags
	routineListCmd.Flags().StringVar(&flagRoutineCompany, "company", "", "Company ID (required)")
	_ = routineListCmd.MarkFlagRequired("company")

	routineCmd.AddCommand(routineCreateCmd, routineListCmd)
	rootCmd.AddCommand(routineCmd)
}

func runRoutineCreate(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := routines.New(s)
	ctx := cmd.Context()

	routine, err := svc.Create(ctx, flagRoutineCompany, flagRoutineAgent, flagRoutineName, flagRoutineCron)
	if err != nil {
		if err == routines.ErrInvalidCron {
			fmt.Fprintf(os.Stderr, "error: invalid cron expression: %s\n", flagRoutineCron)
		} else if err == routines.ErrNameConflict {
			fmt.Fprintf(os.Stderr, "error: routine name already exists: %s\n", flagRoutineName)
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		return err
	}

	data, _ := json.MarshalIndent(routine, "", "  ")
	fmt.Println(string(data))
	return nil
}

func runRoutineList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	svc := routines.New(s)
	list, err := svc.ListByCompany(cmd.Context(), flagRoutineCompany)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCRON\tENABLED\tLAST RUN\tAGENT")

	for _, r := range list {
		lastRun := "never"
		if r.LastRunAt != nil {
			lastRun = r.LastRunAt.Format("2006-01-02 15:04:05")
		}
		enabled := "yes"
		if !r.Enabled {
			enabled = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.Name, r.CronExpr, enabled, lastRun, r.AgentID)
	}
	w.Flush()

	return nil
}
