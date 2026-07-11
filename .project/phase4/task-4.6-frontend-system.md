# Task 4.6: System Admin Dashboard (Frontend)

## Yêu cầu
Frontend page trong React app hiển thị system health và metrics.

## UI Components
1. **System Status Card**: health status (ok/degraded/down) từng service
2. **Metrics Row**: live counters (active users, kernels, jobs, notebooks)
3. **Spark Jobs Chart**: mini chart jobs theo status
4. **Usage Chart**: resource usage mini chart
5. **Uptime & Version**: system info footer

## API cần dùng
- `GET /api/v1/admin/system/health`
- `GET /api/v1/admin/system/info`
- `GET /api/v1/admin/resource-usage/summary`

Đặt route ở `/admin/system` trong frontend.
