# Task 5.1: Data Catalog

## Backend (main)
1. Migration: `data_catalog` table (id, name, type, parent_id, schema_json, metadata JSONB, created_at, updated_at)
2. Service: `services/catalog.go` — CRUD catalog entries, search, browse tree
3. Handler: `handlers/catalog.go`:
   - `GET /api/v1/catalog/tree` — full catalog tree
   - `GET /api/v1/catalog/search?q=` — search tables/columns
   - `GET /api/v1/catalog/:id` — detail với schema + lineage
   - `GET /api/v1/catalog/:id/preview?limit=10` — preview data
   - `POST /api/v1/catalog/:id/lineage` — ghi lineage info
4. Sync từ Iceberg/Trino metadata

## Frontend (agy)
1. Page: `frontend/src/pages/Catalog.tsx` — tree browser + search + preview
2. Route: `/catalog`
