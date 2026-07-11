# Task 5.4: Notebook Scheduler

## Backend (agy)
1. Migration: `scheduled_notebooks` table
2. Service: `services/notebook_scheduler.go`:
   - Lưu cron schedule cho notebook
   - Tại mỗi trigger: spawn kernel, execute cells, collect output
   - Export result (HTML/ipynb) lưu vào storage
3. Handler: `handlers/notebook_scheduler.go`:
   - CRUD scheduled notebooks
   - `GET /api/v1/notebook-schedules` — list
   - `POST /api/v1/notebook-schedules` — create
   - `GET /api/v1/notebook-schedules/:id/runs` — run history

## Frontend (main)
1. UI trong NotebookPage: schedule dialog
2. List page ở `/notebook-schedules`
