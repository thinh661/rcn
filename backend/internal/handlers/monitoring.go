package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/config"
	"github.com/rcn/rcn/backend/internal/services"
)

// MonitoringHandler exposes health check, metrics, and system info endpoints.
type MonitoringHandler struct {
	metrics  *services.MetricsCollector
	checker  *services.HealthChecker
	cfg      *config.Config
}

// NewMonitoringHandler creates a new monitoring handler.
func NewMonitoringHandler(
	metrics *services.MetricsCollector,
	checker *services.HealthChecker,
	cfg *config.Config,
) *MonitoringHandler {
	return &MonitoringHandler{
		metrics: metrics,
		checker: checker,
		cfg:     cfg,
	}
}

// GetHealth is a lightweight liveness check for K8s.
// GET /health
func (h *MonitoringHandler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   h.cfg.ServiceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetSystemHealth returns detailed health of all dependencies.
// GET /api/v1/admin/system/health
func (h *MonitoringHandler) GetSystemHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	checks := h.checker.RunChecks(ctx)

	overallStatus := "ok"
	for _, check := range checks {
		if check.Status == "down" {
			overallStatus = "down"
			break
		}
		if check.Status == "degraded" {
			overallStatus = "degraded"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  overallStatus,
		"checks":  checks,
		"uptime":  h.metrics.GetUptime().Round(time.Second).String(),
		"version": h.cfg.ServiceName,
	})
}

// GetMetrics returns snapshot of all platform metrics.
// GET /api/v1/admin/metrics
func (h *MonitoringHandler) GetMetrics(c *gin.Context) {
	snapshot := h.metrics.Snapshot()
	snapshot["service"] = h.cfg.ServiceName
	c.JSON(http.StatusOK, snapshot)
}

// GetSystemInfo returns system metadata.
// GET /api/v1/admin/system/info
func (h *MonitoringHandler) GetSystemInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	info, err := h.checker.SystemInfo(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get system info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get system info"})
		return
	}

	info["service"] = h.cfg.ServiceName
	info["uptime"] = h.metrics.GetUptime().Round(time.Second).String()

	c.JSON(http.StatusOK, info)
}
