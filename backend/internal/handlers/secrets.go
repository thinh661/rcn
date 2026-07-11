package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/services"
)

// SecretHandler manages app secrets via the encrypted vault. All routes are
// superadmin-only (wired with middleware.RequireSuperAdmin()).
type SecretHandler struct {
	vault *services.SecretVault
}

// NewSecretHandler creates a handler backed by the given vault.
func NewSecretHandler(vault *services.SecretVault) *SecretHandler {
	return &SecretHandler{vault: vault}
}

// ListSecrets returns all secret key names (no values).
// GET /api/v1/admin/secrets
func (h *SecretHandler) ListSecrets(c *gin.Context) {
	keys, err := h.vault.ListSecrets()
	if err != nil {
		log.Error().Err(err).Msg("list secrets failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list secrets"})
		return
	}
	if keys == nil {
		keys = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

// GetSecret returns a single secret's decrypted value.
// GET /api/v1/admin/secrets/:key
func (h *SecretHandler) GetSecret(c *gin.Context) {
	key := c.Param("key")
	value, err := h.vault.GetSecret(key)
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("get secret failed")
		c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "value": value})
}

type setSecretRequest struct {
	Value string `json:"value"`
}

// SetSecret creates or updates a secret.
// PUT /api/v1/admin/secrets/:key
func (h *SecretHandler) SetSecret(c *gin.Context) {
	key := c.Param("key")

	var req setSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.vault.SetSecret(key, req.Value); err != nil {
		log.Error().Err(err).Str("key", key).Msg("set secret failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save secret"})
		return
	}

	log.Info().Str("key", key).Str("admin", c.GetString("admin_id")).Msg("secret set")
	c.JSON(http.StatusOK, gin.H{"key": key, "updated": true})
}

// DeleteSecret removes a secret.
// DELETE /api/v1/admin/secrets/:key
func (h *SecretHandler) DeleteSecret(c *gin.Context) {
	key := c.Param("key")
	if err := h.vault.DeleteSecret(key); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("delete secret failed")
		c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
		return
	}
	log.Info().Str("key", key).Str("admin", c.GetString("admin_id")).Msg("secret deleted")
	c.JSON(http.StatusOK, gin.H{"deleted": key})
}

// RotateSecrets re-encrypts all secrets and increments rotation_version.
// POST /api/v1/admin/secrets/rotate
func (h *SecretHandler) RotateSecrets(c *gin.Context) {
	count, err := h.vault.RotateAllSecrets()
	if err != nil {
		log.Error().Err(err).Int("rotated", count).Msg("secret rotation failed partway through")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "rotation failed partway through",
			"rotated":    count,
		})
		return
	}
	log.Info().Int("count", count).Str("admin", c.GetString("admin_id")).Msg("secrets rotated")
	c.JSON(http.StatusOK, gin.H{"rotated": count, "message": "secrets re-encrypted"})
}
