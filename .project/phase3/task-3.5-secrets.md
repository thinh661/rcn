# Task 3.5: Secret Management

## Mục tiêu
Tăng cường bảo mật cho connector passwords và credentials bằng encrypted secrets store.

## Bối cảnh
- Hiện tại connector passwords đã được AES-GCM encrypt trước khi lưu vào DB (services.MinIOIAM.EncryptSecret)
- App secrets lưu trong `app_secrets` table

## Phạm vi
1. Thêm `secret_vault` table: (key VARCHAR PK, encrypted_value TEXT, rotation_version INT, created_at, updated_at)
2. Key rotation support: re-encrypt secrets với new key
3. API endpoints (superadmin only):
   - `GET /api/v1/admin/secrets` — list secret keys (không show value)
   - `PUT /api/v1/admin/secrets/:key` — update secret
   - `DELETE /api/v1/admin/secrets/:key` — delete secret
4. Frontend: secret management page trong admin settings
5. Auto-rotation hook: trigger re-encryption khi master key thay đổi
