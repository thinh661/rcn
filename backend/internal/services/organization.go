package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rcn/rcn/backend/internal/database"
)

type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ParentID    *string   `json:"parent_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OrgService struct{}

// NewOrgService initializes and returns a new OrgService
func NewOrgService() *OrgService {
	return &OrgService{}
}

// Create inserts a new organization into the database
func (s *OrgService) Create(ctx context.Context, name, desc string, parentID *string) (*Organization, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	id := uuid.New().String()
	query := `
		INSERT INTO organizations (id, name, description, parent_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, name, description, parent_id, created_at, updated_at
	`

	var org Organization
	err := db.QueryRowContext(ctx, query, id, name, desc, parentID).Scan(
		&org.ID,
		&org.Name,
		&org.Description,
		&org.ParentID,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	return &org, nil
}

// List retrieves all organizations sorted by name
func (s *OrgService) List(ctx context.Context) ([]Organization, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, name, description, parent_id, created_at, updated_at
		FROM organizations
		ORDER BY name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var list []Organization
	for rows.Next() {
		var org Organization
		err := rows.Scan(
			&org.ID,
			&org.Name,
			&org.Description,
			&org.ParentID,
			&org.CreatedAt,
			&org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		list = append(list, org)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// Get retrieves a specific organization by its ID
func (s *OrgService) Get(ctx context.Context, id string) (*Organization, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, name, description, parent_id, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	var org Organization
	err := db.QueryRowContext(ctx, query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Description,
		&org.ParentID,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &org, nil
}

// Update updates an organization's details
func (s *OrgService) Update(ctx context.Context, id, name, desc string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `
		UPDATE organizations
		SET name = $2, description = $3, updated_at = NOW()
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, id, name, desc)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

// Delete deletes an organization by its ID
func (s *OrgService) Delete(ctx context.Context, id string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `DELETE FROM organizations WHERE id = $1`
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

// GetTree retrieves a nested hierarchical structure of all organizations
func (s *OrgService) GetTree(ctx context.Context) ([]map[string]interface{}, error) {
	orgs, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make(map[string]map[string]interface{})
	for _, org := range orgs {
		nodes[org.ID] = map[string]interface{}{
			"id":          org.ID,
			"name":        org.Name,
			"description": org.Description,
			"parent_id":   org.ParentID,
			"created_at":  org.CreatedAt,
			"updated_at":  org.UpdatedAt,
			"children":    []map[string]interface{}{},
		}
	}

	var tree []map[string]interface{}
	for _, org := range orgs {
		node := nodes[org.ID]
		if org.ParentID == nil || *org.ParentID == "" {
			tree = append(tree, node)
		} else {
			parent, exists := nodes[*org.ParentID]
			if exists {
				parent["children"] = append(parent["children"].([]map[string]interface{}), node)
			} else {
				tree = append(tree, node)
			}
		}
	}

	return tree, nil
}

// AssignUser links an admin user to an organization
func (s *OrgService) AssignUser(ctx context.Context, userID, orgID string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `
		UPDATE admins
		SET organization_id = $1
		WHERE id = $2
	`

	result, err := db.ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to assign user to organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("admin user not found")
	}

	return nil
}

// GetUsers retrieves all admin users belonging to a specific organization
func (s *OrgService) GetUsers(ctx context.Context, orgID string) ([]map[string]interface{}, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, username, email
		FROM admins
		WHERE organization_id = $1
	`

	rows, err := db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users for organization: %w", err)
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, username, email string
		err := rows.Scan(&id, &username, &email)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user record: %w", err)
		}
		users = append(users, map[string]interface{}{
			"id":       id,
			"username": username,
			"email":    email,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
