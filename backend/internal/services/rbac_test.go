package services

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rcn/rcn/backend/internal/database"
	"github.com/rcn/rcn/backend/internal/middleware"
)

// Define mock database driver to simulate SQL queries without database dependency.
type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{query: query}, nil
}
func (c *mockConn) Begin() (driver.Tx, error) { return &mockTx{}, nil }
func (c *mockConn) Close() error               { return nil }

type mockTx struct{}

func (t *mockTx) Commit() error   { return nil }
func (t *mockTx) Rollback() error { return nil }

type mockStmt struct {
	query string
}

func (s *mockStmt) Close() error { return nil }
func (s *mockStmt) NumInput() int {
	return -1
}
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &mockResult{}, nil
}

type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) { return 1, nil }
func (r *mockResult) RowsAffected() (int64, error) { return 1, nil }

// Thread-safe mock data store for database simulation
var (
	mockMu       sync.Mutex
	mockOwnerID  = "owner-123"
	mockIsPublic = false
)

func setMockDBData(ownerID string, isPublic bool) {
	mockMu.Lock()
	mockOwnerID = ownerID
	mockIsPublic = isPublic
	mockMu.Unlock()
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	mockMu.Lock()
	defer mockMu.Unlock()
	return &mockRows{
		columns: []string{"owner_id", "is_public"},
		rows: [][]driver.Value{
			{mockOwnerID, mockIsPublic},
		},
		idx: 0,
	}, nil
}

type mockRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *mockRows) Columns() []string { return r.columns }
func (r *mockRows) Close() error      { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	for i, v := range r.rows[r.idx] {
		dest[i] = v
	}
	r.idx++
	return nil
}

var registerOnce sync.Once

func setupMockDB(t *testing.T) {
	registerOnce.Do(func() {
		sql.Register("rbacmockdriver", &mockDriver{})
	})
	db, err := sql.Open("rbacmockdriver", "")
	if err != nil {
		t.Fatalf("Failed to open mock db: %v", err)
	}
	database.SetDB(db)
}

// TestRequireRole verifies correct role passes and wrong role blocks.
func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userRole       string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "Matching role passes",
			userRole:       "editor",
			requiredRoles:  []string{"editor"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "One of multiple allowed roles passes",
			userRole:       "admin",
			requiredRoles:  []string{"editor", "admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Mismatched role blocks",
			userRole:       "viewer",
			requiredRoles:  []string{"editor"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Missing role context blocks",
			userRole:       "",
			requiredRoles:  []string{"editor"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				if tt.userRole != "" {
					c.Set("admin_role", tt.userRole)
				}
				c.Next()
			}, middleware.RequireRole(tt.requiredRoles...), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestNotebookAccess checks notebook permission logic.
// - Viewer: read-only access (blocked on non-owned private notebooks, allowed on public/owned).
// - Editor: write/unrestricted access (passes notebook access middleware).
func TestNotebookAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupMockDB(t)

	tests := []struct {
		name             string
		userRole         string
		userID           string
		notebookOwner    string
		notebookIsPublic bool
		expectedStatus   int
	}{
		{
			name:             "Viewer accessing own private notebook - Allowed",
			userRole:         "viewer",
			userID:           "user-123",
			notebookOwner:    "user-123",
			notebookIsPublic: false,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "Viewer accessing someone else's public notebook - Allowed",
			userRole:         "viewer",
			userID:           "user-123",
			notebookOwner:    "other-456",
			notebookIsPublic: true,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "Viewer accessing someone else's private notebook - Forbidden",
			userRole:         "viewer",
			userID:           "user-123",
			notebookOwner:    "other-456",
			notebookIsPublic: false,
			expectedStatus:   http.StatusForbidden,
		},
		{
			name:             "Editor accessing someone else's private notebook - Allowed (middleware bypasses editors)",
			userRole:         "editor",
			userID:           "user-123",
			notebookOwner:    "other-456",
			notebookIsPublic: false,
			expectedStatus:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setMockDBData(tt.notebookOwner, tt.notebookIsPublic)

			r := gin.New()
			r.GET("/notebook/:id", func(c *gin.Context) {
				c.Set("admin_role", tt.userRole)
				c.Set("admin_id", tt.userID)
				c.Next()
			}, middleware.RequireNotebookAccess(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req, _ := http.NewRequest("GET", "/notebook/nb-abc", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
