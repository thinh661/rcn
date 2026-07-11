package database

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/rcn/rcn/backend/internal/config"
)

// MigrateAndSeed runs schema migrations and seeds initial data.
// Notebook-lite scope: admins + notebooks + storage + kernel infra.
// Exam-specific tables (sessions, candidates, submissions, grading, provision_logs)
// are intentionally absent — see main branch if you need them.
func MigrateAndSeed(cfg *config.Config) error {
	db := GetDB()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(20) DEFAULT 'admin',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`ALTER TABLE admins ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'admin'`,
		`UPDATE admins SET role = 'superadmin' WHERE username = (SELECT username FROM admins ORDER BY created_at ASC LIMIT 1) AND role = 'admin'`,
		// Encrypted MinIO IAM secret. Each admin gets a per-user MinIO account so
		// the kernel pod can only access their own users/<slug>/* prefix +
		// public/*. Empty until first login post-IAM-rollout — backfilled lazily.
		`ALTER TABLE admins ADD COLUMN IF NOT EXISTS minio_secret_enc TEXT NOT NULL DEFAULT ''`,
		// Encrypted OIDC tokens from SSO login, retained so the kernel can
		// authenticate to external services (e.g. Trino) as the logged-in user
		// via token passthrough. Empty unless the user logged in via OIDC SSO.
		`ALTER TABLE admins ADD COLUMN IF NOT EXISTS oidc_access_token_enc TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE admins ADD COLUMN IF NOT EXISTS oidc_refresh_token_enc TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE admins ADD COLUMN IF NOT EXISTS oidc_token_expires_at TIMESTAMPTZ`,

		`CREATE TABLE IF NOT EXISTS notebooks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL DEFAULT 'Untitled',
			description TEXT DEFAULT '',
			language VARCHAR(50) NOT NULL DEFAULT 'python',
			owner_id UUID NOT NULL,
			owner_type VARCHAR(20) NOT NULL DEFAULT 'admin',
			is_public BOOLEAN NOT NULL DEFAULT false,
			cluster_config JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notebooks_owner ON notebooks(owner_id, owner_type)`,

		`CREATE TABLE IF NOT EXISTS notebook_cells (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			notebook_id UUID NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
			type VARCHAR(20) NOT NULL DEFAULT 'code',
			source TEXT DEFAULT '',
			cell_order INTEGER NOT NULL DEFAULT 0,
			execution_count INTEGER,
			last_output JSONB,
			last_execution_time_ms INTEGER,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notebook_cells_notebook ON notebook_cells(notebook_id, cell_order)`,

		`CREATE TABLE IF NOT EXISTS allowed_email_rules (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			rule_type VARCHAR(20) NOT NULL,
			value VARCHAR(255) NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			added_by VARCHAR(255) NOT NULL DEFAULT '',
			note TEXT DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			CONSTRAINT allowed_email_rules_type_check CHECK (rule_type IN ('domain', 'exact_email')),
			CONSTRAINT allowed_email_rules_unique UNIQUE (rule_type, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_allowed_email_rules_lookup ON allowed_email_rules(enabled, rule_type, value)`,

		// Admin-managed data connectors (Trino, Postgres, MySQL, …). The TRINO_URL
		// env still seeds a non-deletable "trino" connector; these are the ones
		// added from the UI. password_enc is AES-GCM (services.MinIOIAM key) for
		// broker-mapped (user/password) sources; empty for app-jwt/idp-passthrough.
		`CREATE TABLE IF NOT EXISTS connectors (
			id VARCHAR(64) PRIMARY KEY,
			type VARCHAR(32) NOT NULL,
			label VARCHAR(128) NOT NULL,
			url TEXT NOT NULL,
			auth VARCHAR(32) NOT NULL,
			username VARCHAR(255) NOT NULL DEFAULT '',
			password_enc TEXT NOT NULL DEFAULT '',
			added_by VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		// Connector scope: owner_id = '' is a shared (org-wide) source; a non-empty
		// admin id makes it personal (visible only to that user). Existing rows
		// default to shared.
		`ALTER TABLE connectors ADD COLUMN IF NOT EXISTS owner_id VARCHAR(64) NOT NULL DEFAULT ''`,
		// Connectors are personal (owned by their creator). The id is the notebook
		// helper name and must be unique PER OWNER — so two users can each have a
		// "trino" — not globally. Swap the global PK on id for UNIQUE(owner_id, id).
		`DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'connectors_pkey') THEN
				ALTER TABLE connectors DROP CONSTRAINT connectors_pkey;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'connectors_owner_id_id_key') THEN
				ALTER TABLE connectors ADD CONSTRAINT connectors_owner_id_id_key UNIQUE (owner_id, id);
			END IF;
		END $$`,
		// Generic app-managed secrets (e.g. the connector signing key) — persisted
		// in the DB so they survive restarts without a dedicated volume.
		`CREATE TABLE IF NOT EXISTS app_secrets (key VARCHAR(64) PRIMARY KEY, value TEXT NOT NULL)`,
		`ALTER TABLE app_secrets ADD COLUMN IF NOT EXISTS rotation_version INTEGER NOT NULL DEFAULT 0`,

		// K8s per-user pod tracking
		`CREATE TABLE IF NOT EXISTS user_kernel_pods (
			user_id        TEXT PRIMARY KEY,
			pod_name       TEXT NOT NULL,
			pod_namespace  TEXT NOT NULL,
			pod_url        TEXT NOT NULL DEFAULT '',
			status         TEXT NOT NULL DEFAULT 'pending',
			phase_message  TEXT NOT NULL DEFAULT '',
			created_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			last_used_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_kernel_pods_idle ON user_kernel_pods(status, last_used_at)`,
		// Per-pod resources chosen at connect time (issue #41, k8s_per_user
		// presets). Empty string on legacy rows → gateway falls back to cluster
		// defaults, so no backfill is needed.
		`ALTER TABLE user_kernel_pods ADD COLUMN IF NOT EXISTS cpu_request TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE user_kernel_pods ADD COLUMN IF NOT EXISTS mem_request TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE user_kernel_pods ADD COLUMN IF NOT EXISTS cpu_limit TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE user_kernel_pods ADD COLUMN IF NOT EXISTS mem_limit TEXT NOT NULL DEFAULT ''`,

		// Notebook→kernel cache (UNLOGGED, regenerable)
		`CREATE UNLOGGED TABLE IF NOT EXISTS notebook_kernels (
			notebook_id  TEXT NOT NULL,
			user_id      TEXT NOT NULL,
			kernel_id    TEXT NOT NULL,
			last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (notebook_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notebook_kernels_user ON notebook_kernels(user_id)`,

			// Spark batch jobs table — persisted metadata for SparkApplication CRDs.
		`CREATE TABLE IF NOT EXISTS spark_jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES admins(id),
			name VARCHAR(255) NOT NULL,
			type VARCHAR(20) NOT NULL DEFAULT 'Scala',
			main_class TEXT NOT NULL DEFAULT '',
			main_app_file TEXT NOT NULL,
			arguments TEXT[] DEFAULT '{}',
			spark_conf JSONB DEFAULT '{}',
			driver_cpu VARCHAR(20) DEFAULT '1',
			driver_memory VARCHAR(20) DEFAULT '2g',
			executor_cpu VARCHAR(20) DEFAULT '1',
			executor_memory VARCHAR(20) DEFAULT '2g',
			executor_instances INTEGER DEFAULT 1,
			status VARCHAR(50) DEFAULT 'SUBMITTED',
			spark_application_name VARCHAR(255) DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_spark_jobs_user_id ON spark_jobs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spark_jobs_status ON spark_jobs(status)`,
		`ALTER TABLE spark_jobs ADD COLUMN IF NOT EXISTS webhook_url TEXT NOT NULL DEFAULT ''`,

		// Admin-managed resource presets for kernel pod sizing (issue #41, k8s_per_user).
		// id is the preset key (e.g. "small", "medium"), is_default controls which
		// one is pre-selected in the UI. Managed via the admin CRUD API.
		`CREATE TABLE IF NOT EXISTS resource_presets (
			id VARCHAR(64) PRIMARY KEY,
			label VARCHAR(128) NOT NULL DEFAULT '',
			cpu VARCHAR(20) NOT NULL,
			memory VARCHAR(20) NOT NULL,
			is_default BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Spark scheduled jobs table — persisted metadata for ScheduledSparkApplication CRDs.
		`CREATE TABLE IF NOT EXISTS spark_scheduled_jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES admins(id),
			name VARCHAR(255) NOT NULL,
			schedule VARCHAR(100) NOT NULL,
			template JSONB NOT NULL DEFAULT '{}',
			enabled BOOLEAN NOT NULL DEFAULT true,
			status VARCHAR(50) DEFAULT 'ACTIVE',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_spark_scheduled_jobs_user_id ON spark_scheduled_jobs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spark_scheduled_jobs_enabled ON spark_scheduled_jobs(enabled)`,

		// Reusable Spark job templates — persisted config that users submit jobs from.
		`CREATE TABLE IF NOT EXISTS spark_job_templates (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES admins(id),
			name VARCHAR(255) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			type VARCHAR(20) NOT NULL DEFAULT 'Scala',
			main_class TEXT NOT NULL DEFAULT '',
			main_app_file TEXT NOT NULL,
			arguments TEXT[] DEFAULT '{}',
			spark_conf JSONB DEFAULT '{}',
			driver_cpu VARCHAR(20) DEFAULT '1',
			driver_memory VARCHAR(20) DEFAULT '2g',
			executor_cpu VARCHAR(20) DEFAULT '1',
			executor_memory VARCHAR(20) DEFAULT '2g',
			executor_instances INTEGER DEFAULT 1,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_spark_job_templates_user_id ON spark_job_templates(user_id)`,

		// Audit log for tracking admin actions (create/update/delete) with
		// user context, resource identifiers, and request metadata.
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID,
			username VARCHAR(255) DEFAULT '',
			action VARCHAR(50) NOT NULL,
			resource_type VARCHAR(50) NOT NULL,
			resource_id VARCHAR(255) DEFAULT '',
			details JSONB DEFAULT '{}',
			ip_address VARCHAR(45) DEFAULT '',
			user_agent TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id)`,

	// Extend role values to support multi-tenancy: 'editor' and 'viewer' in
		// addition to 'admin' and 'superadmin'. Add a CHECK constraint to ensure only
		// valid roles are stored.
		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'admins_role_check'
			) THEN
				ALTER TABLE admins ADD CONSTRAINT admins_role_check
				CHECK (role IN ('admin', 'superadmin', 'editor', 'viewer'));
			END IF;
		END $$`,

		// Backfill: OAuth admins originally got username = full email. Storage
		// now uses username as a path segment (users/<username>/...), and "@" in
		// S3 keys triggers SigV4 SignatureDoesNotMatch under some clients. Rewrite
		// to a clean slug from the email local-part. PL/pgSQL handles UNIQUE
		// collisions by appending a short random suffix.
		`DO $$
		DECLARE
			r RECORD;
			new_slug TEXT;
		BEGIN
			FOR r IN SELECT id, email FROM admins WHERE username LIKE '%@%' LOOP
				new_slug := LOWER(REGEXP_REPLACE(
					SPLIT_PART(SPLIT_PART(r.email, '+', 1), '@', 1),
					'[^a-z0-9._-]+', '-', 'g'
				));
				new_slug := TRIM(BOTH '.-_' FROM new_slug);
				IF new_slug = '' THEN new_slug := 'user'; END IF;
				BEGIN
					UPDATE admins SET username = new_slug WHERE id = r.id;
				EXCEPTION WHEN unique_violation THEN
					UPDATE admins
					SET username = new_slug || '-' || SUBSTR(MD5(RANDOM()::TEXT), 1, 4)
					WHERE id = r.id;
				END;
			END LOOP;
		END $$`,

		// Phase 3.3: Notebook Git links for versioning
		`CREATE TABLE IF NOT EXISTS notebook_git_links (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			notebook_id UUID UNIQUE NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
			repo_url TEXT NOT NULL,
			branch VARCHAR(255) NOT NULL DEFAULT 'main',
			file_path TEXT NOT NULL,
			auth_token_enc TEXT NOT NULL DEFAULT '',
			last_committed_at TIMESTAMPTZ,
			last_commit_sha VARCHAR(64) DEFAULT '',
			last_committer VARCHAR(255) DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Phase 3.4: Spark Connect sessions
		`CREATE TABLE IF NOT EXISTS spark_connect_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES admins(id),
			session_id VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			endpoint VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			last_active_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_spark_connect_sessions_user ON spark_connect_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spark_connect_sessions_status ON spark_connect_sessions(status)`,

		// Phase 3.6: Resource usage and cost tracking
		`CREATE TABLE IF NOT EXISTS resource_usage (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES admins(id),
			resource_type VARCHAR(50) NOT NULL,
			amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			unit VARCHAR(20) NOT NULL DEFAULT '',
			cost_estimate DOUBLE PRECISION NOT NULL DEFAULT 0,
			currency VARCHAR(10) NOT NULL DEFAULT 'VND',
			period_start TIMESTAMPTZ NOT NULL,
			period_end TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resource_usage_user ON resource_usage(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_resource_usage_period ON resource_usage(period_start, period_end)`,
		`CREATE INDEX IF NOT EXISTS idx_resource_usage_type ON resource_usage(resource_type)`,

		`CREATE TABLE IF NOT EXISTS cost_rates (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			resource_type VARCHAR(50) UNIQUE NOT NULL,
			rate_per_unit DOUBLE PRECISION NOT NULL DEFAULT 0,
			unit VARCHAR(20) NOT NULL DEFAULT '',
			currency VARCHAR(10) NOT NULL DEFAULT 'VND',
			effective_from TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Phase 5.1: Data Catalog
		`CREATE TABLE IF NOT EXISTS data_catalog (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			parent_id UUID REFERENCES data_catalog(id) ON DELETE CASCADE,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_catalog_parent ON data_catalog(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_catalog_type ON data_catalog(type)`,

		// Phase 5.2: Workflows
		`CREATE TABLE IF NOT EXISTS workflows (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			user_id UUID NOT NULL REFERENCES admins(id),
			schedule VARCHAR(100) DEFAULT '',
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS workflow_tasks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			task_type VARCHAR(50) NOT NULL DEFAULT 'spark_job',
			config JSONB DEFAULT '{}',
			depends_on UUID[] DEFAULT '{}',
			retry_count INTEGER DEFAULT 0,
			timeout_seconds INTEGER DEFAULT 3600,
			task_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_wf_tasks_workflow ON workflow_tasks(workflow_id, task_order)`,
		`CREATE TABLE IF NOT EXISTS workflow_runs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			triggered_by VARCHAR(255) DEFAULT '',
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			task_statuses JSONB DEFAULT '{}',
			error_message TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_wf_runs_workflow ON workflow_runs(workflow_id, created_at DESC)`,

		// Phase 5.4: Notebook Scheduler
		`CREATE TABLE IF NOT EXISTS notebook_schedules (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			notebook_id UUID NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES admins(id),
			schedule VARCHAR(100) NOT NULL,
			enabled BOOLEAN DEFAULT true,
			export_format VARCHAR(20) DEFAULT 'html',
			notification_email VARCHAR(255) DEFAULT '',
			last_run_at TIMESTAMPTZ,
			next_run_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS notebook_schedule_runs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			schedule_id UUID NOT NULL REFERENCES notebook_schedules(id) ON DELETE CASCADE,
			status VARCHAR(50) DEFAULT 'pending',
			output_path TEXT DEFAULT '',
			error_message TEXT DEFAULT '',
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nb_schedule_runs ON notebook_schedule_runs(schedule_id, created_at DESC)`,

		// Phase 5.7: Organizations
		`CREATE TABLE IF NOT EXISTS organizations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) UNIQUE NOT NULL,
			description TEXT DEFAULT '',
			parent_id UUID REFERENCES organizations(id),
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`ALTER TABLE admins ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id)`,
		`ALTER TABLE admins ADD COLUMN IF NOT EXISTS team_id VARCHAR(255) DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_orgs_parent ON organizations(parent_id)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Error().Err(err).Str("migration", m[:60]).Msg("migration failed")
			return err
		}
	}
	log.Info().Msg("database migrations completed")

	// Seed admin user
	if cfg.SeedAdminUsername != "" && cfg.SeedAdminPassword != "" {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM admins WHERE username = $1)", cfg.SeedAdminUsername).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			hash, err := bcrypt.GenerateFromPassword([]byte(cfg.SeedAdminPassword), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			_, err = db.Exec(
				"INSERT INTO admins (id, username, email, password_hash, role) VALUES (gen_random_uuid(), $1, $2, $3, 'superadmin')",
				cfg.SeedAdminUsername, cfg.SeedAdminEmail, string(hash),
			)
			if err != nil {
				return err
			}
			log.Info().Str("username", cfg.SeedAdminUsername).Msg("seed admin user created")
		}
	}

	return nil
}
