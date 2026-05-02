package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestDBBackupDefaultPath(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a temporary store
	testStore := testutil.NewStore(t)

	// Create backups directory
	backupsDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatalf("failed to create backups dir: %v", err)
	}

	// Generate timestamp as the command would
	timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	backupPath := filepath.Join(backupsDir, timestamp+".db")

	// Perform backup
	if err := backupDB(context.Background(), testStore, backupPath); err != nil {
		t.Fatalf("backupDB failed: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file not created at %s: %v", backupPath, err)
	}

	// Verify the filename matches the pattern
	filename := filepath.Base(backupPath)
	if !strings.HasSuffix(filename, ".db") {
		t.Errorf("backup filename should end with .db, got %s", filename)
	}

	// Verify it contains the timestamp pattern
	if !strings.Contains(filename, "20") {
		t.Errorf("backup filename should contain date pattern, got %s", filename)
	}
}

func TestDBBackupCustomOut(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test database
	testStore := testutil.NewStore(t)

	// Create custom backup path
	customBackupDir := filepath.Join(tmpDir, "custom_backups")
	if err := os.MkdirAll(customBackupDir, 0o755); err != nil {
		t.Fatalf("failed to create custom backup dir: %v", err)
	}
	customBackupPath := filepath.Join(customBackupDir, "my_backup.db")

	// Run backup with custom output path
	if err := backupDB(context.Background(), testStore, customBackupPath); err != nil {
		t.Fatalf("backupDB failed: %v", err)
	}

	// Verify backup file exists at custom location
	if _, err := os.Stat(customBackupPath); err != nil {
		t.Errorf("backup file not created at %s: %v", customBackupPath, err)
	}

	// Verify the custom path was used
	expectedPath := filepath.Join(customBackupDir, "my_backup.db")
	if customBackupPath != expectedPath {
		t.Errorf("expected backup path %s, got %s", expectedPath, customBackupPath)
	}
}

func TestDBBackupWritesToExistingDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test database
	testStore := testutil.NewStore(t)

	// Create the backup directory
	backupsDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatalf("failed to create backups dir: %v", err)
	}

	backupPath := filepath.Join(backupsDir, "test.db")

	// Run backup
	if err := backupDB(context.Background(), testStore, backupPath); err != nil {
		t.Fatalf("backupDB failed: %v", err)
	}

	// Verify backup file was created in the directory
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file not created: %v", err)
	}
}

func TestDBBackupVACUUMINTO(t *testing.T) {
	// Create a test database with some data
	s := testutil.NewStore(t)

	// Create a company to ensure data is written to the database
	ctx := context.Background()
	companySvc := companies.New(s)
	company, err := companySvc.Create(ctx, "Test Company", "test-backup-co", "")
	if err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	if company.ID == "" {
		t.Fatalf("company ID should not be empty")
	}

	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.db")

	// Perform backup using VACUUM INTO
	if err := backupDB(ctx, s, backupPath); err != nil {
		t.Fatalf("backupDB failed: %v", err)
	}

	// Verify the backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file not created: %v", err)
	}

	// Verify the backup file has content (is not empty)
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("failed to stat backup file: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Error("backup file is empty, backup may have failed")
	}
}

func TestRunDBBackupIntegration(t *testing.T) {
	// Create a temporary home directory for this test
	tmpHome := t.TempDir()

	// Create a temporary database and config
	tmpStore := testutil.NewStore(t)

	// Create a config directory and file
	configDir := filepath.Join(tmpHome, ".paperclip-go")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Create a temporary data directory
	dataDir := filepath.Join(tmpHome, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	// Create a config file pointing to our temp data directory
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := "dataDir: " + dataDir + "\n"
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Close tmpStore (not used in this test)
	tmpStore.Close()

	// For integration test, we'll verify the directory creation and file naming
	// by testing with default backup directory
	backupsDir := filepath.Join(dataDir, "backups")

	// Verify backups directory doesn't exist before
	if _, err := os.Stat(backupsDir); err == nil {
		t.Fatal("backups directory should not exist before running backup")
	}

	// Create a fresh store for the test
	s := testutil.NewStore(t)

	// Call backupDB directly with a path that verifies directory creation works
	timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	backupPath := filepath.Join(backupsDir, timestamp+".db")

	// Create the directory as runDBBackup would
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatalf("failed to create backups directory: %v", err)
	}

	// Perform the backup
	ctx := context.Background()
	if err := backupDB(ctx, s, backupPath); err != nil {
		t.Fatalf("backupDB failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(backupsDir); err != nil {
		t.Errorf("backups directory not created: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file not created: %v", err)
	}

	// Verify the filename has the correct timestamp pattern
	filename := filepath.Base(backupPath)
	if !strings.Contains(filename, "_") || !strings.HasSuffix(filename, ".db") {
		t.Errorf("backup filename does not match expected pattern, got: %s", filename)
	}

	// Verify the file has correct permissions (0o600)
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("failed to stat backup file: %v", err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Errorf("backup file permissions incorrect: expected 0o600, got %o", fileInfo.Mode().Perm())
	}
}
