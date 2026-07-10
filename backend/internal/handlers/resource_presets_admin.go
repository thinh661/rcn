package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ResourcePresetsAdminHandler provides CRUD admin endpoints for kernel resource presets.
// Phase 2 stub — presets stay in-memory; these endpoints are placeholders.
type ResourcePresetsAdminHandler struct {
	kernelHandler *LocalKernelHandler
}

// NewResourcePresetsAdminHandler creates a new ResourcePresetsAdminHandler.
func NewResourcePresetsAdminHandler(kh *LocalKernelHandler) *ResourcePresetsAdminHandler {
	return &ResourcePresetsAdminHandler{kernelHandler: kh}
}

// List returns all configured resource presets.
// GET /api/v1/admin/resource-presets
func (h *ResourcePresetsAdminHandler) List(c *gin.Context) {
	// Delegate to the kernel handler's existing ResourcePresets endpoint.
	h.kernelHandler.ResourcePresets(c)
}

// Upsert creates or updates a resource preset.
// POST /api/v1/admin/resource-presets
func (h *ResourcePresetsAdminHandler) Upsert(c *gin.Context) {
	log.Warn().Msg("ResourcePresetsAdminHandler.Upsert: stub — presets are in-memory only")
	c.JSON(http.StatusNotImplemented, gin.H{"error": "resource preset CRUD not implemented (Phase 2 stub)"})
}

// Delete removes a resource preset by ID.
// DELETE /api/v1/admin/resource-presets/:id
func (h *ResourcePresetsAdminHandler) Delete(c *gin.Context) {
	log.Warn().Msg("ResourcePresetsAdminHandler.Delete: stub — presets are in-memory only")
	c.JSON(http.StatusNotImplemented, gin.H{"error": "resource preset CRUD not implemented (Phase 2 stub)"})
}
