package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rcn/rcn/backend/internal/database"
)

// SparkJobTemplate is the domain model for a reusable Spark job template.
type SparkJobTemplate struct {
	ID                string            `json:"id"`
	UserID            string            `json:"user_id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Type              string            `json:"type"` // Scala or Python
	MainClass         string            `json:"main_class"`
	MainAppFile       string            `json:"main_app_file"`
	Arguments         []string          `json:"arguments"`
	SparkConf         map[string]string `json:"spark_conf"`
	DriverCPU         string            `json:"driver_cpu"`
	DriverMemory      string            `json:"driver_memory"`
	ExecutorCPU       string            `json:"executor_cpu"`
	ExecutorMemory    string            `json:"executor_memory"`
	ExecutorInstances int32             `json:"executor_instances"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// CreateJobTemplateRequest is the JSON body for creating a job template.
type CreateJobTemplateRequest struct {
	Name              string            `json:"name" binding:"required"`
	Description       string            `json:"description"`
	Type              string            `json:"type"`
	MainClass         string            `json:"main_class"`
	MainAppFile       string            `json:"main_app_file" binding:"required"`
	Arguments         []string          `json:"arguments"`
	SparkConf         map[string]string `json:"spark_conf"`
	DriverCPU         string            `json:"driver_cpu"`
	DriverMemory      string            `json:"driver_memory"`
	ExecutorCPU       string            `json:"executor_cpu"`
	ExecutorMemory    string            `json:"executor_memory"`
	ExecutorInstances int32             `json:"executor_instances"`
}

// UpdateJobTemplateRequest is the JSON body for updating a job template.
type UpdateJobTemplateRequest struct {
	Name              *string           `json:"name"`
	Description       *string           `json:"description"`
	Type              *string           `json:"type"`
	MainClass         *string           `json:"main_class"`
	MainAppFile       *string           `json:"main_app_file"`
	Arguments         []string          `json:"arguments"`
	SparkConf         map[string]string `json:"spark_conf"`
	DriverCPU         *string           `json:"driver_cpu"`
	DriverMemory      *string           `json:"driver_memory"`
	ExecutorCPU       *string           `json:"executor_cpu"`
	ExecutorMemory    *string           `json:"executor_memory"`
	ExecutorInstances *int32            `json:"executor_instances"`
}

// SparkJobTemplateService manages reusable Spark job templates (DB-only, no K8s CRD).
type SparkJobTemplateService struct{}

// NewSparkJobTemplateService creates a new SparkJobTemplateService.
func NewSparkJobTemplateService() *SparkJobTemplateService {
	return &SparkJobTemplateService{}
}

// Create persists a new job template.
func (s *SparkJobTemplateService) Create(ctx context.Context, req *CreateJobTemplateRequest, userID string) (*SparkJobTemplate, error) {
	db := database.GetDB()

	id := uuid.New().String()
	now := time.Now()

	jobType := req.Type
	if jobType == "" {
		jobType = "Scala"
	}

	driverCPU := req.DriverCPU
	if driverCPU == "" {
		driverCPU = "1"
	}
	driverMem := req.DriverMemory
	if driverMem == "" {
		driverMem = "2g"
	}
	execCPU := req.ExecutorCPU
	if execCPU == "" {
		execCPU = "1"
	}
	execMem := req.ExecutorMemory
	if execMem == "" {
		execMem = "2g"
	}
	execInst := req.ExecutorInstances
	if execInst == 0 {
		execInst = 1
	}

	t := &SparkJobTemplate{
		ID:                id,
		UserID:            userID,
		Name:              req.Name,
		Description:       req.Description,
		Type:              jobType,
		MainClass:         req.MainClass,
		MainAppFile:       req.MainAppFile,
		Arguments:         req.Arguments,
		SparkConf:         req.SparkConf,
		DriverCPU:         driverCPU,
		DriverMemory:      driverMem,
		ExecutorCPU:       execCPU,
		ExecutorMemory:    execMem,
		ExecutorInstances: execInst,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	_, err := db.Exec(`
		INSERT INTO spark_job_templates (
			id, user_id, name, description, type, main_class, main_app_file, arguments,
			spark_conf, driver_cpu, driver_memory, executor_cpu, executor_memory, executor_instances
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		t.ID, t.UserID, t.Name, t.Description, t.Type, t.MainClass, t.MainAppFile,
		t.Arguments, toJSONB(t.SparkConf), t.DriverCPU, t.DriverMemory,
		t.ExecutorCPU, t.ExecutorMemory, t.ExecutorInstances,
	)
	if err != nil {
		return nil, fmt.Errorf("insert spark_job_template: %w", err)
	}

	return t, nil
}

// List returns all templates visible to the given user. Admin sees all.
func (s *SparkJobTemplateService) List(ctx context.Context, userID string, showAll bool) ([]SparkJobTemplate, error) {
	db := database.GetDB()

	query := `SELECT id, user_id, name, description, type, main_class, main_app_file, arguments,
		spark_conf, driver_cpu, driver_memory, executor_cpu, executor_memory,
		executor_instances, created_at, updated_at
		FROM spark_job_templates`
	var args []interface{}

	if showAll || userID == "" {
		query += ` ORDER BY created_at DESC`
	} else {
		query += ` WHERE user_id = $1 ORDER BY created_at DESC`
		args = append(args, userID)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query spark_job_templates: %w", err)
	}
	defer rows.Close()

	var templates []SparkJobTemplate
	for rows.Next() {
		var t SparkJobTemplate
		var argsJSON []byte
		var confJSON []byte
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Name, &t.Description, &t.Type, &t.MainClass, &t.MainAppFile,
			&argsJSON, &confJSON,
			&t.DriverCPU, &t.DriverMemory, &t.ExecutorCPU, &t.ExecutorMemory,
			&t.ExecutorInstances, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan spark_job_template: %w", err)
		}
		if argsJSON != nil {
			t.Arguments = parseTextArray(string(argsJSON))
		}
		if confJSON != nil {
			t.SparkConf = parseJSONBMap(confJSON)
		}
		templates = append(templates, t)
	}
	if templates == nil {
		templates = []SparkJobTemplate{}
	}
	return templates, nil
}

// Get returns a single template by ID.
func (s *SparkJobTemplateService) Get(ctx context.Context, id string) (*SparkJobTemplate, error) {
	db := database.GetDB()
	var t SparkJobTemplate
	var argsJSON []byte
	var confJSON []byte

	err := db.QueryRow(`
		SELECT id, user_id, name, description, type, main_class, main_app_file, arguments,
			spark_conf, driver_cpu, driver_memory, executor_cpu, executor_memory,
			executor_instances, created_at, updated_at
		FROM spark_job_templates WHERE id = $1`, id,
	).Scan(
		&t.ID, &t.UserID, &t.Name, &t.Description, &t.Type, &t.MainClass, &t.MainAppFile,
		&argsJSON, &confJSON,
		&t.DriverCPU, &t.DriverMemory, &t.ExecutorCPU, &t.ExecutorMemory,
		&t.ExecutorInstances, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get spark_job_template %s: %w", id, err)
	}
	if argsJSON != nil {
		t.Arguments = parseTextArray(string(argsJSON))
	}
	if confJSON != nil {
		t.SparkConf = parseJSONBMap(confJSON)
	}
	return &t, nil
}

// Update updates a job template.
func (s *SparkJobTemplateService) Update(ctx context.Context, id string, req *UpdateJobTemplateRequest) (*SparkJobTemplate, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	db := database.GetDB()

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	description := existing.Description
	if req.Description != nil {
		description = *req.Description
	}
	jobType := existing.Type
	if req.Type != nil {
		jobType = *req.Type
	}
	mainClass := existing.MainClass
	if req.MainClass != nil {
		mainClass = *req.MainClass
	}
	mainAppFile := existing.MainAppFile
	if req.MainAppFile != nil {
		mainAppFile = *req.MainAppFile
	}
	arguments := existing.Arguments
	if req.Arguments != nil {
		arguments = req.Arguments
	}
	sparkConf := existing.SparkConf
	if req.SparkConf != nil {
		sparkConf = req.SparkConf
	}
	driverCPU := existing.DriverCPU
	if req.DriverCPU != nil {
		driverCPU = *req.DriverCPU
	}
	driverMemory := existing.DriverMemory
	if req.DriverMemory != nil {
		driverMemory = *req.DriverMemory
	}
	executorCPU := existing.ExecutorCPU
	if req.ExecutorCPU != nil {
		executorCPU = *req.ExecutorCPU
	}
	executorMemory := existing.ExecutorMemory
	if req.ExecutorMemory != nil {
		executorMemory = *req.ExecutorMemory
	}
	executorInstances := existing.ExecutorInstances
	if req.ExecutorInstances != nil {
		executorInstances = *req.ExecutorInstances
	}

	_, err = db.Exec(`
		UPDATE spark_job_templates SET
			name = $1, description = $2, type = $3, main_class = $4, main_app_file = $5,
			arguments = $6, spark_conf = $7, driver_cpu = $8, driver_memory = $9,
			executor_cpu = $10, executor_memory = $11, executor_instances = $12,
			updated_at = NOW()
		WHERE id = $13`,
		name, description, jobType, mainClass, mainAppFile,
		arguments, toJSONB(sparkConf), driverCPU, driverMemory,
		executorCPU, executorMemory, executorInstances, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update spark_job_template %s: %w", id, err)
	}

	return s.Get(ctx, id)
}

// Delete removes a job template.
func (s *SparkJobTemplateService) Delete(ctx context.Context, id string) error {
	db := database.GetDB()
	result, err := db.Exec(`DELETE FROM spark_job_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete spark_job_template %s: %w", id, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("spark_job_template %s not found", id)
	}
	return nil
}

// ToSubmitJobRequest converts a template into a SubmitJobRequest so it can be submitted.
func (t *SparkJobTemplate) ToSubmitJobRequest(overrides *RunJobFromTemplateRequest) *SubmitJobRequest {
	req := &SubmitJobRequest{
		Name:              t.Name,
		Type:              t.Type,
		MainClass:         t.MainClass,
		MainAppFile:       t.MainAppFile,
		Arguments:         t.Arguments,
		SparkConf:         t.SparkConf,
		DriverCPU:         t.DriverCPU,
		DriverMemory:      t.DriverMemory,
		ExecutorCPU:       t.ExecutorCPU,
		ExecutorMemory:    t.ExecutorMemory,
		ExecutorInstances: t.ExecutorInstances,
	}

	// Apply overrides if provided.
	if overrides != nil {
		if overrides.Name != "" {
			req.Name = overrides.Name
		}
		if overrides.Type != "" {
			req.Type = overrides.Type
		}
		if overrides.MainClass != "" {
			req.MainClass = overrides.MainClass
		}
		if overrides.MainAppFile != "" {
			req.MainAppFile = overrides.MainAppFile
		}
		if overrides.Arguments != nil {
			req.Arguments = overrides.Arguments
		}
		if overrides.SparkConf != nil {
			if req.SparkConf == nil {
				req.SparkConf = overrides.SparkConf
			} else {
				for k, v := range overrides.SparkConf {
					req.SparkConf[k] = v
				}
			}
		}
		if overrides.DriverCPU != "" {
			req.DriverCPU = overrides.DriverCPU
		}
		if overrides.DriverMemory != "" {
			req.DriverMemory = overrides.DriverMemory
		}
		if overrides.ExecutorCPU != "" {
			req.ExecutorCPU = overrides.ExecutorCPU
		}
		if overrides.ExecutorMemory != "" {
			req.ExecutorMemory = overrides.ExecutorMemory
		}
		if overrides.ExecutorInstances > 0 {
			req.ExecutorInstances = overrides.ExecutorInstances
		}
		if overrides.WebhookURL != "" {
			req.WebhookURL = overrides.WebhookURL
		}
	}

	return req
}

// RunJobFromTemplateRequest is the JSON body for submitting a job from a template with overrides.
type RunJobFromTemplateRequest struct {
	Name              string            `json:"name"`
	Type              string            `json:"type"`
	MainClass         string            `json:"main_class"`
	MainAppFile       string            `json:"main_app_file"`
	Arguments         []string          `json:"arguments"`
	SparkConf         map[string]string `json:"spark_conf"`
	DriverCPU         string            `json:"driver_cpu"`
	DriverMemory      string            `json:"driver_memory"`
	ExecutorCPU       string            `json:"executor_cpu"`
	ExecutorMemory    string            `json:"executor_memory"`
	ExecutorInstances int32             `json:"executor_instances"`
	WebhookURL        string            `json:"webhook_url"`
}

