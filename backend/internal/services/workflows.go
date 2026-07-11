package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rcn/rcn/backend/internal/database"
)

type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UserID      string    `json:"user_id"`
	Schedule    string    `json:"schedule"`
	Status      string    `json:"status"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WorkflowTask struct {
	ID             string          `json:"id"`
	WorkflowID     string          `json:"workflow_id"`
	Name           string          `json:"name"`
	TaskType       string          `json:"task_type"`
	Config         json.RawMessage `json:"config"`
	DependsOn      []string        `json:"depends_on"`
	RetryCount     int             `json:"retry_count"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	TaskOrder      int             `json:"task_order"`
	CreatedAt      time.Time       `json:"created_at"`
}

type WorkflowRun struct {
	ID             string          `json:"id"`
	WorkflowID     string          `json:"workflow_id"`
	Status         string          `json:"status"`
	TriggeredBy    string          `json:"triggered_by"`
	ErrorMessage   string          `json:"error_message"`
	TaskStatuses   json.RawMessage `json:"task_statuses"`
	StartedAt      *time.Time      `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at"`
	CreatedAt      time.Time       `json:"created_at"`
}

type WorkflowService struct{}

// NewWorkflowService initializes and returns a new WorkflowService
func NewWorkflowService() *WorkflowService {
	return &WorkflowService{}
}

// Create inserts a new workflow record into the database
func (s *WorkflowService) Create(ctx context.Context, userID, name, desc, schedule string) (*Workflow, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	id := uuid.New().String()
	query := `
		INSERT INTO workflows (id, name, description, user_id, schedule, status, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'active', true, NOW(), NOW())
		RETURNING id, name, description, user_id, schedule, status, enabled, created_at, updated_at
	`

	var w Workflow
	err := db.QueryRowContext(ctx, query, id, name, desc, schedule).Scan(
		&w.ID,
		&w.Name,
		&w.Description,
		&w.UserID,
		&w.Schedule,
		&w.Status,
		&w.Enabled,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	return &w, nil
}

// List retrieves all workflows sorted by creation time descending
func (s *WorkflowService) List(ctx context.Context) ([]Workflow, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, name, description, user_id, schedule, status, enabled, created_at, updated_at
		FROM workflows
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()

	var list []Workflow
	for rows.Next() {
		var w Workflow
		err := rows.Scan(
			&w.ID,
			&w.Name,
			&w.Description,
			&w.UserID,
			&w.Schedule,
			&w.Status,
			&w.Enabled,
			&w.CreatedAt,
			&w.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}
		list = append(list, w)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// Get retrieves a workflow by its ID
func (s *WorkflowService) Get(ctx context.Context, id string) (*Workflow, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, name, description, user_id, schedule, status, enabled, created_at, updated_at
		FROM workflows
		WHERE id = $1
	`

	var w Workflow
	err := db.QueryRowContext(ctx, query, id).Scan(
		&w.ID,
		&w.Name,
		&w.Description,
		&w.UserID,
		&w.Schedule,
		&w.Status,
		&w.Enabled,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return &w, nil
}

// Update updates an existing workflow
func (s *WorkflowService) Update(ctx context.Context, id, name, desc, schedule string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `
		UPDATE workflows
		SET name = $2, description = $3, schedule = $4, updated_at = NOW()
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, id, name, desc, schedule)
	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("workflow not found")
	}

	return nil
}

// Delete deletes a workflow by its ID
func (s *WorkflowService) Delete(ctx context.Context, id string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `DELETE FROM workflows WHERE id = $1`
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("workflow not found")
	}

	return nil
}

// AddTask adds a new task to a workflow
func (s *WorkflowService) AddTask(ctx context.Context, workflowID, name, taskType string, config json.RawMessage, dependsOn []string) (*WorkflowTask, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	id := uuid.New().String()
	query := `
		INSERT INTO workflow_tasks (id, workflow_id, name, task_type, config, depends_on, retry_count, timeout_seconds, task_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 3, 3600, 0, NOW())
		RETURNING id, workflow_id, name, task_type, config, depends_on, retry_count, timeout_seconds, task_order, created_at
	`

	var t WorkflowTask
	err := db.QueryRowContext(ctx, query, id, workflowID, name, taskType, config, pq.Array(dependsOn)).Scan(
		&t.ID,
		&t.WorkflowID,
		&t.Name,
		&t.TaskType,
		&t.Config,
		pq.Array(&t.DependsOn),
		&t.RetryCount,
		&t.TimeoutSeconds,
		&t.TaskOrder,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add task to workflow: %w", err)
	}

	return &t, nil
}

// RemoveTask removes a task from a workflow by its ID
func (s *WorkflowService) RemoveTask(ctx context.Context, taskID string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `DELETE FROM workflow_tasks WHERE id = $1`
	result, err := db.ExecContext(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("failed to remove workflow task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("workflow task not found")
	}

	return nil
}

// ListTasks retrieves all tasks belonging to a workflow
func (s *WorkflowService) ListTasks(ctx context.Context, workflowID string) ([]WorkflowTask, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, workflow_id, name, task_type, config, depends_on, retry_count, timeout_seconds, task_order, created_at
		FROM workflow_tasks
		WHERE workflow_id = $1
		ORDER BY task_order ASC, created_at ASC
	`

	rows, err := db.QueryContext(ctx, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow tasks: %w", err)
	}
	defer rows.Close()

	var list []WorkflowTask
	for rows.Next() {
		var t WorkflowTask
		err := rows.Scan(
			&t.ID,
			&t.WorkflowID,
			&t.Name,
			&t.TaskType,
			&t.Config,
			pq.Array(&t.DependsOn),
			&t.RetryCount,
			&t.TimeoutSeconds,
			&t.TaskOrder,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow task: %w", err)
		}
		list = append(list, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// TriggerRun triggers a run for the workflow
func (s *WorkflowService) TriggerRun(ctx context.Context, workflowID, triggeredBy string) (*WorkflowRun, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	id := uuid.New().String()
	query := `
		INSERT INTO workflow_runs (id, workflow_id, status, triggered_by, error_message, task_statuses, started_at, finished_at, created_at)
		VALUES ($1, $2, 'running', $3, '', '{}', NOW(), NULL, NOW())
		RETURNING id, workflow_id, status, triggered_by, error_message, task_statuses, started_at, finished_at, created_at
	`

	var r WorkflowRun
	err := db.QueryRowContext(ctx, query, id, workflowID, triggeredBy).Scan(
		&r.ID,
		&r.WorkflowID,
		&r.Status,
		&r.TriggeredBy,
		&r.ErrorMessage,
		&r.TaskStatuses,
		&r.StartedAt,
		&r.FinishedAt,
		&r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger workflow run: %w", err)
	}

	return &r, nil
}

// GetRun retrieves a specific workflow run by its ID
func (s *WorkflowService) GetRun(ctx context.Context, runID string) (*WorkflowRun, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, workflow_id, status, triggered_by, error_message, task_statuses, started_at, finished_at, created_at
		FROM workflow_runs
		WHERE id = $1
	`

	var r WorkflowRun
	err := db.QueryRowContext(ctx, query, runID).Scan(
		&r.ID,
		&r.WorkflowID,
		&r.Status,
		&r.TriggeredBy,
		&r.ErrorMessage,
		&r.TaskStatuses,
		&r.StartedAt,
		&r.FinishedAt,
		&r.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow run not found")
		}
		return nil, fmt.Errorf("failed to get workflow run: %w", err)
	}

	return &r, nil
}

// ListRuns retrieves all workflow runs belonging to a workflow sorted by creation time descending
func (s *WorkflowService) ListRuns(ctx context.Context, workflowID string) ([]WorkflowRun, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, workflow_id, status, triggered_by, error_message, task_statuses, started_at, finished_at, created_at
		FROM workflow_runs
		WHERE workflow_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}
	defer rows.Close()

	var list []WorkflowRun
	for rows.Next() {
		var r WorkflowRun
		err := rows.Scan(
			&r.ID,
			&r.WorkflowID,
			&r.Status,
			&r.TriggeredBy,
			&r.ErrorMessage,
			&r.TaskStatuses,
			&r.StartedAt,
			&r.FinishedAt,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow run: %w", err)
		}
		list = append(list, r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}
