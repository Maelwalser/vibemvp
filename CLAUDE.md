# VibeMenu — Project Description & Engineering Standards

## 1. Project Overview

**VibeMenu** is an interactive Terminal User Interface (TUI) CLI tool for declaratively specifying a complete software system architecture. It implements a vim-inspired editor that lets developers and architects define comprehensive system manifests across 7 pillars — backend, data, contracts, frontend, infrastructure, cross-cutting concerns, and code generation configuration.

The resulting manifest is serialized to `manifest.json` and intended for downstream consumption by code-generation agents or tooling via the `cmd/realize` pipeline.

**Key design principles:**
- Vim-modal editing (Normal / Insert / Command modes)
- Tokyo Night dark theme throughout
- Non-linear editing — users can fill any tab in any order
- Pillar-based dependency graph: Data → Backend → Contracts → Frontend → Infrastructure → Cross-Cutting → Realize

---

## 2. Technology Stack

| Concern | Choice |
|---------|--------|
| Language | Go 1.26.1 |
| TUI framework | `github.com/charmbracelet/bubbletea` v1.3.10 |
| TUI components | `github.com/charmbracelet/bubbles` v1.0.0 (textarea, textinput) |
| Styling/layout | `github.com/charmbracelet/lipgloss` v1.1.0 |
| TUI entry point | `cmd/agent/main.go` |
| Realize entry point | `cmd/realize/main.go` |
| Manifest types | `internal/manifest/` (8 files, split by pillar) |
| UI components | `internal/ui/` (22 files, ~17,800 lines) |
| Code generation engine | `internal/realize/` (DAG, agent, skills, verifiers, orchestrator) |
| Claude model | `claude-opus-4-6` (realize default) |

---

## 3. Project Structure

```
VibeMenu/
├── cmd/
│   ├── agent/
│   │   └── main.go              # TUI entry point — sets up save callback, runs Bubble Tea program
│   └── realize/
│       └── main.go              # Code-gen entry point — CLI flags, runs orchestrator
├── internal/
│   ├── manifest/
│   │   ├── manifest.go          # Root Manifest struct + Save(); legacy pillar types (112 lines)
│   │   ├── manifest_enums.go    # All enum type declarations (154 lines)
│   │   ├── manifest_data.go     # DataPillar, DBSourceDef, DomainDef, caching types (168 lines)
│   │   ├── manifest_backend.go  # BackendPillar, ServiceDef, CommLink, AuthConfig, RoleDef, PermissionDef (155 lines)
│   │   ├── manifest_contracts.go # ContractsPillar, DTODef, EndpointDef, ExternalAPIDef (123 lines)
│   │   ├── manifest_frontend.go  # FrontendPillar, FrontendTech, PageDef (116 lines)
│   │   ├── manifest_infra.go     # InfraPillar, NetworkingConfig, CICDConfig (55 lines)
│   │   ├── manifest_crosscut.go  # CrossCutPillar, TestingConfig, DocsConfig (32 lines)
│   │   └── recent.go            # Recent manifest tracking (64 lines)
│   ├── ui/
│   │   ├── model.go             # Root TUI model, vim modes, Update + command dispatch
│   │   ├── model_sections.go    # Section registry: editor getters + update closures (one entry per pillar)
│   │   ├── model_view.go        # Root View() and all render helpers
│   │   ├── editor.go            # Editor interface (Mode, HintLine, View)
│   │   ├── nav.go               # NavigateTab(), VimNav struct — shared navigation helpers
│   │   ├── styles.go            # Tokyo Night palette, all lipgloss styles
│   │   ├── sections.go          # Section/field definitions, FieldKind enum
│   │   ├── render_helpers.go    # Shared rendering utilities (fillTildes, renderFormFields, …)
│   │   ├── field_options.go     # Shared field option slices (OptionsOnOff, OptionsOffOn)
│   │   ├── animation.go         # Animation utilities
│   │   ├── app.go               # App initialization and setup
│   │   ├── backend_editor.go    # Backend: struct, init, ToManifest, Update dispatcher
│   │   ├── backend_fields.go    # Backend: default field constructors and service/comm/auth form helpers
│   │   ├── backend_services.go  # Backend: service list/form + comm list/form + messaging update handlers
│   │   ├── backend_update.go    # Backend: dropdown, insert, jobs update handlers
│   │   ├── backend_view.go      # Backend: HintLine, View, all sub-tab render functions
│   │   ├── backend_auth_security.go # Backend: auth + security update handlers and view
│   │   ├── data_tab_editor.go   # Data: struct, init, ToManifest, Update dispatcher, View
│   │   ├── data_tab_fields.go   # Data: domain/caching/fs field constructors + helpers
│   │   ├── data_domains.go      # Data: domain list/form + attr/rel update handlers + viewDomains
│   │   ├── data_caching_storage.go # Data: caching/governance/file-storage update handlers + views
│   │   ├── data_editor.go       # Entity/column schema editor: struct, init, Mode, HintLine, Update
│   │   ├── data_editor_fields.go # Entity editor: column + entity settings form helpers
│   │   ├── data_editor_update.go # Entity editor: all update handlers
│   │   ├── data_editor_view.go  # Entity editor: all view functions
│   │   ├── db_editor.go         # Database source editor
│   │   ├── contracts_editor.go  # Contracts: struct, init, ToManifest, Update dispatcher
│   │   ├── contracts_fields.go  # Contracts: DTO/endpoint/versioning/external field constructors
│   │   ├── contracts_dtos.go    # Contracts: DTO list/form + field drill-down update + viewDTOs
│   │   ├── contracts_endpoints.go # Contracts: endpoint/versioning/external update + views
│   │   ├── frontend_editor.go   # Frontend: struct, init, ToManifest, Update dispatcher
│   │   ├── frontend_fields.go   # Frontend: compatibility maps + default field constructors
│   │   ├── frontend_update.go   # Frontend: tech/theme/pages/nav update handlers + View
│   │   ├── frontend_i18n_a11y.go # Frontend: i18n + a11y/SEO update handlers + viewPages
│   │   ├── infra_editor.go      # Infra: struct, init, ToManifest, Update dispatcher, View
│   │   ├── infra_fields.go      # Infra: provider maps, deploy strategies, default field constructors
│   │   ├── crosscut_editor.go   # Crosscut: struct, init, ToManifest, Update dispatcher, View
│   │   ├── crosscut_fields.go   # Crosscut: testing/docs/standards field constructors
│   │   ├── realize_editor.go    # Realize: code generation configuration form
│   │   ├── provider_menu.go     # Provider modal: types, struct, init, state helpers
│   │   ├── provider_menu_oauth.go # Provider modal: OAuth 2.0 PKCE flow + credential step
│   │   ├── provider_menu_update.go # Provider modal: Update handler
│   │   ├── provider_menu_view.go # Provider modal: View + all render functions
│   │   ├── realization_screen.go # Code generation output screen
│   │   ├── frontend_assets.go   # Frontend asset management
│   │   ├── welcome.go           # Welcome/initialization screen
│   │   └── open_file_modal.go   # File open dialog
│   └── realize/
│       ├── agent/
│       │   ├── agent.go         # Claude API client, tool-use loop (~134 lines)
│       │   ├── context.go       # Agent context struct (~13 lines)
│       │   └── prompt.go        # System/user prompt builders (~118 lines)
│       ├── dag/
│       │   ├── dag.go           # DAG struct, topological sort, cycle detection (~149 lines)
│       │   ├── builder.go       # Manifest → DAG task graph construction (~399 lines)
│       │   └── payload.go       # Task payload types (~39 lines)
│       ├── config/
│       │   └── defaults.go      # Tunable constants (DefaultModel, DefaultMaxTokens, MaxSkillBytes, …)
│       ├── orchestrator/
│       │   ├── orchestrator.go  # Config, entrypoint, task dispatch
│       │   ├── models.go        # Provider model registry, resolveModelID()
│       │   └── runner.go        # Per-task runner: agent call + verify + retry
│       ├── output/
│       │   └── writer.go        # File output writer
│       ├── skills/
│       │   ├── registry.go      # In-memory skill registry
│       │   ├── aliases.go       # Technology alias map + universal skills per task kind
│       │   └── loader.go        # Load skill markdown files from disk
│       ├── deps/
│       │   ├── resolver_interface.go # LanguageResolver interface + ResolverRegistry
│       │   ├── resolver.go      # deps.Agent — runs package manager to lock deps
│       │   ├── modules.go       # Shared: ResolvedDeps types, LibraryAPIDocs, PromptContext, Save/Load
│       │   ├── go_modules.go    # Go-specific: WellKnownGoModules, GoModForService, ValidateGoMod
│       │   └── npm_modules.go   # npm-specific: WellKnownNpmPackages, resolveNpmVersion
│       └── verify/
│           ├── verifier.go      # Verifier interface + Registry (Register() for extension)
│           ├── go_verifier.go   # go build + go vet verifier
│           ├── ts_verifier.go   # tsc verifier
│           ├── python_verifier.go # python -m py_compile verifier
│           ├── tf_verifier.go   # terraform validate verifier
│           └── null_verifier.go # No-op verifier for unknown languages
├── system-declaration-menu.md   # Full specification: all options for every field
├── go.mod / go.sum
└── LICENSE
```

File size budget: **800 lines max** per file. Extract utilities if approaching this limit.

> All files are currently under 800 lines. When adding new features, extract helpers to `render_helpers.go` or split sub-tab logic into a new dedicated file (e.g. `backend_new_subtab.go`).

---

## 4. Architecture

### 4.1 Vim Modal System

The root `Model` (`model.go`) owns three modes:

```go
type Mode int
const (
    ModeNormal   // Navigation: Tab/Shift-Tab between sections, j/k within
    ModeInsert   // Text input: i to enter, Esc to exit
    ModeCommand  // :w :q :wq :tabn :tabp :1-6 :help
)
```

### 4.2 Editor Interface + Polymorphic Dispatch

All sub-editors implement the `Editor` interface defined in `editor.go`:

```go
type Editor interface {
    Mode() Mode
    HintLine() string
    View(w, h int) string
}
```

The root `Model` uses `activeEditor() Editor` and `delegateUpdate()` — both dispatch through `sectionRegistry` in `model_sections.go` rather than switch statements. Adding a new pillar requires one `sectionEntry` registration in `buildSectionRegistry()`; no other files need changing.

Each sub-editor also implements:
- `ToManifest[X]Pillar()` — serializes editor state to manifest types

The **KindDataModel** sentinel field in `sections.go` signals full delegation to the sub-editor.

### 4.3 Shared Navigation Utilities (`nav.go`)

Two reusable helpers replace duplicated navigation code across all sub-editors:

**`NavigateTab(key, active, maxTabs int) int`** — handles `h`/`l`/`left`/`right` tab switching. Used in every editor that has sub-tabs.

**`VimNav` struct** — stateful count-prefix + vim motion handler:
```go
type VimNav struct { CountBuf string; GBuf bool }
// Handle returns (newIdx, consumed). consumed=false for enter/space/i/a (caller handles those).
func (v *VimNav) Handle(key string, idx, n int) (int, bool)
func (v *VimNav) Reset()
```
Handles: digit accumulation, `j`/`k` with count multiplier, `gg` (top), `G` (bottom). Used by `InfraEditor` and `CrossCutEditor`.

### 4.4 List+Form Pattern (used in most sub-editors)

```
SubView: List → user presses Enter → SubView: Form → Esc → SubView: List
```

Lists show items with `j/k` navigation. `a` adds, `d` deletes, `Enter`/`i` edits. Forms use unified `renderFormFields()` from `render_helpers.go`.

### 4.5 Manifest Builder Pattern

Each sub-editor implements `ToManifest[X]Pillar()` converting in-memory form state to the canonical manifest structs. `BuildManifest()` in `model.go` calls all six to assemble the final `manifest.Manifest`.

### 4.6 Model Sub-Structs

The root `Model` struct groups related fields into sub-structs to reduce coupling:

```go
type cmdState    struct { buffer, status string; isErr bool }
type modalState  struct { open bool; menu ProviderMenu }
type realizeState struct { screen RealizationScreen; show, triggered bool }
```

### 4.7 Rendering Layout

All form fields use a consistent vim-style layout via `renderFormFields()`:
```
[LineNo] [Label          ] = [Value]
   3          14            3    (remaining width)
```

Tab bars use `renderSubTabBar()`. Bottom hints use `hintBar()`.

### 4.8 New UI Components

Several new UI modules support the expanded functionality:

- **RealizeEditor** (`realize_editor.go`): Configuration form for code generation with per-section LLM model overrides, concurrency, and verification settings.
- **ProviderMenu** (`provider_menu.go`): Interactive modal for selecting and configuring LLM providers (Claude, ChatGPT, Gemini, Mistral, Llama, Custom) with tier selection.
- **RealizationScreen** (`realization_screen.go`): Display for code generation progress and output status.
- **WelcomeScreen** (`welcome.go`): Initial welcome/tutorial screen and manifest initialization.
- **FrontendAssets** (`frontend_assets.go`): Asset management utilities for frontend design assets.
- **Animation** (`animation.go`): Reusable animation primitives for TUI effects.
- **App** (`app.go`): High-level app initialization and lifecycle management.

These components follow the same polymorphic dispatch pattern via the `Editor` interface.

---

## 5. The 7 Architectural Pillars

### Pillar 1 — Backend (`BackendEditor`)
Sub-tabs: **Env** · **Services** · **Communication** · **Messaging** · **API Gateway** · **Jobs** · **Security** · **Auth**

- Architecture pattern selector (Monolith / Modular Monolith / Microservices / Event-Driven / Hybrid) conditionally shows/hides sub-tabs
- Services list with per-service: name, responsibility, language, framework (dynamically filtered by language), pattern tag
- Communication links: from/to service, protocol, direction, trigger, sync/async, resilience patterns
- Messaging: broker config + repeatable event catalog
- API Gateway: technology, routing, features
- Jobs: background job queues and cron jobs configuration
- Security: WAF configuration, CORS settings, session management
- Auth: strategy, identity provider (with RoleDef list for authorization roles), permission definitions, authorization model, token storage, MFA
  - Supports role-based access control (RBAC) with role inheritance
  - Roles can be referenced in endpoint auth_required fields and frontend page access control

### Pillar 2 — Data (`DataTabEditor` + `DBEditor` + `DataEditor`)
Sub-tabs: **Databases** · **Domains** · **Caching** · **File Storage**

- Databases: alias, category, technology (filtered by category), hosting, HA mode — with type-conditional fields (SSL mode, eviction policy, replication factor, etc.)
- Domains: bounded contexts with repeatable attributes (name, type, constraints, default, sensitive, validation) and relationships (type, FK field, cascade)
- Entities (legacy model): similar to domains but in separate `data_editor.go`
- Caching layer config; File/object storage config

### Pillar 3 — Contracts (`ContractsEditor`)
Sub-tabs: **DTOs** · **Endpoints** · **API Versioning** · **External APIs**

- DTOs: name, category (Request/Response/Event Payload/Shared), source domain, protocol (REST/JSON, Protobuf, Avro, MessagePack, Thrift, FlatBuffers, Cap'n Proto), nested fields with protocol-specific types and validation
  - Protocol-specific fields: Protobuf (package, syntax, options), Avro (namespace, schema registry), Thrift (namespace, language), FlatBuffers/Cap'n Proto (namespace)
- Endpoints: service unit, name/path, protocol (REST/GraphQL/gRPC/WebSocket/Event), auth_required, auth_roles (multi-select from backend roles), request/response DTOs
  - Protocol-specific: HTTP method + pagination strategy (REST), operation type (GraphQL), stream type (gRPC), direction (WebSocket)
- API Versioning: strategy (URL path, header, query param, none), current version, deprecation policy
- External APIs: integration with third-party services with protocol-specific configurations
  - Provider, protocol (REST/GraphQL/gRPC/WebSocket/Webhook/SOAP), auth mechanism (API Key, OAuth2, Bearer, Basic, mTLS, None), failure strategy
  - Protocol-conditional fields: REST (base URL, HTTP method, content type, rate limit, webhook endpoint), GraphQL (operation type), gRPC (stream type, TLS mode), WebSocket (subprotocol, message format), Webhook (HMAC header, retry policy), SOAP (version)
  - Request/response DTOs filtered by protocol (backwards compatible with untagged DTOs)

### Pillar 4 — Frontend (`FrontendEditor`)
Sub-tabs: **Tech** · **Theme** · **Pages** · **Navigation**

- Tech: language, platform, framework (filtered by language+platform), meta-framework, package manager, styling, component library, state management, data fetching, form handling, validation, PWA support, realtime strategy, image optimization, auth flow type, error boundary, bundle optimization, frontend testing, frontend linter
- Theme: dark mode strategy, border radius, spacing scale, elevation, motion, vibe, colors, description
- Pages: route, auth_required, layout, core actions, loading strategy, error handling, auth_roles (multi-select from backend roles for role-based page access), linked pages
- Navigation: nav type (sidebar, top bar, etc.), breadcrumbs toggle, auth-aware navigation toggle
- Assets: frontend design assets (images, icons, fonts, videos, mockups, etc.) with usage classification (project or inspiration)

### Pillar 5 — Infrastructure (`InfraEditor`)
Sub-tabs: **Networking** · **CI/CD** · **Observability**

- Networking: DNS, TLS, reverse proxy, CDN
- CI/CD: platform, container registry, deploy strategy, IaC tool, secrets management
- Observability: logging, metrics, tracing, error tracking, health checks, alerting

### Pillar 6 — Cross-Cutting (`CrosscutEditor`)
Sub-tabs: **Testing** · **Docs**

- Testing: testing framework selections dynamically filtered by backend languages and frontend tech choices
  - Unit: language-specific test framework (Jest, Vitest for JavaScript/TypeScript; pytest, Go testing, JUnit, xUnit for others)
  - Integration: integration test framework
  - E2E: end-to-end test tool (Playwright, Cypress, Nightwatch, Selenium, etc.)
  - API: API testing tool (REST, GraphQL, gRPC specific)
  - Load: load testing tool (k6, Locust, Apache JMeter, etc.)
  - Contract: contract testing tool (Pact, Spring Cloud Contract)
- Docs: API doc format (OpenAPI/Swagger, GraphQL schema doc, AsyncAPI, etc.), auto-generation toggle, changelog strategy

### Pillar 7 — Realize (`RealizeEditor`)
Configuration tab for downstream code generation pipeline:

- app_name: application name for generated code
- output_dir: destination directory for generated files
- model: LLM model selection (claude-haiku-4-5-20251001, claude-sonnet-4-6, claude-opus-4-6) — controls intelligence level for code generation
- concurrency: parallel task execution limit (1, 2, 4, 8)
- verify: enable/disable code verification after generation (default: true)
- dry_run: print task plan without executing agent calls
- Per-section model overrides: allow different LLM models for each pillar (backend, data, contracts, frontend, infra, crosscut)

---

## 6. Realize Engine (Code Generation)

`cmd/realize` is the downstream consumer of `manifest.json`. It drives an agentic code-generation pipeline.

### 6.1 Pipeline Overview

```
manifest.json
    ↓
dag.Builder.Build()   → execution DAG (tasks with dependency edges)
    ↓
orchestrator.Run()    → parallel task dispatch (bounded by --parallel flag)
    ↓  (per task)
runner.Run()          → agent.Call() → verify.Check() → retry up to MaxRetries
    ↓
output.Writer         → writes generated files under --output directory
```

### 6.2 DAG Task IDs

Tasks follow a naming convention derived from manifest entries:

| Pattern | Example |
|---------|---------|
| `data.<alias>` | `data.postgres` |
| `svc.<name>` | `svc.api-gateway` |
| `contracts` | `contracts` |
| `frontend` | `frontend` |
| `infra.<component>` | `infra.networking` |
| `crosscut.<component>` | `crosscut.testing` |

### 6.3 Skills System

Skills are markdown files in `.vibemenu/skills/` (configurable via `--skills`). Each file defines a named generation skill. The `skills.Loader` reads them at startup; the `skills.Registry` makes them available to the agent prompt builder.

### 6.4 Verifiers

After each agent call, a language-appropriate verifier checks the output:

| Language | Verifier | Check |
|----------|----------|-------|
| Go | `go_verifier` | `go build` + `go vet` |
| TypeScript | `ts_verifier` | `tsc --noEmit` |
| Python | `python_verifier` | `python -m py_compile` |
| Terraform | `tf_verifier` | `terraform validate` |
| Other | `null_verifier` | always passes |

### 6.5 CLI Flags

```
--manifest  path to manifest.json      (default: manifest.json)
--output    output directory            (default: output)
--skills    skills directory            (default: .vibemenu/skills)
--retries   max retry attempts per task (default: 3)
--parallel  max concurrent tasks        (default: 1)
--dry-run   print task plan, no agents
--verbose   print token usage + thinking logs
```

---

## 7. Manifest Output

Saved to `manifest.json` on `:w` / `Ctrl+S`. Structure:

```json
{
  "created_at": "2026-...",
  "backend":    { "arch_pattern": "...", "services": [...], "auth": { "roles": [...], ... }, ... },
  "data":       { "databases": [...], "domains": [...], ... },
  "contracts":  { "dtos": [...], "endpoints": [...], "external_apis": [...], ... },
  "frontend":   { "tech": {...}, "pages": [...], "assets": [...], ... },
  "infrastructure": { "networking": {...}, "cicd": {...}, ... },
  "cross_cutting":  { "testing": {...}, "docs": {...} },
  "realize":    { "app_name": "...", "output_dir": "...", "model": "...", ... },
  "configured_providers": { ... }
}
```

---

## 8. Key Recent Additions (v2.0)

- **Auth Roles (Backend Pillar):** Auth tab now supports defining RoleDef entries with hierarchical role inheritance. Roles are made available to Contracts (endpoint auth_required filter) and Frontend (page access control).
- **External APIs (Contracts Pillar):** New fourth sub-tab for configuring third-party API integrations with protocol-specific configuration options (REST, GraphQL, gRPC, WebSocket, Webhook, SOAP).
- **Protocol-Tagged DTOs:** DTOs now have a protocol field allowing filtering by serialization format. External API DTOs are filtered to match the selected protocol.
- **Testing Framework Filtering (Cross-Cutting Pillar):** Testing tool options are dynamically filtered based on selected backend languages and frontend framework/language.
- **Realize Tab (7th Pillar):** New configuration tab for code generation pipeline with per-section LLM model overrides and concurrency/verification settings.
- **Provider Modal:** Interactive provider selection menu for configuring LLM providers and tiers for the Realize pipeline.
- **Frontend Assets:** Pages now support asset definitions (images, icons, fonts, videos, mockups) with usage classification.
- **Job Queues & Security Tab:** Backend now includes job scheduling (cron jobs, worker queues) and security configuration (WAF, CORS, session management) as dedicated sub-tabs.

---

## 9. Key Bindings Reference

### Global (Normal Mode)
| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Next / previous main section |
| `j` / `k` | Navigate within section |
| `Space` | Cycle select field |
| `i` | Enter insert mode |
| `:` | Enter command mode |
| `Ctrl+S` | Save manifest |
| `Ctrl+C` | Quit |

### Command Mode
| Command | Action |
|---------|--------|
| `:w` / `:write` | Save |
| `:q` / `:quit` | Quit without save |
| `:wq` / `:x` | Save and quit |
| `:tabn` / `:bn` | Next section |
| `:tabp` / `:bp` | Previous section |
| `:1`–`:7` | Jump to section N |

### Sub-Editor (varies by tab)
| Key | Action |
|-----|--------|
| `a` | Add item (list view) |
| `d` | Delete item (list view) |
| `Enter` / `i` | Edit / insert mode |
| `h` / `l` | Switch sub-tab |
| `b` / `Esc` | Back to parent / exit insert |
| `F` | Drill into nested fields (DTOs) |
| `A` | Drill into attributes (Domains) |

---

## 10. Go Engineering Standards

- **Error handling:** Never swallow errors. Use `fmt.Errorf("context: %w", err)` for wrapping.
- **Immutability:** Favor passing structs by value. Return new copies rather than mutating in place.
- **File size:** 200–400 lines typical, 800 lines hard max. Split by feature/domain.
- **Formatting:** `gofmt` enforced. Run `go vet` before committing.
- **No cobra/viper:** This project uses raw `bubbletea` — do not add cobra or viper unless adding a non-interactive CLI mode.
- **Style constants:** All colors and styles live in `styles.go`. Do not inline lipgloss colors elsewhere.
- **Shared rendering:** Add new rendering helpers to `render_helpers.go`, not inline in sub-editors.
- **Field abstraction:** New form fields use the `Field` struct with `KindText`, `KindSelect`, or `KindTextArea`. Never render raw text inputs directly in sub-editors.
- **Tab navigation:** Use `NavigateTab()` from `nav.go` for `h`/`l` sub-tab switching — do not duplicate this switch in new editors.
- **Vim list navigation:** Use `VimNav` from `nav.go` for `j`/`k`/`gg`/`G`/count-prefix in any new editor with a navigable list.
- **Editor interface:** New sub-editors must implement the `Editor` interface (`Mode()`, `HintLine()`, `View()`). Register them in `buildSectionRegistry()` in `model_sections.go` — add one `sectionEntry` with `editor` and `update` closures, and add the section ID to `sectionOrder`.
- **Manifest types:** Add new pillar types to the appropriate `manifest_*.go` file, not to `manifest.go`. Only the root `Manifest` struct and `Save()` belong in `manifest.go`.
- **Model registry:** Add new AI providers or model tiers to `providerModels` in `orchestrator/models.go`. Do not add new switch cases to `resolveAgent()`.
- **Skill aliases:** Add new technology aliases to `aliasMap` in `skills/aliases.go`. Universal skills for a task kind go in `universalSkillsForKind` in the same file.

---

## 11. Specification Reference

`system-declaration-menu.md` is the canonical specification for all menu options, field names, and valid values across all 7 pillars. When adding or modifying any editor field, cross-reference this document to ensure alignment.

The dependency graph for non-linear resolution:
```
Data (Domains, Databases)
    ↓
Backend (Service Units reference Domains; defines Auth Roles)
    ↓
Contracts (DTOs reference Domains; Endpoints reference Service Units + Auth Roles; External APIs)
    ↓
Frontend (Pages reference Endpoints + DTOs + Auth Roles from Backend)
    ↓
Infrastructure (references all deployable units)
    ↓
Cross-Cutting (Testing frameworks filtered by Backend languages + Frontend tech; Docs formats)
    ↓
Realize (Code generation config — orchestrates generation for all pillars)
```

Empty references show as "unlinked" placeholders — the UI must allow editing in any order.
