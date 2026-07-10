# Task 1.3: Iceberg Support

## Mục tiêu
Thêm Apache Iceberg support vào RCN kernel image và SparkSession.

## Phạm vi
- Thêm Iceberg jars vào kernel Dockerfile
- Cấu hình SparkSession với Iceberg catalog (Hadoop catalog, warehouse trên MinIO)
- Thêm helper function cho notebook (Python + Scala)
- Test Iceberg table creation và query

## Chi tiết kỹ thuật
- Iceberg Spark runtime jar: `org.apache.iceberg:iceberg-spark-runtime-3.5_2.12:1.6.1`
- Catalog: Hadoop catalog với warehouse trên MinIO
  ```
  spark.sql.catalog.iceberg = org.apache.iceberg.spark.SparkCatalog
  spark.sql.catalog.iceberg.type = hadoop
  spark.sql.catalog.iceberg.warehouse = s3a://workspace/iceberg-warehouse/
  ```

## Task này sẽ làm sau khi hoàn thành Task 1.1 (Spark Jobs API)
