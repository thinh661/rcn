# Task 5.10: Delta Lake / Delta Sharing

## Backend (agy)
1. Spark config: Delta Lake support trong kernel image
2. Delta Sharing server config:
   - Delta Sharing Docker service (docker-compose.yml)
   - API proxy: `GET /api/v1/delta-sharing/shares`
   - Create/revoke shares
3. Notebook helper: `delta.read()` function

## Config
1. Kernel: thêm delta-spark dependency
2. docker-compose: delta-sharing-server service
3. Chart: delta-sharing deployment
