package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/database"
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

// SetWebhookRequest is the JSON body for setting a job's webhook URL.
type SetWebhookRequest struct {
	WebhookURL string `json:"webhook_url" binding:"required"`
}

// SetWebhook configures the user-facing webhook URL for a job.
// POST /api/v1/spark/jobs/:id/webhook
func (h *SparkJobHandler) SetWebhook(c *gin.Context) {
	id := c.Param("id")

	var req SetWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.SetWebhook(c.Request.Context(), id, req.WebhookURL); err != nil {
		log.Warn().Err(err).Str("job_id", id).Msg("set webhook failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set webhook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "webhook URL updated"})
}

// SparkOperatorNotification is the payload sent by the Spark Operator webhook.
type SparkOperatorNotification struct {
	ApplicationName string `json:"applicationName"`
	Namespace       string `json:"namespace"`
	State           string `json:"state"`
	PreviousState   string `json:"previousState,omitempty"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
	SubmissionTime  string `json:"submissionTime,omitempty"`
	CompletionTime  string `json:"completionTime,omitempty"`
}

// Callback receives webhook notifications from the Spark Operator on
// application state changes. It updates the job status in the database
// and forwards the notification to the user's configured webhook URL.
// POST /api/v1/spark/callback
func (h *SparkJobHandler) Callback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Warn().Err(err).Msg("spark callback: failed to read body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}
	defer c.Request.Body.Close()

	var notif SparkOperatorNotification
	if err := json.Unmarshal(body, &notif); err != nil || notif.ApplicationName == "" {
		// Some operator versions wrap in a "notification" object.
		var wrapped struct {
			Notification SparkOperatorNotification `json:"notification"`
		}
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil || wrapped.Notification.ApplicationName == "" {
			log.Warn().Err(err).Msg("spark callback: failed to parse notification")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification payload"})
			return
		}
		notif = wrapped.Notification
	}

	appName := notif.ApplicationName
	state := notif.State

	log.Info().Str("app", appName).Str("state", state).Msg("spark callback received")

	db := database.GetDB()

	var jobID, webhookURL string
	err = db.QueryRow(
		`SELECT id, webhook_url FROM spark_jobs WHERE spark_application_name = $1`, appName,
	).Scan(&jobID, &webhookURL)
	if err != nil {
		log.Warn().Err(err).Str("app", appName).Msg("spark callback: job not found in db")
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	if jobStatus := mapSparkStateToStatus(state); jobStatus != "" {
		if err := h.svc.UpdateStatus(c.Request.Context(), jobID, jobStatus); err != nil {
			log.Warn().Err(err).Str("job_id", jobID).Msg("spark callback: failed to update status")
		}
	}

	if webhookURL != "" {
		go forwardWebhook(webhookURL, jobID, appName, state, notif)
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// mapSparkStateToStatus maps Spark Operator application states to internal status.
func mapSparkStateToStatus(state string) services.SparkJobStatus {
	switch state {
	case "SUBMITTED", "NEW":
		return services.JobSubmitted
	case "RUNNING":
		return services.JobRunning
	case "COMPLETED", "FINISHED":
		return services.JobCompleted
	case "FAILED", "SUBMISSION_FAILED", "UNKNOWN":
		return services.JobFailed
	case "INVALIDATING":
		return services.JobStopped
	default:
		return ""
	}
}

// forwardWebhook sends the Spark Operator notification to the user's configured webhook URL.
func forwardWebhook(webhookURL, jobID, appName, state string, notif SparkOperatorNotification) {
	payload := map[string]interface{}{
		"job_id":           jobID,
		"application_name": appName,
		"state":            state,
		"notification":     notif,
		"timestamp":        time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Warn().Err(err).Msg("forward webhook: failed to marshal payload")
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Warn().Err(err).Str("url", webhookURL).Msg("forward webhook: request failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		log.Warn().
			Int("status", resp.StatusCode).
			Str("url", webhookURL).
			Str("response", string(respBody)).
			Msg("forward webhook: non-2xx response")
	}
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
