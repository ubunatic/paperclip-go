package cli

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/api"
	"github.com/ubunatic/paperclip-go/internal/config"
	"github.com/ubunatic/paperclip-go/internal/skills"
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

	s, err := openStoreFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close()

	skillsList, err := skills.Load(cfg.SkillsDir)
	if err != nil {
		log.Printf("warning: failed to load skills: %v", err)
	}

	router := api.NewRouter(s, skillsList)
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
	defer signal.Stop(sigChan)

	select {
	case <-sigChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		return nil
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
