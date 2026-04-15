package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/api"
	"github.com/ubunatic/paperclip-go/internal/config"
	"github.com/ubunatic/paperclip-go/internal/store"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the paperclip-go server",
	Long:  "Start the HTTP server listening on 127.0.0.1:3200.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serveRun()
	},
}

func serveRun() error {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Ensure data directory exists and open the database.
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	s, err := store.Open(cfg.DBPath())
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close()

	router := api.NewRouter(s)
	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: router,
	}

	done := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stdout, "server listening on %s\n", cfg.ListenAddr)
		done <- server.ListenAndServe()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		return nil
	case err := <-done:
		return err
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
