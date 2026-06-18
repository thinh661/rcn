# RCN Project Rules & Context (AI Guidelines)

This file serves as a context stabilizer and guide to help AI coding assistants quickly understand the project architecture, enforce design patterns, and save prompt tokens.

---

## 1. Project Overview & Architecture

**SparkLabX** is a self-hosted, Jupyter-style notebook platform designed for **Apache Spark (PySpark & Scala)**. It features end-to-end user isolation via MinIO S3 IAM and dynamic kernel container provisioning.

### Technology Stack
*   **Frontend**: React 19 (TypeScript), Vite, Tailwind CSS v4, Monaco Editor (code editor), Radix UI (UI primitives), TanStack React Query (server-state).
*   **Backend**: Go (Gin Web Framework), PostgreSQL (metadata storage), MinIO Client SDK & Admin SDK (`madmin-go`).
*   **Kernel Engine**: Jupyter Kernel Gateway, Apache Spark (with S3A Hadoop client), Almond Fork (Scala Kernel), DataFlint (Spark performance plugin).

---

## 2. Directory Layout & Core Components

```
📂 sparklabx/
 ├── 📂 backend/
 │    ├── 📂 cmd/server/main.go        # HTTP Server Entry & Service Dependency Wiring
 │    └── 📂 internal/
 │         ├── 📂 config/               # Environment Configuration (.env mapping)
 │         ├── 📂 database/             # Schema Migrations (migrations.go) & DB Pool
 │         ├── 📂 handlers/             # Gin Controllers (local_kernel.go, storage.go, auth.go)
 │         ├── 📂 middleware/           # RequireAdmin, RateLimit, RequireKernelToken
 │         └── 📂 services/             # MinIO IAM, Kernel Gateways (shared/docker/k8s), Recorder
 ├── 📂 frontend/
 │    └── 📂 src/
 │         ├── 📂 components/           # Radix UI + Tailwind components (NotebookPage.tsx is main)
 │         ├── 📂 hooks/                # useJupyterKernel.ts (WebSocket management), useNotebook.ts
 │         └── 📂 services/             # Backend API Clients (authService.ts, notebookService.ts)
 ├── 📂 kernel/
 │    ├── 🐳 Dockerfile                # Kernel Base with PySpark, Almond, Trino, DataFlint Jars
 │    ├── 📄 entrypoint.sh             # Dynamic hadoop-aws / spark S3 config builder
 │    └── 📂 helpers/                  # Python/Scala helper functions (query(), display())
 └── 📂 chart/                         # Kubernetes Helm chart
```

---

## 3. Core Design Patterns & Constraints

### ⚠️ Token-Saving Tips (Read Before Analyzing Files)
*   **`local_kernel.go` is very large (~40KB)**. Do NOT read the entire file unless necessary. Only view specific line numbers for handlers you need to modify (e.g., `Connect`, `WebSocket`, `Usage`, `Shutdown`).
*   **`useJupyterKernel.ts` is very large (~80KB)**. It handles complex WebSocket messaging, cells execution outputs, and state transitions. Read specific methods instead of the entire file.

### A. User Storage Isolation (S3 prefix constraints)
*   **MinIO IAM is the source of truth**: Do NOT implement access control for user files only at the application layer. Every user gets a dedicated MinIO credentials key/secret generated at first login and stored AES-GCM encrypted in the DB.
*   **Prefix layout**: `users/<username>/` (private) and `public/` (shared read-write).
*   **Policy**: Enforced directly at MinIO using `madmin-go` policy generation (see `minio_iam.go`). The kernel pod runs with the user's specific credentials, preventing cross-user S3 access (`AccessDenied`).

### B. OIDC & Connector Authentication (SSO Passthrough & App-as-Issuer)
*   **Decoupled Authentication**: The application acts as its own OIDC Issuer (RS256 JWTs) and publishes its JWKS at `/api/v1/.well-known/jwks.json`.
*   **Connecting from Kernel**: When code running on the Jupyter kernel needs to execute a query via `query("connector_id", "SELECT...")`, it contacts the backend API `/api/v1/connectors/:id/credentials` using a short-lived `SPARKLABX_KERNEL_TOKEN`.
*   **Credentials Resolving**:
    *   `app-jwt` (default): The app mints a custom RS256 JWT representing the user's SSO identity. External systems (like Trino) validate this token against the app's JWKS.
    *   `broker-mapped`: Uses personal credentials (username/password) stored AES-GCM encrypted in the database.

### C. Kernel Provisioning (KERNEL_MODE)
*   Three modes exist: `shared` (dev/demo), `docker_per_user` (single host, socket mount), and `k8s_per_user` (production).
*   Pods are spawned dynamically on-demand at notebook connect, updated via DB status polling (`user_kernel_pods` table), and auto-reaped when idle.

### D. WebSockets & Persistence (Jupyter Kernel Gateway)
*   WebSocket proxying (`/api/v1/notebooks/:id/kernel/ws/...`) sniffs `execute_request` to map message IDs to specific notebook cells.
*   A background **Kernel Recorder** runs in the Go backend to process output streams asynchronously. Cell outputs and execution state persist to PostgreSQL, allowing users to reload the page or disconnect without losing cell outputs.

---

## 4. Coding & Modification Standards

1.  **Preserve existing comments**: When modifying backend or frontend files, keep existing architectural comments intact.
2.  **Database schema updates**: Add new database schema modifications at the end of the `migrations` slice in `backend/internal/database/migrations.go` using idempotent SQL (`ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`).
3.  **Tailwind CSS**: The frontend uses Tailwind CSS v4. Standard Tailwind classes and class merging (`cn`) utilities should be used for styling.
4.  **Error boundaries**: Cell outputs render inside `CellOutputErrorBoundary` in `CellOutputRenderer.tsx` to isolate rendering crashes. Maintain this boundary protection for any modifications to the output viewer.
