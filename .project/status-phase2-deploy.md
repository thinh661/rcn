# Status: Phase 2 Deploy — Scheduled Jobs API + Webhooks + Frontend UI

**Date:** 2026-07-10T16:59 UTC
**Status:** ✅ COMPLETED

## What was done

### Build Fixes
Commit `4728b17` and `3a5091a` introduced code with compilation errors:

1. **`spark_jobs.go:190: undefined: webhookURL`** — variable used before declaration. Moved `webhookURL := req.WebhookURL` before the `job := &SparkJob{...}` struct literal.
2. **`spark_job_templates.go: unused import encoding/json`** — Removed the unused import.
3. **Missing functions referenced in `main.go`:**
   - `database.SeedResourcePresetsFromEnv(cfg)` — Added stub in `database.go`
   - `(*LocalKernelHandler).UpdateResourcePresetsFromDB()` — Added stub method on `LocalKernelHandler` in `local_kernel.go`
   - `handlers.NewResourcePresetsAdminHandler(kh)` — Created new file `resource_presets_admin.go` with `List`, `Upsert`, `Delete` endpoints (stubs returning 501).

### Image Build & Deployment

1. **DinD pod** (`builder-pod`) created in `rcn` namespace with privileged mode + docker
2. **Backend image** (`ghcr.io/sparklabx/backend:latest`) built in pod, exported as tar
3. **Frontend image** (`ghcr.io/sparklabx/frontend:latest`) built in pod, exported as tar
4. Imports into k3s containerd via `host-access` privileged pod with `/` host mount
5. `imagePullPolicy` already set to `IfNotPresent`
6. Rollout restart of backend + frontend deployments

### Verification

| Check | Result |
|-------|--------|
| `GET /health` | ✅ 200 `{"status":"ok"}` |
| `GET /api/v1/spark/scheduled-jobs` | ✅ 401 (route exists, auth required) |
| `POST /api/v1/spark/callback` | ✅ 200 `{"received":true}` |
| `POST /api/v1/spark/jobs/:id/webhook` | ✅ 401 (route exists, auth required) |
| Frontend pod | ✅ Running, serves HTML |
| Backend pod | ✅ Running, no CrashLoopBackOff |
| `kubectl get scheduledsparkapplications` | ✅ CRD registered |

## Files Changed (not yet committed)

| File | Change |
|------|--------|
| `backend/internal/services/spark_jobs.go` | Fixed `webhookURL` use-before-declare |
| `backend/internal/services/spark_job_templates.go` | Removed unused `encoding/json` import |
| `backend/internal/database/database.go` | Added `SeedResourcePresetsFromEnv` stub |
| `backend/internal/handlers/local_kernel.go` | Added `UpdateResourcePresetsFromDB` stub method |
| `backend/internal/handlers/resource_presets_admin.go` | **New file** — admin CRUD handler (stub) |

## Commands Used

```bash
# DinD setup
kubectl -n rcn run builder-pod --image=ubuntu:22.04 --privileged -- sleep infinity
kubectl -n rcn exec builder-pod -- apt-get install -y docker.io docker-buildx
kubectl -n rcn exec builder-pod -- dockerd --storage-driver=vfs &

# Build backend
kubectl -n rcn exec builder-pod -- docker build -t ghcr.io/sparklabx/backend:latest -f backend/Dockerfile .

# Build frontend
kubectl -n rcn exec builder-pod -- docker build -t ghcr.io/sparklabx/frontend:latest -f frontend/Dockerfile frontend/

# Export + import
kubectl -n rcn exec builder-pod -- docker save ... -o /tmp/backend-image.tar
kubectl cp builder-pod:/tmp/backend-image.tar local.tar
kubectl cp local.tar host-access:/host/tmp/
kubectl exec host-access -- chroot /host ctr -n k8s.io images import /tmp/backend-image.tar

# Deploy
kubectl -n rcn rollout restart deploy/backend
kubectl -n rcn rollout restart deploy/frontend

# Verify
kubectl -n rcn exec deploy/backend -- wget -q -O- http://localhost:10000/health
kubectl -n rcn exec deploy/backend -- wget -q -O- http://localhost:10000/api/v1/spark/scheduled-jobs
kubectl -n rcn exec deploy/backend -- wget -q -O- --post-data='{"applicationName":"test",...}' http://localhost:10000/api/v1/spark/callback
```

## Known Issues

1. **Uncommitted fixes** — The 3 compilation fixes + new handler file need to be committed to `main`
2. **No ghcr.io push** — Images only in local containerd, not pushed to registry
3. **Resource presets admin CRUD** — Stubbed out; full DB-backed implementation needed later
4. **Spotless/ScheduledSparkApplication CRD check** — The route exists but if the CRD isn't deployed by Spark Operator, the service init will log a warning and disable scheduled jobs
