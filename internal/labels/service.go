// Package labels provides CRUD operations for Label entities and label-issue associations.
package labels

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
	"modernc.org/sqlite"
)

// ErrNotFound is returned when a requested label does not exist.
var ErrNotFound = errors.New("label not found")

// ErrDuplicate is returned when attempting to create a label with a duplicate name within a company.
var ErrDuplicate = errors.New("label name already exists in this company")

// ErrIssueNotFound is returned when a requested issue does not exist.
var ErrIssueNotFound = errors.New("issue not found")

// ErrCompanyMismatch is returned when issue and label belong to different companies.
var ErrCompanyMismatch = errors.New("issue and label are in different companies")

// ErrAssociationNotFound is returned when a label-issue association does not exist.
var ErrAssociationNotFound = errors.New("label is not associated with this issue")

// SQLite extended error codes (https://www.sqlite.org/rescode.html#extrc)
// Used for translating constraint violations to domain-specific errors.
const (
	sqliteConstraintUnique      = 2067 // SQLITE_CONSTRAINT_UNIQUE - primary key or unique constraint violation
	sqliteConstraintForeignKey  = 787  // SQLITE_CONSTRAINT_FOREIGNKEY - foreign key constraint violation
)

// Service provides label CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new label and returns the created entity.
// Returns ErrDuplicate if a label with the same name already exists for this company.
func (s *Service) Create(ctx context.Context, companyID, name, color string) (*domain.Label, error) {
	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)
	l := &domain.Label{
		ID:        ids.NewUUID(),
		CompanyID: companyID,
		Name:      name,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO labels(id, company_id, name, color, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		l.ID, l.CompanyID, l.Name, l.Color, ts, ts,
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintUnique {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("inserting label: %w", err)
	}
	return l, nil
}

// Get returns the label with the given ID, or ErrNotFound if it doesn't exist.
func (s *Service) Get(ctx context.Context, id string) (*domain.Label, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, name, color, created_at, updated_at
		 FROM labels WHERE id = ?`, id,
	)
	l, err := scanLabel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return l, err
}

// ListByCompany returns all labels for a given company, ordered by creation time ascending.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.Label, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, name, color, created_at, updated_at
		 FROM labels WHERE company_id = ? ORDER BY created_at ASC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing labels by company: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Label, 0)
	for rows.Next() {
		l, err := scanLabel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating labels: %w", err)
	}
	return out, nil
}

// Delete removes the label with the given ID.
// Returns ErrNotFound if the label does not exist.
func (s *Service) Delete(ctx context.Context, id string) error {
	result, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM labels WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("deleting label: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// LinkToIssue associates a label with an issue (idempotent via INSERT OR IGNORE).
// Atomically verifies that both issue and label exist and belong to the same company within a transaction.
// Returns ErrIssueNotFound if the issue doesn't exist, ErrNotFound if the label doesn't exist.
func (s *Service) LinkToIssue(ctx context.Context, issueID, labelID string) error {
	return s.store.WithTx(ctx, func(tx *sql.Tx) error {
		// Verify both issue and label exist and belong to same company
		var issueCompanyID, labelCompanyID string
		err := tx.QueryRowContext(ctx,
			`SELECT i.company_id, l.company_id
			 FROM issues i, labels l
			 WHERE i.id = ? AND l.id = ?`,
			issueID, labelID,
		).Scan(&issueCompanyID, &labelCompanyID)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Determine which one doesn't exist by querying within the transaction
				var issueExists int
				errIssue := tx.QueryRowContext(ctx,
					`SELECT 1 FROM issues WHERE id = ? LIMIT 1`,
					issueID,
				).Scan(&issueExists)
				if errors.Is(errIssue, sql.ErrNoRows) {
					return ErrIssueNotFound
				}
				if errIssue != nil {
					return fmt.Errorf("checking issue existence: %w", errIssue)
				}
				return ErrNotFound // label doesn't exist
			}
			return fmt.Errorf("verifying issue and label: %w", err)
		}

		// Verify same company
		if issueCompanyID != labelCompanyID {
			return ErrCompanyMismatch
		}

		// INSERT OR IGNORE into issue_labels
		now := time.Now().UTC().Truncate(time.Second)
		ts := now.Format(time.RFC3339)
		_, err = tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO issue_labels(issue_id, label_id, created_at)
			 VALUES (?, ?, ?)`,
			issueID, labelID, ts,
		)
		if err != nil {
			var sqliteErr *sqlite.Error
			if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintForeignKey {
				// FK constraint violation: one of the entities was deleted
				// Query to distinguish between deleted issue vs deleted label
				var issueExists int
				errIssue := tx.QueryRowContext(ctx,
					`SELECT 1 FROM issues WHERE id = ? LIMIT 1`,
					issueID,
				).Scan(&issueExists)
				if errors.Is(errIssue, sql.ErrNoRows) {
					return ErrIssueNotFound
				}
				if errIssue != nil {
					return fmt.Errorf("checking issue existence: %w", errIssue)
				}
				// Issue exists, so label must not exist
				var labelExists int
				errLabel := tx.QueryRowContext(ctx,
					`SELECT 1 FROM labels WHERE id = ? LIMIT 1`,
					labelID,
				).Scan(&labelExists)
				if errors.Is(errLabel, sql.ErrNoRows) {
					return ErrNotFound
				}
				if errLabel != nil {
					return fmt.Errorf("checking label existence: %w", errLabel)
				}
				// Both exist but FK failed for some other reason
				return fmt.Errorf("linking label to issue: %w", err)
			}
			return fmt.Errorf("linking label to issue: %w", err)
		}
		return nil
	})
}

// UnlinkFromIssue removes the association between a label and an issue.
// Atomically verifies that both issue and label exist and belong to the same company.
// Returns ErrAssociationNotFound if the association does not exist.
// Returns ErrCompanyMismatch if issue and label belong to different companies.
func (s *Service) UnlinkFromIssue(ctx context.Context, issueID, labelID string) error {
	return s.store.WithTx(ctx, func(tx *sql.Tx) error {
		// Verify both issue and label exist and belong to same company
		var issueCompanyID, labelCompanyID string
		err := tx.QueryRowContext(ctx,
			`SELECT i.company_id, l.company_id
			 FROM issues i, labels l
			 WHERE i.id = ? AND l.id = ?`,
			issueID, labelID,
		).Scan(&issueCompanyID, &labelCompanyID)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Determine which one doesn't exist by querying within the transaction
				var issueExists int
				errIssue := tx.QueryRowContext(ctx,
					`SELECT 1 FROM issues WHERE id = ? LIMIT 1`,
					issueID,
				).Scan(&issueExists)
				if errors.Is(errIssue, sql.ErrNoRows) {
					return ErrAssociationNotFound
				}
				if errIssue != nil {
					return fmt.Errorf("checking issue existence: %w", errIssue)
				}
				return ErrAssociationNotFound // label doesn't exist
			}
			return fmt.Errorf("verifying issue and label: %w", err)
		}

		// Verify same company. For unlink operations, preserve the existing
		// not-found behavior for callers that only special-case
		// ErrAssociationNotFound.
		if issueCompanyID != labelCompanyID {
			return ErrAssociationNotFound
		}

		// Delete the association
		result, err := tx.ExecContext(ctx,
			`DELETE FROM issue_labels WHERE issue_id = ? AND label_id = ?`, issueID, labelID,
		)
		if err != nil {
			return fmt.Errorf("deleting label from issue: %w", err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking rows affected: %w", err)
		}
		if n == 0 {
			return ErrAssociationNotFound
		}
		return nil
	})
}

// GetLabelsForIssue returns all labels associated with the given issue.
func (s *Service) GetLabelsForIssue(ctx context.Context, issueID string) ([]*domain.Label, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT l.id, l.company_id, l.name, l.color, l.created_at, l.updated_at
		 FROM labels l
		 INNER JOIN issue_labels il ON l.id = il.label_id
		 WHERE il.issue_id = ? ORDER BY l.created_at ASC`,
		issueID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing labels for issue: %w", err)
	}
	defer rows.Close()

	out := make([]*domain.Label, 0)
	for rows.Next() {
		l, err := scanLabel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating labels: %w", err)
	}
	return out, nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanLabel(s scanner) (*domain.Label, error) {
	var l domain.Label
	var createdAt string
	var updatedAt string

	if err := s.Scan(&l.ID, &l.CompanyID, &l.Name, &l.Color, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	var err error
	l.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAt, err)
	}

	l.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAt, err)
	}

	return &l, nil
}
