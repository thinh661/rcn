# Task 1.1: Batch Jobs API + SparkApplication CRD Integration

## Mục tiêu
Thêm REST API cho phép submit, list, stop Spark batch jobs thông qua Spark Operator CRD (`SparkApplication`).

## Bối cảnh
- Spark Operator đã cài (namespace: spark-operator)
- CRD: `sparkapplications.sparkoperator.k8s.io` đã register
- RCN backend hiện chỉ support notebook (WebSocket proxy) chưa có batch job
- Backend chạy in-cluster (K8s service account đã có RBAC)

## Thiết kế

### 1. Backend Service: `spark_jobs_service.go`
File mới: `backend/internal/services/spark_jobs.go`

Interface và implementation giao tiếp với K8s API để manage SparkApplication CRDs:
```go
type SparkJob struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    UserID      string    `json:"user_id"`
    MainClass   string    `json:"main_class"`
    MainAppFile string    `json:"main_app_file"` // s3a:// or local://
    Arguments   []string  `json:"arguments"`
    SparkConf   map[string]string `json:"spark_conf"`
    DriverCPU   string    `json:"driver_cpu"`
    DriverMem   string    `json:"driver_memory"`
    ExecutorCPU string    `json:"executor_cpu"`
    ExecutorMem string    `json:"executor_memory"`
    ExecutorNum int32     `json:"executor_instances"`
    Status      string    `json:"status"` // SUBMITTED, RUNNING, COMPLETED, FAILED
    CreatedAt   time.Time `json:"created_at"`
}

type SparkJobService interface {
    Submit(ctx context.Context, job *SparkJob, userID string) error
    List(ctx context.Context, userID string) ([]SparkJob, error)
    Get(ctx context.Context, id string) (*SparkJob, error)
    Stop(ctx context.Context, id string) error
    GetLogs(ctx context.Context, id string, tail int) (string, error)
}
```

### 2. Backend Handlers: `spark_jobs_handler.go`
File mới: `backend/internal/handlers/spark_jobs.go`

REST endpoints:
```
GET    /api/v1/spark/jobs          → List jobs
POST   /api/v1/spark/jobs          → Submit job
GET    /api/v1/spark/jobs/:id      → Get job details
DELETE /api/v1/spark/jobs/:id      → Stop/delete job
GET    /api/v1/spark/jobs/:id/logs → Get job logs
```

Mỗi endpoint cần `middleware.RequireAdmin` (user phải login)

### 3. Database Migration
Thêm vào `backend/internal/database/migrations.go`:
```sql
CREATE TABLE IF NOT EXISTS spark_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES admins(id),
    name VARCHAR(255) NOT NULL,
    main_class TEXT NOT NULL DEFAULT '',
    main_app_file TEXT NOT NULL,
    arguments TEXT[] DEFAULT '{}',
    spark_conf JSONB DEFAULT '{}',
    driver_cpu VARCHAR(20) DEFAULT '1',
    driver_memory VARCHAR(20) DEFAULT '2g',
    executor_cpu VARCHAR(20) DEFAULT '1',
    executor_memory VARCHAR(20) DEFAULT '2g',
    executor_instances INTEGER DEFAULT 1,
    status VARCHAR(50) DEFAULT 'SUBMITTED',
    spark_application_name VARCHAR(255) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_spark_jobs_user_id ON spark_jobs(user_id);
```

### 4. Cấu trúc SparkApplication CRD
Khi submit job, backend cần tạo SparkApplication CRD object:
```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: rcn-job-{jobID}
  namespace: rcn
spec:
  type: Scala  # hoặc Python
  mode: cluster
  image: ghcr.io/sparklabx/kernel:latest
  imagePullPolicy: IfNotPresent
  mainClass: {mainClass}
  mainApplicationFile: {mainAppFile}
  arguments: [...]
  sparkVersion: "3.5.0"
  driver:
    cores: {driverCPU}
    coreLimit: {driverCPU}
    memory: {driverMem}
    serviceAccount: spark
  executor:
    cores: {executorCPU}
    instances: {executorNum}
    memory: {executorMem}
  sparkConf:
    spark.hadoop.fs.s3a.endpoint: http://minio:9000
    # ... more from user config
```

## Kết quả mong đợi
- REST API hoàn chỉnh cho batch job CRUD
- Job submit thành SparkApplication CRD
- Spark Operator quản lý lifecycle của job
- Frontend có thể gọi API này (UI sẽ làm task riêng)

## Điều kiện nghiệm thu
1. `curl POST /api/v1/spark/jobs` → job được submit, SparkApplication CRD xuất hiện
2. `curl GET /api/v1/spark/jobs` → list jobs
3. `curl DELETE /api/v1/spark/jobs/:id` → job stopped
4. Backend restart không mất job history (Postgres-backed)

## Ràng buộc
- Go client sử dụng `k8s.io/client-go` (in-cluster config vì backend chạy trong K8s)
- SparkApplication spec cần tương thích Spark 3.5.x
- Per-user isolation: job chạy với MinIO credentials của user đó
- Phải xử lý error cases: image pull fail, resource insufficient
