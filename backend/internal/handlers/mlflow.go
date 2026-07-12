package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rcn/rcn/backend/internal/services"
)

type MLflowHandler struct {
	svc *services.MLflowService
}

// NewMLflowHandler initializes the MLflow handler
func NewMLflowHandler(svc *services.MLflowService) *MLflowHandler {
	return &MLflowHandler{
		svc: svc,
	}
}

// Proxy routes the request to the MLflow service
// Route: ANY /api/v1/mlflow/*path
func (h *MLflowHandler) Proxy(c *gin.Context) {
	if !h.svc.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mlflow service is disabled"})
		return
	}

	// Retrieve path with query parameters if present
	path := c.Param("path")
	if c.Request.URL.RawQuery != "" {
		path = path + "?" + c.Request.URL.RawQuery
	}

	// Store client request headers in the context to be forwarded by the service
	ctx := context.WithValue(c.Request.Context(), "headers", c.Request.Header)

	// Forward request
	resp, err := h.svc.ProxyRequest(ctx, c.Request.Method, path, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Copy response headers to the client
	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Header(k, v)
		}
	}

	// Write response status and body
	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		// Response headers/status are already sent, so we can't send a new JSON error response.
		// Just finish the request.
		return
	}
}
