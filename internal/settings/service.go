// Package settings provides CRUD operations for Setting entities.
package settings

import (
	"context"
	"fmt"
	"time"

	"github.com/ubunatic/paperclip-go/internal/store"
)

// Service provides setting CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// GetAll returns all instance settings as a map of key-value pairs.
// Returns empty map (not nil) when no rows exist.
func (s *Service) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT key, value FROM instance_settings ORDER BY key ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("querying instance_settings: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scanning setting row: %w", err)
		}
		result[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating settings: %w", err)
	}
	return result, nil
}

// Patch performs a merge-update (UPSERT) of the provided settings and returns the full map after update.
// Empty updates map is valid and returns the current state unchanged.
func (s *Service) Patch(ctx context.Context, updates map[string]string) (map[string]string, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	// UPSERT each setting key-value pair
	for key, value := range updates {
		_, err := s.store.DB.ExecContext(ctx,
			`INSERT INTO instance_settings(key, value, updated_at)
			 VALUES (?, ?, ?)
			 ON CONFLICT(key) DO UPDATE SET
			 value = excluded.value,
			 updated_at = excluded.updated_at`,
			key, value, ts,
		)
		if err != nil {
			return nil, fmt.Errorf("upserting setting %q: %w", key, err)
		}
	}

	// Return the full map after update
	return s.GetAll(ctx)
}

// SeedDefaults inserts default settings if they don't already exist (INSERT OR IGNORE).
// Returns without error if settings already exist.
func (s *Service) SeedDefaults(ctx context.Context, defaults map[string]string) error {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	for key, value := range defaults {
		_, err := s.store.DB.ExecContext(ctx,
			`INSERT OR IGNORE INTO instance_settings(key, value, updated_at)
			 VALUES (?, ?, ?)`,
			key, value, ts,
		)
		if err != nil {
			return fmt.Errorf("seeding default %q: %w", key, err)
		}
	}
	return nil
}
