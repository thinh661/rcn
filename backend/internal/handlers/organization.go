package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rcn/rcn/backend/internal/services"
)

type OrgHandler struct {
	svc *services.OrgService
}

// NewOrgHandler creates a new instance of OrgHandler
func NewOrgHandler(svc *services.OrgService) *OrgHandler {
	return &OrgHandler{
		svc: svc,
	}
}

type CreateOrgRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id"`
}

type UpdateOrgRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type AssignUserRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// GetTree retrieves the organization tree
// GET /api/v1/admin/orgs/tree
func (h *OrgHandler) GetTree(c *gin.Context) {
	tree, err := h.svc.GetTree(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tree)
}

// ListOrgs lists all organizations sorted by name
// GET /api/v1/admin/orgs
func (h *OrgHandler) ListOrgs(c *gin.Context) {
	orgs, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orgs)
}

// CreateOrg creates a new organization
// POST /api/v1/admin/orgs
func (h *OrgHandler) CreateOrg(c *gin.Context) {
	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	org, err := h.svc.Create(c.Request.Context(), req.Name, req.Description, req.ParentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, org)
}

// GetOrg retrieves an organization by ID
// GET /api/v1/admin/orgs/:id
func (h *OrgHandler) GetOrg(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id is required"})
		return
	}

	org, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, org)
}

// UpdateOrg updates an organization's details
// PUT /api/v1/admin/orgs/:id
func (h *OrgHandler) UpdateOrg(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id is required"})
		return
	}

	var req UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.svc.Update(c.Request.Context(), id, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "organization updated successfully"})
}

// DeleteOrg deletes an organization by ID
// DELETE /api/v1/admin/orgs/:id
func (h *OrgHandler) DeleteOrg(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id is required"})
		return
	}

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "organization deleted successfully"})
}

// AssignUser assigns a user to an organization
// POST /api/v1/admin/orgs/:id/assign
func (h *OrgHandler) AssignUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id is required"})
		return
	}

	var req AssignUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.svc.AssignUser(c.Request.Context(), req.UserID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user assigned to organization successfully"})
}

// GetUsers retrieves all users assigned to an organization
// GET /api/v1/admin/orgs/:id/users
func (h *OrgHandler) GetUsers(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization id is required"})
		return
	}

	users, err := h.svc.GetUsers(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}
