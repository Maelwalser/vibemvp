package ui

import "strings"

// containerRuntimeByLang maps backend language → preferred container base images.
var containerRuntimeByLang = map[string][]string{
	"Go":              {"scratch", "distroless", "alpine"},
	"TypeScript/Node": {"node:alpine", "node:slim", "distroless/nodejs"},
	"Python":          {"python:slim", "python:alpine", "distroless/python3"},
	"Java":            {"eclipse-temurin:alpine", "distroless/java", "amazoncorretto"},
	"Kotlin":          {"eclipse-temurin:alpine", "distroless/java", "amazoncorretto"},
	"C#/.NET":         {"mcr.microsoft.com/dotnet/aspnet", "alpine"},
	"Rust":            {"scratch", "distroless", "alpine"},
	"Ruby":            {"ruby:slim", "ruby:alpine"},
	"PHP":             {"php:fpm-alpine", "php:cli-alpine"},
	"Elixir":          {"elixir:alpine", "elixir:slim"},
}

// containerRuntimeAllOptions is the union of every language-specific option,
// shown when no backend languages have been configured yet.
var containerRuntimeAllOptions = func() []string {
	seen := make(map[string]bool)
	var out []string
	for _, opts := range containerRuntimeByLang {
		for _, o := range opts {
			if !seen[o] {
				seen[o] = true
				out = append(out, o)
			}
		}
	}
	return out
}()

// runtimeOptionsForLangs returns the deduplicated set of container base images
// appropriate for the given backend languages. Falls back to the full set when
// langs is empty or contains only unrecognised values.
func runtimeOptionsForLangs(langs []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, lang := range langs {
		for _, o := range containerRuntimeByLang[lang] {
			if !seen[o] {
				seen[o] = true
				out = append(out, o)
			}
		}
	}
	if len(out) == 0 {
		return containerRuntimeAllOptions
	}
	return out
}

// deployStrategiesByOrchestrator maps orchestrator → valid deploy strategies.
var deployStrategiesByOrchestrator = map[string][]string{
	"Docker Compose": {"Recreate"},
	"K3s":            {"Rolling", "Blue-green", "Canary", "Recreate"},
	"K8s (managed)":  {"Rolling", "Blue-green", "Canary", "Recreate"},
	"ECS":            {"Rolling", "Blue-green", "Canary"},
	"Cloud Run":      {"Rolling", "Canary"},
	"Nomad":          {"Rolling", "Blue-green", "Canary"},
	"None":           {"Recreate"},
}

// deployStrategyAllOptions is the union of all orchestrator-specific strategies,
// shown before any orchestrator is configured.
var deployStrategyAllOptions = []string{"Rolling", "Blue-green", "Canary", "Recreate"}

// infraAllOptions is the full (provider-agnostic) option set for each
// cloud-provider-aware field. Used when no provider is selected.
var infraAllOptions = map[string][]string{
	"registry":     {"Docker Hub", "GHCR", "ECR", "GCR", "Artifact Registry", "ACR", "Self-hosted"},
	"cdn":          {"Cloudflare", "CloudFront", "Cloud CDN", "Azure CDN", "Fastly", "Vercel Edge", "None"},
	"dns_provider": {"Cloudflare", "Route53", "Cloud DNS", "Azure DNS", "Other"},
	"secrets_mgmt": {"GitHub Secrets", "HashiCorp Vault", "AWS Secrets Manager", "GCP Secret Manager", "Azure Key Vault", "None"},
	"logging":      {"Loki + Grafana", "ELK Stack", "CloudWatch", "Cloud Logging", "Azure Monitor", "Datadog", "Stdout/file"},
	"metrics":      {"Prometheus + Grafana", "Datadog", "CloudWatch", "Cloud Monitoring", "Azure Monitor", "New Relic", "None"},
	"ssl_cert":     {"Auto-renew (certbot/ACME)", "ACM", "GCP-managed", "Azure-managed", "Cloudflare proxy", "Manual"},
}

// infraProviderOptions maps field key → cloud_provider → recommended option slice.
var infraProviderOptions = map[string]map[string][]string{
	"registry": {
		"AWS":         {"ECR", "GHCR", "Docker Hub"},
		"GCP":         {"GCR", "Artifact Registry", "GHCR"},
		"Azure":       {"ACR", "GHCR"},
		"Cloudflare":  {"GHCR", "Docker Hub"},
		"Hetzner":     {"GHCR", "Docker Hub", "Self-hosted"},
		"Self-hosted": {"GHCR", "Docker Hub", "Self-hosted"},
	},
	"cdn": {
		"AWS":         {"CloudFront", "Cloudflare", "None"},
		"GCP":         {"Cloud CDN", "Cloudflare", "None"},
		"Azure":       {"Azure CDN", "Cloudflare", "None"},
		"Cloudflare":  {"Cloudflare", "None"},
		"Hetzner":     {"Cloudflare", "None"},
		"Self-hosted": {"Cloudflare", "None"},
	},
	"dns_provider": {
		"AWS":         {"Route53", "Cloudflare", "Other"},
		"GCP":         {"Cloud DNS", "Cloudflare", "Other"},
		"Azure":       {"Azure DNS", "Cloudflare", "Other"},
		"Cloudflare":  {"Cloudflare", "Other"},
		"Hetzner":     {"Cloudflare", "Other"},
		"Self-hosted": {"Cloudflare", "Other"},
	},
	"secrets_mgmt": {
		"AWS":         {"AWS Secrets Manager", "HashiCorp Vault", "GitHub Secrets"},
		"GCP":         {"GCP Secret Manager", "HashiCorp Vault", "GitHub Secrets"},
		"Azure":       {"Azure Key Vault", "HashiCorp Vault", "GitHub Secrets"},
		"Cloudflare":  {"HashiCorp Vault", "GitHub Secrets"},
		"Hetzner":     {"HashiCorp Vault", "GitHub Secrets"},
		"Self-hosted": {"HashiCorp Vault", "GitHub Secrets"},
	},
	"logging": {
		"AWS":         {"CloudWatch", "ELK Stack", "Loki + Grafana", "Datadog", "Stdout/file"},
		"GCP":         {"Cloud Logging", "ELK Stack", "Loki + Grafana", "Datadog", "Stdout/file"},
		"Azure":       {"Azure Monitor", "ELK Stack", "Loki + Grafana", "Datadog", "Stdout/file"},
		"Cloudflare":  {"Loki + Grafana", "ELK Stack", "Datadog", "Stdout/file"},
		"Hetzner":     {"Loki + Grafana", "ELK Stack", "Datadog", "Stdout/file"},
		"Self-hosted": {"Loki + Grafana", "ELK Stack", "Datadog", "Stdout/file"},
	},
	"metrics": {
		"AWS":         {"CloudWatch", "Prometheus + Grafana", "Datadog", "New Relic", "None"},
		"GCP":         {"Cloud Monitoring", "Prometheus + Grafana", "Datadog", "New Relic", "None"},
		"Azure":       {"Azure Monitor", "Prometheus + Grafana", "Datadog", "New Relic", "None"},
		"Cloudflare":  {"Prometheus + Grafana", "Datadog", "New Relic", "None"},
		"Hetzner":     {"Prometheus + Grafana", "Datadog", "New Relic", "None"},
		"Self-hosted": {"Prometheus + Grafana", "Datadog", "New Relic", "None"},
	},
	"ssl_cert": {
		"AWS":         {"ACM", "Auto-renew (certbot/ACME)", "Manual"},
		"GCP":         {"GCP-managed", "Auto-renew (certbot/ACME)", "Manual"},
		"Azure":       {"Azure-managed", "Auto-renew (certbot/ACME)", "Manual"},
		"Cloudflare":  {"Cloudflare proxy", "Auto-renew (certbot/ACME)", "Manual"},
		"Hetzner":     {"Auto-renew (certbot/ACME)", "Manual"},
		"Self-hosted": {"Auto-renew (certbot/ACME)", "Manual"},
	},
}

// applyCloudProviderToFields returns a copy of fields with Options narrowed (and
// Value/SelIdx reconciled) for any field present in infraProviderOptions.
func applyCloudProviderToFields(fields []Field, provider string) []Field {
	// Strip " (specify)" suffix produced by "Other (specify)" backend option.
	cp := provider
	if idx := strings.Index(cp, " ("); idx >= 0 {
		cp = cp[:idx]
	}
	out := make([]Field, len(fields))
	copy(out, fields)
	for i := range out {
		providerMap, ok := infraProviderOptions[out[i].Key]
		if !ok {
			continue
		}
		var opts []string
		if filtered, has := providerMap[cp]; has {
			opts = filtered
		} else {
			opts = infraAllOptions[out[i].Key]
		}
		out[i].Options = opts
		// Keep current value when still valid; otherwise fall back to first option.
		found := false
		for j, o := range opts {
			if o == out[i].Value {
				out[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			out[i].Value = opts[0]
			out[i].SelIdx = 0
		}
	}
	return out
}

// ── sub-tabs ──────────────────────────────────────────────────────────────────

type infraTabIdx int

const (
	infraTabNetworking infraTabIdx = iota
	infraTabCICD
	infraTabObservability
	infraTabEnvironments
)

var infraTabLabels = []string{"NETWORKING", "CI/CD", "OBSERVABILITY", "ENVIRONMENTS"}

// ── field definitions ─────────────────────────────────────────────────────────

func defaultNetworkingFields() []Field {
	return []Field{
		{
			Key: "dns_provider", Label: "dns_provider  ", Kind: KindSelect,
			Options: infraAllOptions["dns_provider"],
			Value:   "Cloudflare",
		},
		{
			Key: "tls_ssl", Label: "tls_ssl       ", Kind: KindSelect,
			Options: []string{"Let's Encrypt", "Cloudflare", "ACM", "Manual", "None (dev)"},
			Value:   "Let's Encrypt",
		},
		{
			Key: "reverse_proxy", Label: "reverse_proxy ", Kind: KindSelect,
			Options: []string{"Nginx", "Caddy", "Traefik", "Cloudflare Tunnel", "Cloud LB"},
			Value:   "Caddy", SelIdx: 1,
		},
		{
			Key: "cdn", Label: "cdn           ", Kind: KindSelect,
			Options: infraAllOptions["cdn"],
			Value:   "None", SelIdx: 6,
		},
		{Key: "primary_domain", Label: "Primary Domain", Kind: KindText},
		{
			Key: "domain_strategy", Label: "Domain Strat. ", Kind: KindSelect,
			Options: []string{"Subdomain per service", "Path-based routing", "Single domain", "Custom"},
			Value:   "Single domain", SelIdx: 2,
		},
		{
			Key: "cors_infra", Label: "CORS Enforced ", Kind: KindSelect,
			Options: []string{"Reverse proxy (Nginx/Caddy)", "Application-level", "CDN/WAF", "Both"},
			Value:   "Application-level", SelIdx: 1,
		},
		{
			Key: "ssl_cert", Label: "SSL Cert Mgmt ", Kind: KindSelect,
			Options: infraAllOptions["ssl_cert"],
			Value:   "Auto-renew (certbot/ACME)",
		},
	}
}

func defaultInfraCICDFields() []Field {
	return []Field{
		{
			Key: "platform", Label: "platform      ", Kind: KindSelect,
			Options: []string{
				"GitHub Actions", "GitLab CI", "Jenkins",
				"CircleCI", "ArgoCD", "Tekton",
			},
			Value: "GitHub Actions",
		},
		{
			Key: "registry", Label: "registry      ", Kind: KindSelect,
			Options: infraAllOptions["registry"],
			Value:   "GHCR", SelIdx: 1,
		},
		{
			Key: "deploy_strategy", Label: "deploy_strat  ", Kind: KindSelect,
			Options: []string{"Rolling", "Blue-green", "Canary", "Recreate"},
			Value:   "Rolling",
		},
		{
			Key: "iac_tool", Label: "iac_tool      ", Kind: KindSelect,
			Options: []string{"Terraform", "Pulumi", "CloudFormation", "Ansible", "None"},
			Value:   "Terraform",
		},
		{
			Key: "secrets_mgmt", Label: "secrets_mgmt  ", Kind: KindSelect,
			Options: infraAllOptions["secrets_mgmt"],
			Value:   "GitHub Secrets",
		},
		{
			Key:     "container_runtime",
			Label:   "Container     ",
			Kind:    KindSelect,
			Options: containerRuntimeAllOptions,
			Value:   containerRuntimeAllOptions[0],
		},
		{
			Key: "backup_dr", Label: "Backup/DR     ", Kind: KindSelect,
			Options: []string{"Cross-region replication", "Daily snapshots", "Managed provider DR", "None"},
			Value:   "None", SelIdx: 3,
		},
	}
}

func defaultEnvTopologyFields() []Field {
	return []Field{
		{
			Key: "stages", Label: "stages        ", Kind: KindSelect,
			Options: []string{
				"dev + prod", "dev + staging + prod",
				"dev + qa + staging + prod", "dev + staging + qa + preview + prod", "Custom",
			},
			Value: "dev + staging + prod", SelIdx: 1,
		},
		{
			Key: "promotion_pipeline", Label: "promotion     ", Kind: KindSelect,
			Options: []string{
				"Dev → Staging → Prod", "Dev → QA → Staging → Prod",
				"Dev → Prod (direct)", "Manual", "None",
			},
			Value: "Dev → Staging → Prod",
		},
		{
			Key: "secret_key_strategy", Label: "secret_keys   ", Kind: KindSelect,
			Options: []string{"Per-environment", "Shared base + overrides", "Fully shared", "None"},
			Value:   "Per-environment",
		},
		{
			Key: "migration_strategy", Label: "db_migrations ", Kind: KindSelect,
			Options: []string{
				"Auto on deploy", "Manual CI step", "Flyway", "Liquibase",
				"Atlas", "golang-migrate", "None",
			},
			Value: "Manual CI step", SelIdx: 1,
		},
		{
			Key: "db_seeding", Label: "db_seeding    ", Kind: KindSelect,
			Options: []string{"Automatic (fixtures)", "Manual", "None"},
			Value:   "None", SelIdx: 2,
		},
		{
			Key: "preview_envs", Label: "preview_envs  ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "false",
		},
	}
}

func defaultObservabilityFields() []Field {
	return []Field{
		{
			Key:     "logging",
			Label:   "logging       ",
			Kind:    KindSelect,
			Options: infraAllOptions["logging"],
			Value:   "Loki + Grafana",
		},
		{
			Key:     "metrics",
			Label:   "metrics       ",
			Kind:    KindSelect,
			Options: infraAllOptions["metrics"],
			Value:   "Prometheus + Grafana",
		},
		{
			Key: "tracing", Label: "tracing       ", Kind: KindSelect,
			Options: []string{
				"OpenTelemetry + Jaeger", "OpenTelemetry + Tempo", "Datadog APM", "None",
			},
			Value: "OpenTelemetry + Jaeger",
		},
		{
			Key: "error_tracking", Label: "error_tracking", Kind: KindSelect,
			Options: []string{"Sentry", "Datadog", "Rollbar", "Built-in", "None"},
			Value:   "Sentry",
		},
		{
			Key: "health_checks", Label: "health_checks ", Kind: KindSelect,
			Options: OptionsOffOn, Value: "true", SelIdx: 1,
		},
		{
			Key: "alerting", Label: "alerting      ", Kind: KindSelect,
			Options: []string{
				"Grafana Alerting", "PagerDuty", "OpsGenie",
				"CloudWatch Alarms", "None",
			},
			Value: "Grafana Alerting",
		},
		{
			Key: "log_retention", Label: "Log Retention ", Kind: KindSelect,
			Options: []string{"7 days", "30 days", "90 days", "1 year", "Indefinite"},
			Value:   "30 days", SelIdx: 1,
		},
	}
}

// ── Runtime field population ──────────────────────────────────────────────────

// SetOrchestrator narrows the deploy_strategy options to those supported by
// the given orchestrator. A no-op when the orchestrator has not changed.
func (ie *InfraEditor) SetOrchestrator(orch string) {
	if ie.orchestrator == orch {
		return
	}
	ie.orchestrator = orch
	opts, ok := deployStrategiesByOrchestrator[orch]
	if !ok {
		opts = deployStrategyAllOptions
	}
	for i := range ie.cicdFields {
		if ie.cicdFields[i].Key != "deploy_strategy" {
			continue
		}
		ie.cicdFields[i].Options = opts
		found := false
		for j, o := range opts {
			if o == ie.cicdFields[i].Value {
				ie.cicdFields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			ie.cicdFields[i].Value = opts[0]
			ie.cicdFields[i].SelIdx = 0
		}
		break
	}
}

// SetCloudProvider narrows cloud-aware field options to those appropriate for
// the given provider. A no-op when the provider has not changed.
func (ie *InfraEditor) SetCloudProvider(cp string) {
	if ie.cloudProvider == cp {
		return
	}
	ie.cloudProvider = cp
	ie.networkingFields = applyCloudProviderToFields(ie.networkingFields, cp)
	ie.cicdFields = applyCloudProviderToFields(ie.cicdFields, cp)
	ie.obsFields = applyCloudProviderToFields(ie.obsFields, cp)
}

// SetBackendLanguages narrows the container_runtime options to images that are
// appropriate for the given backend languages. A no-op when languages are unchanged.
func (ie *InfraEditor) SetBackendLanguages(langs []string) {
	key := strings.Join(langs, ",")
	if ie.backendLanguages == key {
		return
	}
	ie.backendLanguages = key
	opts := runtimeOptionsForLangs(langs)
	for i := range ie.cicdFields {
		if ie.cicdFields[i].Key != "container_runtime" {
			continue
		}
		ie.cicdFields[i].Options = opts
		// Keep value if still valid, else reset to first option.
		found := false
		for j, o := range opts {
			if o == ie.cicdFields[i].Value {
				ie.cicdFields[i].SelIdx = j
				found = true
				break
			}
		}
		if !found && len(opts) > 0 {
			ie.cicdFields[i].Value = opts[0]
			ie.cicdFields[i].SelIdx = 0
		}
		break
	}
}
