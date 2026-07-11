# Task 5.6: Billing Dashboard

## Backend (main)
1. Service: mở rộng `resource_usage.go` thêm:
   - `GET /api/v1/admin/billing/forecast` — dự báo chi phí
   - `GET /api/v1/admin/billing/daily?from=&to=` — chi phí theo ngày
   - `GET /api/v1/admin/billing/by-user` — cost breakdown theo user
   - `GET /api/v1/admin/billing/invoices` — invoice history

## Frontend (agy)
1. Page: `frontend/src/pages/Billing.tsx`
2. Charts: daily cost, by-user, forecast
3. Route: `/admin/billing`
