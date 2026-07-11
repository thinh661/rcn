package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rcn/rcn/backend/internal/database"
)

// CatalogEntry represents a node in the data catalog tree.
type CatalogEntry struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"` // database, schema, table, column
	ParentID  *string         `json:"parent_id,omitempty"`
	Meta      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// CatalogService manages the data catalog.
type CatalogService struct{}

// NewCatalogService creates a new catalog service.
func NewCatalogService() *CatalogService {
	return &CatalogService{}
}

// GetTree returns the full catalog tree structure.
func (s *CatalogService) GetTree(ctx context.Context) ([]map[string]interface{}, error) {
	db := database.GetDB()
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, type, parent_id, metadata, created_at, updated_at
		FROM data_catalog
		WHERE parent_id IS NULL
		ORDER BY type, name
	`)
	if err != nil {
		return nil, fmt.Errorf("query catalog tree: %w", err)
	}
	defer rows.Close()

	var roots []map[string]interface{}
	for rows.Next() {
		node, err := scanCatalogEntry(rows.Scan)
		if err != nil {
			return nil, err
		}
		children, _ := s.getChildren(ctx, node["id"].(string))
		node["children"] = children
		roots = append(roots, node)
	}
	return roots, rows.Err()
}

func (s *CatalogService) getChildren(ctx context.Context, parentID string) ([]map[string]interface{}, error) {
	db := database.GetDB()
	rows, err := db.QueryContext(ctx, `
		SELECT id, name, type, parent_id, metadata, created_at, updated_at
		FROM data_catalog
		WHERE parent_id = $1
		ORDER BY type, name
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []map[string]interface{}
	for rows.Next() {
		node, err := scanCatalogEntry(rows.Scan)
		if err != nil {
			return nil, err
		}
		grandchildren, _ := s.getChildren(ctx, node["id"].(string))
		node["children"] = grandchildren
		children = append(children, node)
	}
	return children, rows.Err()
}

func scanCatalogEntry(scan func(dest ...interface{}) error) (map[string]interface{}, error) {
	var (
		id, name, etype string
		parentID        *string
		meta            []byte
		createdAt, updatedAt time.Time
	)
	if err := scan(&id, &name, &etype, &parentID, &meta, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	node := map[string]interface{}{
		"id":         id,
		"name":       name,
		"type":       etype,
		"created_at": createdAt,
		"updated_at": updatedAt,
	}
	if parentID != nil {
		node["parent_id"] = *parentID
	}
	if len(meta) > 0 {
		var parsed interface{}
		if json.Unmarshal(meta, &parsed) == nil {
			node["metadata"] = parsed
		}
	}
	return node, nil
}

// Search searches catalog entries by name.
func (s *CatalogService) Search(ctx context.Context, query string, limit int) ([]CatalogEntry, error) {
	db := database.GetDB()
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, name, type, parent_id, metadata, created_at, updated_at
		FROM data_catalog
		WHERE name ILIKE '%' || $1 || '%'
		ORDER BY type, name
		LIMIT $2
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search catalog: %w", err)
	}
	defer rows.Close()

	var entries []CatalogEntry
	for rows.Next() {
		var e CatalogEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.ParentID, &e.Meta, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetEntry returns a single catalog entry by ID.
func (s *CatalogService) GetEntry(ctx context.Context, id string) (*CatalogEntry, error) {
	db := database.GetDB()
	var e CatalogEntry
	err := db.QueryRowContext(ctx, `
		SELECT id, name, type, parent_id, metadata, created_at, updated_at
		FROM data_catalog WHERE id = $1
	`, id).Scan(&e.ID, &e.Name, &e.Type, &e.ParentID, &e.Meta, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get catalog entry: %w", err)
	}
	return &e, nil
}

// UpsertEntry creates or updates a catalog entry.
func (s *CatalogService) UpsertEntry(ctx context.Context, name, etype string, parentID *string, metadata json.RawMessage) (*CatalogEntry, error) {
	db := database.GetDB()
	id := uuid.New().String()

	var e CatalogEntry
	err := db.QueryRowContext(ctx, `
		INSERT INTO data_catalog (id, name, type, parent_id, metadata)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (name, type, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'))
		DO UPDATE SET metadata = EXCLUDED.metadata, updated_at = NOW()
		RETURNING id, name, type, parent_id, metadata, created_at, updated_at
	`, id, name, etype, parentID, metadata).Scan(
		&e.ID, &e.Name, &e.Type, &e.ParentID, &e.Meta, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert catalog entry: %w", err)
	}
	return &e, nil
}

// DeleteEntry deletes a catalog entry and its children.
func (s *CatalogService) DeleteEntry(ctx context.Context, id string) error {
	db := database.GetDB()
	_, err := db.ExecContext(ctx, `DELETE FROM data_catalog WHERE id = $1`, id)
	return err
}

// SyncFromTrino syncs catalog entries from Trino metadata.
func (s *CatalogService) SyncFromTrino(ctx context.Context) error {
	// Trino system tables: information_schema.schemata, information_schema.tables
	// This is a placeholder — actual sync depends on Trino connector setup.
	return nil
}
