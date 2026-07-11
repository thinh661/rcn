# Task 3.6: Resource Usage & Cost Tracking

## Mục tiêu
Theo dõi resource usage (CPU, memory, storage, Spark jobs) và ước tính chi phí theo user/team.

## Phạm vi
1. DB migration:
   - `resource_usage` table (id, user_id, resource_type, amount, unit, cost_estimate, currency, period_start, period_end, created_at)
   - `cost_rates` table (id, resource_type, rate_per_unit, currency, effective_from)
2. Service: collect usage từ kernel pods + spark jobs + storage usage
3. API endpoints:
   - `GET /api/v1/admin/resource-usage` — superadmin: all usage (filter: user_id, from, to, page, limit)
   - `GET /api/v1/resource-usage/my` — current user's usage
   - `GET /api/v1/admin/resource-usage/summary` — aggregated summary
   - `PUT /api/v1/admin/cost-rates/:resource_type` — superadmin: set cost rate
   - `GET /api/v1/admin/cost-rates` — list current rates
4. Background worker: collect usage định kỳ (cron job 5 phút)

## Rate defaults (VND):
- Spark CPU: 500 VND/CPU-hour
- Spark Memory: 100 VND/GB-hour
- Kernel CPU: 1000 VND/CPU-hour
- Kernel Memory: 200 VND/GB-hour
- Storage: 50 VND/GB-month
