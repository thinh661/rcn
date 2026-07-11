package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rcn/rcn/backend/internal/services"
)

type NotebookSchedulerHandler struct {
	svc *services.NotebookSchedulerService
}

// NewNotebookSchedulerHandler creates a new instance of NotebookSchedulerHandler
func NewNotebookSchedulerHandler(svc *services.NotebookSchedulerService) *NotebookSchedulerHandler {
	return &NotebookSchedulerHandler{
		svc: svc,
	}
}

type CreateNotebookScheduleRequest struct {
	NotebookID        string `json:"notebook_id" binding:"required"`
	Schedule          string `json:"schedule" binding:"required"`
	ExportFormat      string `json:"export_format" binding:"required"`
	NotificationEmail string `json:"notification_email"`
}

type UpdateNotebookScheduleRequest struct {
	Schedule          string `json:"schedule" binding:"required"`
	ExportFormat      string `json:"export_format" binding:"required"`
	NotificationEmail string `json:"notification_email"`
}

// ListSchedules lists all notebook schedules for the authenticated user
// GET /api/v1/notebook-schedules
func (h *NotebookSchedulerHandler) ListSchedules(c *gin.Context) {
	userID := c.GetString("admin_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	schedules, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedules)
}

// CreateSchedule creates a new notebook schedule
// POST /api/v1/notebook-schedules
func (h *NotebookSchedulerHandler) CreateSchedule(c *gin.Context) {
	userID := c.GetString("admin_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	var req CreateNotebookScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schedule, err := h.svc.Create(c.Request.Context(), req.NotebookID, userID, req.Schedule, req.ExportFormat, req.NotificationEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

// GetSchedule gets a notebook schedule by its ID
// GET /api/v1/notebook-schedules/:id
func (h *NotebookSchedulerHandler) GetSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "schedule id is required"})
		return
	}

	schedule, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// UpdateSchedule updates a notebook schedule by ID
// PUT /api/v1/notebook-schedules/:id
func (h *NotebookSchedulerHandler) UpdateSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "schedule id is required"})
		return
	}

	var req UpdateNotebookScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.svc.Update(c.Request.Context(), id, req.Schedule, req.ExportFormat, req.NotificationEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notebook schedule updated successfully"})
}

// DeleteSchedule deletes a notebook schedule by ID
// DELETE /api/v1/notebook-schedules/:id
func (h *NotebookSchedulerHandler) DeleteSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "schedule id is required"})
		return
	}

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notebook schedule deleted successfully"})
}

// ListRuns lists all execution runs for a specific notebook schedule
// GET /api/v1/notebook-schedules/:id/runs
func (h *NotebookSchedulerHandler) ListRuns(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "schedule id is required"})
		return
	}

	runs, err := h.svc.ListRuns(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runs)
}
