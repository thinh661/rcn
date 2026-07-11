# Task 5.9: Unit + Integration Tests

## Backend (agy)
1. Unit tests cho services:
   - `services/crypto_test.go` (đã có) — expand
   - `services/git_integration_test.go`
   - `services/resource_usage_test.go`
   - `services/spark_connect_test.go`
   - `services/monitoring_test.go`
2. Integration tests:
   - DB migration tests
   - API endpoint tests with httptest
   - Test containers (PostgreSQL + MinIO)

## Frontend (agy)
1. Component tests với Vitest + Testing Library
2. Test các page: Login, Notebook, Spark Jobs

## Tooling
1. `go test ./...` — backend
2. `npx vitest` — frontend
