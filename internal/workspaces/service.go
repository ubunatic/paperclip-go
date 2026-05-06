// Package workspaces provides CRUD operations for Workspace entities.
package workspaces

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ubunatic/paperclip-go/internal/domain"
	"github.com/ubunatic/paperclip-go/internal/ids"
	"github.com/ubunatic/paperclip-go/internal/store"
)

// ErrNotFound is returned when a requested workspace does not exist.
var ErrNotFound = errors.New("workspace not found")

// ErrDuplicate is returned when attempting to create a workspace with a non-unique agent_id, path pair.
var ErrDuplicate = errors.New("workspace with this agent and path already exists")

// Service provides workspace CRUD backed by the store.
type Service struct {
	store *store.Store
}

// New returns a Service using the given store.
func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Create inserts a new workspace and returns the created entity.
func (s *Service) Create(ctx context.Context, companyID, agentID, path string, issueID *string, status string) (*domain.Workspace, error) {
	if !domain.IsValidWorkspaceStatus(status) {
		return nil, fmt.Errorf("invalid status %q", status)
	}

	now := time.Now().UTC().Truncate(time.Second)
	ts := now.Format(time.RFC3339)

	workspace := &domain.Workspace{
		ID:        ids.NewUUID(),
		CompanyID: companyID,
		AgentID:   agentID,
		IssueID:   issueID,
		Path:      path,
		Status:    domain.WorkspaceStatus(status),
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := s.store.DB.ExecContext(ctx,
		`INSERT INTO execution_workspaces(id, company_id, agent_id, issue_id, path, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		workspace.ID, workspace.CompanyID, workspace.AgentID, workspace.IssueID, workspace.Path, workspace.Status, ts, ts,
	)
	if err != nil {
		// Check for UNIQUE constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	return workspace, nil
}

// GetByID retrieves a workspace by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Workspace, error) {
	row := s.store.DB.QueryRowContext(ctx,
		`SELECT id, company_id, agent_id, issue_id, path, status, created_at, updated_at
		 FROM execution_workspaces WHERE id = ?`,
		id,
	)
	workspace, err := scanWorkspace(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	return workspace, nil
}

// ListByCompany retrieves all workspaces for a given company.
func (s *Service) ListByCompany(ctx context.Context, companyID string) ([]*domain.Workspace, error) {
	rows, err := s.store.DB.QueryContext(ctx,
		`SELECT id, company_id, agent_id, issue_id, path, status, created_at, updated_at
		 FROM execution_workspaces WHERE company_id = ? ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	workspaces := make([]*domain.Workspace, 0)
	for rows.Next() {
		w, err := scanWorkspace(rows)
		if err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		workspaces = append(workspaces, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating workspaces: %w", err)
	}

	return workspaces, nil
}

// Delete removes a workspace by its ID.
// Returns ErrNotFound if the workspace does not exist.
func (s *Service) Delete(ctx context.Context, id string) error {
	result, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM execution_workspaces WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("determining if workspace was deleted: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanWorkspace(s scanner) (*domain.Workspace, error) {
	var w domain.Workspace
	var createdAtStr, updatedAtStr string
	var issueID *string

	if err := s.Scan(&w.ID, &w.CompanyID, &w.AgentID, &issueID, &w.Path, &w.Status, &createdAtStr, &updatedAtStr); err != nil {
		return nil, err
	}

	w.IssueID = issueID

	var err error
	w.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at %q: %w", createdAtStr, err)
	}

	w.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at %q: %w", updatedAtStr, err)
	}

	return &w, nil
}
