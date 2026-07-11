package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/config"
	"github.com/rcn/rcn/backend/internal/services"
)

type ResourceUsageHandler struct {
	svc *services.ResourceUsageService
	cfg *config.Config
}

type SetCostRateRequest struct {
	RatePerUnit float64 `json:"rate_per_unit" binding:"required"`
	Unit        string  `json:"unit" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
}

// NewResourceUsageHandler creates a new instance of ResourceUsageHandler
func NewResourceUsageHandler(svc *services.ResourceUsageService, cfg *config.Config) *ResourceUsageHandler {
	return &ResourceUsageHandler{
		svc: svc,
		cfg: cfg,
	}
}

// ListAllUsage retrieves resource usage records for all users (admin-only)
// GET /api/v1/admin/resource-usage
func (h *ResourceUsageHandler) ListAllUsage(c *gin.Context) {
	userID := c.Query("user_id")

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	from, to, err := parseTimeRange(c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	records, total, err := h.svc.GetAllUsage(c.Request.Context(), from, to, userID, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to list all resource usage records")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve resource usage records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   records,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// MyUsage retrieves resource usage records for the authenticated user
// GET /api/v1/resource-usage/my
func (h *ResourceUsageHandler) MyUsage(c *gin.Context) {
	userID := c.GetString("admin_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	from, to, err := parseTimeRange(c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	records, total, err := h.svc.GetUserUsage(c.Request.Context(), userID, from, to, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("failed to retrieve user usage records")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve usage records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   records,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UsageSummary retrieves aggregated resource usage cost summary (admin-only)
// GET /api/v1/admin/resource-usage/summary
func (h *ResourceUsageHandler) UsageSummary(c *gin.Context) {
	from, to, err := parseTimeRange(c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	summary, err := h.svc.GetSummary(c.Request.Context(), from, to)
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve usage summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate usage summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// SetCostRate sets or updates the cost rate for a resource type (admin-only)
// PUT /api/v1/admin/cost-rates/:resource_type
func (h *ResourceUsageHandler) SetCostRate(c *gin.Context) {
	resourceType := c.Param("resource_type")
	if resourceType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_type parameter is required"})
		return
	}

	var req SetCostRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request body: %s", err.Error())})
		return
	}

	err := h.svc.UpsertCostRate(c.Request.Context(), resourceType, req.RatePerUnit, req.Unit, req.Currency)
	if err != nil {
		log.Error().Err(err).Str("resource_type", resourceType).Msg("failed to upsert cost rate")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert cost rate"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "cost rate updated successfully",
	})
}

// ListCostRates retrieves all configured cost rates (admin-only)
// GET /api/v1/admin/cost-rates
func (h *ResourceUsageHandler) ListCostRates(c *gin.Context) {
	rates, err := h.svc.GetCostRates()
	if err != nil {
		log.Error().Err(err).Msg("failed to retrieve cost rates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve cost rates"})
		return
	}

	c.JSON(http.StatusOK, rates)
}

// parseTimeRange parses from and to timestamps, defaulting to last 30 days and now respectively
func parseTimeRange(fromStr, toStr string) (time.Time, time.Time, error) {
	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			// fallback to common Date format (YYYY-MM-DD)
			from, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date format; must be RFC3339 or YYYY-MM-DD")
			}
		}
	} else {
		from = time.Now().AddDate(0, 0, -30)
	}

	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			// fallback to common Date format (YYYY-MM-DD)
			to, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' date format; must be RFC3339 or YYYY-MM-DD")
			}
		}
	} else {
		to = time.Now()
	}

	return from, to, nil
}
