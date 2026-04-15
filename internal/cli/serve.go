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
	// Load config
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	listenAddr := cfg.ListenAddr

	// Create router and server
	router := api.NewRouter()
	server := &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	// Channel to signal when server starts
	done := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		fmt.Fprintf(os.Stdout, "server listening on %s\n", listenAddr)
		done <- server.ListenAndServe()
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		// Graceful shutdown
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
