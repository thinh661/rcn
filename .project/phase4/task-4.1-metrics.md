# Task 4.1: Prometheus Metrics

## Mục tiêu
Export metrics từ Go backend để Prometheus scrape: HTTP request metrics, Go runtime, custom Spark/kernel metrics.

## Implementation
1. Go: `services/metrics.go` — custom metrics collector với `prometheus/client_golang`
2. Middleware: `middleware/metrics.go` — ghi request count, duration, status codes
3. Endpoint: `GET /api/v1/admin/metrics` — prometheus endpoint (superadmin)
4. Metrics:
   - `rcn_http_requests_total{method,path,status}`
   - `rcn_http_request_duration_seconds{method,path}`
   - `rcn_spark_jobs_total{status}`
   - `rcn_kernel_sessions_active`
   - `rcn_git_commits_total`
   - Go runtime: `go_goroutines`, `go_memstats_alloc_bytes`, etc.

Dùng promhttp.Handler() cho exposition format.
