package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/services"
)

// GitIntegrationHandler handles notebook git versioning endpoints.
type GitIntegrationHandler struct {
	svc *services.GitIntegrationService
}

// NewGitIntegrationHandler creates a new GitIntegrationHandler.
func NewGitIntegrationHandler(svc *services.GitIntegrationService) *GitIntegrationHandler {
	return &GitIntegrationHandler{svc: svc}
}

// LinkNotebook links a notebook to a git repository.
// POST /api/v1/notebooks/:id/git/link
func (h *GitIntegrationHandler) LinkNotebook(c *gin.Context) {
	if !h.svc.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git integration is not enabled"})
		return
	}

	notebookID := c.Param("id")

	var req struct {
		RepoURL   string `json:"repo_url" binding:"required"`
		Branch    string `json:"branch"`
		FilePath  string `json:"file_path" binding:"required"`
		AuthToken string `json:"auth_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	link, err := h.svc.LinkNotebook(c.Request.Context(), notebookID, req.RepoURL, req.Branch, req.FilePath, req.AuthToken)
	if err != nil {
		log.Error().Err(err).Str("notebook_id", notebookID).Msg("failed to link notebook to git")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": link})
}

// GetLinkStatus returns the git sync status of a notebook.
// GET /api/v1/notebooks/:id/git/status
func (h *GitIntegrationHandler) GetLinkStatus(c *gin.Context) {
	if !h.svc.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git integration is not enabled"})
		return
	}

	notebookID := c.Param("id")

	status, err := h.svc.GetStatus(c.Request.Context(), notebookID)
	if err != nil {
		log.Error().Err(err).Str("notebook_id", notebookID).Msg("failed to get git status")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": status})
}

// CommitNotebook commits the current notebook version to git.
// POST /api/v1/notebooks/:id/git/commit
func (h *GitIntegrationHandler) CommitNotebook(c *gin.Context) {
	if !h.svc.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git integration is not enabled"})
		return
	}

	notebookID := c.Param("id")
	userID := c.GetString("admin_id")

	var req struct {
		Message string `json:"message"`
	}
	_ = c.ShouldBindJSON(&req)

	// Export notebook to JSON first.
	nbJSON, err := h.svc.ExportNotebookJSON(c.Request.Context(), notebookID)
	if err != nil {
		log.Error().Err(err).Str("notebook_id", notebookID).Msg("failed to export notebook for git commit")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Record the commit in the DB.
	result, err := h.svc.CommitNotebook(c.Request.Context(), notebookID, userID, req.Message)
	if err != nil {
		log.Error().Err(err).Str("notebook_id", notebookID).Msg("failed to commit notebook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     result,
		"notebook": nbJSON,
	})
}

// UnlinkNotebook removes the git link from a notebook.
// DELETE /api/v1/notebooks/:id/git/link
func (h *GitIntegrationHandler) UnlinkNotebook(c *gin.Context) {
	if !h.svc.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git integration is not enabled"})
		return
	}

	notebookID := c.Param("id")

	if err := h.svc.UnlinkNotebook(c.Request.Context(), notebookID); err != nil {
		log.Error().Err(err).Str("notebook_id", notebookID).Msg("failed to unlink notebook from git")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notebook unlinked from git"})
}
