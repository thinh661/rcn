# Task 2.2: Scheduled Jobs API (Cron Scheduling)

## Mục tiêu
Hỗ trợ lập lịch Spark batch jobs định kỳ qua `ScheduledSparkApplication` CRD.

## Bối cảnh
- Spark Operator đã cài, CRD `ScheduledSparkApplication` đã register
- Backend có `SparkJobService` cho ad-hoc jobs (Task 1.1)
- Cần thêm API cho scheduled jobs

## Phạm vi
1. Backend service + handler cho scheduled jobs
2. Sử dụng CRD `ScheduledSparkApplication` với `schedule` (cron expression)
3. DB migration: `spark_scheduled_jobs` table
4. REST API: CRUD scheduled jobs (name, schedule, template, enabled)

## API
```
GET    /api/v1/spark/scheduled-jobs          → List
POST   /api/v1/spark/scheduled-jobs          → Create
GET    /api/v1/spark/scheduled-jobs/:id      → Get
PUT    /api/v1/spark/scheduled-jobs/:id      → Update
DELETE /api/v1/spark/scheduled-jobs/:id      → Delete
PATCH  /api/v1/spark/scheduled-jobs/:id/toggle → Enable/disable
```

## Kết quả mong đợi
- User tạo scheduled job với cron expression
- Spark Operator tự động tạo SparkApplication theo lịch
- Backend theo dõi trạng thái từ CRD
