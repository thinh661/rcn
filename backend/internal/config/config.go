package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Database
	DatabaseURL string

	// JWT
	JWTSecretKey     string
	JWTExpireMinutes int

	// Seed admin
	SeedAdminUsername string
	SeedAdminEmail    string
	SeedAdminPassword string

	// Google OAuth2
	GoogleClientID     string
	GoogleClientSecret string

	// Microsoft OAuth2
	MicrosoftClientID     string
	MicrosoftClientSecret string

	// Generic OIDC SSO — any enterprise IdP (Keycloak, Okta, Auth0, Azure AD,
	// Google, ...). Provider-agnostic backend authorization-code flow; endpoints
	// are discovered from {issuer}/.well-known/openid-configuration.
	OIDCIssuerURL         string // external, browser-facing issuer
	OIDCInternalIssuerURL string // back-channel issuer base (token/userinfo) when the backend can't reach the external URL (e.g. local docker). Empty → same as OIDCIssuerURL.
	OIDCClientID          string
	OIDCClientSecret      string
	OIDCScopes            string // space-separated; default "openid email profile"
	OIDCProviderName      string // login button label, e.g. "Acme SSO"
	OIDCRedirectURL       string // registered redirect_uri (browser-reachable callback)
	OIDCPostLoginRedirect string // frontend URL to land on after a successful login

	// AWS
	AWSProfile    string
	TFStateBucket string
	TFStateRegion string

	// Service
	ServiceName string
	ServicePort string
	Environment string
	LogLevel    string

	// MinIO (local S3-compatible storage for grading)
	MinIOEndpoint        string
	MinIOAccessKey       string
	MinIOSecretKey       string
	MinIOWorkspaceBucket string // single shared bucket; users isolated via prefix

	// Jupyter
	JupyterGatewayURL string

	// Kernel deployment (see KERNEL_MODE in .env.example)
	KernelMode           string
	KernelPodImage       string
	KernelPodNamespace   string
	KernelPodIdleMinutes int
	KernelPodMaxTotal    int
	KernelDockerNetwork  string
	KernelPullSecret     string // optional K8s imagePullSecret for private forks
	KernelCallbackURL    string // backend base URL reachable FROM kernels; the data helpers call it to fetch a fresh OIDC token per query

	// Connector auth (see docs/connectors-design.md). The app mints RS256 JWTs for
	// connectors; ConnectorJWTPrivateKey is the signing key (PEM) — empty → loaded
	// from ConnectorJWTKeyFile, else generated once and persisted in the DB.
	// ConnectorIssuer is the token `iss` (what a connector's required-issuer must
	// match). Connectors themselves are added/managed at runtime in the UI.
	ConnectorJWTPrivateKey string
	ConnectorJWTKeyFile    string
	ConnectorIssuer        string

	// Per-user kernel pod resource requests/limits. Strings in k8s quantity
	// format ("500m", "1Gi"). For docker_per_user mode only the *Limit values
	// apply — Docker has no separate "request" concept.
	KernelPodCPURequest    string
	KernelPodMemoryRequest string
	KernelPodCPULimit      string
	KernelPodMemoryLimit   string

	// Per-notebook resource presets (k8s_per_user only). When the preset list is
	// empty the feature is disabled and every pod uses the KernelPod*Limit values
	// above. Each preset's cpu/memory is applied as BOTH request and limit
	// (Guaranteed QoS) so a user gets exactly what they pick.
	KernelResourcePresets       []ResourcePreset
	KernelResourceDefaultPreset string // id of the preset pre-selected in the UI
	KernelResourceAllowCustom   bool   // show the "Custom" row in the dialog
	KernelResourceCustomMaxCPU  string // hard ceiling for custom cpu, e.g. "8"
	KernelResourceCustomMaxMem  string // hard ceiling for custom memory, e.g. "16Gi"

	// Git Integration (notebook versioning)
	GitEnabled   bool
	GitSSHKeyPath string
	GitAuthorName string
	GitAuthorEmail string

	// Spark Connect (gRPC remote SparkSession)
	SparkConnectEnabled  bool
	SparkConnectEndpoint string
	SparkConnectPort     string

	// Cost Tracking rates (VND per unit)
	CostRateSparkCPU     float64 // VND per CPU-hour
	CostRateSparkMemory  float64 // VND per GB-hour
	CostRateKernelCPU    float64 // VND per CPU-hour
	CostRateKernelMemory float64 // VND per GB-hour
	CostRateStorage      float64 // VND per GB-month

	// AI Assistant
	AIEnabled  bool
	AIProvider string
	AIAPIKey   string
	AIModel    string
	AIEndpoint string

	// MLflow
	MLflowEnabled     bool
	MLflowTrackingURI string

	// Delta Lake / Delta Sharing
	DeltaSharingEnabled bool
	DeltaSharingURL     string

	// CORS
	CORSOrigins []string
}

// ResourcePreset is one selectable kernel-pod size offered to the user.
type ResourcePreset struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

func Load() *Config {
	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		JWTSecretKey:     getEnv("JWT_SECRET_KEY", ""),
		JWTExpireMinutes: getEnvInt("JWT_EXPIRE_MINUTES", 60),

		SeedAdminUsername: getEnv("SEED_ADMIN_USERNAME", ""),
		SeedAdminEmail:    getEnv("SEED_ADMIN_EMAIL", ""),
		SeedAdminPassword: getEnv("SEED_ADMIN_PASSWORD", ""),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),

		MicrosoftClientID:     getEnv("MICROSOFT_CLIENT_ID", ""),
		MicrosoftClientSecret: getEnv("MICROSOFT_CLIENT_SECRET", ""),

		OIDCIssuerURL:         getEnv("OIDC_ISSUER_URL", ""),
		OIDCInternalIssuerURL: getEnv("OIDC_INTERNAL_ISSUER_URL", ""),
		OIDCClientID:          getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:      getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCScopes:            getEnv("OIDC_SCOPES", "openid email profile"),
		OIDCProviderName:      getEnv("OIDC_PROVIDER_NAME", "SSO"),
		OIDCRedirectURL:       getEnv("OIDC_REDIRECT_URL", ""),
		OIDCPostLoginRedirect: getEnv("OIDC_POST_LOGIN_REDIRECT", "/"),

		AWSProfile:    getEnv("AWS_PROFILE", ""),
		TFStateBucket: getEnv("TF_STATE_BUCKET", ""),
		TFStateRegion: getEnv("TF_STATE_REGION", ""),

		ServiceName: getEnv("SERVICE_NAME", "RCN"),
		ServicePort: getEnv("SERVICE_PORT", "10000"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		MinIOEndpoint:        getEnv("MINIO_ENDPOINT", ""),
		MinIOAccessKey:       getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey:       getEnv("MINIO_SECRET_KEY", ""),
		MinIOWorkspaceBucket: getEnv("MINIO_WORKSPACE_BUCKET", "workspace"),

		JupyterGatewayURL: getEnv("JUPYTER_GATEWAY_URL", "http://jupyter:8888"),

		KernelMode:           getEnv("KERNEL_MODE", "shared"),
		KernelPodImage:       getEnv("KERNEL_POD_IMAGE", "ghcr.io/sparklabx/kernel:latest"),
		KernelPodNamespace:   getEnv("KERNEL_POD_NAMESPACE", "RCN"),
		KernelPodIdleMinutes: getEnvInt("KERNEL_POD_IDLE_MINUTES", 30),
		KernelPodMaxTotal:    getEnvInt("KERNEL_POD_MAX_TOTAL", 50),
		KernelDockerNetwork:  getEnv("KERNEL_DOCKER_NETWORK", "RCN_default"),
		KernelPullSecret:     getEnv("KERNEL_PULL_SECRET", ""), // empty → no imagePullSecret
		KernelCallbackURL:    getEnv("KERNEL_CALLBACK_URL", "http://RCN-backend:10000"),

		ConnectorJWTPrivateKey: getEnv("CONNECTOR_JWT_PRIVATE_KEY", ""),
		ConnectorJWTKeyFile:    getEnv("CONNECTOR_JWT_PRIVATE_KEY_FILE", ""),
		ConnectorIssuer:        getEnv("CONNECTOR_ISSUER", "RCN"),

		KernelPodCPURequest:    getEnv("KERNEL_POD_CPU_REQUEST", "500m"),
		KernelPodMemoryRequest: getEnv("KERNEL_POD_MEMORY_REQUEST", "1Gi"),
		KernelPodCPULimit:      getEnv("KERNEL_POD_CPU_LIMIT", "2000m"),
		KernelPodMemoryLimit:   getEnv("KERNEL_POD_MEMORY_LIMIT", "4Gi"),

		KernelResourcePresets:       parseResourcePresets(getEnv("KERNEL_RESOURCE_PRESETS", "")),
		KernelResourceDefaultPreset: getEnv("KERNEL_RESOURCE_DEFAULT_PRESET", ""),
		KernelResourceAllowCustom:   getEnvBool("KERNEL_RESOURCE_ALLOW_CUSTOM", false),
		KernelResourceCustomMaxCPU:  getEnv("KERNEL_RESOURCE_CUSTOM_MAX_CPU", ""),
		KernelResourceCustomMaxMem:  getEnv("KERNEL_RESOURCE_CUSTOM_MAX_MEMORY", ""),

		GitEnabled:      getEnvBool("GIT_ENABLED", false),
		GitSSHKeyPath:   getEnv("GIT_SSH_KEY_PATH", ""),
		GitAuthorName:   getEnv("GIT_AUTHOR_NAME", "RCN Notebook"),
		GitAuthorEmail:  getEnv("GIT_AUTHOR_EMAIL", "notebook@rcn.local"),

		SparkConnectEnabled:  getEnvBool("SPARK_CONNECT_ENABLED", false),
		SparkConnectEndpoint: getEnv("SPARK_CONNECT_ENDPOINT", ""),
		SparkConnectPort:     getEnv("SPARK_CONNECT_PORT", "15002"),

		CostRateSparkCPU:     getEnvFloat("COST_RATE_SPARK_CPU", 500.0),
		CostRateSparkMemory:  getEnvFloat("COST_RATE_SPARK_MEMORY", 100.0),
		CostRateKernelCPU:    getEnvFloat("COST_RATE_KERNEL_CPU", 1000.0),
		CostRateKernelMemory: getEnvFloat("COST_RATE_KERNEL_MEMORY", 200.0),
		CostRateStorage:      getEnvFloat("COST_RATE_STORAGE", 50.0),

		AIEnabled:  getEnvBool("AI_ENABLED", false),
		AIProvider: getEnv("AI_PROVIDER", "openai"),
		AIAPIKey:   getEnv("AI_API_KEY", ""),
		AIModel:    getEnv("AI_MODEL", "gpt-4o-mini"),
		AIEndpoint: getEnv("AI_ENDPOINT", ""),

		MLflowEnabled:     getEnvBool("MLFLOW_ENABLED", false),
		MLflowTrackingURI: getEnv("MLFLOW_TRACKING_URI", "http://mlflow:5000"),

		DeltaSharingEnabled: getEnvBool("DELTA_SHARING_ENABLED", false),
		DeltaSharingURL:     getEnv("DELTA_SHARING_URL", "http://delta-sharing:8080"),

		CORSOrigins: splitCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000")),
	}
}

// OIDCEnabled reports whether generic OIDC SSO is configured. The login UI shows
// the SSO button and the /auth/oidc/* routes are active only when this is true.
func (c *Config) OIDCEnabled() bool {
	return c.OIDCIssuerURL != "" && c.OIDCClientID != "" && c.OIDCRedirectURL != ""
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}

// parseResourcePresets reads the KERNEL_RESOURCE_PRESETS JSON array. Malformed
// JSON or entries missing cpu/memory are dropped (logged at the call site is
// overkill for config) so a typo disables presets rather than crashing boot.
func parseResourcePresets(raw string) []ResourcePreset {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var presets []ResourcePreset
	if err := json.Unmarshal([]byte(raw), &presets); err != nil {
		return nil
	}
	out := presets[:0]
	for _, p := range presets {
		if p.ID == "" || p.CPU == "" || p.Memory == "" {
			continue
		}
		if p.Label == "" {
			p.Label = p.ID
		}
		out = append(out, p)
	}
	return out
}

// splitCORSOrigins splits a comma-separated list of CORS origins, strips
// whitespace from each token, and skips empty tokens. This prevents subtle
// bugs from config values like "https://a.com, https://b.com" or trailing
// commas.
func splitCORSOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
