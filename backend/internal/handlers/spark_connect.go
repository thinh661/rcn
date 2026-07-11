package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rcn/rcn/backend/internal/services"
)

type SparkConnectHandler struct {
	svc *services.SparkConnectService
}

// NewSparkConnectHandler creates a new instance of SparkConnectHandler
func NewSparkConnectHandler(svc *services.SparkConnectService) *SparkConnectHandler {
	return &SparkConnectHandler{
		svc: svc,
	}
}

// GetConfig returns the connection info for Spark Connect
// GET /api/v1/spark-connect/config
func (h *SparkConnectHandler) GetConfig(c *gin.Context) {
	config := h.svc.GetConfig()
	c.JSON(http.StatusOK, config)
}

// CreateSession creates a new session for the current authenticated user
// POST /api/v1/spark-connect/sessions
func (h *SparkConnectHandler) CreateSession(c *gin.Context) {
	// Extract admin_id as userID from the Gin context as required
	userID := c.GetString("admin_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	session, err := h.svc.CreateSession(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// ListSessions retrieves all sessions of the authenticated user
// GET /api/v1/spark-connect/sessions
func (h *SparkConnectHandler) ListSessions(c *gin.Context) {
	userID := c.GetString("admin_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	sessions, err := h.svc.ListSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// CloseSession closes a specific session by its ID
// DELETE /api/v1/spark-connect/sessions/:id
func (h *SparkConnectHandler) CloseSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}

	userID := c.GetString("admin_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	err := h.svc.CloseSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session closed successfully"})
}
