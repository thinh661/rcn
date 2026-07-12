# RCN Code Review Report

## 🐛 Critical Bugs
- [ ] **Bug 1: Race Condition / Concurrent `spawnAndWait` Execution**
  - **File**: `backend/internal/services/k8s_per_user_gateway.go:367`
  - **Impact**: In `GetGatewayURL`, the backend initiates an asynchronous goroutine to spawn a pod via `EnsureSpawning` and then immediately calls `spawnAndWait` synchronously in the same thread. This results in two concurrent execution flows executing `.Create()` and `.Watch()` on the same pod name, creating race conditions, API conflicts in K8s, and duplicate state persistence/updates.
  - **Fix**: Modify `GetGatewayURL` to only trigger `EnsureSpawning` and wait for the status in the DB or via a channel, rather than calling `spawnAndWait` again.

- [ ] **Bug 2: Server Crash on Invalid Resource Specs (`MustParse` Panic)**
  - **File**: `backend/internal/services/k8s_per_user_gateway.go:775-780` & `backend/internal/services/spark_jobs.go:467`
  - **Impact**: The application utilizes `resource.MustParse` to parse CPU/Memory configurations directly from HTTP request payloads (in Spark Job submission) or from DB rows (in K8s Per User Gateway). If a user inputs an invalid resource string (e.g. `"invalid-cpu"`), `MustParse` will trigger a panic, causing the current goroutine or the entire backend server to crash.
  - **Fix**: Replace `resource.MustParse` with `resource.ParseQuantity` and handle errors gracefully (e.g., return `400 Bad Request` or fall back to defaults).

- [ ] **Bug 3: K8s API Context Timeout Leak in Reaper Loops**
  - **File**: `backend/internal/services/k8s_per_user_gateway.go:1039` & `backend/internal/services/k8s_per_user_gateway.go:1099`
  - **Impact**: Both `reapDeadPods()` and `sweepOrphanPods()` create a single context with a 30-second timeout at the start of the function. Inside the loop, they make sequential K8s API requests (Get and Delete) for each pod. If there are a large number of pods, the accumulated network latency will quickly exceed the 30-second timeout, causing the context to expire and failing all subsequent pod cleanups.
  - **Fix**: Use individual contexts with small, isolated timeouts for each API request inside loops, or perform operations concurrently.

- [ ] **Bug 4: K8s CRD Leak on Database Persistence Failure**
  - **File**: `backend/internal/services/spark_jobs.go:146-209`
  - **Impact**: In `Submit()`, the SparkApplication CRD is created in Kubernetes first. The backend then inserts the metadata into PostgreSQL. If the database insertion fails, the backend merely logs a warning and proceeds. This leaks the running CRD in K8s (creating an orphan job) which the backend has no record of, making it unmanageable and un-stoppable.
  - **Fix**: Wrap the K8s creation and SQL insertion in a transaction-like rollback structure, or delete the CRD from K8s if the SQL query fails.

- [ ] **Bug 5: Missing DAG Validation & Topological Order Execution**
  - **File**: `backend/internal/handlers/workflows.go` & `backend/internal/services/workflows.go`
  - **Impact**: The workflow engine is missing logic to detect cycles when tasks are added, allowing users to define circular task dependencies that can hang the system or run indefinitely. Additionally, topological order execution is completely unimplemented (runs are marked `running` but no tasks are scheduled).
  - **Fix**: Implement cycle detection and topological sorting in the service layer (now implemented via `ValidateDAG` and `TopologicalSort` helpers).

## ⚠️ Warnings
- [ ] **Warning 1: Broken Kernel Connections on Backend Restart (In-memory State Loss)**
  - **File**: `backend/internal/handlers/local_kernel.go:937` & `backend/internal/handlers/local_kernel.go:1103`
  - **Impact**: The websocket proxy and HTTP proxy verify notebook ownership via a package-level, in-memory map `kernelMap`. If the backend restarts, this map is wiped clean. Active proxy connections will be rejected with `403 Forbidden` despite the kernel pods still running. Users are forced to restart their kernels, losing all in-memory computation state.
  - **Fix**: Persist `kernelMap` associations in a persistent store like Redis or PostgreSQL.

- [ ] **Warning 2: Sensitive OIDC Tokens Leaking in Pod Environment Specs**
  - **File**: `backend/internal/services/k8s_per_user_gateway.go:731-740`
  - **Impact**: The user's OIDC access token (`RCN_KERNEL_TOKEN`) is passed as a plaintext environment variable in the pod specification. Anyone with read access to the pod configuration (e.g. `kubectl get pod -o yaml`) can read and hijack the user's identity.
  - **Fix**: Inject the token dynamically via a temporary, scoped Kubernetes Secret or mount it as a projected volume.

- [ ] **Warning 3: Tight-Loop API Hammer on K8s Watch Disconnect**
  - **File**: `backend/internal/services/k8s_per_user_gateway.go:522-543`
  - **Impact**: If the K8s watch channel drops (normal after ~30m or due to network blips), the watch loop in `watchUntilReady` immediately attempts to reopen it in a tight loop. If the API server is down or throttling, this will overwhelm the control plane.
  - **Fix**: Implement an exponential backoff with jitter between watch retries.

- [ ] **Warning 4: Incomplete RBAC Checks in Middleware**
  - **File**: `backend/internal/middleware/auth.go:191-222`
  - **Impact**: The `RequireNotebookAccess` middleware only restricts access if the role is explicitly `"viewer"`. For `"editor"` or other roles (like candidates/students), it bypasses authorization checks. This creates a false sense of security and leaves endpoints exposed if developers forget to check permissions inside handlers.
  - **Fix**: Check authorization constraints and ownership for all roles uniformly in the middleware.

- [ ] **Warning 5: No Role Hierarchy Support in `RequireRole`**
  - **File**: `backend/internal/middleware/auth.go:163-186`
  - **Impact**: The middleware performs an exact string comparison (`roleStr == allowed`). If a route requires `"editor"` access, it will deny `"admin"` or `"superadmin"` users unless they are explicitly enumerated in the arguments.
  - **Fix**: Implement hierarchical role validation where higher-privilege roles inherit permissions of lower-privilege roles.

## 💡 Suggestions
- [ ] **Suggestion 1: Unbuffered Database Writes in Batch Touch**
  - **File**: `backend/internal/services/k8s_per_user_gateway.go:907-909`
  - **Description**: `flushTouchLoop` iterates through the touch buffer and executes `db.Exec` sequentially for each user. Under high concurrency, this can exhaust the database connection pool. Implementing a bulk update query or batch transactions would improve database throughput.

- [ ] **Suggestion 2: Hop-by-Hop Headers Propagation in HTTP Proxy**
  - **File**: `backend/internal/handlers/local_kernel.go:1128-1130`
  - **Description**: The HTTP proxy replicates all incoming headers to the Jupyter Gateway, including hop-by-hop headers (e.g. `Connection`, `Keep-Alive`, `Host`). This violates RFC standards and can cause connection failures. Hop-by-hop headers should be filtered out before proxying.

- [ ] **Suggestion 3: Missing Context Propagation in DB Queries**
  - **File**: `backend/internal/services/k8s_per_user_gateway.go:908`, `:1005`, `:1073`
  - **Description**: Background loops and touch handlers use `db.Exec` and `db.Query` instead of `db.ExecContext` and `db.QueryContext`. This prevents context cancellation and timeouts from propagating to DB operations, potentially leading to dangling queries during application shutdown.
