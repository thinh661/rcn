# Task 5.7: Team/Org Tree

## Backend (agy)
1. Migration: `organizations` + `teams` tables
2. Service: `services/organization.go`:
   - CRUD organizations, teams
   - Assign users to org/team
   - Org-level resource quotas
3. Handler: `handlers/organization.go`:
   - `GET /api/v1/admin/orgs` — list orgs (superadmin)
   - `POST /api/v1/admin/orgs` — create org
   - `GET /api/v1/admin/orgs/:id/teams` — list teams
   - `PUT /api/v1/admin/orgs/:id/quota` — set quota
   - `GET /api/v1/admin/orgs/tree` — full org tree

## Frontend (main)
1. Page: `frontend/src/pages/Organization.tsx` — org tree + member list
2. Route: `/admin/organization`
