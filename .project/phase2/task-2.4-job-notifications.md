# Task 2.4: Job Notifications (Webhook)

## Mục tiêu
Gửi thông báo khi Spark batch job hoàn thành hoặc thất bại.

## Bối cảnh
- SparkOperator có sẵn cơ chế webhook notification
- Backend Go với Gin, có thể thêm handler nhận callback
- Hoặc dùng SparkOperator's `notification` field trong CRD spec

## Phạm vi
1. Add `webhook_url` field to `spark_jobs` table
2. Backend endpoint: `POST /api/v1/spark/jobs/:id/webhook` để đặt URL
3. SparkApplication spec: thêm `notification` field:
   ```yaml
   spec:
     notification:
       api:
         url: http://backend:10000/api/v1/spark/callback
   ```
4. Backend callback handler nhận SparkOperator webhook → update job status
5. Hoặc gửi webhook ra ngoài (Slack, Teams, Email) khi job done/fail

## Result
- User nhận notification khi job chạy xong
- Webhook URL configurable per job
