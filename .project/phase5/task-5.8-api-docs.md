# Task 5.8: API Docs (Swagger/OpenAPI)

## Backend (main)
1. Add gin-swagger: `github.com/swaggo/gin-swagger` + `github.com/swaggo/files`
2. Annotate tất cả handlers với Swagger comments
3. Endpoint: `GET /api/v1/swagger/*any`
4. Generate: `swag init` tạo `docs/` folder
5. UI: Swagger UI tại `/api/v1/swagger/index.html`

## Scope
- Tất cả handlers có comment đầy đủ
- Request/response models
- Auth header documentation
