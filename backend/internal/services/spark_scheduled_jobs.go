package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/rcn/rcn/backend/internal/database"
)

// ScheduledSparkAppGVR is the GVR for ScheduledSparkApplication CRD.
var scheduledSparkAppGVR = schema.GroupVersionResource{
	Group:    "sparkoperator.k8s.io",
	Version:  "v1beta2",
	Resource: "scheduledsparkapplications",
}

// ScheduledJobStatus represents the status of a scheduled Spark job.
type ScheduledJobStatus string

const (
	ScheduledJobActive   ScheduledJobStatus = "ACTIVE"
	ScheduledJobPaused   ScheduledJobStatus = "PAUSED"
	ScheduledJobDisabled ScheduledJobStatus = "DISABLED"
)

// ScheduledSparkJob is the domain model for a scheduled Spark batch job.
type ScheduledSparkJob struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	Name           string                 `json:"name"`
	Schedule       string                 `json:"schedule"`
	Template       map[string]interface{} `json:"template"`
	Enabled        bool                   `json:"enabled"`
	Status         ScheduledJobStatus     `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// CreateScheduledJobRequest is the JSON body for creating a scheduled job.
type CreateScheduledJobRequest struct {
	Name     string                 `json:"name" binding:"required"`
	Schedule string                 `json:"schedule" binding:"required"`
	Template map[string]interface{} `json:"template" binding:"required"`
}

// UpdateScheduledJobRequest is the JSON body for updating a scheduled job.
type UpdateScheduledJobRequest struct {
	Name     *string                `json:"name"`
	Schedule *string                `json:"schedule"`
	Template map[string]interface{} `json:"template"`
	Enabled  *bool                  `json:"enabled"`
}

// SparkScheduledJobService manages ScheduledSparkApplication CRDs via the Spark Operator.
type SparkScheduledJobService struct {
	dynamicClient dynamic.Interface
	namespace     string
}

// NewSparkScheduledJobService creates a new SparkScheduledJobService.
func NewSparkScheduledJobService(namespace string) (*SparkScheduledJobService, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("k8s in-cluster config: %w", err)
	}
	dynClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("k8s dynamic client: %w", err)
	}
	if namespace == "" {
		namespace = "rcn"
	}
	return &SparkScheduledJobService{
		dynamicClient: dynClient,
		namespace:     namespace,
	}, nil
}

// NewSparkScheduledJobServiceWithClient creates a service with a pre-existing dynamic client (for testing).
func NewSparkScheduledJobServiceWithClient(dynamicClient dynamic.Interface, namespace string) *SparkScheduledJobService {
	if namespace == "" {
		namespace = "rcn"
	}
	return &SparkScheduledJobService{
		dynamicClient: dynamicClient,
		namespace:     namespace,
	}
}

// Create creates a new ScheduledSparkApplication CRD and persists the record in Postgres.
func (s *SparkScheduledJobService) Create(ctx context.Context, req *CreateScheduledJobRequest, userID string) (*ScheduledSparkJob, error) {
	db := database.GetDB()

	jobID := uuid.New().String()
	crdName := fmt.Sprintf("rcn-scheduled-%s", jobID[:8])

	// Build the ScheduledSparkApplication CRD.
	crd := s.buildScheduledApp(crdName, req)

	created, err := s.dynamicClient.Resource(scheduledSparkAppGVR).Namespace(s.namespace).
		Create(ctx, crd, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create ScheduledSparkApplication CRD: %w", err)
	}

	// Serialize template to JSONB.
	templateJSON, err := json.Marshal(req.Template)
	if err != nil {
		return nil, fmt.Errorf("marshal template: %w", err)
	}

	job := &ScheduledSparkJob{
		ID:        jobID,
		UserID:    userID,
		Name:      req.Name,
		Schedule:  req.Schedule,
		Template:  req.Template,
		Enabled:   true,
		Status:    ScheduledJobActive,
		CreatedAt: created.GetCreationTimestamp().Time,
	}

	_, err = db.Exec(`
		INSERT INTO spark_scheduled_jobs (id, user_id, name, schedule, template, enabled, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		job.ID, job.UserID, job.Name, job.Schedule, templateJSON, job.Enabled, string(job.Status),
	)
	if err != nil {
		log.Warn().Err(err).Str("job_id", jobID).Msg("failed to persist spark_scheduled_job to DB")
	}

	return job, nil
}

// List returns all scheduled jobs. Admin sees all.
func (s *SparkScheduledJobService) List(ctx context.Context, userID string, showAll bool) ([]ScheduledSparkJob, error) {
	db := database.GetDB()

	query := `SELECT id, user_id, name, schedule, template, enabled, status, created_at, updated_at
		FROM spark_scheduled_jobs`
	var args []interface{}

	if showAll || userID == "" {
		query += ` ORDER BY created_at DESC`
	} else {
		query += ` WHERE user_id = $1 ORDER BY created_at DESC`
		args = append(args, userID)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query spark_scheduled_jobs: %w", err)
	}
	defer rows.Close()

	var jobs []ScheduledSparkJob
	for rows.Next() {
		var j ScheduledSparkJob
		var templateJSON []byte
		var statusStr string

		if err := rows.Scan(
			&j.ID, &j.UserID, &j.Name, &j.Schedule,
			&templateJSON, &j.Enabled, &statusStr,
			&j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan spark_scheduled_job: %w", err)
		}
		j.Status = ScheduledJobStatus(statusStr)
		if templateJSON != nil {
			_ = json.Unmarshal(templateJSON, &j.Template)
		}
		if j.Template == nil {
			j.Template = make(map[string]interface{})
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// Get returns a single scheduled job by ID.
func (s *SparkScheduledJobService) Get(ctx context.Context, id string) (*ScheduledSparkJob, error) {
	db := database.GetDB()
	var j ScheduledSparkJob
	var templateJSON []byte
	var statusStr string

	err := db.QueryRow(`
		SELECT id, user_id, name, schedule, template, enabled, status, created_at, updated_at
		FROM spark_scheduled_jobs WHERE id = $1`, id,
	).Scan(
		&j.ID, &j.UserID, &j.Name, &j.Schedule,
		&templateJSON, &j.Enabled, &statusStr,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get spark_scheduled_job %s: %w", id, err)
	}
	j.Status = ScheduledJobStatus(statusStr)
	if templateJSON != nil {
		_ = json.Unmarshal(templateJSON, &j.Template)
	}
	if j.Template == nil {
		j.Template = make(map[string]interface{})
	}
	return &j, nil
}

// Update updates a scheduled job's definition and the corresponding CRD.
func (s *SparkScheduledJobService) Update(ctx context.Context, id string, req *UpdateScheduledJobRequest) (*ScheduledSparkJob, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	db := database.GetDB()

	// Build update fields.
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	schedule := existing.Schedule
	if req.Schedule != nil {
		schedule = *req.Schedule
	}
	template := existing.Template
	if req.Template != nil {
		template = req.Template
	}
	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	templateJSON, err := json.Marshal(template)
	if err != nil {
		return nil, fmt.Errorf("marshal template: %w", err)
	}

	// Determine the CRD name from the DB record (derived from the ID).
	crdName := fmt.Sprintf("rcn-scheduled-%s", id[:8])

	// Update the CRD spec.
	crd, err := s.dynamicClient.Resource(scheduledSparkAppGVR).Namespace(s.namespace).
		Get(ctx, crdName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("get ScheduledSparkApplication CRD: %w", err)
	}

	if err == nil {
		// CRD exists — update schedule and template.
		_ = unstructured.SetNestedField(crd.Object, schedule, "spec", "schedule")
		if template, ok := template["spec"]; ok {
			if specMap, ok := template.(map[string]interface{}); ok {
				if templateSpec, ok := specMap["template"]; ok {
					if templateSpecMap, ok := templateSpec.(map[string]interface{}); ok {
						_ = unstructured.SetNestedMap(crd.Object, templateSpecMap, "spec", "template")
					}
				}
			}
		}
		if !enabled {
			_ = unstructured.SetNestedField(crd.Object, true, "spec", "suspend")
		} else {
			unstructured.RemoveNestedField(crd.Object, "spec", "suspend")
		}

		_, err = s.dynamicClient.Resource(scheduledSparkAppGVR).Namespace(s.namespace).
			Update(ctx, crd, metav1.UpdateOptions{})
		if err != nil {
			log.Warn().Err(err).Str("crd", crdName).Msg("failed to update ScheduledSparkApplication CRD")
		}
	}

	// Determine new status
	status := existing.Status
	if req.Enabled != nil {
		if *req.Enabled {
			status = ScheduledJobActive
		} else {
			status = ScheduledJobDisabled
		}
	}

	_, err = db.Exec(`
		UPDATE spark_scheduled_jobs
		SET name = $1, schedule = $2, template = $3, enabled = $4, status = $5, updated_at = NOW()
		WHERE id = $6`,
		name, schedule, templateJSON, enabled, string(status), id,
	)
	if err != nil {
		return nil, fmt.Errorf("update spark_scheduled_job %s: %w", id, err)
	}

	// Re-fetch to get the updated at timestamp.
	return s.Get(ctx, id)
}

// Delete deletes a scheduled job and the corresponding CRD.
func (s *SparkScheduledJobService) Delete(ctx context.Context, id string) error {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Derive CRD name from ID.
	crdName := fmt.Sprintf("rcn-scheduled-%s", id[:8])

	err = s.dynamicClient.Resource(scheduledSparkAppGVR).Namespace(s.namespace).
		Delete(ctx, crdName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Warn().Err(err).Str("crd", crdName).Msg("failed to delete ScheduledSparkApplication CRD")
	}

	db := database.GetDB()
	// Mark as deleted / remove from DB.
	_, err = db.Exec(`DELETE FROM spark_scheduled_jobs WHERE id = $1`, existing.ID)
	return err
}

// Toggle enables or disables a scheduled job.
func (s *SparkScheduledJobService) Toggle(ctx context.Context, id string) (*ScheduledSparkJob, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	newEnabled := !existing.Enabled

	// Toggle the CRD suspend field.
	crdName := fmt.Sprintf("rcn-scheduled-%s", id[:8])
	crd, err := s.dynamicClient.Resource(scheduledSparkAppGVR).Namespace(s.namespace).
		Get(ctx, crdName, metav1.GetOptions{})
	if err == nil {
		if !newEnabled {
			_ = unstructured.SetNestedField(crd.Object, true, "spec", "suspend")
		} else {
			unstructured.RemoveNestedField(crd.Object, "spec", "suspend")
		}
		_, err = s.dynamicClient.Resource(scheduledSparkAppGVR).Namespace(s.namespace).
			Update(ctx, crd, metav1.UpdateOptions{})
		if err != nil {
			log.Warn().Err(err).Str("crd", crdName).Msg("failed to update ScheduledSparkApplication suspend field")
		}
	} else if !errors.IsNotFound(err) {
		log.Warn().Err(err).Str("crd", crdName).Msg("failed to get ScheduledSparkApplication CRD for toggle")
	}

	newStatus := ScheduledJobActive
	if !newEnabled {
		newStatus = ScheduledJobDisabled
	}

	db := database.GetDB()
	_, err = db.Exec(`
		UPDATE spark_scheduled_jobs SET enabled = $1, status = $2, updated_at = NOW() WHERE id = $3`,
		newEnabled, string(newStatus), id,
	)
	if err != nil {
		return nil, fmt.Errorf("toggle spark_scheduled_job %s: %w", id, err)
	}

	return s.Get(ctx, id)
}

// buildScheduledApp constructs the ScheduledSparkApplication CRD unstructured object.
func (s *SparkScheduledJobService) buildScheduledApp(name string, req *CreateScheduledJobRequest) *unstructured.Unstructured {
	app := &unstructured.Unstructured{}
	app.SetAPIVersion("sparkoperator.k8s.io/v1beta2")
	app.SetKind("ScheduledSparkApplication")
	app.SetName(name)
	app.SetNamespace(s.namespace)

	_ = unstructured.SetNestedField(app.Object, req.Schedule, "spec", "schedule")
	_ = unstructured.SetNestedField(app.Object, true, "spec", "concurrencyPolicy")

	// Merge the user-provided template into the spec.template field.
	// The template should contain the SparkApplication spec (type, mode, image, driver, executor, etc.).
	if template, ok := req.Template["spec"]; ok {
		if specMap, ok := template.(map[string]interface{}); ok {
			if templateSpec, ok := specMap["template"]; ok {
				if templateSpecMap, ok := templateSpec.(map[string]interface{}); ok {
					_ = unstructured.SetNestedMap(app.Object, templateSpecMap, "spec", "template")
				}
			}
		}
	}

	return app
}
