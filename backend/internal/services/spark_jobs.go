package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"

	"github.com/rcn/rcn/backend/internal/database"
)

// SparkApplication GVR for the Spark Operator CRD.
var sparkAppGVR = schema.GroupVersionResource{
	Group:    "sparkoperator.k8s.io",
	Version:  "v1beta2",
	Resource: "sparkapplications",
}

// SparkJobStatus mirrors the SparkApplication driver state.
type SparkJobStatus string

const (
	JobSubmitted  SparkJobStatus = "SUBMITTED"
	JobRunning    SparkJobStatus = "RUNNING"
	JobCompleted  SparkJobStatus = "COMPLETED"
	JobFailed     SparkJobStatus = "FAILED"
	JobStopped    SparkJobStatus = "STOPPED"
	JobUnknown    SparkJobStatus = "UNKNOWN"
)

// SparkJob is the domain model for a submitted Spark batch job.
type SparkJob struct {
	ID                   string            `json:"id"`
	UserID               string            `json:"user_id"`
	Name                 string            `json:"name"`
	Type                 string            `json:"type"` // Scala or Python
	MainClass            string            `json:"main_class"`
	MainAppFile          string            `json:"main_app_file"` // s3a:// or local://
	Arguments            []string          `json:"arguments"`
	SparkConf            map[string]string `json:"spark_conf"`
	DriverCPU            string            `json:"driver_cpu"`
	DriverMemory         string            `json:"driver_memory"`
	ExecutorCPU          string            `json:"executor_cpu"`
	ExecutorMemory       string            `json:"executor_memory"`
	ExecutorInstances    int32             `json:"executor_instances"`
	Status               SparkJobStatus    `json:"status"`
	SparkApplicationName string            `json:"spark_application_name"`
	WebhookURL           string            `json:"webhook_url"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// SubmitJobRequest is the JSON body for submitting a new Spark job.
type SubmitJobRequest struct {
	Name              string            `json:"name" binding:"required"`
	Type              string            `json:"type"` // Scala (default) or Python
	MainClass         string            `json:"main_class"`
	MainAppFile       string            `json:"main_app_file" binding:"required"`
	Arguments         []string          `json:"arguments"`
	SparkConf         map[string]string `json:"spark_conf"`
	DriverCPU         string            `json:"driver_cpu"`
	DriverMemory      string            `json:"driver_memory"`
	ExecutorCPU       string            `json:"executor_cpu"`
	ExecutorMemory    string            `json:"executor_memory"`
	ExecutorInstances int32             `json:"executor_instances"`
	WebhookURL        string            `json:"webhook_url"`
}

// SparkJobService manages Spark batch jobs via the Spark Operator CRD.
type SparkJobService struct {
	dynamicClient dynamic.Interface
	clientSet     *kubernetes.Clientset
	namespace     string
}

func NewSparkJobService(namespace string) (*SparkJobService, error) {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("k8s in-cluster config: %w", err)
	}
	dynClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("k8s dynamic client: %w", err)
	}
	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("k8s clientset: %w", err)
	}
	if namespace == "" {
		namespace = "rcn"
	}
	return &SparkJobService{
		dynamicClient: dynClient,
		clientSet:     cs,
		namespace:     namespace,
	}, nil
}

// Submit creates a new SparkApplication CRD and persists the job in Postgres.
func (s *SparkJobService) Submit(ctx context.Context, req *SubmitJobRequest, userID string) (*SparkJob, error) {
	db := database.GetDB()

	jobID := uuid.New().String()
	appName := fmt.Sprintf("rcn-job-%s", jobID[:8])
	jobType := req.Type
	if jobType == "" {
		jobType = "Scala"
	}

	// Build default SparkConf with MinIO connectivity.
	sparkConf := map[string]string{
		"spark.hadoop.fs.s3a.endpoint":           "http://minio:9000",
		"spark.hadoop.fs.s3a.path.style.access":  "true",
		"spark.hadoop.fs.s3a.connection.ssl.enabled": "false",
		"spark.eventLog.enabled":                 "true",
		"spark.eventLog.dir":                     "s3a://workspace/event-logs/",
		"spark.history.fs.logDirectory":          "s3a://workspace/event-logs/",
		"spark.kubernetes.allocation.driver.readinessTimeout": "60s",
		"spark.kubernetes.executor.deleteOnTermination": "true",
	}
	for k, v := range req.SparkConf {
		sparkConf[k] = v
	}

	// Build SparkApplication CRD object.
	app := s.buildSparkApplication(appName, jobType, req, sparkConf)

	// Inject notification webhook so the Spark Operator calls back on state changes.
	notificationURL := fmt.Sprintf("http://backend:10000/api/v1/spark/callback?job_id=%s", jobID)
	_ = unstructured.SetNestedField(app.Object, notificationURL, "spec", "notification", "api", "url")

	// Create the CRD in K8s.
	created, err := s.dynamicClient.Resource(sparkAppGVR).Namespace(s.namespace).
		Create(ctx, app, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create SparkApplication CRD: %w", err)
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

	// Persist to Postgres.
	job := &SparkJob{
		ID:                   jobID,
		UserID:               userID,
		Name:                 req.Name,
		Type:                 jobType,
		MainClass:            req.MainClass,
		MainAppFile:          req.MainAppFile,
		Arguments:            req.Arguments,
		SparkConf:            sparkConf,
		DriverCPU:            driverCPU,
		DriverMemory:         driverMem,
		ExecutorCPU:          execCPU,
		ExecutorMemory:       execMem,
		ExecutorInstances:    execInst,
		Status:               JobSubmitted,
		SparkApplicationName: appName,
		WebhookURL:           webhookURL,
		CreatedAt:            created.GetCreationTimestamp().Time,
	}

	webhookURL := req.WebhookURL

	_, err = db.Exec(`
		INSERT INTO spark_jobs (
			id, user_id, name, type, main_class, main_app_file, arguments,
			spark_conf, driver_cpu, driver_memory, executor_cpu, executor_memory,
			executor_instances, status, spark_application_name, webhook_url
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		job.ID, job.UserID, job.Name, job.Type, job.MainClass, job.MainAppFile,
		job.Arguments, toJSONB(job.SparkConf), job.DriverCPU, job.DriverMemory,
		job.ExecutorCPU, job.ExecutorMemory, job.ExecutorInstances,
		string(job.Status), job.SparkApplicationName, webhookURL,
	)
	if err != nil {
		log.Warn().Err(err).Str("job_id", jobID).Msg("failed to persist spark_job to DB")
	}

	s.syncApplicationStatus(ctx, appName)
	return job, nil
}

// List returns all jobs visible to the given user. Admin sees all.
func (s *SparkJobService) List(ctx context.Context, userID string, showAll bool) ([]SparkJob, error) {
	db := database.GetDB()

	query := `SELECT id, user_id, name, type, main_class, main_app_file, arguments,
		spark_conf, driver_cpu, driver_memory, executor_cpu, executor_memory,
		executor_instances, status, spark_application_name, webhook_url, created_at, updated_at
		FROM spark_jobs`
	var args []interface{}

	if showAll || userID == "" {
		query += ` ORDER BY created_at DESC`
	} else {
		query += ` WHERE user_id = $1 ORDER BY created_at DESC`
		args = append(args, userID)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query spark_jobs: %w", err)
	}
	defer rows.Close()

	var jobs []SparkJob
	for rows.Next() {
		var j SparkJob
		var argsJSON []byte
		var confJSON []byte
		if err := rows.Scan(
			&j.ID, &j.UserID, &j.Name, &j.Type, &j.MainClass, &j.MainAppFile,
			&argsJSON, &confJSON,
			&j.DriverCPU, &j.DriverMemory, &j.ExecutorCPU, &j.ExecutorMemory,
			&j.ExecutorInstances, (*string)(&j.Status), &j.SparkApplicationName,
			&j.WebhookURL, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan spark_job: %w", err)
		}
		if argsJSON != nil {
			// Postgres TEXT[] is scanned as a JSON-like byte array; parse it.
			j.Arguments = parseTextArray(string(argsJSON))
		}
		if confJSON != nil {
			j.SparkConf = parseJSONBMap(confJSON)
		}
		if j.SparkApplicationName != "" {
			liveStatus := s.getCRDStatus(ctx, j.SparkApplicationName)
			if liveStatus != "" && liveStatus != JobUnknown {
				j.Status = liveStatus
			}
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// Get returns a single job by ID.
func (s *SparkJobService) Get(ctx context.Context, id string) (*SparkJob, error) {
	db := database.GetDB()
	var j SparkJob
	var argsJSON []byte
	var confJSON []byte

	err := db.QueryRow(`
		SELECT id, user_id, name, type, main_class, main_app_file, arguments,
			spark_conf, driver_cpu, driver_memory, executor_cpu, executor_memory,
			executor_instances, status, spark_application_name, webhook_url, created_at, updated_at
		FROM spark_jobs WHERE id = $1`, id,
	).Scan(
		&j.ID, &j.UserID, &j.Name, &j.Type, &j.MainClass, &j.MainAppFile,
		&argsJSON, &confJSON,
		&j.DriverCPU, &j.DriverMemory, &j.ExecutorCPU, &j.ExecutorMemory,
		&j.ExecutorInstances, (*string)(&j.Status), &j.SparkApplicationName,
		&j.WebhookURL, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get spark_job %s: %w", id, err)
	}
	if argsJSON != nil {
		j.Arguments = parseTextArray(string(argsJSON))
	}
	if confJSON != nil {
		j.SparkConf = parseJSONBMap(confJSON)
	}
	if j.SparkApplicationName != "" {
		liveStatus := s.getCRDStatus(ctx, j.SparkApplicationName)
		if liveStatus != "" && liveStatus != JobUnknown {
			j.Status = liveStatus
		}
	}
	return &j, nil
}

// Stop deletes the SparkApplication CRD and marks the job as stopped.
func (s *SparkJobService) Stop(ctx context.Context, id string) error {
	db := database.GetDB()

	var appName string
	err := db.QueryRow(`SELECT spark_application_name FROM spark_jobs WHERE id = $1`, id).Scan(&appName)
	if err != nil {
		return fmt.Errorf("find job %s: %w", id, err)
	}

	if appName != "" {
		err = s.dynamicClient.Resource(sparkAppGVR).Namespace(s.namespace).
			Delete(ctx, appName, metav1.DeleteOptions{
				GracePeriodSeconds: ptr.To(int64(0)),
			})
		if err != nil && !errors.IsNotFound(err) {
			log.Warn().Err(err).Str("app", appName).Msg("failed to delete SparkApplication CRD")
		}
	}

	_, err = db.Exec(`UPDATE spark_jobs SET status = $1, updated_at = NOW() WHERE id = $2`,
		string(JobStopped), id)
	return err
}

// SetWebhook updates the webhook URL for a given job.
func (s *SparkJobService) SetWebhook(ctx context.Context, id string, webhookURL string) error {
	db := database.GetDB()
	_, err := db.Exec(`UPDATE spark_jobs SET webhook_url = $1, updated_at = NOW() WHERE id = $2`, webhookURL, id)
	return err
}

// UpdateStatus updates the job status in the database.
func (s *SparkJobService) UpdateStatus(ctx context.Context, id string, status SparkJobStatus) error {
	db := database.GetDB()
	_, err := db.Exec(`UPDATE spark_jobs SET status = $1, updated_at = NOW() WHERE id = $2`, string(status), id)
	return err
}

// GetLogs returns the driver pod logs for a completed/running job.
func (s *SparkJobService) GetLogs(ctx context.Context, id string, tail int) (string, error) {
	db := database.GetDB()

	var appName string
	err := db.QueryRow(`SELECT spark_application_name FROM spark_jobs WHERE id = $1`, id).Scan(&appName)
	if err != nil {
		return "", fmt.Errorf("find job %s: %w", id, err)
	}
	if appName == "" {
		return "", fmt.Errorf("no SparkApplication for job %s", id)
	}

	// The driver pod is named after the SparkApplication.
	driverPodName := appName + "-driver"
	tailLines := int64(tail)
	if tailLines <= 0 {
		tailLines = 100
	}

	req := s.clientSet.CoreV1().Pods(s.namespace).GetLogs(driverPodName, &corev1.PodLogOptions{
		TailLines: &tailLines,
	})
	raw, err := req.DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("get driver logs: %w", err)
	}
	return string(raw), nil
}

// syncApplicationStatus queries the SparkApplication CRD and updates the DB.
func (s *SparkJobService) syncApplicationStatus(ctx context.Context, appName string) {
	status := s.getCRDStatus(ctx, appName)
	if status == "" || status == JobUnknown {
		return
	}
	db := database.GetDB()
	_, _ = db.Exec(`UPDATE spark_jobs SET status = $1, updated_at = NOW() WHERE spark_application_name = $2`,
		string(status), appName)
}

// getCRDStatus reads the SparkApplication CRD's application state.
func (s *SparkJobService) getCRDStatus(ctx context.Context, appName string) SparkJobStatus {
	obj, err := s.dynamicClient.Resource(sparkAppGVR).Namespace(s.namespace).
		Get(ctx, appName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return JobFailed
		}
		return JobUnknown
	}

	appState, found, err := unstructured.NestedString(obj.Object, "status", "appState", "state")
	if err != nil || !found {
		return JobUnknown
	}

	switch appState {
	case "SUBMITTED", "NEW":
		return JobSubmitted
	case "RUNNING":
		return JobRunning
	case "COMPLETED", "FINISHED":
		return JobCompleted
	case "FAILED", "SUBMISSION_FAILED":
		return JobFailed
	case "INVALIDATING":
		return JobStopped
	default:
		return JobUnknown
	}
}

// buildSparkApplication constructs the SparkApplication CRD unstructured object.
func (s *SparkJobService) buildSparkApplication(name, jobType string, req *SubmitJobRequest, sparkConf map[string]string) *unstructured.Unstructured {
	app := &unstructured.Unstructured{}
	app.SetAPIVersion("sparkoperator.k8s.io/v1beta2")
	app.SetKind("SparkApplication")
	app.SetName(name)
	app.SetNamespace(s.namespace)

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

	_ = unstructured.SetNestedField(app.Object, jobType, "spec", "type")
	_ = unstructured.SetNestedField(app.Object, "cluster", "spec", "mode")
	_ = unstructured.SetNestedField(app.Object, "ghcr.io/sparklabx/kernel:latest", "spec", "image")
	_ = unstructured.SetNestedField(app.Object, "IfNotPresent", "spec", "imagePullPolicy")
	_ = unstructured.SetNestedField(app.Object, "3.5.0", "spec", "sparkVersion")

	if jobType == "Python" {
		_ = unstructured.SetNestedField(app.Object, req.MainAppFile, "spec", "mainApplicationFile")
	} else {
		_ = unstructured.SetNestedField(app.Object, req.MainClass, "spec", "mainClass")
		_ = unstructured.SetNestedField(app.Object, req.MainAppFile, "spec", "mainApplicationFile")
	}

	if len(req.Arguments) > 0 {
		_ = unstructured.SetNestedStringSlice(app.Object, req.Arguments, "spec", "arguments")
	}

	// Driver spec
	_ = unstructured.SetNestedField(app.Object, driverCPU, "spec", "driver", "cores")
	coreLimitQty := resource.MustParse(driverCPU)
	_ = unstructured.SetNestedField(app.Object, coreLimitQty.String(), "spec", "driver", "coreLimit")
	_ = unstructured.SetNestedField(app.Object, driverMem, "spec", "driver", "memory")
	_ = unstructured.SetNestedField(app.Object, "spark", "spec", "driver", "serviceAccount")

	// Executor spec
	_ = unstructured.SetNestedField(app.Object, execCPU, "spec", "executor", "cores")
	_ = unstructured.SetNestedField(app.Object, int64(execInst), "spec", "executor", "instances")
	_ = unstructured.SetNestedField(app.Object, execMem, "spec", "executor", "memory")

	// SparkConf
	_ = unstructured.SetNestedStringMap(app.Object, sparkConf, "spec", "sparkConf")

	// MinIO secret ref
	_ = unstructured.SetNestedField(app.Object, "rcn-secrets", "spec", "hadoopConfSecret")
	_ = unstructured.SetNestedField(app.Object, "spark", "spec", "driver", "serviceAccount")

	return app
}

// Health checks that the Spark Operator CRD is accessible.
func (s *SparkJobService) Health(ctx context.Context) error {
	_, err := s.dynamicClient.Resource(sparkAppGVR).Namespace(s.namespace).
		List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

// SyncAllStatuses refreshes the status of all non-terminal jobs from K8s.
func (s *SparkJobService) SyncAllStatuses(ctx context.Context) {
	db := database.GetDB()
	rows, err := db.Query(`
		SELECT id, spark_application_name FROM spark_jobs
		WHERE status IN ($1, $2, $3)`,
		string(JobSubmitted), string(JobRunning), string(JobUnknown),
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, appName string
		if err := rows.Scan(&id, &appName); err != nil || appName == "" {
			continue
		}
		liveStatus := s.getCRDStatus(ctx, appName)
		if liveStatus != "" && liveStatus != JobUnknown {
			_, _ = db.Exec(`UPDATE spark_jobs SET status = $1, updated_at = NOW() WHERE id = $2`,
				string(liveStatus), id)
		}
	}
}

// parseTextArray is a minimal parser for Postgres TEXT[] literals.
func parseTextArray(s string) []string {
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return nil
	}
	inner := s[1 : len(s)-1]
	if inner == "" {
		return nil
	}
	var result []string
	var buf []byte
	inQuotes := false
	escaped := false
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if escaped {
			buf = append(buf, c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == '"' {
			inQuotes = !inQuotes
			continue
		}
		if c == ',' && !inQuotes {
			result = append(result, string(buf))
			buf = nil
			continue
		}
		buf = append(buf, c)
	}
	if len(buf) > 0 || len(result) > 0 {
		result = append(result, string(buf))
	}
	return result
}

// parseJSONBMap converts a JSONB byte payload to map[string]string.
func parseJSONBMap(data []byte) map[string]string {
	if len(data) == 0 {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

// toJSONB converts a map to a JSON byte slice for Postgres JSONB columns.
func toJSONB(m map[string]string) []byte {
	if m == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(m)
	return b
}
