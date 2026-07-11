package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/database"
	"github.com/rcn/rcn/backend/internal/middleware"
)

// AuditLogHandler exposes admin-facing audit log queries.
type AuditLogHandler struct{}

func NewAuditLogHandler() *AuditLogHandler {
	return &AuditLogHandler{}
}

// AuditLogEntry represents a single row from the audit_logs table.
type AuditLogEntry struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id,omitempty"`
	Username     string                 `json:"username,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	CreatedAt    string                 `json:"created_at"`
}

// ListAuditLogs returns paginated audit log entries, superadmin only.
// GET /api/v1/admin/audit-logs
//
// Query parameters (all optional):
//   - user_id:     Filter by specific user ID
//   - action:      Filter by action (create, update, delete, patch)
//   - resource_type: Filter by resource type (e.g. "admin_users", "notebooks")
//   - resource_id: Filter by specific resource ID
//   - from:        Start date (RFC3339 or YYYY-MM-DD)
//   - to:          End date (RFC3339 or YYYY-MM-DD)
//   - limit:       Max results per page (default 50, max 200)
//   - offset:      Pagination offset (default 0)
func (h *AuditLogHandler) ListAuditLogs(c *gin.Context) {
	if !middleware.IsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only superadmin can view audit logs"})
		return
	}

	db := database.GetDB()

	// Build dynamic WHERE clause from query params
	var conditions []string
	var args []interface{}
	argIdx := 1

	if userID := c.Query("user_id"); userID != "" {
		conditions = append(conditions, fmt.Sprintf("al.user_id = $%d", argIdx))
		args = append(args, userID)
		argIdx++
	}

	if action := c.Query("action"); action != "" {
		conditions = append(conditions, fmt.Sprintf("al.action = $%d", argIdx))
		args = append(args, action)
		argIdx++
	}

	if resourceType := c.Query("resource_type"); resourceType != "" {
		conditions = append(conditions, fmt.Sprintf("al.resource_type = $%d", argIdx))
		args = append(args, resourceType)
		argIdx++
	}

	if resourceID := c.Query("resource_id"); resourceID != "" {
		conditions = append(conditions, fmt.Sprintf("al.resource_id = $%d", argIdx))
		args = append(args, resourceID)
		argIdx++
	}

	if from := c.Query("from"); from != "" {
		t, err := parseTimeParam(from)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("al.created_at >= $%d", argIdx))
			args = append(args, t)
			argIdx++
		}
	}

	if to := c.Query("to"); to != "" {
		t, err := parseTimeParam(to)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("al.created_at <= $%d", argIdx))
			args = append(args, t)
			argIdx++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Pagination
	limit := 50
	maxLimit := 200
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > maxLimit {
				limit = maxLimit
			} else {
				limit = parsed
			}
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get total count first (for pagination metadata)
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs al %s", whereClause)
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		log.Error().Err(err).Msg("failed to count audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query audit logs"})
		return
	}

	// Get paginated results
	query := fmt.Sprintf(
		`SELECT al.id, COALESCE(al.user_id, ''), al.username, al.action, al.resource_type,
		        al.resource_id, al.details, al.ip_address, al.user_agent, al.created_at
		 FROM audit_logs al %s
		 ORDER BY al.created_at DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	queryArgs := append(args, limit, offset)

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		log.Error().Err(err).Msg("failed to query audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query audit logs"})
		return
	}
	defer rows.Close()

	entries := []AuditLogEntry{}
	for rows.Next() {
		var entry AuditLogEntry
		var detailsStr sql.NullString
		var createdAt time.Time

		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Username, &entry.Action,
			&entry.ResourceType, &entry.ResourceID, &detailsStr,
			&entry.IPAddress, &entry.UserAgent, &createdAt,
		); err != nil {
			log.Error().Err(err).Msg("failed to scan audit log entry")
			continue
		}

		entry.CreatedAt = createdAt.Format(time.RFC3339)

		if detailsStr.Valid && detailsStr.String != "" && detailsStr.String != "{}" {
			entry.Details = make(map[string]interface{})
			// Best-effort parse — if it fails, leave details nil
			_ = json.Unmarshal([]byte(detailsStr.String), &entry.Details)
		}

		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("error iterating audit log rows")
	}

	if entries == nil {
		entries = []AuditLogEntry{}
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// parseTimeParam parses a time query parameter that is either an RFC3339
// string or a YYYY-MM-DD date.
func parseTimeParam(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	// Try YYYY-MM-DDTHH:MM (without seconds or timezone)
	if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}
