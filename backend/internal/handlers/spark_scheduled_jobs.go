package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/services"
)

// SparkScheduledJobHandler exposes REST endpoints for managing scheduled Spark jobs.
type SparkScheduledJobHandler struct {
	svc *services.SparkScheduledJobService
}

// NewSparkScheduledJobHandler creates a new SparkScheduledJobHandler.
func NewSparkScheduledJobHandler(svc *services.SparkScheduledJobService) *SparkScheduledJobHandler {
	return &SparkScheduledJobHandler{svc: svc}
}

// ListScheduledJobs returns all scheduled Spark jobs. Admins see all; users see their own.
// GET /api/v1/spark/scheduled-jobs
func (h *SparkScheduledJobHandler) ListScheduledJobs(c *gin.Context) {
	userID := adminID(c)
	showAll := c.Query("all") == "true"

	jobs, err := h.svc.List(c.Request.Context(), userID, showAll)
	if err != nil {
		log.Warn().Err(err).Msg("list scheduled spark jobs failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list scheduled jobs"})
		return
	}
	if jobs == nil {
		jobs = []services.ScheduledSparkJob{}
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// CreateScheduledJob creates a new scheduled Spark batch job.
// POST /api/v1/spark/scheduled-jobs
func (h *SparkScheduledJobHandler) CreateScheduledJob(c *gin.Context) {
	userID := adminID(c)

	var req services.CreateScheduledJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.svc.Create(c.Request.Context(), &req, userID)
	if err != nil {
		log.Warn().Err(err).Msg("create scheduled spark job failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"job": job})
}

// GetScheduledJob returns details for a single scheduled Spark job.
// GET /api/v1/spark/scheduled-jobs/:id
func (h *SparkScheduledJobHandler) GetScheduledJob(c *gin.Context) {
	id := c.Param("id")

	job, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scheduled job not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": job})
}

// UpdateScheduledJob updates a scheduled Spark job's configuration.
// PUT /api/v1/spark/scheduled-jobs/:id
func (h *SparkScheduledJobHandler) UpdateScheduledJob(c *gin.Context) {
	id := c.Param("id")

	var req services.UpdateScheduledJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		log.Warn().Err(err).Str("job_id", id).Msg("update scheduled spark job failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

// DeleteScheduledJob deletes a scheduled Spark job.
// DELETE /api/v1/spark/scheduled-jobs/:id
func (h *SparkScheduledJobHandler) DeleteScheduledJob(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		log.Warn().Err(err).Str("job_id", id).Msg("delete scheduled spark job failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete scheduled job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "scheduled job deleted"})
}

// ToggleScheduledJob enables or disables a scheduled Spark job.
// PATCH /api/v1/spark/scheduled-jobs/:id/toggle
func (h *SparkScheduledJobHandler) ToggleScheduledJob(c *gin.Context) {
	id := c.Param("id")

	job, err := h.svc.Toggle(c.Request.Context(), id)
	if err != nil {
		log.Warn().Err(err).Str("job_id", id).Msg("toggle scheduled spark job failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}
