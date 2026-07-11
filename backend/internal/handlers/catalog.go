package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/database"
	"github.com/rcn/rcn/backend/internal/services"
)

// CatalogHandler serves data catalog endpoints.
type CatalogHandler struct {
	svc *services.CatalogService
}

// NewCatalogHandler creates a new catalog handler.
func NewCatalogHandler(svc *services.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

// GetTree returns the full catalog tree.
// GET /api/v1/catalog/tree
func (h *CatalogHandler) GetTree(c *gin.Context) {
	tree, err := h.svc.GetTree(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get catalog tree")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get catalog tree"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tree})
}

// Search searches catalog entries.
// GET /api/v1/catalog/search?q=&limit=
func (h *CatalogHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	entries, err := h.svc.Search(c.Request.Context(), query, limit)
	if err != nil {
		log.Error().Err(err).Msg("catalog search failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": entries})
}

// GetEntry returns a catalog entry by ID.
// GET /api/v1/catalog/:id
func (h *CatalogHandler) GetEntry(c *gin.Context) {
	id := c.Param("id")

	entry, err := h.svc.GetEntry(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entry not found"})
		return
	}

	// Also get children for tree navigation
	var children []map[string]interface{}
	// Fetch children via direct query
	db := database.GetDB()
	rows, _ := db.QueryContext(c.Request.Context(),
		`SELECT id, name, type FROM data_catalog WHERE parent_id = $1 ORDER BY type, name`, id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var childID, childName, childType string
			if rows.Scan(&childID, &childName, &childType) == nil {
				children = append(children, map[string]interface{}{
					"id": childID, "name": childName, "type": childType,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     entry,
		"children": children,
	})
}

// UpsertEntry creates or updates a catalog entry.
// POST /api/v1/catalog
func (h *CatalogHandler) UpsertEntry(c *gin.Context) {
	var req struct {
		Name     string          `json:"name" binding:"required"`
		Type     string          `json:"type" binding:"required"`
		ParentID *string         `json:"parent_id"`
		Metadata json.RawMessage `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := h.svc.UpsertEntry(c.Request.Context(), req.Name, req.Type, req.ParentID, req.Metadata)
	if err != nil {
		log.Error().Err(err).Msg("failed to upsert catalog entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": entry})
}

// DeleteEntry deletes a catalog entry.
// DELETE /api/v1/catalog/:id
func (h *CatalogHandler) DeleteEntry(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteEntry(c.Request.Context(), id); err != nil {
		log.Error().Err(err).Str("id", id).Msg("failed to delete catalog entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
