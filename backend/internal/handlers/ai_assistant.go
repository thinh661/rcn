package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/services"
)

// AIAssistantHandler exposes AI chat completions.
type AIAssistantHandler struct {
	svc *services.AIAssistantService
}

// NewAIAssistantHandler creates a new AI assistant handler.
func NewAIAssistantHandler(svc *services.AIAssistantService) *AIAssistantHandler {
	return &AIAssistantHandler{svc: svc}
}

// Ask handles a prompt to the AI assistant.
// POST /api/v1/ai/ask
func (h *AIAssistantHandler) Ask(c *gin.Context) {
	if !h.svc.IsEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI assistant is not enabled"})
		return
	}

	var req struct {
		Prompt      string `json:"prompt" binding:"required"`
		NotebookID  string `json:"notebook_id"`
		CodeContext string `json:"code_context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Ask(c.Request.Context(), req.Prompt, req.NotebookID, req.CodeContext)
	if err != nil {
		log.Error().Err(err).Msg("AI assistant query failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI query failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// GetConfig returns the AI assistant configuration status.
// GET /api/v1/ai/config
func (h *AIAssistantHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled":  h.svc.IsEnabled(),
		"provider": "configured",
	})
}
