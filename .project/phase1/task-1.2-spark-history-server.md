# Task 1.2: Spark History Server + Event Logs

## Mục tiêu
Deploy Spark History Server trên k3s, cấu hình event log lưu vào MinIO, và tích hợp vào RCN frontend.

## Bối cảnh
- Spark Operator đã được cài (namespace: spark-operator)
- MinIO đang chạy ở namespace rcn bucket: workspace
- RCN kernel image cần được cập nhật để bật event log
- Spark History Server cần được deploy và cấu hình kết nối MinIO S3

## Phạm vi công việc

### 1. Backend - Spark History Server deployment
- Tạo Spark History Server deployment/service/ingress trong Helm chart
- Sử dụng image: `apache/spark:3.5.5` với entrypoint là `org.apache.spark.deploy.history.HistoryServer`
- Cấu hình:
  - `SPARK_HISTORY_OPTS`:
    - `-Dspark.history.fs.logDirectory=s3a://workspace/event-logs/`
    - `-Dspark.hadoop.fs.s3a.endpoint=http://minio:9000`
    - `-Dspark.hadoop.fs.s3a.access.key=...` (từ secret)
    - `-Dspark.hadoop.fs.s3a.secret.key=...` (từ secret)
    - `-Dspark.hadoop.fs.s3a.path.style.access=true`
    - `-Dspark.history.ui.port=18080`
  - Service port 18080
  - Ingress: `shs.rcn.lakehouse.local` hoặc `rcn.lakehouse.local/spark-history`
  - Dùng cùng MinIO credentials từ Helm secret

### 2. Kernel image - event log configuration
- Cập nhật `kernel/entrypoint.sh`: thêm spark event log conf vào spark-defaults.conf
  ```
  spark.eventLog.enabled=true
  spark.eventLog.dir=s3a://workspace/event-logs/
  spark.history.fs.logDirectory=s3a://workspace/event-logs/
  ```
- Cập nhật `kernel/Dockerfile` nếu cần thêm Spark History Server libs

### 3. Frontend - Spark History Server UI link
- Thêm nút "Spark History Server" vào header hoặc sidebar
- Mở Spark History Server UI trong tab mới hoặc iframe
- Sử dụng cùng pattern như Spark UI proxy hiện có

## Kết quả mong đợi
- Spark History Server có thể truy cập được
- Kernel pod ghi event log vào MinIO khi chạy
- Spark History Server hiển thị lịch sử job sau khi kernel chạy
- Nút "Spark History" xuất hiện trong RCN frontend

## Điều kiện nghiệm thu
1. `kubectl get pods -n rcn | grep spark-history` → Running
2. Spark job chạy xong → event log file xuất hiện trong MinIO bucket
3. Spark History Server UI hiển thị được job đã chạy
4. Frontend có button link đến Spark History Server

## Ràng buộc
- Dùng MinIO (không PVC riêng) cho event log storage (PO đã duyệt)
- Event log phải support Spark 3.5.x
- S3A path style access (MinIO compatible)
