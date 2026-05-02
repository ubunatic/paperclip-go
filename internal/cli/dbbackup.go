package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ubunatic/paperclip-go/internal/config"
	"github.com/ubunatic/paperclip-go/internal/store"
)

var dbBackupCmd = &cobra.Command{
	Use:   "db:backup",
	Short: "Create a backup of the SQLite database",
	RunE:  runDBBackup,
}

var flagBackupOut string

func init() {
	dbBackupCmd.Flags().StringVar(&flagBackupOut, "out", "", "Optional destination path for backup file")
	rootCmd.AddCommand(dbBackupCmd)
}

// backupDB performs the core backup operation using an open store and running VACUUM INTO.
func backupDB(ctx context.Context, s *store.Store, backupPath string) error {
	// Validate that backupPath does not contain single-quote characters
	if strings.Contains(backupPath, "'") {
		return fmt.Errorf("backup path must not contain single-quote characters: %s", backupPath)
	}

	// Run VACUUM INTO to create the backup
	if _, err := s.DB.ExecContext(ctx, fmt.Sprintf(`VACUUM INTO '%s'`, backupPath)); err != nil {
		return fmt.Errorf("running VACUUM INTO: %w", err)
	}

	// Set backup file permissions to 0o600 (read/write for owner only)
	if err := os.Chmod(backupPath, 0o600); err != nil {
		return fmt.Errorf("setting backup file permissions: %w", err)
	}

	// Verify the backup file was created and is readable
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not created or not readable: %w", err)
	}

	return nil
}

// runDBBackup resolves backup path, creates the backups directory, and calls backupDB.
func runDBBackup(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Open the store
	s, err := openStoreFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer s.Close()

	// Determine backup destination path
	var backupPath string
	if flagBackupOut != "" {
		// Use custom output path
		backupPath = flagBackupOut
		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
			return fmt.Errorf("creating backup directory: %w", err)
		}
	} else {
		// Use default: backups/<timestamp>.db
		backupsDir := cfg.BackupsDir()
		if err := os.MkdirAll(backupsDir, 0o755); err != nil {
			return fmt.Errorf("creating backups directory: %w", err)
		}
		timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
		backupPath = filepath.Join(backupsDir, timestamp+".db")
	}

	// Perform the backup
	if err := backupDB(ctx, s, backupPath); err != nil {
		return err
	}

	// Print the backup path to stdout
	fmt.Fprintf(os.Stdout, "%s\n", backupPath)

	return nil
}
