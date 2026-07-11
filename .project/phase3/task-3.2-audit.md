# Task 3.2: Audit Logging

## Mục tiêu
Ghi log tất cả API actions (CRUD) vào Postgres để superadmin có thể xem lịch sử.

## Phạm vi
1. DB migration: `audit_logs` table (id, user_id, action, resource_type, resource_id, details JSONB, ip_address, created_at)
2. Backend middleware: `AuditLog(action, resourceType)` ghi log sau mỗi request thành công
3. API endpoint: `GET /api/v1/admin/audit-logs` — list audit logs (superadmin only) với filters:
   - `?user_id=`, `?action=`, `?resource_type=`, `?from=`, `?to=`, `?page=`, `?limit=`
4. Integration: gắn middleware vào các route chính (notebooks, connectors, spark jobs, users)

## Thiết kế
- Gin middleware pattern: log after c.Next() when status < 500
- Capture: method, path, user_id, status, latency
- Auto-cleanup: xóa logs > 90 ngày (optional)
