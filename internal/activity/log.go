// Package activity provides activity logging operations.
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// Log provides activity logging backed by the store.
type Log struct {
	store *store.Store
}

// New returns a Log using the given store.
func New(s *store.Store) *Log {
	return &Log{store: s}
}

// Record inserts a new activity log entry.
func (l *Log) Record(ctx context.Context, companyID, actorKind, actorID, action, entityKind, entityID, metaJSON string) error {
	// Default empty metaJSON to '{}'
	if metaJSON == "" {
		metaJSON = "{}"
	}
	// Validate metaJSON is valid JSON
	if !json.Valid([]byte(metaJSON)) {
		return fmt.Errorf("metaJSON is not valid JSON: %q", metaJSON)
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	_, err := l.store.DB.ExecContext(ctx,
		`INSERT INTO activity_log(id, company_id, actor_kind, actor_id, action, entity_kind, entity_id, meta_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ids.NewUUID(), companyID, actorKind, actorID, action, entityKind, entityID, metaJSON, ts,
	)
	if err != nil {
		return fmt.Errorf("inserting activity: %w", err)
	}
	return nil
}

// List returns the most recent activity log entries for a company, limited by count.
func (l *Log) List(ctx context.Context, companyID string, limit int) ([]*domain.Activity, error) {
	rows, err := l.store.DB.QueryContext(ctx,
		`SELECT id, company_id, actor_kind, actor_id, action, entity_kind, entity_id, meta_json, created_at
		 FROM activity_log WHERE company_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`,
		companyID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying activity log: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Activity, 0)
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating activity log: %w", err)
	}
	return out, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanActivity(s scanner) (*domain.Activity, error) {
	var a domain.Activity
	var createdAt string
	var metaJSONBytes []byte
	if err := s.Scan(&a.ID, &a.CompanyID, &a.ActorKind, &a.ActorID, &a.Action, &a.EntityKind, &a.EntityID, &metaJSONBytes, &createdAt); err != nil {
		return nil, err
	}
	a.MetaJSON = json.RawMessage(metaJSONBytes)
	var err error
	a.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	return &a, nil
}
