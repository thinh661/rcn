package services

import (
	"context"
	"fmt"
	"time"

	"github.com/rcn/rcn/backend/internal/database"
)

// DeltaShare represents a record in the delta_shares table
type DeltaShare struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Table     string    `json:"table"`
	ShareWith string    `json:"share_with"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// DeltaSharingService handles database operations and configuration for Delta Sharing
type DeltaSharingService struct {
	enabled   bool
	serverURL string
}

// NewDeltaSharingService initializes a new DeltaSharingService
func NewDeltaSharingService(enabled bool, serverURL string) *DeltaSharingService {
	return &DeltaSharingService{
		enabled:   enabled,
		serverURL: serverURL,
	}
}

// IsEnabled returns true if Delta Sharing is enabled
func (s *DeltaSharingService) IsEnabled() bool {
	return s.enabled
}

// ListShares retrieves all delta shares ordered by creation date descending
func (s *DeltaSharingService) ListShares(ctx context.Context) ([]DeltaShare, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, name, table_name, share_with, status, created_at
		FROM delta_shares
		ORDER BY created_at DESC
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list delta shares: %w", err)
	}
	defer rows.Close()

	var shares []DeltaShare
	for rows.Next() {
		var share DeltaShare
		err := rows.Scan(
			&share.ID,
			&share.Name,
			&share.Table,
			&share.ShareWith,
			&share.Status,
			&share.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan delta share: %w", err)
		}
		shares = append(shares, share)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return shares, nil
}

// CreateShare inserts a new delta share into the database
func (s *DeltaSharingService) CreateShare(ctx context.Context, name, table, shareWith string) (*DeltaShare, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		INSERT INTO delta_shares (name, table_name, share_with, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, name, table_name, share_with, status, created_at
	`

	var share DeltaShare
	err := db.QueryRowContext(ctx, query, name, table, shareWith).Scan(
		&share.ID,
		&share.Name,
		&share.Table,
		&share.ShareWith,
		&share.Status,
		&share.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create delta share: %w", err)
	}

	return &share, nil
}

// RevokeShare updates the status of the specified delta share to 'revoked'
func (s *DeltaSharingService) RevokeShare(ctx context.Context, id string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `
		UPDATE delta_shares
		SET status = 'revoked'
		WHERE id = $1
	`
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke delta share: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("delta share not found")
	}

	return nil
}

// GetRecipients retrieves unique recipient entities who have active shares
func (s *DeltaSharingService) GetRecipients(ctx context.Context) ([]string, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT DISTINCT share_with
		FROM delta_shares
		WHERE status = 'active'
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipients: %w", err)
	}
	defer rows.Close()

	var recipients []string
	for rows.Next() {
		var recipient string
		if err := rows.Scan(&recipient); err != nil {
			return nil, fmt.Errorf("failed to scan recipient: %w", err)
		}
		recipients = append(recipients, recipient)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return recipients, nil
}

// GetConfig returns the configuration details of Delta Sharing
func (s *DeltaSharingService) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled":           s.enabled,
		"server_url":        s.serverURL,
		"connection_string": s.serverURL,
	}
}
