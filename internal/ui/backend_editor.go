package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-mvp/internal/manifest"
)

// ── modes ─────────────────────────────────────────────────────────────────────

type beMode int

const (
	beNormal beMode = iota
	beInsert
)

// ── arch options ──────────────────────────────────────────────────────────────

type archOption struct {
	value string
	label string
	desc  string
}

var beArchOptions = []archOption{
	{"monolith", "Monolith", "Single deployable unit — all features in one codebase"},
	{"modular-monolith", "Modular Monolith", "Clear domain boundaries, single deployment"},
	{"microservices", "Microservices", "Independent services communicating over a network"},
	{"event-driven", "Event-Driven", "Services communicate asynchronously via events"},
	{"hybrid", "Hybrid", "Mix of patterns — each service unit tagged with its own pattern"},
}

// ── sub-tab IDs per arch ──────────────────────────────────────────────────────

// backendSubTab enumerates the logical sub-tabs in the backend section.
type backendSubTab int

const (
	beTabEnv backendSubTab = iota
	beTabServices
	beTabComm
	beTabMessaging
	beTabAPIGW
	beTabAuth
)

// subTabsForArch returns the ordered list of sub-tabs for the given arch value.
func subTabsForArch(arch string) []backendSubTab {
	switch arch {
	case "monolith":
		return []backendSubTab{beTabEnv, beTabServices, beTabAuth}
	case "modular-monolith":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabAuth}
	case "microservices":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabAPIGW, beTabAuth}
	case "event-driven":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabMessaging, beTabAuth}
	case "hybrid":
		return []backendSubTab{beTabEnv, beTabServices, beTabComm, beTabMessaging, beTabAPIGW, beTabAuth}
	default:
		return []backendSubTab{beTabEnv, beTabServices, beTabAuth}
	}
}

func subTabLabel(t backendSubTab) string {
	switch t {
	case beTabEnv:
		return "ENV"
	case beTabServices:
		return "SERVICES"
	case beTabComm:
		return "COMM"
	case beTabMessaging:
		return "MESSAGING"
	case beTabAPIGW:
		return "API GW"
	case beTabAuth:
		return "AUTH"
	}
	return "?"
}

// ── framework options per language ────────────────────────────────────────────

var backendFrameworksByLang = map[string][]string{
	"Go":              {"Fiber", "Gin", "Echo", "Chi", "net/http (stdlib)", "Connect"},
	"TypeScript/Node": {"Express", "Fastify", "NestJS", "Hono", "tRPC", "Elysia (Bun)"},
	"Python":          {"FastAPI", "Django", "Flask", "Litestar", "Starlette"},
	"Java":            {"Spring Boot", "Quarkus", "Micronaut", "Jakarta EE"},
	"Kotlin":          {"Ktor", "Spring Boot (Kotlin)", "http4k"},
	"C#/.NET":         {"ASP.NET Core", "Minimal APIs", "Carter"},
	"Rust":            {"Axum", "Actix-web", "Rocket", "Warp"},
	"Ruby":            {"Rails", "Sinatra", "Hanami", "Roda"},
	"PHP":             {"Laravel", "Symfony", "Slim", "Laminas"},
	"Elixir":          {"Phoenix", "Plug", "Bandit"},
	"Other":           {"Other"},
}

var backendLanguages = []string{
	"Go", "TypeScript/Node", "Python", "Java", "Kotlin",
	"C#/.NET", "Rust", "Ruby", "PHP", "Elixir", "Other",
}

// ── field definitions ─────────────────────────────────────────────────────────

func defaultEnvFields() []Field {
	return []Field{
		{
			Key: "compute_env", Label: "compute_env   ", Kind: KindSelect,
			Options: []string{
				"Bare Metal", "VM", "Containers (Docker)", "Kubernetes",
				"Serverless (FaaS)", "PaaS",
			},
			Value: "Containers (Docker)", SelIdx: 2,
		},
		{
			Key: "cloud_provider", Label: "cloud_provider", Kind: KindSelect,
			Options: []string{
				"AWS", "GCP", "Azure", "Cloudflare", "Hetzner",
				"Self-hosted", "Other (specify)",
			},
			Value: "AWS",
		},
		{
			Key: "orchestrator", Label: "orchestrator  ", Kind: KindSelect,
			Options: []string{
				"Docker Compose", "K3s", "K8s (managed)", "Nomad",
				"ECS", "Cloud Run", "None",
			},
			Value: "Docker Compose",
		},
		{Key: "regions", Label: "regions       ", Kind: KindText},
		{
			Key: "stages", Label: "stages        ", Kind: KindSelect,
			Options: []string{
				"Development", "Development + Staging", "Development + Staging + Production",
				"Staging + Production", "Production only",
			},
			Value: "Development + Staging + Production", SelIdx: 2,
		},
	}
}

func defaultServiceFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "responsibility", Label: "responsibility", Kind: KindText},
		{
			Key: "language", Label: "language      ", Kind: KindSelect,
			Options: backendLanguages,
			Value:   "Go",
		},
		{
			Key: "framework", Label: "framework     ", Kind: KindSelect,
			Options: backendFrameworksByLang["Go"],
			Value:   "Fiber",
		},
		{
			Key: "pattern_tag", Label: "pattern_tag   ", Kind: KindSelect,
			Options: []string{
				"Monolith part", "Modular module", "Microservice",
				"Event processor", "Serverless function",
			},
			Value: "Microservice",
		},
	}
}

func serviceFieldsFromDef(s manifest.ServiceDef) []Field {
	f := defaultServiceFields()
	f = setFieldValue(f, "name", s.Name)
	f = setFieldValue(f, "responsibility", s.Responsibility)
	if s.Language != "" {
		f = setFieldValue(f, "language", s.Language)
		// update framework options based on language
		if opts, ok := backendFrameworksByLang[s.Language]; ok {
			for i := range f {
				if f[i].Key == "framework" {
					f[i].Options = opts
					f[i].SelIdx = 0
					f[i].Value = opts[0]
					if s.Framework != "" {
						f = setFieldValue(f, "framework", s.Framework)
					}
					break
				}
			}
		}
	}
	if s.PatternTag != "" {
		f = setFieldValue(f, "pattern_tag", s.PatternTag)
	}
	return f
}

func serviceDefFromFields(fields []Field) manifest.ServiceDef {
	return manifest.ServiceDef{
		Name:           fieldGet(fields, "name"),
		Responsibility: fieldGet(fields, "responsibility"),
		Language:       fieldGet(fields, "language"),
		Framework:      fieldGet(fields, "framework"),
		PatternTag:     fieldGet(fields, "pattern_tag"),
	}
}

func defaultCommFields() []Field {
	return []Field{
		{Key: "from", Label: "from          ", Kind: KindText},
		{Key: "to", Label: "to            ", Kind: KindText},
		{
			Key: "direction", Label: "direction     ", Kind: KindSelect,
			Options: []string{
				"Unidirectional (→)", "Bidirectional (↔)", "Pub/Sub (fan-out)",
			},
			Value: "Unidirectional (→)",
		},
		{
			Key: "protocol", Label: "protocol      ", Kind: KindSelect,
			Options: []string{
				"REST (HTTP)", "gRPC", "GraphQL", "WebSocket",
				"Message Queue", "Event Bus", "Internal (in-process)",
			},
			Value: "REST (HTTP)",
		},
		{Key: "trigger", Label: "trigger       ", Kind: KindText},
		{
			Key: "sync_async", Label: "sync_async    ", Kind: KindSelect,
			Options: []string{"Synchronous", "Asynchronous", "Fire-and-forget"},
			Value:   "Synchronous",
		},
	}
}

func commLinkFromFields(fields []Field) manifest.CommLink {
	return manifest.CommLink{
		From:      fieldGet(fields, "from"),
		To:        fieldGet(fields, "to"),
		Direction: fieldGet(fields, "direction"),
		Protocol:  fieldGet(fields, "protocol"),
		Trigger:   fieldGet(fields, "trigger"),
		SyncAsync: fieldGet(fields, "sync_async"),
	}
}

func commFieldsFromLink(l manifest.CommLink) []Field {
	f := defaultCommFields()
	f = setFieldValue(f, "from", l.From)
	f = setFieldValue(f, "to", l.To)
	if l.Direction != "" {
		f = setFieldValue(f, "direction", l.Direction)
	}
	if l.Protocol != "" {
		f = setFieldValue(f, "protocol", l.Protocol)
	}
	f = setFieldValue(f, "trigger", l.Trigger)
	if l.SyncAsync != "" {
		f = setFieldValue(f, "sync_async", l.SyncAsync)
	}
	return f
}

func defaultMessagingFields() []Field {
	return []Field{
		{
			Key: "broker_tech", Label: "broker_tech   ", Kind: KindSelect,
			Options: []string{
				"Kafka", "NATS", "RabbitMQ", "Redis Streams",
				"AWS SQS/SNS", "Google Pub/Sub", "Azure Service Bus", "Pulsar",
			},
			Value: "Kafka",
		},
		{
			Key: "deployment", Label: "deployment    ", Kind: KindSelect,
			Options: []string{"Managed (cloud)", "Self-hosted", "Embedded"},
			Value:   "Managed (cloud)",
		},
		{
			Key: "serialization", Label: "serialization ", Kind: KindSelect,
			Options: []string{"JSON", "Protobuf", "Avro", "MessagePack", "CloudEvents"},
			Value:   "JSON",
		},
		{
			Key: "delivery", Label: "delivery      ", Kind: KindSelect,
			Options: []string{"At-most-once", "At-least-once", "Exactly-once"},
			Value:   "At-least-once",
		},
	}
}

func defaultEventFields() []Field {
	return []Field{
		{Key: "name", Label: "name          ", Kind: KindText},
		{Key: "domain", Label: "domain        ", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
}

func defaultAPIGWFields() []Field {
	return []Field{
		{
			Key: "technology", Label: "technology    ", Kind: KindSelect,
			Options: []string{
				"Kong", "Traefik", "NGINX", "Envoy",
				"AWS API Gateway", "Cloudflare Workers", "Custom (specify)", "None",
			},
			Value: "Kong",
		},
		{
			Key: "routing", Label: "routing       ", Kind: KindSelect,
			Options: []string{"Path-based", "Header-based", "Domain-based"},
			Value:   "Path-based",
		},
		{Key: "features", Label: "features      ", Kind: KindText},
	}
}

func defaultAuthFields() []Field {
	return []Field{
		{
			Key: "strategy", Label: "strategy      ", Kind: KindSelect,
			Options: []string{
				"JWT (stateless)", "Session-based", "OAuth 2.0 / OIDC",
				"API Keys", "mTLS", "None",
			},
			Value: "JWT (stateless)",
		},
		{
			Key: "provider", Label: "provider      ", Kind: KindSelect,
			Options: []string{
				"Self-managed", "Auth0", "Clerk", "Supabase Auth",
				"Firebase Auth", "Keycloak", "AWS Cognito", "Other",
			},
			Value: "Self-managed",
		},
		{
			Key: "authz_model", Label: "authz_model   ", Kind: KindSelect,
			Options: []string{"RBAC", "ABAC", "ACL", "ReBAC", "Policy-based (OPA/Cedar)", "Custom"},
			Value:   "RBAC",
		},
		{
			Key: "token_storage", Label: "token_storage ", Kind: KindSelect,
			Options: []string{
				"HttpOnly cookie", "Authorization header (Bearer)",
				"WebSocket protocol header", "Other",
			},
			Value: "HttpOnly cookie",
		},
		{
			Key: "mfa", Label: "mfa           ", Kind: KindSelect,
			Options: []string{"None", "TOTP", "SMS", "Email", "Passkeys/WebAuthn"},
			Value:   "None",
		},
	}
}

// ── list + form sub-editor for services and comm links ───────────────────────

type beListView int

const (
	beListViewList beListView = iota
	beListViewForm
)

type beListEditor struct {
	items    [][]Field        // each item is a slice of fields
	itemView beListView
	itemIdx  int
	formIdx  int
	form     []Field
}

func newBeListEditor() beListEditor {
	return beListEditor{itemView: beListViewList}
}

// ── BackendEditor ─────────────────────────────────────────────────────────────

// BackendEditor manages the BACKEND section.
type BackendEditor struct {
	// Arch selection
	ArchIdx       int
	ArchConfirmed bool
	dropdownOpen  bool
	dropdownIdx   int

	// Sub-tab state
	activeTabIdx int // index into subTabsForArch(arch)
	activeField  int

	// Field stores
	EnvFields      []Field
	MessagingFields []Field
	APIGWFields    []Field
	AuthFields     []Field

	// List editors
	serviceEditor beListEditor
	commEditor    beListEditor
	eventEditor   beListEditor // event catalog within messaging

	// Internal mode
	internalMode beMode
	formInput    textinput.Model
	width        int

	// Services and comm links for manifest export
	Services  []manifest.ServiceDef
	CommLinks []manifest.CommLink
	Events    []manifest.EventDef
}

func newBackendEditor() BackendEditor {
	return BackendEditor{
		EnvFields:       defaultEnvFields(),
		MessagingFields: defaultMessagingFields(),
		APIGWFields:     defaultAPIGWFields(),
		AuthFields:      defaultAuthFields(),
		serviceEditor:   newBeListEditor(),
		commEditor:      newBeListEditor(),
		eventEditor:     newBeListEditor(),
		formInput:       newFormInput(),
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (be BackendEditor) currentArch() string {
	if be.ArchIdx >= 0 && be.ArchIdx < len(beArchOptions) {
		return beArchOptions[be.ArchIdx].value
	}
	return beArchOptions[0].value
}

func (be BackendEditor) activeTabs() []backendSubTab {
	return subTabsForArch(be.currentArch())
}

func (be BackendEditor) activeTab() backendSubTab {
	tabs := be.activeTabs()
	if be.activeTabIdx >= 0 && be.activeTabIdx < len(tabs) {
		return tabs[be.activeTabIdx]
	}
	return beTabEnv
}

func (be BackendEditor) tabLabels() []string {
	tabs := be.activeTabs()
	labels := make([]string, len(tabs))
	for i, t := range tabs {
		labels[i] = subTabLabel(t)
	}
	return labels
}

// ── ToManifest ────────────────────────────────────────────────────────────────

func (be BackendEditor) ToManifest() manifest.BackendPillar {
	arch := be.currentArch()

	env := manifest.EnvConfig{
		ComputeEnv:    fieldGet(be.EnvFields, "compute_env"),
		CloudProvider: fieldGet(be.EnvFields, "cloud_provider"),
		Orchestrator:  fieldGet(be.EnvFields, "orchestrator"),
		Regions:       fieldGet(be.EnvFields, "regions"),
		Stages:        fieldGet(be.EnvFields, "stages"),
	}

	auth := manifest.AuthConfig{
		Strategy:     fieldGet(be.AuthFields, "strategy"),
		Provider:     fieldGet(be.AuthFields, "provider"),
		AuthzModel:   fieldGet(be.AuthFields, "authz_model"),
		TokenStorage: fieldGet(be.AuthFields, "token_storage"),
		MFA:          fieldGet(be.AuthFields, "mfa"),
	}

	bp := manifest.BackendPillar{
		ArchPattern: manifest.ArchPattern(arch),
		Env:         env,
		Services:    be.Services,
		CommLinks:   be.CommLinks,
		Auth:        auth,
	}

	tabs := subTabsForArch(arch)
	for _, t := range tabs {
		if t == beTabMessaging {
			mc := manifest.MessagingConfig{
				BrokerTech:    fieldGet(be.MessagingFields, "broker_tech"),
				Deployment:    fieldGet(be.MessagingFields, "deployment"),
				Serialization: fieldGet(be.MessagingFields, "serialization"),
				Delivery:      fieldGet(be.MessagingFields, "delivery"),
			}
			bp.Messaging = &mc
		}
		if t == beTabAPIGW {
			gw := manifest.APIGatewayConfig{
				Technology: fieldGet(be.APIGWFields, "technology"),
				Routing:    fieldGet(be.APIGWFields, "routing"),
				Features:   fieldGet(be.APIGWFields, "features"),
			}
			bp.APIGateway = &gw
		}
	}

	// Legacy compat fields
	bp.ComputeEnv = manifest.ComputeEnv(env.ComputeEnv)
	bp.CloudProvider = env.CloudProvider
	if len(be.Services) > 0 {
		bp.Language = be.Services[0].Language
		bp.Framework = be.Services[0].Framework
	}
	return bp
}

// ── Mode / HintLine ───────────────────────────────────────────────────────────

func (be BackendEditor) Mode() Mode {
	if be.internalMode == beInsert {
		return ModeInsert
	}
	return ModeNormal
}

func (be BackendEditor) HintLine() string {
	if !be.ArchConfirmed {
		if be.dropdownOpen {
			return hintBar("j/k", "navigate", "Enter", "confirm", "Esc", "close")
		}
		return hintBar("Enter", "open arch selector")
	}
	if be.internalMode == beInsert {
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}

	tab := be.activeTab()
	switch tab {
	case beTabServices:
		ed := be.serviceEditor
		if ed.itemView == beListViewList {
			return hintBar("j/k", "navigate", "a", "add service", "d", "delete", "Enter", "edit", "h/l", "sub-tab", "b", "change arch")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back", "Tab", "next field")
	case beTabComm:
		ed := be.commEditor
		if ed.itemView == beListViewList {
			return hintBar("j/k", "navigate", "a", "add link", "d", "delete", "Enter", "edit", "h/l", "sub-tab")
		}
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
	case beTabMessaging:
		if be.eventEditor.itemView == beListViewForm {
			return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "b/Esc", "back")
		}
		return hintBar("j/k", "navigate", "Space", "cycle", "a", "add event", "d", "del event", "h/l", "sub-tab")
	default:
		return hintBar("j/k", "navigate", "i/Enter", "edit", "Space", "cycle", "H", "cycle back", "h/l", "sub-tab", "b", "change arch")
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (be BackendEditor) Update(msg tea.Msg) (BackendEditor, tea.Cmd) {
	if !be.ArchConfirmed {
		return be.updateArchSelect(msg)
	}
	if be.internalMode == beInsert {
		return be.updateInsert(msg)
	}
	return be.updateNormal(msg)
}

func (be BackendEditor) updateArchSelect(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return be, nil
	}
	if be.dropdownOpen {
		switch key.String() {
		case "j", "down":
			if be.dropdownIdx < len(beArchOptions)-1 {
				be.dropdownIdx++
			}
		case "k", "up":
			if be.dropdownIdx > 0 {
				be.dropdownIdx--
			}
		case "g":
			be.dropdownIdx = 0
		case "G":
			be.dropdownIdx = len(beArchOptions) - 1
		case "enter", " ":
			be.ArchIdx = be.dropdownIdx
			be.dropdownOpen = false
			be.ArchConfirmed = true
			be.activeTabIdx = 0
			be.activeField = 0
		case "esc":
			be.dropdownOpen = false
		}
		return be, nil
	}
	switch key.String() {
	case "enter", " ":
		be.dropdownOpen = true
		be.dropdownIdx = be.ArchIdx
	}
	return be, nil
}

func (be BackendEditor) updateInsert(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			be.saveInput()
			be.internalMode = beNormal
			be.formInput.Blur()
			return be, nil
		case "tab":
			be.saveInput()
			fields := be.currentEditableFields()
			if fields != nil {
				be.activeField = (be.activeField + 1) % len(*fields)
				return be.tryEnterInsert()
			}
		case "shift+tab":
			be.saveInput()
			fields := be.currentEditableFields()
			if fields != nil {
				n := len(*fields)
				be.activeField = (be.activeField - 1 + n) % n
				return be.tryEnterInsert()
			}
		}
	}
	var cmd tea.Cmd
	be.formInput, cmd = be.formInput.Update(msg)
	return be, cmd
}

func (be BackendEditor) updateNormal(msg tea.Msg) (BackendEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return be, nil
	}

	tab := be.activeTab()

	// Delegate list editors
	switch tab {
	case beTabServices:
		if be.serviceEditor.itemView == beListViewList {
			return be.updateServiceList(key)
		}
		return be.updateServiceForm(key)
	case beTabComm:
		if be.commEditor.itemView == beListViewList {
			return be.updateCommList(key)
		}
		return be.updateCommForm(key)
	case beTabMessaging:
		if be.eventEditor.itemView == beListViewForm {
			return be.updateEventForm(key)
		}
		return be.updateMessaging(key)
	}

	// Generic field navigation for ENV, API GW, AUTH
	switch key.String() {
	case "b":
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
			be.activeField = 0
		}
	case "l", "right":
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
			be.activeField = 0
		}
	case "j", "down":
		if fields := be.currentEditableFields(); fields != nil && be.activeField < len(*fields)-1 {
			be.activeField++
		}
	case "k", "up":
		if be.activeField > 0 {
			be.activeField--
		}
	case "g":
		be.activeField = 0
	case "G":
		if fields := be.currentEditableFields(); fields != nil {
			be.activeField = len(*fields) - 1
		}
	case "enter", " ":
		if f := be.mutableFieldPtr(); f != nil && f.Kind == KindSelect {
			f.CycleNext()
		} else {
			return be.tryEnterInsert()
		}
	case "H", "shift+left":
		if f := be.mutableFieldPtr(); f != nil && f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i":
		return be.tryEnterInsert()
	}
	return be, nil
}

func (be BackendEditor) updateServiceList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	n := len(ed.items)
	switch key.String() {
	case "j", "down":
		if n > 0 && ed.itemIdx < n-1 {
			ed.itemIdx++
		}
	case "k", "up":
		if ed.itemIdx > 0 {
			ed.itemIdx--
		}
	case "a":
		svc := manifest.ServiceDef{}
		be.Services = append(be.Services, svc)
		ed.items = append(ed.items, defaultServiceFields())
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.Services = append(be.Services[:ed.itemIdx], be.Services[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter", "l", "right":
		if n > 0 {
			ed.form = copyFields(ed.items[ed.itemIdx])
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.activeField = 0
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
			be.activeField = 0
		}
	case "b":
		be.ArchConfirmed = false
		be.dropdownOpen = false
		be.dropdownIdx = be.ArchIdx
		be.activeTabIdx = 0
		be.activeField = 0
	}
	return be, nil
}

func (be BackendEditor) updateServiceForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = (ed.formIdx + 1) % len(ed.form)
	case "k", "up":
		n := len(ed.form)
		ed.formIdx = (ed.formIdx - 1 + n) % n
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CycleNext()
			// If language changed, update framework options
			if f.Key == "language" {
				be.updateServiceFrameworkOptions(ed)
			}
		} else {
			return be.enterServiceFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
			if f.Key == "language" {
				be.updateServiceFrameworkOptions(ed)
			}
		}
	case "i", "a":
		if ed.form[ed.formIdx].Kind == KindText {
			return be.enterServiceFormInsert()
		}
	case "b", "esc":
		be.saveServiceForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be *BackendEditor) updateServiceFrameworkOptions(ed *beListEditor) {
	lang := fieldGet(ed.form, "language")
	opts, ok := backendFrameworksByLang[lang]
	if !ok {
		opts = []string{"Other"}
	}
	for i := range ed.form {
		if ed.form[i].Key == "framework" {
			ed.form[i].Options = opts
			ed.form[i].SelIdx = 0
			ed.form[i].Value = opts[0]
			break
		}
	}
}

func (be BackendEditor) enterServiceFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.serviceEditor
	f := ed.form[ed.formIdx]
	if f.Kind != KindText {
		return be, nil
	}
	be.internalMode = beInsert
	be.formInput.SetValue(f.Value)
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveServiceForm() {
	ed := &be.serviceEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	svc := serviceDefFromFields(ed.form)
	if ed.itemIdx < len(be.Services) {
		be.Services[ed.itemIdx] = svc
	}
}

func (be BackendEditor) updateCommList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	n := len(ed.items)
	switch key.String() {
	case "j", "down":
		if n > 0 && ed.itemIdx < n-1 {
			ed.itemIdx++
		}
	case "k", "up":
		if ed.itemIdx > 0 {
			ed.itemIdx--
		}
	case "a":
		be.CommLinks = append(be.CommLinks, manifest.CommLink{})
		ed.items = append(ed.items, defaultCommFields())
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = 0
	case "d":
		if n > 0 {
			be.CommLinks = append(be.CommLinks[:ed.itemIdx], be.CommLinks[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter", "l", "right":
		if n > 0 {
			ed.form = copyFields(ed.items[ed.itemIdx])
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.activeField = 0
		}
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
			be.activeField = 0
		}
	case "b":
		be.ArchConfirmed = false
	}
	return be, nil
}

func (be BackendEditor) updateCommForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = (ed.formIdx + 1) % len(ed.form)
	case "k", "up":
		n := len(ed.form)
		ed.formIdx = (ed.formIdx - 1 + n) % n
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CycleNext()
		} else {
			return be.enterCommFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ed.form[ed.formIdx].Kind == KindText {
			return be.enterCommFormInsert()
		}
	case "b", "esc":
		be.saveCommForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be BackendEditor) enterCommFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.commEditor
	f := ed.form[ed.formIdx]
	if f.Kind != KindText {
		return be, nil
	}
	be.internalMode = beInsert
	be.formInput.SetValue(f.Value)
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveCommForm() {
	ed := &be.commEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	link := commLinkFromFields(ed.form)
	if ed.itemIdx < len(be.CommLinks) {
		be.CommLinks[ed.itemIdx] = link
	}
}

func (be BackendEditor) updateMessaging(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	// Upper section: messaging broker config fields
	// Lower section: event catalog list
	// We use a split: first len(MessagingFields) positions are broker fields,
	// then event list items below.
	brokerCount := len(be.MessagingFields)
	eventCount := len(ed.items)
	total := brokerCount + eventCount

	switch key.String() {
	case "h", "left":
		if be.activeTabIdx > 0 {
			be.activeTabIdx--
			be.activeField = 0
		}
	case "l", "right":
		if be.activeTabIdx < len(be.activeTabs())-1 {
			be.activeTabIdx++
			be.activeField = 0
		}
	case "b":
		be.ArchConfirmed = false
	case "j", "down":
		if be.activeField < total-1 {
			be.activeField++
		}
	case "k", "up":
		if be.activeField > 0 {
			be.activeField--
		}
	case "enter", " ":
		if be.activeField < brokerCount {
			f := &be.MessagingFields[be.activeField]
			if f.Kind == KindSelect {
				f.CycleNext()
			}
		} else {
			eventIdx := be.activeField - brokerCount
			if eventIdx < eventCount {
				ed.form = copyFields(ed.items[eventIdx])
				ed.formIdx = 0
				ed.itemIdx = eventIdx
				ed.itemView = beListViewForm
			}
		}
	case "H", "shift+left":
		if be.activeField < brokerCount {
			f := &be.MessagingFields[be.activeField]
			if f.Kind == KindSelect {
				f.CyclePrev()
			}
		}
	case "a":
		be.Events = append(be.Events, manifest.EventDef{})
		ed.items = append(ed.items, defaultEventFields())
		ed.itemIdx = len(ed.items) - 1
		ed.form = copyFields(ed.items[ed.itemIdx])
		ed.formIdx = 0
		ed.itemView = beListViewForm
		be.activeField = brokerCount + ed.itemIdx
	case "d":
		eventIdx := be.activeField - brokerCount
		if eventIdx >= 0 && eventIdx < eventCount {
			be.Events = append(be.Events[:eventIdx], be.Events[eventIdx+1:]...)
			ed.items = append(ed.items[:eventIdx], ed.items[eventIdx+1:]...)
			if be.activeField > brokerCount && be.activeField >= brokerCount+len(ed.items) {
				be.activeField = brokerCount + len(ed.items) - 1
			}
		}
	case "i":
		if be.activeField < brokerCount {
			return be.tryEnterInsert()
		}
	}
	return be, nil
}

func (be BackendEditor) updateEventForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	switch key.String() {
	case "j", "down":
		ed.formIdx = (ed.formIdx + 1) % len(ed.form)
	case "k", "up":
		n := len(ed.form)
		ed.formIdx = (ed.formIdx - 1 + n) % n
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CycleNext()
		} else {
			return be.enterEventFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ed.form[ed.formIdx].Kind == KindText {
			return be.enterEventFormInsert()
		}
	case "b", "esc":
		be.saveEventForm()
		ed.itemView = beListViewList
	}
	return be, nil
}

func (be BackendEditor) enterEventFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.eventEditor
	f := ed.form[ed.formIdx]
	if f.Kind != KindText {
		return be, nil
	}
	be.internalMode = beInsert
	be.formInput.SetValue(f.Value)
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveEventForm() {
	ed := &be.eventEditor
	if ed.itemIdx >= len(ed.items) {
		return
	}
	ed.items[ed.itemIdx] = copyFields(ed.form)
	evt := manifest.EventDef{
		Name:        fieldGet(ed.form, "name"),
		Domain:      fieldGet(ed.form, "domain"),
		Description: fieldGet(ed.form, "description"),
	}
	if ed.itemIdx < len(be.Events) {
		be.Events[ed.itemIdx] = evt
	}
}

// currentEditableFields returns a pointer to the current tab's field slice.
func (be *BackendEditor) currentEditableFields() *[]Field {
	switch be.activeTab() {
	case beTabEnv:
		return &be.EnvFields
	case beTabMessaging:
		return &be.MessagingFields
	case beTabAPIGW:
		return &be.APIGWFields
	case beTabAuth:
		return &be.AuthFields
	}
	return nil
}

// mutableFieldPtr returns a pointer to the active field for the current tab.
func (be *BackendEditor) mutableFieldPtr() *Field {
	fields := be.currentEditableFields()
	if fields == nil {
		return nil
	}
	if be.activeField >= 0 && be.activeField < len(*fields) {
		return &(*fields)[be.activeField]
	}
	return nil
}

func (be BackendEditor) tryEnterInsert() (BackendEditor, tea.Cmd) {
	fields := be.currentEditableFields()
	if fields == nil || be.activeField >= len(*fields) {
		return be, nil
	}
	f := (*fields)[be.activeField]
	if f.Kind != KindText {
		return be, nil
	}
	be.internalMode = beInsert
	be.formInput.SetValue(f.Value)
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}

func (be *BackendEditor) saveInput() {
	val := be.formInput.Value()

	// Check if we're in a service form
	if be.serviceEditor.itemView == beListViewForm {
		ed := &be.serviceEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindText {
			ed.form[ed.formIdx].Value = val
		}
		return
	}
	// Check if we're in a comm form
	if be.commEditor.itemView == beListViewForm {
		ed := &be.commEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindText {
			ed.form[ed.formIdx].Value = val
		}
		return
	}
	// Check if we're in an event form
	if be.eventEditor.itemView == beListViewForm {
		ed := &be.eventEditor
		if ed.formIdx < len(ed.form) && ed.form[ed.formIdx].Kind == KindText {
			ed.form[ed.formIdx].Value = val
		}
		return
	}
	// Generic field stores
	fields := be.currentEditableFields()
	if fields == nil {
		return
	}
	if be.activeField < len(*fields) && (*fields)[be.activeField].Kind == KindText {
		(*fields)[be.activeField].Value = val
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (be BackendEditor) View(w, h int) string {
	be.width = w
	if !be.ArchConfirmed {
		return be.viewArchSelect(w, h)
	}
	return be.viewSubTabs(w, h)
}

func (be BackendEditor) viewArchSelect(w, h int) string {
	var lines []string
	lines = append(lines,
		StyleSectionDesc.Render("  # Backend — Choose an architecture pattern"),
		"",
	)

	current := beArchOptions[be.ArchIdx]
	label := StyleFieldKey.Render("arch_pattern  ")
	val := StyleFieldValActive.Render(current.label) + StyleSelectArrow.Render(" ▾")
	row := "     " + label + StyleEquals.Render(" = ") + val
	raw := lipgloss.Width(row)
	if raw < w {
		row += strings.Repeat(" ", w-raw)
	}
	lines = append(lines, StyleCurLine.Render(row))

	if be.dropdownOpen {
		lines = append(lines, "")
		for i, opt := range beArchOptions {
			isCur := i == be.dropdownIdx
			var cursor string
			if isCur {
				cursor = StyleCurLineNum.Render("  ▶ ")
			} else {
				cursor = "      "
			}
			labelPart := fmt.Sprintf("%-20s", opt.label)
			var optRow string
			if isCur {
				optRow = cursor +
					StyleFieldValActive.Render(labelPart) +
					StyleSectionDesc.Render(opt.desc)
				rw := lipgloss.Width(optRow)
				if rw < w {
					optRow += strings.Repeat(" ", w-rw)
				}
				optRow = StyleCurLine.Render(optRow)
			} else {
				optRow = cursor +
					StyleFieldKey.Render(labelPart) +
					StyleSectionDesc.Render(opt.desc)
			}
			lines = append(lines, optRow)
		}
	}

	return fillTildes(lines, h)
}

func (be BackendEditor) viewSubTabs(w, h int) string {
	var lines []string

	opt := beArchOptions[be.ArchIdx]
	archStr := StyleFieldValActive.Render(opt.label)
	hint := StyleSectionDesc.Render("  (b: change arch)")
	lines = append(lines,
		StyleSectionDesc.Render("  # Backend · ")+archStr+hint,
		"",
		renderSubTabBar(be.tabLabels(), be.activeTabIdx),
		"",
	)

	tab := be.activeTab()
	switch tab {
	case beTabEnv:
		lines = append(lines, renderFormFields(w, be.EnvFields, be.activeField, be.internalMode == beInsert, be.formInput)...)
	case beTabServices:
		lines = append(lines, be.viewServiceEditor(w)...)
	case beTabComm:
		lines = append(lines, be.viewCommEditor(w)...)
	case beTabMessaging:
		lines = append(lines, be.viewMessaging(w)...)
	case beTabAPIGW:
		lines = append(lines, renderFormFields(w, be.APIGWFields, be.activeField, be.internalMode == beInsert, be.formInput)...)
	case beTabAuth:
		lines = append(lines, renderFormFields(w, be.AuthFields, be.activeField, be.internalMode == beInsert, be.formInput)...)
	}

	return fillTildes(lines, h)
}

func (be BackendEditor) viewServiceEditor(w int) []string {
	ed := be.serviceEditor
	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Service Units — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no services yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				name := fieldGet(item, "name")
				if name == "" {
					name = fmt.Sprintf("(service #%d)", i+1)
				}
				lang := fieldGet(item, "language")
				fw := fieldGet(item, "framework")
				extra := lang
				if fw != "" {
					extra += " / " + fw
				}
				lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, extra))
			}
		}
		return lines
	}
	// Form view
	idx := ed.itemIdx
	name := "(new service)"
	if idx < len(ed.items) {
		n := fieldGet(ed.form, "name")
		if n != "" {
			name = n
		}
	}

	arch := be.currentArch()
	var fields []Field
	if arch != "hybrid" {
		// Hide pattern_tag for non-hybrid arches
		for _, f := range ed.form {
			if f.Key == "pattern_tag" {
				continue
			}
			fields = append(fields, f)
		}
	} else {
		fields = ed.form
	}

	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
	lines = append(lines, renderFormFields(w, fields, ed.formIdx, be.internalMode == beInsert, be.formInput)...)
	return lines
}

func (be BackendEditor) viewCommEditor(w int) []string {
	ed := be.commEditor
	if ed.itemView == beListViewList {
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Communication Links — a: add  d: delete  Enter: edit"), "")
		if len(ed.items) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no links yet — press 'a' to add)"))
		} else {
			for i, item := range ed.items {
				from := fieldGet(item, "from")
				to := fieldGet(item, "to")
				if from == "" {
					from = "?"
				}
				if to == "" {
					to = "?"
				}
				proto := fieldGet(item, "protocol")
				name := from + " → " + to
				lines = append(lines, renderListItem(w, i == ed.itemIdx, "  ▶ ", name, proto))
			}
		}
		return lines
	}
	from := fieldGet(be.commEditor.form, "from")
	to := fieldGet(be.commEditor.form, "to")
	title := from + " → " + to
	if from == "" && to == "" {
		title = "(new link)"
	}
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(title), "")
	lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == beInsert, be.formInput)...)
	return lines
}

func (be BackendEditor) viewMessaging(w int) []string {
	ed := be.eventEditor
	if ed.itemView == beListViewForm {
		name := fieldGet(ed.form, "name")
		if name == "" {
			name = "(new event)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, ed.form, ed.formIdx, be.internalMode == beInsert, be.formInput)...)
		return lines
	}

	// Combined view: broker config fields + event catalog list
	var lines []string
	lines = append(lines, StyleSectionDesc.Render("  # Messaging Broker + Event Catalog"), "")

	brokerCount := len(be.MessagingFields)
	// Render broker fields in upper section
	for i, f := range be.MessagingFields {
		isCur := i == be.activeField
		lineNo := StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}
		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}
		eq := StyleEquals.Render(" = ")
		val := f.DisplayValue()
		var valStr string
		if isCur {
			valStr = StyleFieldValActive.Render(val) + StyleSelectArrow.Render(" ▾")
		} else {
			valStr = StyleFieldVal.Render(val) + StyleSelectArrow.Render(" ▾")
		}
		row := lineNo + keyStr + eq + valStr
		if isCur {
			rw := lipgloss.Width(row)
			if rw < w {
				row += strings.Repeat(" ", w-rw)
			}
			row = StyleCurLine.Render(row)
		}
		lines = append(lines, row)
	}

	// Divider + event catalog
	lines = append(lines, "", StyleSectionDesc.Render("  ── Event Catalog (a: add  d: delete  Enter: edit) ──"), "")

	if len(ed.items) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no events yet — press 'a' to add)"))
	} else {
		for i, item := range ed.items {
			globalIdx := brokerCount + i
			isCur := globalIdx == be.activeField
			name := fieldGet(item, "name")
			if name == "" {
				name = fmt.Sprintf("(event #%d)", i+1)
			}
			domain := fieldGet(item, "domain")
			lines = append(lines, renderListItem(w, isCur, "  ▶ ", name, domain))
		}
	}
	return lines
}

// copyFields makes a deep copy of a field slice.
func copyFields(src []Field) []Field {
	dst := make([]Field, len(src))
	for i, f := range src {
		dst[i] = f
		if f.Options != nil {
			dst[i].Options = make([]string, len(f.Options))
			copy(dst[i].Options, f.Options)
		}
	}
	return dst
}
