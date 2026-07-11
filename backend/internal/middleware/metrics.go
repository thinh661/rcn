package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/services"
)

// MetricsMiddleware records HTTP request metrics in the collector.
// It must be registered AFTER gin.Recovery() and before all route handlers.
func MetricsMiddleware(collector *services.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		collector.RecordHTTPRequest(method, path, status, duration)

		// Skip metrics endpoint itself to avoid infinite recording
		if path != "/api/v1/admin/metrics" {
			log.Debug().
				Str("method", method).
				Str("path", path).
				Int("status", status).
				Dur("duration", duration).
				Msg("recorded metric")
		}
	}
}
