package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rcn/rcn/backend/internal/services"
)

// DeltaSharingHandler handles all HTTP routing/requests for Delta Sharing
type DeltaSharingHandler struct {
	svc *services.DeltaSharingService
}

// NewDeltaSharingHandler creates a new DeltaSharingHandler
func NewDeltaSharingHandler(svc *services.DeltaSharingService) *DeltaSharingHandler {
	return &DeltaSharingHandler{
		svc: svc,
	}
}

// CreateShareRequest defines the request body structure for creating a new share
type CreateShareRequest struct {
	Name      string `json:"name" binding:"required"`
	Table     string `json:"table" binding:"required"`
	ShareWith string `json:"share_with"`
}

// ListShares lists all delta shares
// GET /api/v1/delta-sharing/shares
func (h *DeltaSharingHandler) ListShares(c *gin.Context) {
	adminID := c.GetString("admin_id")
	if adminID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	shares, err := h.svc.ListShares(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, shares)
}

// CreateShare creates a new delta share
// POST /api/v1/delta-sharing/shares
func (h *DeltaSharingHandler) CreateShare(c *gin.Context) {
	adminID := c.GetString("admin_id")
	if adminID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	var req CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	share, err := h.svc.CreateShare(c.Request.Context(), req.Name, req.Table, req.ShareWith)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, share)
}

// RevokeShare revokes a share by status update
// DELETE /api/v1/delta-sharing/shares/:id
func (h *DeltaSharingHandler) RevokeShare(c *gin.Context) {
	adminID := c.GetString("admin_id")
	if adminID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	err := h.svc.RevokeShare(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "share revoked successfully"})
}

// GetConfig returns configuration info for Delta Sharing
// GET /api/v1/delta-sharing/config
func (h *DeltaSharingHandler) GetConfig(c *gin.Context) {
	// Delta sharing config config endpoint
	adminID := c.GetString("admin_id")
	if adminID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	c.JSON(http.StatusOK, h.svc.GetConfig())
}
