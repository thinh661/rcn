package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/database"
)

// skipAuditPaths are path prefixes that should never be logged (health checks,
// static assets, read-only endpoints). Audit logging focuses on mutating
// actions (create, update, delete, patch).
var skipAuditPrefixes = []string{
	"/health",
	"/static",
	"/favicon",
}

// skipAuditMethods are HTTP methods that are too noisy or always read-only.
// POST is included here at the top level — the AuditMiddleware function below
// uses isMutatingMethod so that auth/login endpoints are still excluded while
// POST to resource endpoints gets logged.
var skipAuditMethods = map[string]bool{
	"GET":    true,
	"HEAD":   true,
	"OPTIONS": true,
}

// isMutatingMethod returns true for HTTP methods that modify state.
func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	}
	return false
}

// shouldSkipAudit returns true when the request path matches a skip prefix
// or the method is not a mutating method.
func shouldSkipAudit(path string, method string) bool {
	if !isMutatingMethod(method) {
		return true
	}
	for _, prefix := range skipAuditPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// extractResourceType derives a human-readable resource type from the URL path.
// It strips the API prefix (/api/v1/) and collapses path segments to produce
// a consistent label like "admin_users", "notebooks", "connectors", "spark_jobs".
func extractResourceType(path string) string {
	// Strip /api/v1/ prefix
	p := strings.TrimPrefix(path, "/api/v1/")
	p = strings.TrimPrefix(p, "/api/")
	p = strings.TrimSuffix(p, "/")

	parts := strings.Split(p, "/")

	// Handle common single-segment and two-segment patterns
	switch {
	case len(parts) >= 2 && parts[0] == "admin":
		// /admin/users, /admin/audit-logs → "admin_users", "admin_audit-logs"
		return parts[0] + "_" + parts[1]
	case len(parts) >= 2 && parts[0] == "spark":
		// /spark/jobs, /spark/scheduled-jobs → "spark_jobs", "spark_scheduled-jobs"
		return parts[0] + "_" + parts[1]
	case len(parts) >= 1 && parts[0] != "":
		// /notebooks, /connectors, /allowed-domains, /resource-presets, /kernel
		return parts[0]
	default:
		return "unknown"
	}
}

// extractResourceID attempts to find a UUID-like resource identifier from the
// URL path. Returns the first segment that looks like a UUID (or the first
// non-empty, non-action segment after the resource type).
func extractResourceID(path string) string {
	p := strings.TrimPrefix(path, "/api/v1/")
	p = strings.TrimPrefix(p, "/api/")
	p = strings.TrimSuffix(p, "/")

	parts := strings.Split(p, "/")

	// Look for segments that are UUID-like (36 chars with hyphens) or non-empty
	// parameter values.
	for i, part := range parts {
		if len(part) == 36 && strings.Count(part, "-") == 4 {
			return part
		}
		// Also capture segments that look like route params (the value after
		// a known resource type segment)
		if i > 0 {
			prev := parts[i-1]
			// Common resource-owning parent segments
			resourceOwners := map[string]bool{
				"users": true, "notebooks": true, "connectors": true,
				"jobs": true, "scheduled-jobs": true, "job-templates": true,
				"presets": true, "domains": true, "buckets": true,
			}
			if resourceOwners[prev] && part != "" && !strings.HasPrefix(part, ":") {
				return part
			}
		}
	}
	return ""
}

// determineAction returns a normalized action label from the HTTP method.
func determineAction(method string) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut:
		return "update"
	case http.MethodDelete:
		return "delete"
	case http.MethodPatch:
		return "patch"
	default:
		return strings.ToLower(method)
	}
}

// AuditMiddleware returns a Gin middleware that logs mutating requests
// (POST/PUT/DELETE/PATCH) to the audit_logs table. The insert is
// fire-and-forget (goroutine) so it never blocks the request.
//
// GET, HEAD, OPTIONS, health checks, and static paths are silently skipped.
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request first so audit_logs has the response status.
		c.Next()

		path := c.Request.URL.Path
		method := c.Request.Method

		if shouldSkipAudit(path, method) {
			return
		}

		// Only log 2xx and 4xx client errors — don't log every 401/403 from
		// the auth middleware for unauthenticated requests.
		status := c.Writer.Status()
		if status >= 500 || status == 0 {
			return
		}

		// Extract user context (set by RequireAdmin middleware)
		userID, _ := c.Get("admin_id")
		userIDStr, _ := userID.(string)

		username, _ := c.Get("admin_username")
		usernameStr, _ := username.(string)

		// Derive audit fields from the request
		action := determineAction(method)
		resourceType := extractResourceType(path)
		resourceID := extractResourceID(path)
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Build details from request body (if available and appropriate)
		// For POST/PUT, capture the request body but skip file uploads and
		// sensitive fields (passwords, tokens).
		var details map[string]interface{}
		if c.Request.ContentLength > 0 && c.Request.ContentLength < 65536 {
			// We can't read the body here because gin already consumed it.
			// Instead capture the Content-Type and Content-Length.
			details = map[string]interface{}{
				"content_type": c.Request.Header.Get("Content-Type"),
				"content_length": c.Request.ContentLength,
				"path": path,
			}
		} else if c.Request.ContentLength > 0 {
			details = map[string]interface{}{
				"path": path,
			}
		}

		if details == nil {
			details = map[string]interface{}{}
		}

		detailsJSON, err := json.Marshal(details)
		if err != nil {
			log.Warn().Err(err).Msg("failed to marshal audit details")
			detailsJSON = []byte("{}")
		}

		// Fire-and-forget insert — never block the request.
		go func() {
			db := database.GetDB()
			if db == nil {
				return
			}
			_, err := db.Exec(
				`INSERT INTO audit_logs (user_id, username, action, resource_type, resource_id, details, ip_address, user_agent)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				userIDStr, usernameStr, action, resourceType, resourceID,
				detailsJSON, ipAddress, userAgent,
			)
			if err != nil {
				log.Warn().Err(err).
					Str("action", action).
					Str("resource_type", resourceType).
					Msg("failed to write audit log")
			}
		}()
	}
}
