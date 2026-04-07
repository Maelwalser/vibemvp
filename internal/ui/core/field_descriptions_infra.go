package core

func init() {
	// ── INFRASTRUCTURE: Networking ────────────────────────────────────────────

	fieldDescriptions["dns_provider"] = map[string]string{
		"Cloudflare": "DNS with DDoS protection and edge caching. Generates Terraform Cloudflare DNS resources.",
		"Route53":    "AWS Route 53. Tight integration with AWS services. Generates Terraform aws_route53_zone resources.",
		"Cloud DNS":  "Google Cloud DNS. Generates Terraform google_dns_managed_zone resources.",
		"Azure DNS":  "Azure DNS. Generates Terraform azurerm_dns_zone resources.",
		"Other":      "Custom DNS provider. Generates placeholder DNS configuration.",
	}

	fieldDescriptions["tls_ssl"] = map[string]string{
		"Let's Encrypt": "Free automated TLS certificates via ACME. Generates cert-manager or Caddy configuration.",
		"Cloudflare":    "Cloudflare-managed TLS. Generates Cloudflare SSL configuration and origin certificate.",
		"ACM":           "AWS Certificate Manager. Free TLS for AWS resources. Generates Terraform aws_acm_certificate resources.",
		"Manual":        "Manually managed certificates. Generates TLS secret mounts and renewal reminder configuration.",
		"None (dev)":    "No TLS. HTTP only. For local development only.",
	}

	fieldDescriptions["cdn"] = map[string]string{
		"Cloudflare CDN": "Cloudflare's global edge network. Caches static assets. Generates Cloudflare page rules and cache config.",
		"AWS CloudFront": "AWS CDN. Tight S3 and ALB integration. Generates Terraform CloudFront distribution resources.",
		"GCP Cloud CDN":  "GCP CDN for Load Balancer backends. Generates Terraform backend service with CDN policy.",
		"Azure CDN":      "Azure CDN profiles. Generates Terraform azurerm_cdn_profile resources.",
		"BunnyCDN":       "Cost-effective CDN with image optimization and edge scripting.",
		"None":           "No CDN. Assets served directly from origin.",
	}

	fieldDescriptions["domain_strategy"] = map[string]string{
		"Subdomain per service": "Each service gets its own subdomain (api.example.com, auth.example.com). Clear separation; requires wildcard TLS.",
		"Path-based routing":    "All services under one domain, differentiated by path prefix (/api/*, /auth/*). Single certificate.",
		"Single domain":         "Everything served from one domain and path. Simple; limited separation.",
		"Custom":                "Custom domain strategy. Generates placeholder routing rules.",
	}

	fieldDescriptions["cors_infra"] = map[string]string{
		"Reverse proxy (Nginx/Caddy)": "CORS headers added by the reverse proxy layer. Centralized; no application code needed.",
		"Application-level":           "Each service handles its own CORS middleware. More granular control per endpoint.",
		"CDN/WAF":                     "CORS handled at the CDN or WAF layer. Edge-level enforcement.",
		"Both":                        "Both application-level and proxy/CDN CORS headers. Belt-and-suspenders approach.",
	}

	fieldDescriptions["cors_strategy"] = map[string]string{
		"Permissive":       "Allow all origins (*). Suitable for public APIs with no sensitive data.",
		"Strict allowlist": "Only explicitly listed origins are allowed. Generates allowed-origins environment variable.",
		"Same-origin":      "Requests only allowed from the same origin. Strictest; no cross-origin access.",
	}

	fieldDescriptions["ssl_cert"] = map[string]string{
		"cert-manager (k8s)": "Kubernetes cert-manager operator. Automatically provisions and renews Let's Encrypt certificates.",
		"Caddy (auto)":       "Caddy web server with automatic HTTPS. Handles certificate provisioning and renewal.",
		"AWS ACM":            "AWS Certificate Manager. Generates ACM certificate request and DNS validation records.",
		"Manual rotation":    "Certificates managed and rotated manually or by a custom script.",
		"Cloudflare (edge)":  "Cloudflare manages TLS at the edge. Origin certificate optional.",
		"None":               "No SSL certificate management configured.",
	}

	// ── INFRASTRUCTURE: CI/CD ─────────────────────────────────────────────────

	fieldDescriptions["deploy_strategy"] = map[string]string{
		"Rolling":    "Gradually replaces old instances with new ones. Zero downtime. Some versions run simultaneously briefly.",
		"Blue-green": "Two identical environments; traffic switches instantly. Zero downtime. Easy rollback.",
		"Canary":     "Small percentage of traffic routed to new version first. Monitors for errors before full rollout.",
		"Recreate":   "Old version stopped before new version starts. Brief downtime. Simplest strategy.",
	}

	fieldDescriptions["iac_tool"] = map[string]string{
		"Terraform": "HashiCorp's declarative IaC. Provider-agnostic. Generates .tf files for all infrastructure with state management.",
		"Pulumi":    "IaC using TypeScript, Python, or Go. Generates Pulumi stacks with typed infrastructure components.",
		"CDK":       "AWS Cloud Development Kit. Define infrastructure in TypeScript/Python against AWS constructs.",
		"Ansible":   "Agentless configuration management via YAML playbooks. Generates Ansible roles and inventory.",
		"Helm":      "Kubernetes package manager. Generates Helm chart templates and values.yaml.",
		"None":      "No IaC tooling. Infrastructure managed manually.",
	}

	fieldDescriptions["registry"] = map[string]string{
		"ECR":          "AWS Elastic Container Registry. Tight IAM integration. Generates ECR repository and push permissions.",
		"GCR/Artifact": "Google Container Registry or Artifact Registry. Generates GCP service account and push config.",
		"ACR":          "Azure Container Registry. Generates Azure managed identity and ACR pull config.",
		"Docker Hub":   "Public/private Docker Hub registry. Generates docker login step in CI pipeline.",
		"GHCR":         "GitHub Container Registry. Free for public images; integrated with GitHub Actions.",
		"Self-hosted":  "Self-hosted registry (Harbor, Nexus). Generates registry credentials and trust config.",
	}

	fieldDescriptions["secrets_mgmt"] = map[string]string{
		"AWS Secrets Manager":        "AWS-managed secret storage with automatic rotation. Generates IAM policies and SDK calls.",
		"HashiCorp Vault":            "Open-source secrets platform. Dynamic secrets and encryption-as-a-service. Generates Vault policy files.",
		"GCP Secret Manager":         "Google Cloud-managed secrets. Generates IAM bindings and Secret Manager client calls.",
		"Azure Key Vault":            "Azure-managed secrets and keys. Generates Managed Identity access and Key Vault SDK integration.",
		"Kubernetes Secrets":         "Native Kubernetes Secrets as env vars or files. Generates Secret manifests and volume mounts.",
		"Doppler":                    "Developer-friendly secrets platform with ENV injection. Generates Doppler CLI setup.",
		"Environment variables only": "Plain environment variables. Generates .env.example files.",
	}

	fieldDescriptions["backup_dr"] = map[string]string{
		"Cross-region replication": "Database replicated to another region. Survive regional outage with low RTO/RPO.",
		"Daily snapshots":          "Automated daily database snapshots stored in durable object storage. Simple and cost-effective.",
		"Managed provider DR":      "Cloud provider's managed disaster recovery (AWS RDS Multi-AZ, GCP Cloud SQL HA). Zero-effort DR.",
		"None":                     "No backup or disaster recovery configured.",
	}

	// ── INFRASTRUCTURE: Compute ───────────────────────────────────────────────

	fieldDescriptions["compute_env"] = map[string]string{
		"Bare Metal":          "Physical servers without virtualization. Maximum performance and I/O throughput. Generates server provisioning scripts.",
		"VM":                  "Virtual machines on cloud or on-premises. Familiar deployment model. Generates Terraform VM resources.",
		"Containers (Docker)": "Docker containers on a single host or cluster. Portable and reproducible. Generates Dockerfiles.",
		"Kubernetes":          "Container orchestration at scale. Self-healing, autoscaling. Generates Kubernetes manifests.",
		"Serverless (FaaS)":   "Functions invoked on demand. No server management. Generates Lambda/Cloud Functions handlers.",
		"PaaS":                "Platform as a Service. Push to deploy; infrastructure abstracted. Generates platform config files.",
	}

	fieldDescriptions["cloud_provider"] = map[string]string{
		"AWS":             "Amazon Web Services. Widest service catalog. Generates Terraform AWS resources and IAM configuration.",
		"GCP":             "Google Cloud Platform. Strong in data analytics and ML. Generates Terraform GCP resources.",
		"Azure":           "Microsoft Azure. Strong enterprise and Active Directory integration. Generates Terraform Azure resources.",
		"Cloudflare":      "Cloudflare's edge platform (Workers, R2, D1). Generates Wrangler config and edge function scaffolding.",
		"Hetzner":         "Cost-effective European cloud. VMs, dedicated servers. Generates Terraform Hetzner resources.",
		"Self-hosted":     "On-premises or co-located servers. Full control. Generates Ansible playbooks.",
		"Other (specify)": "Custom cloud provider. Generates generic infrastructure configuration placeholders.",
	}

	fieldDescriptions["orchestrator"] = map[string]string{
		"Kubernetes":           "Full container orchestration. Deployments, Services, Ingress, HPA. Generates complete K8s manifest set.",
		"Docker Swarm":         "Docker's native clustering. Simpler than Kubernetes. Generates docker-compose.yml with Swarm deploy constraints.",
		"ECS (Fargate)":        "AWS fully-managed container runtime. No node management. Generates ECS task definitions.",
		"Cloud Run":            "GCP serverless containers. Scales to zero. Generates Cloud Run service YAML.",
		"Azure Container Apps": "Azure serverless container platform. Generates Container Apps manifests and scaling rules.",
		"Fly.io":               "Global application platform. Generates fly.toml configuration.",
		"Nomad":                "HashiCorp's flexible orchestrator. Supports containers, VMs, and raw executables.",
		"None":                 "No container orchestrator. Manual deployment or systemd services.",
	}

	// ── INFRASTRUCTURE: Observability ─────────────────────────────────────────

	fieldDescriptions["logging"] = map[string]string{
		"CloudWatch":         "AWS CloudWatch Logs. Generates CloudWatch log group Terraform and structured logging config.",
		"Datadog":            "SaaS observability with unified logs, metrics, and traces. Generates Datadog agent config.",
		"ELK Stack":          "Elasticsearch, Logstash, Kibana. Generates Logstash pipeline and Kibana dashboards.",
		"Loki + Grafana":     "Lightweight Grafana-native log aggregation. Cost-effective for Kubernetes. Generates Loki Helm values.",
		"GCP Cloud Logging":  "Google Cloud managed logging. Generates structured logging config and log sink Terraform.",
		"Azure Monitor Logs": "Azure Log Analytics. Generates Diagnostic Settings and KQL query examples.",
		"Fluentd/Fluent Bit": "Open-source log collectors with flexible routing. Generates Fluent Bit DaemonSet.",
		"None":               "No centralized logging. Application writes to stdout only.",
	}

	fieldDescriptions["metrics"] = map[string]string{
		"Prometheus + Grafana": "Open-source metrics stack. Pull-based scraping. Generates Prometheus scrape configs and Grafana dashboards.",
		"Datadog":              "SaaS metrics with APM, dashboards, and anomaly detection. Generates Datadog agent and custom metrics.",
		"CloudWatch":           "AWS CloudWatch Metrics. Generates CloudWatch alarms and dashboard JSON.",
		"New Relic":            "Full-stack observability SaaS. Generates New Relic agent config.",
		"OpenTelemetry":        "Vendor-neutral telemetry standard. Generates OTEL SDK setup, exporters, and collector config.",
		"GCP Cloud Monitoring": "Google Cloud managed metrics. Generates metric descriptors and Terraform dashboard resources.",
		"Azure Monitor":        "Azure unified monitoring. Generates Azure Monitor workspace and alert rules.",
		"None":                 "No metrics collection.",
	}

	fieldDescriptions["tracing"] = map[string]string{
		"Jaeger":             "Open-source distributed tracing. Generates Jaeger agent config and OTEL instrumentation.",
		"Zipkin":             "Open-source distributed tracing. Generates Zipkin reporter config.",
		"Datadog APM":        "Datadog application performance monitoring. Generates Datadog tracer init.",
		"AWS X-Ray":          "AWS-native distributed tracing. Generates X-Ray daemon config and SDK instrumentation.",
		"Google Cloud Trace": "GCP-native distributed tracing. Generates Cloud Trace exporter.",
		"OpenTelemetry":      "Vendor-neutral trace instrumentation. Generates OTEL trace SDK and exporter config.",
		"None":               "No distributed tracing.",
	}

	fieldDescriptions["error_tracking"] = map[string]string{
		"Sentry":   "Error tracking with full stack traces, breadcrumbs, and context. Generates Sentry SDK init.",
		"Datadog":  "Datadog error tracking integrated with APM and logs.",
		"Rollbar":  "Real-time error monitoring with deployment tracking. Generates Rollbar SDK init.",
		"Built-in": "Platform or framework built-in error logging (CloudWatch, GCP Error Reporting). No extra dependency.",
		"None":     "No error tracking configured.",
	}

	fieldDescriptions["health_checks"] = map[string]string{
		"true":  "Health check endpoints generated for each service. Generates /health or /readyz + /livez handlers for Kubernetes probes.",
		"false": "No health check endpoints generated.",
	}

	fieldDescriptions["alerting"] = map[string]string{
		"PagerDuty":         "On-call incident management. Generates PagerDuty integration key and escalation policy.",
		"OpsGenie":          "Atlassian alert management and on-call scheduling. Generates OpsGenie API integration.",
		"Alertmanager":      "Prometheus Alertmanager. Routes alerts to receivers (Slack, email, PagerDuty). Generates alertmanager.yml.",
		"Datadog Monitors":  "Datadog's built-in alerting on metrics and logs. Generates monitor Terraform resources.",
		"CloudWatch Alarms": "AWS CloudWatch threshold-based alarms. Generates Terraform CloudWatch alarm resources.",
		"Slack webhooks":    "Simple Slack notifications for alerts. No on-call management. Generates Slack webhook integration.",
		"None":              "No alerting configured.",
	}

	fieldDescriptions["log_retention"] = map[string]string{
		"7 days":     "Logs retained for 7 days. Low cost; suitable for development and staging.",
		"30 days":    "Logs retained for 30 days. Standard for most production workloads.",
		"90 days":    "Logs retained for 90 days. Required for many compliance frameworks.",
		"1 year":     "Logs retained for 1 year. Meets most audit and compliance requirements.",
		"Indefinite": "Logs never automatically deleted. Highest cost. Use only when indefinite audit trails are required.",
	}

	// ── CROSSCUT: Testing ─────────────────────────────────────────────────────

	fieldDescriptions["unit"] = map[string]string{
		"Go testing": "Go's built-in testing package. Table-driven tests. Generates _test.go files with TestXxx functions.",
		"Testify":    "Go test assertions and mocking. Generates test files using assert, require, and mock packages.",
		"Jest":       "JavaScript/TypeScript test runner with built-in mocking and coverage. Generates jest.config.ts and test files.",
		"Vitest":     "Vite-native unit testing. Faster than Jest for Vite projects. Generates vitest.config.ts.",
		"pytest":     "Python's most popular test framework. Fixtures and parametrize. Generates conftest.py and test_*.py files.",
		"unittest":   "Python's built-in test framework. Generates unittest.TestCase subclasses.",
		"JUnit":      "Java standard unit testing. Generates JUnit 5 test classes with @Test annotations.",
		"Kotest":     "Kotlin-native testing DSL. Generates Kotest Spec classes.",
		"TestNG":     "Java testing with data providers. Generates TestNG annotated test classes.",
		"xUnit":      ".NET community test framework. Generates xUnit test classes with [Fact] and [Theory].",
		"NUnit":      ".NET testing with [Test] attributes. Generates NUnit test fixtures.",
		"MSTest":     "Microsoft's .NET test framework. Generates MSTest test classes.",
		"cargo test": "Rust's built-in test runner. Generates #[test] and #[cfg(test)] annotated functions.",
		"RSpec":      "Ruby BDD-style testing. Generates RSpec spec files with describe/it blocks.",
		"minitest":   "Ruby standard library test framework. Generates Minitest::Test subclasses.",
		"PHPUnit":    "PHP unit testing framework. Generates PHPUnit TestCase subclasses.",
		"Pest":       "PHP elegant testing with expressive syntax. Generates Pest test files.",
		"Other":      "Custom test framework. Generates placeholder test structure.",
	}

	fieldDescriptions["integration"] = map[string]string{
		"Testcontainers":  "Real Docker containers for databases and brokers in tests. Language-specific SDK. Generates container setup helpers.",
		"Docker Compose":  "Multi-container test environments. Generates compose file for integration test dependencies.",
		"In-memory fakes": "In-process implementations of external dependencies. Fast; no Docker. Generates fake adapters.",
		"None":            "No integration testing framework.",
	}

	fieldDescriptions["e2e"] = map[string]string{
		"Playwright":       "Cross-browser E2E testing (Chromium, Firefox, WebKit). Generates Playwright tests with page objects.",
		"Cypress":          "JavaScript E2E testing with time-travel debugging. Generates Cypress spec files.",
		"Selenium":         "Classic browser automation. Language-agnostic. Generates Selenium test classes.",
		"Flutter Driver":   "Flutter's native E2E test framework. Generates flutter_driver test files.",
		"Integration Test": "Flutter official integration test package. Generates integration_test/ directory.",
		"Espresso":         "Android UI testing via Instrumentation. Generates Espresso test classes.",
		"UI Automator":     "Android cross-app UI testing. Generates UIAutomator test classes.",
		"XCUITest":         "Apple's official iOS/macOS UI testing. Generates XCUITest target and classes.",
		"EarlGrey":         "Google's iOS UI testing with synchronization. Generates EarlGrey test files.",
		"None":             "No E2E test framework.",
	}

	fieldDescriptions["fe_testing"] = map[string]string{
		"Vitest":          "Vite-native component testing. Fast HMR-based test runner. Generates Vitest component tests.",
		"Jest":            "Component testing with jsdom. Generates Jest component tests.",
		"Testing Library": "User-centric component testing utilities. Works with Jest or Vitest. Generates RTL render helpers.",
		"Storybook":       "Component development and visual regression testing. Generates Storybook stories and test configurations.",
		"None":            "No frontend component testing.",
	}

	fieldDescriptions["load"] = map[string]string{
		"k6":        "Go-based load testing with JavaScript scripting. Generates k6 test scripts with virtual user scenarios.",
		"Artillery": "Node.js load testing toolkit. YAML-based scenarios. Generates Artillery config and scenario files.",
		"JMeter":    "Java-based load testing with GUI. Generates JMeter JMX test plan.",
		"Locust":    "Python-based distributed load testing. Generates locustfile.py with user tasks.",
		"None":      "No load testing configured.",
	}

	fieldDescriptions["contract"] = map[string]string{
		"Pact":                  "Consumer-driven contract testing. Generates Pact consumer and provider test files.",
		"Schemathesis":          "API schema-based fuzzing and contract validation. Generates Schemathesis test config.",
		"Dredd":                 "HTTP API testing against OpenAPI specs. Generates Dredd hooks and configuration.",
		"AsyncAPI validator":    "Validates async event messages against AsyncAPI schemas. Generates validator setup.",
		"Spring Cloud Contract": "JVM contract testing framework. Generates contract Groovy files and base test classes.",
		"None":                  "No contract testing.",
	}

	// ── CROSSCUT: Standards ────────────────────────────────────────────────────

	fieldDescriptions["dep_updates"] = map[string]string{
		"Dependabot": "GitHub automated dependency update PRs. Generates .github/dependabot.yml with update schedules.",
		"Renovate":   "More configurable automated updates. Monorepo support and custom grouping. Generates renovate.json.",
		"Manual":     "Dependencies updated manually. No automated PR generation.",
		"None":       "No dependency update automation.",
	}

	fieldDescriptions["feature_flags"] = map[string]string{
		"LaunchDarkly":      "Feature flag SaaS with targeting, gradual rollouts, and A/B testing. Generates LaunchDarkly SDK init.",
		"Unleash":           "Open-source feature flags. Self-hosted or cloud. Generates Unleash client setup.",
		"Flagsmith":         "Open-source feature flags and remote config. Generates Flagsmith SDK integration.",
		"Custom (env vars)": "Feature flags as environment variables. No external service. Generates typed accessors.",
		"None":              "No feature flag system.",
	}

	fieldDescriptions["changelog"] = map[string]string{
		"Conventional Commits": "Automated changelog from commit messages following the Conventional Commits spec. Generates .commitlintrc and CHANGELOG.md template.",
		"Manual":               "Changelog maintained by hand. Generates CHANGELOG.md template.",
		"None":                 "No changelog strategy.",
	}

	fieldDescriptions["auto_generate"] = map[string]string{
		"true":  "API documentation auto-generated from code annotations or schemas. Generates tooling configuration.",
		"false": "API documentation written manually.",
	}

	// ── REALIZE ───────────────────────────────────────────────────────────────

	fieldDescriptions["concurrency"] = map[string]string{
		"1": "Sequential task execution. One task at a time. Easiest to debug; slowest for large manifests.",
		"2": "Two parallel tasks. Moderate speed increase with low risk of resource contention.",
		"4": "Four concurrent tasks. Good balance of speed and stability. Recommended for most manifests.",
		"8": "Eight concurrent tasks. Maximum throughput. Best for large manifests on capable machines.",
	}

	fieldDescriptions["verify"] = map[string]string{
		"true":  "Verify generated code after each task. Compiles and lints before moving on. Failures trigger retries with escalating model tiers.",
		"false": "Skip verification. Faster generation but generated code may have compilation errors.",
	}

	fieldDescriptions["dry_run"] = map[string]string{
		"false": "Execute agent calls and generate code. Full pipeline runs.",
		"true":  "Print the task execution plan without calling AI agents. Review generation order and task count before committing.",
	}
}
