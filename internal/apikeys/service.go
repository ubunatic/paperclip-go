// Package apikeys provides creation, validation, and revocation of API keys.
package apikeys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// ErrNotFound is returned when no matching API key exists.
var ErrNotFound = errors.New("api key not found")

// ErrRevoked is returned when the key exists but has been revoked.
var ErrRevoked = errors.New("api key is revoked")

// Service manages API key lifecycle.
type Service struct {
	store *store.Store
}

// New returns a Service backed by the given store.
func New(s *store.Store) *Service { return &Service{store: s} }

// Create generates a new random API key for the given company, stores a SHA-256
// hash of it, and returns the domain record together with the raw key (the only
// time it is ever visible in plaintext).
func (s *Service) Create(ctx context.Context, companyID, name string) (*domain.APIKey, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generating key bytes: %w", err)
	}
	rawKey := base64.RawURLEncoding.EncodeToString(raw)
	keyHash := hashKey(rawKey)

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	key := &domain.APIKey{
		ID:        ids.NewUUID(),
		CompanyID: companyID,
		Name:      name,
		CreatedAt: now,
	}

	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO api_keys(id, company_id, name, key_hash, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		key.ID, key.CompanyID, key.Name, keyHash, ts,
	)
	if err != nil {
		return nil, "", fmt.Errorf("inserting api key: %w", err)
	}
	return key, rawKey, nil
}

// Validate looks up a key by hashing rawKey and returns the associated APIKey.
// Returns ErrNotFound if the hash is unknown, ErrRevoked if revoked_at is set.
func (s *Service) Validate(ctx context.Context, rawKey string) (*domain.APIKey, error) {
	keyHash := hashKey(rawKey)
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, name, created_at, revoked_at
		 FROM api_keys WHERE key_hash = ?`,
		keyHash,
	)
	key, err := scanAPIKey(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying api key: %w", err)
	}
	if key.RevokedAt != nil {
		return nil, ErrRevoked
	}
	return key, nil
}

// ListByCompany returns all non-revoked keys for the given company, ordered by creation time.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.APIKey, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, name, created_at, revoked_at
		 FROM api_keys
		 WHERE company_id = ? AND revoked_at IS NULL
		 ORDER BY created_at ASC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing api keys: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.APIKey, 0)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating api keys: %w", err)
	}
	return out, nil
}

// Revoke soft-deletes the key with the given ID. Returns ErrNotFound if the key
// does not exist or is already revoked.
func (s *Service) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	result, err := s.store.DB.ExecContext(ctx,
		`UPDATE api_keys SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("revoking api key: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for api key revoke: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(s scanner) (*domain.APIKey, error) {
	var key domain.APIKey
	var createdAt string
	var revokedAt *string
	if err := s.Scan(&key.ID, &key.CompanyID, &key.Name, &createdAt, &revokedAt); err != nil {
		return nil, err
	}
	var err error
	key.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}
	if revokedAt != nil {
		t, err := time.Parse(time.RFC3339, *revokedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing revoked_at %q: %w", *revokedAt, err)
		}
		key.RevokedAt = &t
	}
	return &key, nil
}

func hashKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}
