package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rcn/rcn/backend/internal/services"
)

type WorkflowsHandler struct {
	svc *services.WorkflowService
}

// NewWorkflowsHandler creates a new instance of WorkflowsHandler
func NewWorkflowsHandler(svc *services.WorkflowService) *WorkflowsHandler {
	return &WorkflowsHandler{
		svc: svc,
	}
}

type CreateWorkflowRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Schedule    string `json:"schedule"`
}

type UpdateWorkflowRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Schedule    string `json:"schedule"`
}

type AddTaskRequest struct {
	Name      string          `json:"name" binding:"required"`
	TaskType  string          `json:"task_type" binding:"required"`
	Config    json.RawMessage `json:"config" binding:"required"`
	DependsOn []string        `json:"depends_on"`
}

// ListWorkflows retrieves all workflows
// GET /api/v1/workflows
func (h *WorkflowsHandler) ListWorkflows(c *gin.Context) {
	workflows, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, workflows)
}

// CreateWorkflow creates a new workflow
// POST /api/v1/workflows
func (h *WorkflowsHandler) CreateWorkflow(c *gin.Context) {
	userID := c.GetString("admin_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workflow, err := h.svc.Create(c.Request.Context(), userID, req.Name, req.Description, req.Schedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, workflow)
}

// GetWorkflow retrieves a workflow by ID
// GET /api/v1/workflows/:id
func (h *WorkflowsHandler) GetWorkflow(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow id is required"})
		return
	}

	workflow, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// UpdateWorkflow updates a workflow's information
// PUT /api/v1/workflows/:id
func (h *WorkflowsHandler) UpdateWorkflow(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow id is required"})
		return
	}

	var req UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.svc.Update(c.Request.Context(), id, req.Name, req.Description, req.Schedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workflow updated successfully"})
}

// DeleteWorkflow deletes a workflow by ID
// DELETE /api/v1/workflows/:id
func (h *WorkflowsHandler) DeleteWorkflow(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow id is required"})
		return
	}

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "workflow deleted successfully"})
}

// ListTasks lists all tasks for a workflow
// GET /api/v1/workflows/:id/tasks
func (h *WorkflowsHandler) ListTasks(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow id is required"})
		return
	}

	tasks, err := h.svc.ListTasks(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// AddTask adds a task to a workflow
// POST /api/v1/workflows/:id/tasks
func (h *WorkflowsHandler) AddTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow id is required"})
		return
	}

	var req AddTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.svc.AddTask(c.Request.Context(), id, req.Name, req.TaskType, req.Config, req.DependsOn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// RemoveTask removes a task from a workflow
// DELETE /api/v1/workflows/:id/tasks/:taskId
func (h *WorkflowsHandler) RemoveTask(c *gin.Context) {
	taskId := c.Param("taskId")
	if taskId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id is required"})
		return
	}

	err := h.svc.RemoveTask(c.Request.Context(), taskId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task removed successfully"})
}

// TriggerRun triggers a run for the workflow
// POST /api/v1/workflows/:id/run
func (h *WorkflowsHandler) TriggerRun(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow id is required"})
		return
	}

	userID := c.GetString("admin_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: admin_id is required"})
		return
	}

	run, err := h.svc.TriggerRun(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, run)
}

// ListRuns retrieves all runs for a workflow
// GET /api/v1/workflows/:id/runs
func (h *WorkflowsHandler) ListRuns(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workflow id is required"})
		return
	}

	runs, err := h.svc.ListRuns(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runs)
}

// GetRun retrieves a specific workflow run by its run ID
// GET /api/v1/workflows/runs/:runId
func (h *WorkflowsHandler) GetRun(c *gin.Context) {
	runId := c.Param("runId")
	if runId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "run id is required"})
		return
	}

	run, err := h.svc.GetRun(c.Request.Context(), runId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, run)
}
