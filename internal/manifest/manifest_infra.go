package manifest

// ── Infrastructure tab types ──────────────────────────────────────────────────

// NetworkingConfig describes networking and connectivity settings.
type NetworkingConfig struct {
	DNSProvider     string `json:"dns_provider"`
	TLSSSL          string `json:"tls_ssl"`
	ReverseProxy    string `json:"reverse_proxy"`
	CDN             string `json:"cdn"`
	PrimaryDomain   string `json:"primary_domain,omitempty"`
	DomainStrategy  string `json:"domain_strategy,omitempty"`
	CORSEnforcement string `json:"cors_enforcement,omitempty"`
	SSLCertMgmt     string `json:"ssl_cert_mgmt,omitempty"`
}

// CICDConfig describes CI/CD pipeline settings.
type CICDConfig struct {
	Platform          string `json:"platform"`
	ContainerRegistry string `json:"container_registry"`
	DeployStrategy    string `json:"deploy_strategy"`
	IaCTool           string `json:"iac_tool"`
	SecretsMgmt       string `json:"secrets_mgmt,omitempty"`
	ContainerRuntime  string `json:"container_runtime,omitempty"`
	BackupDR          string `json:"backup_dr,omitempty"`
}

// ObservabilityConfig describes logging, metrics, tracing, and alerting settings.
type ObservabilityConfig struct {
	Logging       string `json:"logging"`
	Metrics       string `json:"metrics"`
	Tracing       string `json:"tracing"`
	ErrorTracking string `json:"error_tracking"`
	HealthChecks  bool   `json:"health_checks"`
	Alerting      string `json:"alerting"`
	LogRetention  string `json:"log_retention,omitempty"`
}

// EnvTopologyConfig describes environment staging, promotion, and secret topology.
type EnvTopologyConfig struct {
	Stages            string `json:"stages,omitempty"`
	PromotionPipeline string `json:"promotion_pipeline,omitempty"`
	SecretKeyStrategy string `json:"secret_key_strategy,omitempty"`
	MigrationStrategy string `json:"migration_strategy,omitempty"`
	DBSeeding         string `json:"db_seeding,omitempty"`
	PreviewEnvs       string `json:"preview_envs,omitempty"`
}

// InfraPillar groups infrastructure configuration.
type InfraPillar struct {
	Networking    NetworkingConfig    `json:"networking"`
	CICD          CICDConfig          `json:"cicd"`
	Observability ObservabilityConfig `json:"observability"`
	EnvTopology   EnvTopologyConfig   `json:"env_topology,omitempty"`
}
