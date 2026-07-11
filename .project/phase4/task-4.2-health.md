# Task 4.2: Health Checks + System Health API

## Mục tiêu
Cung cấp health check endpoints cho K8s liveness/readiness probes + system health dashboard API.

## Implementation
1. Backend: Thêm `GET /health` (public) — lightweight ping DB
2. `GET /api/v1/admin/system/health` (superadmin) — detailed health:
   - Database: ping, connection stats
   - MinIO: bucket list test
   - Spark Operator: check CRD availability
   - Kernel Gateway: status
   - Git: config check
   - Spark Connect: service reachability
   - Response: `{status: "ok"|"degraded"|"down", checks: [...]}`
3. `GET /api/v1/admin/system/info` (superadmin) — system metadata:
   - Version (git commit from build)
   - Uptime
   - Active users count
   - Active kernels count
   - Total notebooks count
   - Total Spark jobs count
