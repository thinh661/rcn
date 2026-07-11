# Task 3.1: Multi-tenancy & RBAC

## Mục tiêu
Mở rộng hệ thống phân quyền từ admin/superadmin thành multi-role: admin, editor, viewer.

## Phạm vi
1. DB migration: Add `role` enum ('superadmin', 'admin', 'editor', 'viewer') nếu chưa có
2. Backend middleware mới:
   - `RequireRole(roles ...string)` — check user role from JWT
   - `RequireNotebookAccess(minRole string)` — check user có quyền truy cập notebook
3. Update `admins` table: thêm `organization_id`, `team_id`
4. API endpoints:
   - `GET /api/v1/admin/roles` — list roles & permissions
   - `PUT /api/v1/admin/users/:id/role` — superadmin set user role (đã có)
5. Frontend: role badge hiển thị trong user list, role selector trong user management

## Ràng buộc
- Notebook permission: viewer (read-only), editor (read+write), admin (full control)
- API endpoints mặc định require admin, các endpoint read có thể mở cho viewer
