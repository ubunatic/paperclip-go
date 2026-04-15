package cli

import (
	"fmt"
	"os"

	"github.com/ubunatic/paperclip-go/internal/config"
	"github.com/ubunatic/paperclip-go/internal/store"
)

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
