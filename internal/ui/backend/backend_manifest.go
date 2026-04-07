package backend

import (
	"strings"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── ToManifest ────────────────────────────────────────────────────────────────

func (be BackendEditor) ToManifest() manifest.BackendPillar {
	arch := be.currentArch()

	var auth *manifest.AuthConfig
	if be.authEnabled {
		svcUnit := core.FieldGet(be.AuthFields, "service_unit")
		if svcUnit == "None (external)" || svcUnit == "None" || svcUnit == "(no services configured)" {
			svcUnit = ""
		}
		auth = &manifest.AuthConfig{
			Strategy:     core.NoneToEmpty(core.FieldGetMulti(be.AuthFields, "strategy")),
			Provider:     core.NoneToEmpty(core.FieldGet(be.AuthFields, "provider")),
			ServiceUnit:  svcUnit,
			AuthzModel:   core.NoneToEmpty(core.FieldGet(be.AuthFields, "authz_model")),
			SessionMgmt:  core.NoneToEmpty(core.FieldGet(be.AuthFields, "session_mgmt")),
			Permissions:  be.authPerms,
			Roles:        be.authRoles,
			TokenStorage: core.NoneToEmpty(core.FieldGetMulti(be.AuthFields, "token_storage")),
			RefreshToken: core.NoneToEmpty(core.FieldGet(be.AuthFields, "refresh_token")),
			MFA:          core.NoneToEmpty(core.FieldGet(be.AuthFields, "mfa")),
		}
		// Clean hidden auth fields so incompatible values don't leak.
		if be.isAuthFieldHidden("session_mgmt") {
			auth.SessionMgmt = ""
		}
		if be.isAuthFieldHidden("token_storage") {
			auth.TokenStorage = ""
			auth.RefreshToken = ""
		}
	}

	// Derive stack configs from the list editor items.
	var stackConfigs []manifest.StackConfig
	for _, item := range be.stackConfigEditor.items {
		sc := manifest.StackConfig{
			Name:             core.FieldGet(item, "name"),
			Language:         core.FieldGet(item, "language"),
			LanguageVersion:  core.FieldGet(item, "language_version"),
			Framework:        core.FieldGet(item, "framework"),
			FrameworkVersion: core.FieldGet(item, "framework_version"),
		}
		if sc.Name != "" {
			stackConfigs = append(stackConfigs, sc)
		}
	}

	// Language/framework fields are always hidden from the service form — they are
	// never set per-service. For monolith they live at the pillar level; for all
	// other arches they live in the referenced stack config. Strip them from every
	// service to keep the manifest clean. Also strip architecture-specific fields
	// that don't apply to the current arch.
	services := make([]manifest.ServiceDef, len(be.Services))
	for i, s := range be.Services {
		s.Language = ""
		s.LanguageVersion = ""
		s.Framework = ""
		s.FrameworkVersion = ""
		if arch == "monolith" {
			s.ConfigRef = ""
			s.ServiceDiscovery = ""
			s.Environment = ""
		}
		if arch != "hybrid" {
			s.PatternTag = ""
		}
		services[i] = s
	}

	bp := manifest.BackendPillar{
		ArchPattern: manifest.ArchPattern(arch),
		Services:    services,
		Auth:        auth,
		JobQueues:   be.jobQueues,
	}

	// Stack configs only apply to non-monolith architectures.
	if arch != "monolith" {
		bp.StackConfigs = stackConfigs
	}

	// Comm links only apply to architectures with a COMM tab.
	tabs := subTabsForArch(arch)
	hasTab := func(t backendSubTab) bool {
		for _, tab := range tabs {
			if tab == t {
				return true
			}
		}
		return false
	}
	if hasTab(beTabComm) {
		// Strip response_dtos from non-bidirectional comm links.
		links := make([]manifest.CommLink, len(be.CommLinks))
		for i, l := range be.CommLinks {
			if l.Direction != "Bidirectional (↔)" {
				l.ResponseDTOs = nil
			}
			links[i] = l
		}
		bp.CommLinks = links
	}
	if be.secEnabled {
		bp.WAF = &manifest.WAFConfig{
			Provider:          core.NoneToEmpty(core.FieldGet(be.securityFields, "waf_provider")),
			Ruleset:           core.NoneToEmpty(core.FieldGet(be.securityFields, "waf_ruleset")),
			CAPTCHA:           core.NoneToEmpty(core.FieldGet(be.securityFields, "captcha")),
			BotProtection:     core.NoneToEmpty(core.FieldGet(be.securityFields, "bot_protection")),
			RateLimitStrategy: core.NoneToEmpty(core.FieldGet(be.securityFields, "rate_limit_strategy")),
			RateLimitBackend:  core.NoneToEmpty(core.FieldGet(be.securityFields, "rate_limit_backend")),
			DDoSProtection:    core.NoneToEmpty(core.FieldGet(be.securityFields, "ddos_protection")),
			InternalMTLS:      arch != "monolith" && core.FieldGet(be.securityFields, "internal_mtls") == "Enabled",
		}
		// Clean hidden security fields.
		if be.isSecurityFieldHidden("rate_limit_backend") {
			bp.WAF.RateLimitBackend = ""
		}
	}

	for _, t := range tabs {
		if t == beTabMessaging {
			msgEnv := core.FieldGet(be.MessagingFields, "environment")
			if msgEnv == "(no environments configured)" {
				msgEnv = ""
			}
			mc := manifest.MessagingConfig{
				BrokerTech:    core.NoneToEmpty(core.FieldGet(be.MessagingFields, "broker_tech")),
				Deployment:    core.NoneToEmpty(core.FieldGet(be.MessagingFields, "deployment")),
				Serialization: core.NoneToEmpty(core.FieldGet(be.MessagingFields, "serialization")),
				Delivery:      core.NoneToEmpty(core.FieldGet(be.MessagingFields, "delivery")),
				Environment:   msgEnv,
			}
			bp.Messaging = &mc
			bp.Events = be.Events
		}
		if t == beTabAPIGW && be.apiGWEnabled {
			gw := manifest.APIGatewayConfig{
				Technology:  core.NoneToEmpty(core.FieldGet(be.APIGWFields, "technology")),
				Routing:     core.NoneToEmpty(core.FieldGet(be.APIGWFields, "routing")),
				Features:    core.FieldGetMulti(be.APIGWFields, "features"),
				Endpoints:   core.FieldGetMulti(be.APIGWFields, "endpoints"),
				Environment: core.FieldGet(be.APIGWFields, "environment"),
			}
			if gw.Environment == "(no environments configured)" {
				gw.Environment = ""
			}
			bp.APIGateway = &gw
		}
	}

	// Monolith: language/framework live at the pillar level (CONFIG tab).
	if arch == "monolith" {
		bp.Language = core.FieldGet(be.EnvFields, "monolith_lang")
		bp.LanguageVersion = core.FieldGet(be.EnvFields, "monolith_lang_ver")
		bp.Framework = core.FieldGet(be.EnvFields, "monolith_fw")
		bp.FrameworkVersion = core.FieldGet(be.EnvFields, "monolith_fw_ver")
		envVal := core.FieldGet(be.EnvFields, "environment")
		if envVal != "(no environments configured)" {
			bp.MonolithEnvironment = envVal
		}
		// Global health dependencies for monolith live in EnvConfig.
		healthDeps := core.FieldGetSelectedSlice(be.EnvFields, "health_deps")
		if len(healthDeps) > 0 {
			bp.Env = &manifest.EnvConfig{HealthDeps: healthDeps}
		}
	}
	// For all other arches, stack details live in stack_configs; services reference
	// them via config_ref. No top-level language/framework fields are emitted.
	return bp
}

// FromBackendPillar populates the editor from a saved manifest BackendPillar,
// reversing the ToManifest() operation.
func (be BackendEditor) FromBackendPillar(bp manifest.BackendPillar) BackendEditor {
	// Restore arch selection.
	arch := string(bp.ArchPattern)
	for i, opt := range beArchOptions {
		if opt.value == arch {
			be.ArchIdx = i
			be.dropdownIdx = i
			break
		}
	}
	if arch != "" {
		be.ArchConfirmed = true
		be.dropdownOpen = false
	}

	// Env fields (monolith language/framework/environment only).
	if bp.Language != "" {
		be.envEnabled = true
		if arch == "monolith" {
			be.EnvFields = core.SetFieldValue(be.EnvFields, "monolith_lang", bp.Language)
			be.updateEnvMonolithOptions()
			be.EnvFields = core.SetFieldValue(be.EnvFields, "monolith_lang_ver", bp.LanguageVersion)
			be.EnvFields = core.SetFieldValue(be.EnvFields, "monolith_fw", bp.Framework)
			be.updateEnvMonolithVersionOptions()
			be.EnvFields = core.SetFieldValue(be.EnvFields, "monolith_fw_ver", bp.FrameworkVersion)
			if bp.MonolithEnvironment != "" {
				be.EnvFields = core.SetFieldValue(be.EnvFields, "environment", bp.MonolithEnvironment)
			}
			// Restore global health deps — options populated lazily via SetDBSourceAliases.
			if bp.Env != nil && len(bp.Env.HealthDeps) > 0 {
				for i := range be.EnvFields {
					if be.EnvFields[i].Key == "health_deps" {
						be.EnvFields[i].Value = strings.Join(bp.Env.HealthDeps, ", ")
						break
					}
				}
			}
		}
	}

	// Stack configs for non-monolith arches.
	if len(bp.StackConfigs) > 0 {
		be.envEnabled = true
		be.stackConfigEditor.items = make([][]core.Field, len(bp.StackConfigs))
		for i, sc := range bp.StackConfigs {
			f := defaultStackConfigFields()
			f = core.SetFieldValue(f, "name", sc.Name)
			if sc.Language != "" {
				f = core.SetFieldValue(f, "language", sc.Language)
				if vers, ok := core.LangVersions[sc.Language]; ok {
					for j := range f {
						if f[j].Key == "language_version" {
							f[j].Options = vers
							f[j].SelIdx = 0
							f[j].Value = vers[0]
							break
						}
					}
					if sc.LanguageVersion != "" {
						f = core.SetFieldValue(f, "language_version", sc.LanguageVersion)
					}
				}
				if opts, ok := backendFrameworksByLang[sc.Language]; ok {
					for j := range f {
						if f[j].Key == "framework" {
							f[j].Options = opts
							f[j].SelIdx = 0
							f[j].Value = opts[0]
							break
						}
					}
					if sc.Framework != "" {
						f = core.SetFieldValue(f, "framework", sc.Framework)
					}
				}
				fw := sc.Framework
				if fw == "" {
					if opts, ok := backendFrameworksByLang[sc.Language]; ok && len(opts) > 0 {
						fw = opts[0]
					}
				}
				fwVers := core.CompatibleFrameworkVersions(sc.Language, sc.LanguageVersion, fw)
				for j := range f {
					if f[j].Key == "framework_version" {
						f[j].Options = fwVers
						f[j].SelIdx = 0
						f[j].Value = fwVers[0]
						if sc.FrameworkVersion != "" {
							f = core.SetFieldValue(f, "framework_version", sc.FrameworkVersion)
						}
						break
					}
				}
			}
			be.stackConfigEditor.items[i] = f
		}
		be.StackConfigs = bp.StackConfigs
		be.applyStackConfigNamesToServices()
	}

	// Auth fields.
	if bp.Auth != nil && (bp.Auth.Strategy != "" || bp.Auth.Provider != "") {
		be.authEnabled = true
		be.AuthFields = core.RestoreMultiSelectValue(be.AuthFields, "strategy", bp.Auth.Strategy)
		be.AuthFields = core.SetFieldValue(be.AuthFields, "provider", bp.Auth.Provider)
		if bp.Auth.ServiceUnit != "" {
			// Restore service_unit; options will be repopulated lazily on first open.
			for i := range be.AuthFields {
				if be.AuthFields[i].Key == "service_unit" {
					be.AuthFields[i].Options = []string{bp.Auth.ServiceUnit}
					be.AuthFields[i].Value = bp.Auth.ServiceUnit
					be.AuthFields[i].SelIdx = 0
					break
				}
			}
		}
		be.AuthFields = core.SetFieldValue(be.AuthFields, "authz_model", bp.Auth.AuthzModel)
		be.authPerms = bp.Auth.Permissions
		be.authRoles = bp.Auth.Roles
		be.AuthFields = core.RestoreMultiSelectValue(be.AuthFields, "token_storage", bp.Auth.TokenStorage)
		be.AuthFields = core.SetFieldValue(be.AuthFields, "session_mgmt", bp.Auth.SessionMgmt)
		be.AuthFields = core.SetFieldValue(be.AuthFields, "refresh_token", bp.Auth.RefreshToken)
		// Recompute dynamic options after restoring strategy and provider.
		be.updateAuthTokenStorageOptions()
		be.updateAuthMFAOptions()
		be.AuthFields = core.SetFieldValue(be.AuthFields, "mfa", bp.Auth.MFA)
	}

	// Security / WAF fields.
	if bp.WAF != nil && (bp.WAF.Provider != "" || bp.WAF.Ruleset != "") {
		be.secEnabled = true
		be.securityFields = core.SetFieldValue(be.securityFields, "waf_provider", bp.WAF.Provider)
		be.securityFields = core.SetFieldValue(be.securityFields, "waf_ruleset", bp.WAF.Ruleset)
		be.securityFields = core.SetFieldValue(be.securityFields, "captcha", bp.WAF.CAPTCHA)
		be.securityFields = core.SetFieldValue(be.securityFields, "bot_protection", bp.WAF.BotProtection)
		be.securityFields = core.SetFieldValue(be.securityFields, "rate_limit_strategy", bp.WAF.RateLimitStrategy)
		be.securityFields = core.SetFieldValue(be.securityFields, "rate_limit_backend", bp.WAF.RateLimitBackend)
		be.securityFields = core.SetFieldValue(be.securityFields, "ddos_protection", bp.WAF.DDoSProtection)
		mtlsVal := "Disabled"
		if bp.WAF.InternalMTLS {
			mtlsVal = "Enabled"
		}
		be.securityFields = core.SetFieldValue(be.securityFields, "internal_mtls", mtlsVal)
	}

	// Messaging fields.
	if bp.Messaging != nil {
		be.MessagingFields = core.SetFieldValue(be.MessagingFields, "broker_tech", bp.Messaging.BrokerTech)
		be.MessagingFields = core.SetFieldValue(be.MessagingFields, "deployment", bp.Messaging.Deployment)
		be.MessagingFields = core.SetFieldValue(be.MessagingFields, "serialization", bp.Messaging.Serialization)
		be.MessagingFields = core.SetFieldValue(be.MessagingFields, "delivery", bp.Messaging.Delivery)
		if bp.Messaging.Environment != "" {
			be.MessagingFields = core.SetFieldValue(be.MessagingFields, "environment", bp.Messaging.Environment)
		}
	}

	// Event catalog.
	be.Events = bp.Events
	be.eventEditor.items = make([][]core.Field, len(bp.Events))
	for i, evt := range bp.Events {
		f := defaultEventFields()
		f = core.SetFieldValue(f, "name", evt.Name)
		f = core.SetFieldValue(f, "publisher_service", evt.PublisherService)
		f = core.SetFieldValue(f, "consumer_service", evt.ConsumerService)
		f = core.SetFieldValue(f, "dto", evt.DTO)
		f = core.SetFieldValue(f, "description", evt.Description)
		be.eventEditor.items[i] = f
	}

	// API Gateway fields.
	if bp.APIGateway != nil {
		be.apiGWEnabled = true
		if bp.APIGateway.Environment != "" {
			be.APIGWFields = core.SetFieldValue(be.APIGWFields, "environment", bp.APIGateway.Environment)
		}
		be.APIGWFields = core.SetFieldValue(be.APIGWFields, "technology", bp.APIGateway.Technology)
		be.APIGWFields = core.SetFieldValue(be.APIGWFields, "routing", bp.APIGateway.Routing)
		be.APIGWFields = core.RestoreMultiSelectValue(be.APIGWFields, "features", bp.APIGateway.Features)
		// Endpoint options are injected lazily via SetEndpointNames; store names
		// in Value so they can be restored once options become available.
		for i := range be.APIGWFields {
			if be.APIGWFields[i].Key == "endpoints" {
				be.APIGWFields[i].Value = bp.APIGateway.Endpoints
				break
			}
		}
	}

	// Collections — stored directly; per-item forms are rebuilt lazily on navigation.
	be.Services = bp.Services
	be.serviceEditor.items = make([][]core.Field, len(bp.Services))
	for i, svc := range bp.Services {
		fields := serviceFieldsFromDef(svc)
		// Monolith: health deps are global (CONFIG tab), not per-service.
		if arch == "monolith" {
			fields = core.WithoutField(fields, "health_deps")
		}
		be.serviceEditor.items[i] = fields
	}
	// Apply orchestrator-based service discovery options now that items are populated.
	be.updateServiceDiscoveryOptions()

	be.CommLinks = bp.CommLinks
	be.commEditor.items = make([][]core.Field, len(bp.CommLinks))
	for i, link := range bp.CommLinks {
		be.commEditor.items[i] = commFieldsFromLink(link)
	}

	be.jobQueues = bp.JobQueues

	return be
}

// ── Public accessors ─────────────────────────────────────────────────────────

// Orchestrator returns the primary orchestrator injected from the infra tab.
func (be BackendEditor) Orchestrator() string {
	return be.orchestrator
}

// WAFRateLimitStrategy returns the configured WAF rate-limit strategy so that
// the Contracts editor can set a sensible default for endpoint rate_limit.
func (be BackendEditor) WAFRateLimitStrategy() string {
	return core.FieldGet(be.securityFields, "rate_limit_strategy")
}

// AuthRoleOptions returns role names for use in frontend page forms.
// Returns only explicitly configured roles; empty slice means none configured.
func (be BackendEditor) AuthRoleOptions() []string {
	names := make([]string, 0, len(be.authRoles))
	for _, r := range be.authRoles {
		if r.Name != "" {
			names = append(names, r.Name)
		}
	}
	return names
}

// ServiceNames returns the names of all created service units for cross-reference.
func (be BackendEditor) ServiceNames() []string {
	var names []string
	for _, item := range be.serviceEditor.items {
		name := core.FieldGet(item, "name")
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// ServiceDefs returns full service definitions for technology-based protocol filtering.
func (be BackendEditor) ServiceDefs() []manifest.ServiceDef {
	defs := make([]manifest.ServiceDef, 0, len(be.serviceEditor.items))
	for _, item := range be.serviceEditor.items {
		defs = append(defs, serviceDefFromFields(item))
	}
	return defs
}

// Languages returns the unique set of programming languages configured across
// all stack configs (non-monolith) or the monolith language.
func (be BackendEditor) Languages() []string {
	seen := make(map[string]bool)
	var langs []string
	add := func(l string) {
		if l != "" && !seen[l] {
			seen[l] = true
			langs = append(langs, l)
		}
	}
	if be.currentArch() == "monolith" {
		add(core.FieldGet(be.EnvFields, "monolith_lang"))
	} else {
		for _, item := range be.stackConfigEditor.items {
			add(core.FieldGet(item, "language"))
		}
	}
	return langs
}

// ArchPattern returns the currently selected architecture pattern value (e.g. "monolith", "microservices").
func (be BackendEditor) ArchPattern() string {
	return be.currentArch()
}

// ServiceFrameworks returns the unique set of frameworks used across all configured
// stack configs (non-monolith) or the monolith framework.
func (be BackendEditor) ServiceFrameworks() []string {
	seen := make(map[string]bool)
	var fws []string
	add := func(fw string) {
		if fw != "" && !seen[fw] {
			seen[fw] = true
			fws = append(fws, fw)
		}
	}
	if be.currentArch() == "monolith" {
		add(core.FieldGet(be.EnvFields, "monolith_fw"))
	} else {
		for _, item := range be.stackConfigEditor.items {
			add(core.FieldGet(item, "framework"))
		}
	}
	return fws
}

// AuthStrategy returns the selected backend auth strategies for cross-editor use.
func (be BackendEditor) AuthStrategy() []string {
	raw := core.FieldGetMulti(be.AuthFields, "strategy")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ", ")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// CommProtocols returns the unique set of protocols used across all communication links.
func (be BackendEditor) CommProtocols() []string {
	seen := make(map[string]bool)
	var protos []string
	for _, link := range be.CommLinks {
		p := link.Protocol
		if p != "" && !seen[p] {
			seen[p] = true
			protos = append(protos, p)
		}
	}
	return protos
}
