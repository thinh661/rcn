package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rcn/rcn/backend/internal/database"
)

type SparkConnectSession struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id"`
	Status     string    `json:"status"`
	Endpoint   string    `json:"endpoint"`
	CreatedAt  time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active_at"`
}

type SparkConnectService struct {
	endpoint string
	port     string
	enabled  bool
}

// NewSparkConnectService initializes the SparkConnectService with the given configuration
func NewSparkConnectService(endpoint, port string, enabled bool) *SparkConnectService {
	return &SparkConnectService{
		endpoint: endpoint,
		port:     port,
		enabled:  enabled,
	}
}

// generateShortUUID creates an 8-character long hexadecimal string to use as short UUID
func generateShortUUID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp representation if crypto/rand fails
		return fmt.Sprintf("%08x", time.Now().UnixNano())[:8]
	}
	return hex.EncodeToString(b)
}

// CreateSession inserts a new active session for the user into the database
func (s *SparkConnectService) CreateSession(ctx context.Context, userID string) (*SparkConnectSession, error) {
	sessionID := "sc-" + generateShortUUID()
	endpoint := fmt.Sprintf("sc://%s:%s", s.endpoint, s.port)

	db := database.GetDB()
	
	query := `
		INSERT INTO spark_connect_sessions (user_id, session_id, status, endpoint)
		VALUES ($1, $2, 'active', $3)
		RETURNING id, user_id, session_id, status, endpoint, created_at, last_active_at
	`

	var sess SparkConnectSession
	err := db.QueryRowContext(ctx, query, userID, sessionID, endpoint).Scan(
		&sess.ID,
		&sess.UserID,
		&sess.SessionID,
		&sess.Status,
		&sess.Endpoint,
		&sess.CreatedAt,
		&sess.LastActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create spark connect session: %w", err)
	}

	return &sess, nil
}

// ListSessions retrieves all sessions belonging to the specified user sorted by creation time descending
func (s *SparkConnectService) ListSessions(ctx context.Context, userID string) ([]SparkConnectSession, error) {
	db := database.GetDB()

	query := `
		SELECT id, user_id, session_id, status, endpoint, created_at, last_active_at
		FROM spark_connect_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list spark connect sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SparkConnectSession
	for rows.Next() {
		var sess SparkConnectSession
		err := rows.Scan(
			&sess.ID,
			&sess.UserID,
			&sess.SessionID,
			&sess.Status,
			&sess.Endpoint,
			&sess.CreatedAt,
			&sess.LastActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan spark connect session: %w", err)
		}
		sessions = append(sessions, sess)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// CloseSession marks the session as closed for the given session ID and user ID
func (s *SparkConnectService) CloseSession(ctx context.Context, sessionID string, userID string) error {
	db := database.GetDB()

	query := `
		UPDATE spark_connect_sessions
		SET status = 'closed'
		WHERE id = $1 AND user_id = $2
	`

	result, err := db.ExecContext(ctx, query, sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to close spark connect session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found or not owned by user")
	}

	return nil
}

// GetConfig returns configuration values for Spark Connect
func (s *SparkConnectService) GetConfig() map[string]interface{} {
	connectionString := fmt.Sprintf("sc://%s:%s", s.endpoint, s.port)
	return map[string]interface{}{
		"endpoint":          s.endpoint,
		"port":              s.port,
		"enabled":           s.enabled,
		"connection_string": connectionString,
	}
}
