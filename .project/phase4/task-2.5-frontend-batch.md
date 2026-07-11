# Task 2.5: Batch Dashboard (Frontend)

## Yêu cầu
Frontend page dashboard monitoring cho Spark batch jobs — migrated từ Phase 2 backlog.

## UI Components
1. **Jobs Overview**: total, running, succeeded, failed counts
2. **Jobs Table**: list với filters (status, date, user)
3. **Schedule Status**: cron job health indicators
4. **Execution Timeline**: mini chart jobs over time

## API cần dùng
- `GET /api/v1/spark/jobs?limit=50`
- `GET /api/v1/spark/scheduled-jobs`
- `GET /api/v1/admin/resource-usage?resource_type=spark_cpu&from=...&to=...`

Đặt route ở `/batch/dashboard` trong frontend.
