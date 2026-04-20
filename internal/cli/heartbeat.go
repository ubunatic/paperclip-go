package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/activity"
	"github.com/ubunatic/paperclip-go/internal/agents"
	"github.com/ubunatic/paperclip-go/internal/comments"
	"github.com/ubunatic/paperclip-go/internal/heartbeat"
	"github.com/ubunatic/paperclip-go/internal/issues"
)

var heartbeatCmd = &cobra.Command{
	Use:   "heartbeat",
	Short: "Manage heartbeat runs",
}

var heartbeatRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a heartbeat for an agent",
	RunE:  runHeartbeatRun,
}

var (
	flagHeartbeatAgent string
)

func init() {
	heartbeatRunCmd.Flags().StringVar(&flagHeartbeatAgent, "agent", "", "Agent ID (required)")
	_ = heartbeatRunCmd.MarkFlagRequired("agent")

	heartbeatCmd.AddCommand(heartbeatRunCmd)
	rootCmd.AddCommand(heartbeatCmd)
}

func runHeartbeatRun(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := cmd.Context()

	// Initialize services
	actLog := activity.New(s)
	agentSvc := agents.New(s, actLog)
	issueSvc := issues.New(s)
	commentSvc := comments.New(s)
	registry := heartbeat.NewDefaultRegistry()
	runner := heartbeat.New(s, agentSvc, issueSvc, commentSvc, actLog, registry)

	// Run the heartbeat
	run, err := runner.Run(ctx, flagHeartbeatAgent)
	if err != nil {
		return fmt.Errorf("heartbeat run failed: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(run)
}
