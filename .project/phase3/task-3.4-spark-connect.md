# Task 3.4: Spark Connect (gRPC Remote SparkSession)

## Mục tiêu
Hỗ trợ Spark Connect — gRPC protocol cho phép notebook kết nối tới Spark cluster từ xa qua `spark.connect("sc://...")`.

## Phạm vi
1. Config: `SPARK_CONNECT_ENABLED`, `SPARK_CONNECT_ENDPOINT`, `SPARK_CONNECT_PORT`
2. DB migration: `spark_connect_sessions` table (id, user_id, session_id, status, created_at, last_active_at)
3. API endpoints:
   - `GET /api/v1/spark-connect/config` — connection string cho UI copy
   - `POST /api/v1/spark-connect/sessions` — create new SparkConnect session
   - `GET /api/v1/spark-connect/sessions` — list user's sessions
   - `DELETE /api/v1/spark-connect/sessions/:id` — close session
4. UI: dashboard hiển thị active sessions + connection string có sẵn

## Constraints
- Spark 3.4+ required cho Spark Connect
- gRPC port mặc định: 15002
- Deploy Spark Connect server như một K8s Service riêng
