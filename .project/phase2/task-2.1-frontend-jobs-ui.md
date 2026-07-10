# Task 2.1: Frontend Batch Jobs UI

## Mục tiêu
Tạo giao diện quản lý Spark batch jobs trong RCN frontend.

## Bối cảnh
- Backend API đã có: `GET/POST /api/v1/spark/jobs`, `GET/DELETE /api/v1/spark/jobs/:id`, `GET /api/v1/spark/jobs/:id/logs`
- Frontend là React + Vite + TypeScript (thư mục `frontend/src/`)
- Cần thêm route `/spark-jobs` và components mới

## Phạm vi
1. Route `/spark-jobs` trong frontend router
2. Page `SparkJobsPage.tsx`:
   - List jobs table (name, type, status, created_at, actions)
   - Submit job form (name, type: Scala/Python, main class, main app file, arguments, resources)
   - Job detail view (logs, status, CRD name)
   - Stop job action
3. Integrate với backend API

## Thiết kế
- Route: `/spark-jobs` → `SparkJobsPage`
- Trang list: table với cột Name, Type, Status (badge), Created, Actions (View/Stop)
- Nút "New Job" → modal/dialog form submit
- Trang detail: job info + logs viewer
