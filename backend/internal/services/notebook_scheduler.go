package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rcn/rcn/backend/internal/database"
)

type NotebookSchedule struct {
	ID                string     `json:"id"`
	NotebookID        string     `json:"notebook_id"`
	UserID            string     `json:"user_id"`
	Schedule          string     `json:"schedule"`
	ExportFormat      string     `json:"export_format"`
	NotificationEmail string     `json:"notification_email"`
	Enabled           bool       `json:"enabled"`
	LastRunAt         *time.Time `json:"last_run_at"`
	NextRunAt         *time.Time `json:"next_run_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ScheduleRun struct {
	ID           string     `json:"id"`
	ScheduleID   string     `json:"schedule_id"`
	Status       string     `json:"status"`
	OutputPath   string     `json:"output_path"`
	ErrorMessage string     `json:"error_message"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type NotebookSchedulerService struct{}

// NewNotebookSchedulerService initializes and returns a new NotebookSchedulerService
func NewNotebookSchedulerService() *NotebookSchedulerService {
	return &NotebookSchedulerService{}
}

// Create inserts a new notebook schedule into the database
func (s *NotebookSchedulerService) Create(ctx context.Context, notebookID, userID, schedule, exportFormat, email string) (*NotebookSchedule, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	id := uuid.New().String()
	query := `
		INSERT INTO notebook_schedules (id, notebook_id, user_id, schedule, export_format, notification_email, enabled, last_run_at, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, NULL, NULL, NOW(), NOW())
		RETURNING id, notebook_id, user_id, schedule, export_format, notification_email, enabled, last_run_at, next_run_at, created_at, updated_at
	`

	var ns NotebookSchedule
	err := db.QueryRowContext(ctx, query, id, notebookID, userID, schedule, exportFormat, email).Scan(
		&ns.ID,
		&ns.NotebookID,
		&ns.UserID,
		&ns.Schedule,
		&ns.ExportFormat,
		&ns.NotificationEmail,
		&ns.Enabled,
		&ns.LastRunAt,
		&ns.NextRunAt,
		&ns.CreatedAt,
		&ns.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notebook schedule: %w", err)
	}

	return &ns, nil
}

// List retrieves all notebook schedules belonging to a user sorted by creation time descending
func (s *NotebookSchedulerService) List(ctx context.Context, userID string) ([]NotebookSchedule, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, notebook_id, user_id, schedule, export_format, notification_email, enabled, last_run_at, next_run_at, created_at, updated_at
		FROM notebook_schedules
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list notebook schedules: %w", err)
	}
	defer rows.Close()

	var list []NotebookSchedule
	for rows.Next() {
		var ns NotebookSchedule
		err := rows.Scan(
			&ns.ID,
			&ns.NotebookID,
			&ns.UserID,
			&ns.Schedule,
			&ns.ExportFormat,
			&ns.NotificationEmail,
			&ns.Enabled,
			&ns.LastRunAt,
			&ns.NextRunAt,
			&ns.CreatedAt,
			&ns.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notebook schedule: %w", err)
		}
		list = append(list, ns)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// Get retrieves a notebook schedule by its ID
func (s *NotebookSchedulerService) Get(ctx context.Context, id string) (*NotebookSchedule, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, notebook_id, user_id, schedule, export_format, notification_email, enabled, last_run_at, next_run_at, created_at, updated_at
		FROM notebook_schedules
		WHERE id = $1
	`

	var ns NotebookSchedule
	err := db.QueryRowContext(ctx, query, id).Scan(
		&ns.ID,
		&ns.NotebookID,
		&ns.UserID,
		&ns.Schedule,
		&ns.ExportFormat,
		&ns.NotificationEmail,
		&ns.Enabled,
		&ns.LastRunAt,
		&ns.NextRunAt,
		&ns.CreatedAt,
		&ns.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notebook schedule not found")
		}
		return nil, fmt.Errorf("failed to get notebook schedule: %w", err)
	}

	return &ns, nil
}

// Update updates an existing notebook schedule
func (s *NotebookSchedulerService) Update(ctx context.Context, id, schedule, exportFormat, email string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `
		UPDATE notebook_schedules
		SET schedule = $2, export_format = $3, notification_email = $4, updated_at = NOW()
		WHERE id = $1
	`

	result, err := db.ExecContext(ctx, query, id, schedule, exportFormat, email)
	if err != nil {
		return fmt.Errorf("failed to update notebook schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("notebook schedule not found")
	}

	return nil
}

// Delete deletes a notebook schedule by its ID
func (s *NotebookSchedulerService) Delete(ctx context.Context, id string) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	query := `DELETE FROM notebook_schedules WHERE id = $1`
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notebook schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("notebook schedule not found")
	}

	return nil
}

// ListRuns retrieves all runs for a specific notebook schedule sorted by creation time descending
func (s *NotebookSchedulerService) ListRuns(ctx context.Context, scheduleID string) ([]ScheduleRun, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, schedule_id, status, output_path, error_message, started_at, finished_at, created_at
		FROM schedule_runs
		WHERE schedule_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedule runs: %w", err)
	}
	defer rows.Close()

	var list []ScheduleRun
	for rows.Next() {
		var sr ScheduleRun
		err := rows.Scan(
			&sr.ID,
			&sr.ScheduleID,
			&sr.Status,
			&sr.OutputPath,
			&sr.ErrorMessage,
			&sr.StartedAt,
			&sr.FinishedAt,
			&sr.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule run: %w", err)
		}
		list = append(list, sr)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}
