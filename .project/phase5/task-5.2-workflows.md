# Task 5.2: Workflows (DAG Jobs)

## Backend (main)
1. Migration: `workflows` + `workflow_tasks` + `workflow_runs` tables
2. Service: `services/workflows.go` — DAG engine:
   - Validate DAG (no cycles)
   - Execute tasks in topological order
   - Track run status (pending/running/success/failed)
   - Retry logic + timeout
3. Handler: `handlers/workflows.go`:
   - CRUD workflows + tasks
   - `POST /api/v1/workflows/:id/run` — trigger run
   - `GET /api/v1/workflows/:id/runs` — run history
   - `GET /api/v1/workflows/runs/:runId` — run detail + task statuses

## Frontend (agy)
1. Page: `frontend/src/pages/Workflows.tsx` — DAG visual editor + run monitor
2. Route: `/workflows`
