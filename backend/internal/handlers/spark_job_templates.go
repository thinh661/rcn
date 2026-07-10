package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/services"
)

// SparkJobTemplateHandler exposes REST endpoints for managing reusable Spark job templates.
type SparkJobTemplateHandler struct {
	templateSvc *services.SparkJobTemplateService
	jobSvc      *services.SparkJobService
}

// NewSparkJobTemplateHandler creates a new SparkJobTemplateHandler.
// The jobSvc is required for the POST /:id/run endpoint; pass nil if not available.
func NewSparkJobTemplateHandler(templateSvc *services.SparkJobTemplateService, jobSvc *services.SparkJobService) *SparkJobTemplateHandler {
	return &SparkJobTemplateHandler{
		templateSvc: templateSvc,
		jobSvc:      jobSvc,
	}
}

// ListTemplates returns all job templates. Admins see all; users see their own.
// GET /api/v1/spark/job-templates
func (h *SparkJobTemplateHandler) ListTemplates(c *gin.Context) {
	userID := adminID(c)
	showAll := c.Query("all") == "true"

	templates, err := h.templateSvc.List(c.Request.Context(), userID, showAll)
	if err != nil {
		log.Warn().Err(err).Msg("list spark job templates failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list job templates"})
		return
	}
	if templates == nil {
		templates = []services.SparkJobTemplate{}
	}
	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// CreateTemplate creates a new reusable job template.
// POST /api/v1/spark/job-templates
func (h *SparkJobTemplateHandler) CreateTemplate(c *gin.Context) {
	userID := adminID(c)

	var req services.CreateJobTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.templateSvc.Create(c.Request.Context(), &req, userID)
	if err != nil {
		log.Warn().Err(err).Msg("create spark job template failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"template": t})
}

// GetTemplate returns a single job template by ID.
// GET /api/v1/spark/job-templates/:id
func (h *SparkJobTemplateHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")

	t, err := h.templateSvc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job template not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": t})
}

// UpdateTemplate updates a job template's configuration.
// PUT /api/v1/spark/job-templates/:id
func (h *SparkJobTemplateHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")

	var req services.UpdateJobTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.templateSvc.Update(c.Request.Context(), id, &req)
	if err != nil {
		log.Warn().Err(err).Str("template_id", id).Msg("update spark job template failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"template": t})
}

// DeleteTemplate deletes a job template.
// DELETE /api/v1/spark/job-templates/:id
func (h *SparkJobTemplateHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")

	if err := h.templateSvc.Delete(c.Request.Context(), id); err != nil {
		log.Warn().Err(err).Str("template_id", id).Msg("delete spark job template failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete job template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job template deleted"})
}

// RunTemplate submits a Spark job from a template, with optional overrides.
// POST /api/v1/spark/job-templates/:id/run
func (h *SparkJobTemplateHandler) RunTemplate(c *gin.Context) {
	if h.jobSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Spark job service not available"})
		return
	}

	userID := adminID(c)
	id := c.Param("id")

	t, err := h.templateSvc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job template not found"})
		return
	}

	// Parse optional overrides.
	var overrides *services.RunJobFromTemplateRequest
	var req services.RunJobFromTemplateRequest
	if err := c.ShouldBindJSON(&req); err == nil {
		// Only use overrides if the body is a valid JSON object.
		// An empty JSON body `{}` is valid and results in zero overrides.
		overrides = &req
	}

	submitReq := t.ToSubmitJobRequest(overrides)

	job, err := h.jobSvc.Submit(c.Request.Context(), submitReq, userID)
	if err != nil {
		log.Warn().Err(err).Str("template_id", id).Msg("run spark job from template failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"job": job})
}
