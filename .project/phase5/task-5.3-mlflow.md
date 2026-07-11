# Task 5.3: MLflow Integration

## Backend (main)
1. Config: MLFLOW_TRACKING_URI
2. Service: `services/mlflow.go` — proxy MLflow API:
   - `GET /api/v1/mlflow/experiments` — list experiments
   - `POST /api/v1/mlflow/experiments` — create experiment  
   - `GET /api/v1/mlflow/runs?experiment_id=` — list runs
   - `POST /api/v1/mlflow/runs` — log run
   - `GET /api/v1/mlflow/models` — model registry
   - `POST /api/v1/mlflow/models/register` — register model
3. UI embed: iFrame MLflow UI hoặc proxy API
4. Notebook helper: `mlflow.autolog()` support trong kernel
