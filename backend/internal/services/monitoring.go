package services

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/rcn/rcn/backend/internal/database"
)

// Metric counters for Prometheus-style exposition. This is a lightweight
// in-process metrics collector that exposes data as a simple JSON endpoint.
// For production Prometheus scraping, configure the /api/v1/admin/metrics
// endpoint with promhttp.Handler() and prometheus/client_golang.

// MetricsCollector holds all instrumented counters and gauges for the RCN
// platform. It is safe for concurrent use.
type MetricsCollector struct {
	mu sync.RWMutex

	// HTTP request counters
	HTTPRequestsTotal   map[string]*uint64 // key: "method:path:status"
	HTTPRequestDuration map[string]*int64  // key: "method:path" → cumulative ms

	// Spark job counters
	SparkJobsSubmitted   atomic.Int64
	SparkJobsSucceeded   atomic.Int64
	SparkJobsFailed      atomic.Int64
	SparkJobsRunning     atomic.Int64

	// Kernel counters
	ActiveKernels     atomic.Int64
	TotalKernelStarts atomic.Int64

	// Git counters
	GitCommitsTotal atomic.Int64
	GitLinksTotal   atomic.Int64

	// Start timestamp
	startTime time.Time
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		HTTPRequestsTotal:   make(map[string]*uint64),
		HTTPRequestDuration: make(map[string]*int64),
		startTime:           time.Now().UTC(),
	}
}

// RecordHTTPRequest records an HTTP request metric.
func (mc *MetricsCollector) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	key := fmt.Sprintf("%s:%s:%d", method, path, status)
	durKey := fmt.Sprintf("%s:%s", method, path)

	mc.mu.Lock()
	if _, ok := mc.HTTPRequestsTotal[key]; !ok {
		mc.HTTPRequestsTotal[key] = new(uint64)
	}
	*mc.HTTPRequestsTotal[key]++

	if _, ok := mc.HTTPRequestDuration[durKey]; !ok {
		mc.HTTPRequestDuration[durKey] = new(int64)
	}
	*mc.HTTPRequestDuration[durKey] += duration.Milliseconds()
	mc.mu.Unlock()
}

// RecordSparkJob records a Spark job status change.
func (mc *MetricsCollector) RecordSparkJob(status string) {
	switch status {
	case "SUBMITTED", "RUNNING":
		mc.SparkJobsRunning.Add(1)
		mc.SparkJobsSubmitted.Add(1)
	case "COMPLETED", "SUCCEEDED":
		mc.SparkJobsRunning.Add(-1)
		mc.SparkJobsSucceeded.Add(1)
	case "FAILED":
		mc.SparkJobsRunning.Add(-1)
		mc.SparkJobsFailed.Add(1)
	}
}

// RecordKernelStart increments kernel counter.
func (mc *MetricsCollector) RecordKernelStart() {
	mc.ActiveKernels.Add(1)
	mc.TotalKernelStarts.Add(1)
}

// RecordKernelStop decrements kernel counter.
func (mc *MetricsCollector) RecordKernelStop() {
	mc.ActiveKernels.Add(-1)
}

// RecordGitCommit increments git commit counter.
func (mc *MetricsCollector) RecordGitCommit() {
	mc.GitCommitsTotal.Add(1)
}

// RecordGitLink increments git link counter.
func (mc *MetricsCollector) RecordGitLink() {
	mc.GitLinksTotal.Add(1)
}

// Snapshot returns a point-in-time view of all metrics.
func (mc *MetricsCollector) Snapshot() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	uptime := time.Since(mc.startTime).Round(time.Second).String()

	// Build request counters
	httpTotal := uint64(0)
	httpByStatus := make(map[int]uint64)
	httpDetailed := make(map[string]uint64)

	for key, val := range mc.HTTPRequestsTotal {
		httpTotal += *val
		httpDetailed[key] = *val

		// Extract status code
		var method, path string
		var status int
		fmt.Sscanf(key, "%s:%s:%d", &method, &path, &status)
		httpByStatus[status] += *val
	}

	// Build duration totals
	totalDurationMs := int64(0)
	for _, val := range mc.HTTPRequestDuration {
		totalDurationMs += *val
	}

	return map[string]interface{}{
		"uptime":        uptime,
		"start_time":    mc.startTime.Format(time.RFC3339),
		"http": map[string]interface{}{
			"total_requests":    httpTotal,
			"by_status":         httpByStatus,
			"detailed":          httpDetailed,
			"total_duration_ms": totalDurationMs,
		},
		"spark_jobs": map[string]interface{}{
			"submitted": mc.SparkJobsSubmitted.Load(),
			"succeeded": mc.SparkJobsSucceeded.Load(),
			"failed":    mc.SparkJobsFailed.Load(),
			"running":   mc.SparkJobsRunning.Load(),
		},
		"kernels": map[string]interface{}{
			"active":         mc.ActiveKernels.Load(),
			"total_starts":   mc.TotalKernelStarts.Load(),
		},
		"git": map[string]interface{}{
			"commits_total": mc.GitCommitsTotal.Load(),
			"links_total":   mc.GitLinksTotal.Load(),
		},
	}
}

// GetUptime returns the service uptime.
func (mc *MetricsCollector) GetUptime() time.Duration {
	return time.Since(mc.startTime)
}

// HealthChecker performs dependency health checks.
type HealthChecker struct {
	dbURL        string
	minioEnabled bool
	sparkEnabled bool
}

// NewHealthChecker creates a health checker.
func NewHealthChecker(dbURL string, minioEnabled, sparkEnabled bool) *HealthChecker {
	return &HealthChecker{
		dbURL:        dbURL,
		minioEnabled: minioEnabled,
		sparkEnabled: sparkEnabled,
	}
}

// HealthCheckResult represents one service's health.
type HealthCheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "degraded", "down"
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// RunChecks executes all health checks and returns results.
func (hc *HealthChecker) RunChecks(ctx context.Context) []HealthCheckResult {
	var results []HealthCheckResult

	// Database check
	results = append(results, hc.checkDatabase(ctx))

	// MinIO check
	if hc.minioEnabled {
		results = append(results, hc.checkMinIO(ctx))
	}

	return results
}

func (hc *HealthChecker) checkDatabase(ctx context.Context) HealthCheckResult {
	start := time.Now()
	db := database.GetDB()
	result := HealthCheckResult{Name: "database"}

	if db == nil {
		result.Status = "down"
		result.Message = "database connection is nil"
		return result
	}

	if err := db.PingContext(ctx); err != nil {
		result.Status = "down"
		result.Message = err.Error()
		return result
	}

	// Check connection stats
	var stats struct {
		MaxOpenConns    int
		OpenConns       int
		InUse           int
		Idle            int
		WaitCount       int64
		WaitDuration    time.Duration
		MaxIdleClosed   int64
		MaxIdleTimeClosed int64
		MaxLifetimeClosed  int64
	}

	dbStats := db.Stats()
	stats.MaxOpenConns = dbStats.MaxOpenConns
	stats.OpenConns = dbStats.OpenConns
	stats.InUse = dbStats.InUse
	stats.Idle = dbStats.Idle
	stats.WaitCount = dbStats.WaitCount
	stats.WaitDuration = dbStats.WaitDuration
	stats.MaxIdleClosed = dbStats.MaxIdleClosed

	result.Status = "ok"
	result.Message = fmt.Sprintf("open=%d inuse=%d idle=%d waits=%d",
		stats.OpenConns, stats.InUse, stats.Idle, stats.WaitCount)
	result.Latency = time.Since(start).Round(time.Millisecond).String()

	return result
}

func (hc *HealthChecker) checkMinIO(ctx context.Context) HealthCheckResult {
	start := time.Now()
	result := HealthCheckResult{Name: "minio_s3"}
	result.Status = "ok"
	result.Message = "configured"
	result.Latency = time.Since(start).Round(time.Millisecond).String()
	return result
}

// SystemInfo returns system metadata.
func (hc *HealthChecker) SystemInfo(ctx context.Context) (map[string]interface{}, error) {
	db := database.GetDB()

	var activeUsers int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&activeUsers)

	var totalNotebooks int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notebooks`).Scan(&totalNotebooks)

	var totalSparkJobs int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM spark_jobs`).Scan(&totalSparkJobs)

	return map[string]interface{}{
		"active_users":     activeUsers,
		"total_notebooks":  totalNotebooks,
		"total_spark_jobs": totalSparkJobs,
	}, nil
}
