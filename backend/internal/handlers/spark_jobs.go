package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/services"
)

// SparkJobHandler exposes REST endpoints for managing Spark batch jobs.
type SparkJobHandler struct {
	svc *services.SparkJobService
}

func NewSparkJobHandler(svc *services.SparkJobService) *SparkJobHandler {
	return &SparkJobHandler{svc: svc}
}

// ListJobs returns all Spark jobs. Admins see all; users see their own.
// GET /api/v1/spark/jobs
func (h *SparkJobHandler) ListJobs(c *gin.Context) {
	userID := adminID(c)
	showAll := c.Query("all") == "true"

	jobs, err := h.svc.List(c.Request.Context(), userID, showAll)
	if err != nil {
		log.Warn().Err(err).Msg("list spark jobs failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}
	if jobs == nil {
		jobs = []services.SparkJob{}
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// SubmitJob creates a new Spark batch job.
// POST /api/v1/spark/jobs
func (h *SparkJobHandler) SubmitJob(c *gin.Context) {
	userID := adminID(c)

	var req services.SubmitJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.svc.Submit(c.Request.Context(), &req, userID)
	if err != nil {
		log.Warn().Err(err).Msg("submit spark job failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"job": job})
}

// GetJob returns details for a single Spark job.
// GET /api/v1/spark/jobs/:id
func (h *SparkJobHandler) GetJob(c *gin.Context) {
	id := c.Param("id")

	job, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"job": job})
}

// StopJob stops and deletes a Spark batch job.
// DELETE /api/v1/spark/jobs/:id
func (h *SparkJobHandler) StopJob(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.Stop(c.Request.Context(), id); err != nil {
		log.Warn().Err(err).Str("job_id", id).Msg("stop spark job failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop job"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "job stopped"})
}

// GetJobLogs returns driver logs for a Spark job.
// GET /api/v1/spark/jobs/:id/logs
func (h *SparkJobHandler) GetJobLogs(c *gin.Context) {
	id := c.Param("id")

	tail := 100
	if t := c.Query("tail"); t != "" {
		if parsed, err := parseInt(t); err == nil && parsed > 0 {
			tail = parsed
		}
	}

	logs, err := h.svc.GetLogs(c.Request.Context(), id, tail)
	if err != nil {
		log.Warn().Err(err).Str("job_id", id).Msg("get job logs failed")
		c.JSON(http.StatusNotFound, gin.H{"error": "logs not available"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// adminID extracts the admin_id from the Gin context (set by RequireAdmin middleware).
func adminID(c *gin.Context) string {
	if id, ok := c.Get("admin_id"); ok {
		return id.(string)
	}
	return ""
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}


