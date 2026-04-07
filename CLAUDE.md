# VibeMenu — Project & Engineering Guide

## 1. Overview

**VibeMenu** is a vim-modal TUI for declaratively specifying software system architecture across 8 sections (Description + 7 pillars). Output is `manifest.json`, consumed by `cmd/realize` for agentic code generation.

**Principles:** Vim-modal editing (Normal/Insert/Command) | Tokyo Night theme | Non-linear editing | Pillar dependency graph: Description > Data > Backend > Contracts > Frontend > Infra > Cross-Cutting > Realize

| Concern | Choice |
|---------|--------|
| Language | Go 1.26.1 |
| TUI | `bubbletea` v1.3.10, `bubbles` v1.0.0, `lipgloss` v1.1.0 |
| Claude SDK | `anthropic-sdk-go` v1.28.0 |
| Entry points | `cmd/agent/main.go` (TUI), `cmd/realize/main.go` (codegen) |

---

## 2. Project Structure

```
cmd/agent/main.go          # TUI entry — Bubble Tea program
cmd/realize/main.go         # Codegen entry — CLI flags, orchestrator
internal/
  bundled/                  # Embedded skills markdown + loader patches
  manifest/                 # 11 files: root Manifest, per-pillar types, enums, providers, recent
  ui/                       # 14 subdirectories, no root files
    app/                    # Root Model, section registry, view dispatch (4 files)
    arch/                   # Architecture visualization diagrams (5 files)
    backend/                # Backend pillar editor (15 files, ~8.4k lines)
    contracts/              # Contracts pillar editor (5 files)
    core/                   # Shared: styles, nav, rendering, fields, undo (15 files)
    crosscut/               # Cross-cutting editor (2 files)
    data/                   # Data pillar editor + entity/DB editors (12 files, ~6.3k lines)
    description/            # Free-text textarea (1 file)
    frontend/               # Frontend pillar editor (9 files)
    infra/                  # Infrastructure editor (3 files)
    provider/               # Provider modal + credentials (5 files)
    realization/            # Codegen progress screen (1 file)
    realize_cfg/            # Realize config form (1 file)
    welcome/                # Welcome screen (1 file)
  realize/
    agent/                  # ClaudeAgent, OpenAIAgent, GeminiAgent + prompts + roles
    dag/                    # DAG builder, topological sort, TaskKind enums
    config/                 # Tunable constants (DefaultModel, MaxTokens, etc.)
    orchestrator/           # Config, runner, tier escalation, reconcile, repair, model registry
    memory/                 # SharedMemory + sigscan (signature extraction)
    state/                  # State tracking during generation
    output/                 # File writer
    skills/                 # Skill registry, loader, aliases, extraction
    deps/                   # Dependency resolution (Go modules, npm, library docs)
    verify/                 # Verifiers (Go/TS/Python/Terraform) + deterministic fixes per language
system-declaration-menu.md  # Canonical spec for all field options
```

**File size budget:** 800 lines max. Files near limit: `arch_render.go` (907), `data_tab_editor.go` (790), `backend_view.go` (754), `backend_services.go` (744), `backend_auth.go` (737), `deterministic_fixes_go.go` (713), `contracts_fields_endpoints.go` (707). Extract helpers before adding to these.

---

## 3. Architecture

### 3.1 Vim Modes (`core/editor.go`)

`ModeNormal` (navigation) | `ModeInsert` (text input) | `ModeCommand` (`:w`, `:q`, etc.)

### 3.2 Editor Interface + Registry

All sub-editors implement `Editor` (from `core/editor.go`): `Mode()`, `HintLine()`, `View(w, h)`.

Dispatch uses `sectionRegistry` in `app/model_sections.go` (not switch statements). Adding a pillar = one `sectionEntry` in `buildSectionRegistry()`. Each editor also implements `ToManifest[X]Pillar()`.

`KindDataModel` sentinel in `core/sections.go` signals full delegation to sub-editor.

### 3.3 Shared Navigation (`core/nav.go`)

- **`NavigateTab(key, active, maxTabs)`** — `h`/`l` sub-tab switching
- **`VimNav`** — count-prefix + `j`/`k`/`gg`/`G` motion handler

### 3.4 Undo System (`core/undo.go`)

Generic `UndoStack[T]` (depth 50). `CopySlice`, `CopyFieldItems` for snapshots. Typed snapshots: `SvcSnapshot`, `CommSnapshot`, `EventSnapshot`.

### 3.5 List+Form Pattern

```
List (j/k nav, a=add, d=delete, Enter=edit) → Form (renderFormFields from core/) → Esc → List
```

### 3.6 Rendering (`core/render_helpers.go`, `core/render_util.go`)

Form layout: `[LineNo] [Label] = [Value]` (3 + 14 + 3 + remaining). Tab bars: `renderSubTabBar()`. Hints: `hintBar()`.

### 3.7 Model Sub-Structs (`app/model.go`)

`cmdState` (command buffer), `modalState` (provider menu), `realizeState` (codegen screen).

### 3.8 Field Descriptions (`core/field_descriptions*.go`)

Split across 3 files: general+backend, contracts-specific, infrastructure-specific.

---

## 4. The 8 Sections

**Section 0 — Description** (`description/`): Free-text project description textarea.

**Pillar 1 — Backend** (`backend/`): Sub-tabs: Env, Services, Stack Config, Communication, Messaging, API Gateway, Jobs, Security, Auth. Architecture pattern (Monolith/Microservices/etc.) conditionally shows/hides sub-tabs. RBAC roles referenced by endpoints and frontend pages.

**Pillar 2 — Data** (`data/`): Sub-tabs: Databases, Domains, Caching, File Storage. Domains have repeatable attributes + relationships. Entity editor in `data_editor.go`.

**Pillar 3 — Contracts** (`contracts/`): Sub-tabs: DTOs, Endpoints, API Versioning, External APIs. Protocol-conditional fields throughout (REST/GraphQL/gRPC/WebSocket/etc.). External APIs support 6 protocols with auth mechanisms.

**Pillar 4 — Frontend** (`frontend/`): Sub-tabs: Tech, Theme, Pages, Navigation, i18n, A11y/SEO. Pages define components with 12+ action types. Assets via `AssetDef`.

**Pillar 5 — Infrastructure** (`infra/`): Sub-tabs: Networking, CI/CD, Observability. Server environments with compute/cloud settings.

**Pillar 6 — Cross-Cutting** (`crosscut/`): Sub-tabs: Testing, Docs. Testing frameworks filtered by backend languages + frontend tech.

**Pillar 7 — Realize** (`realize_cfg/`): app_name, output_dir, concurrency, verify, dry_run, provider selection, tier_fast/medium/slow model IDs, provider assignments per section.

---

## 5. Realize Engine

### 5.1 Pipeline

```
manifest.json → dag.Build() → orchestrator.Run() → runner.Run() (per task)
  → deterministic fixes → agent.Call() → verify.Check() → retry (tier escalation)
  → reconcile + repair → memory.SharedMemory → output.Writer
```

### 5.2 Multi-Provider Agents

| Agent | Providers |
|-------|-----------|
| ClaudeAgent | Haiku, Sonnet, Opus |
| OpenAIAgent | o3-mini/4o/o1, Mistral, Llama via Groq |
| GeminiAgent | Flash, Pro, Ultra |

`roles.go` has per-`TaskKind` role instructions injected into system prompts.

### 5.3 Model Tiering (`orchestrator/tier.go`)

| Tier | Default kinds |
|------|--------------|
| fast (Haiku/o3-mini/Flash) | contracts, docs, docker, CI |
| medium (Sonnet/4o/Pro) | services, auth, data, frontend, terraform, testing |
| slow (Opus/o1/Ultra) | escalation fallback |

### 5.4 Shared Memory

Stores completed task outputs. `sigscan.go` extracts signatures (functions, types, interfaces) for downstream agent context.

### 5.5 DAG Task IDs

Pattern: `data.<alias>`, `svc.<name>`, `contracts`, `frontend`, `infra.<component>`, `crosscut.<component>`

### 5.6 Skills

Markdown files in `.vibemenu/skills/` + `internal/bundled/skills/`. Loader reads at startup, Registry provides to prompt builder. Aliases in `skills/aliases.go`.

### 5.7 Verifiers

Go (`go build` + `go vet`), TypeScript (`tsc --noEmit`), Python (`py_compile`), Terraform (`terraform validate`), null (always passes). Deterministic fixes (gofmt, unused imports, etc.) applied before retries to save tokens.

### 5.8 CLI Flags

```
--manifest (default: manifest.json)  --output (default: output)  --skills (default: .vibemenu/skills)
--retries (default: 3)  --parallel (default: 1)  --dry-run  --verbose
```

---

## 6. Key Bindings

**Normal:** Tab/Shift-Tab (sections), j/k (navigate), Space (cycle), i (insert), : (command), Ctrl+S (save), Shift+M (provider menu), Ctrl+C (quit)

**Command:** :w (save), :q (quit), :wq/:x (save+quit), :tabn/:tabp (next/prev section), :1-:8 (jump)

**Sub-Editor:** a (add), d (delete), Enter/i (edit), h/l (sub-tab), b/Esc (back), F (drill fields), A (drill attributes)

---

## 7. Manifest Output

Saved on `:w`/`Ctrl+S`. Top-level keys: `created_at`, `description`, `backend`, `data`, `contracts`, `frontend`, `infrastructure`, `cross_cutting`, `realize`, `configured_providers`.

---

## 8. Go Engineering Standards

- **Errors:** Never swallow. Wrap with `fmt.Errorf("context: %w", err)`.
- **Immutability:** Pass structs by value, return new copies.
- **File size:** 200-400 typical, 800 max. Split by feature/domain.
- **Formatting:** `gofmt` + `go vet` before committing.
- **No cobra/viper** unless adding a non-interactive CLI mode.
- **Styles:** All colors in `core/styles.go`. No inline lipgloss colors.
- **Rendering:** Helpers go in `core/render_helpers.go` or `core/render_util.go`, not inline.
- **Fields:** Use `Field` struct from `core/sections.go` (`KindText`/`KindSelect`/`KindTextArea`). No raw text inputs in sub-editors.
- **Field descriptions:** Add to appropriate `core/field_descriptions*.go` file.
- **Tab nav:** Use `NavigateTab()` from `core/nav.go`. Don't duplicate.
- **List nav:** Use `VimNav` from `core/nav.go` for j/k/gg/G/count-prefix.
- **Undo:** Use `UndoStack[T]` from `core/undo.go`.
- **New editors:** Implement `Editor` interface, register in `buildSectionRegistry()` in `app/model_sections.go`.
- **Package org:** Each pillar = own subdirectory under `internal/ui/`. Shared = `core/`. No files in `ui/` root.
- **Manifest types:** Per-pillar types in `manifest_*.go`. Root struct + `Save()` only in `manifest.go`. Providers in `providers.go`.
- **Model registry:** New providers/tiers in `providerModels` in `orchestrator/models.go`.
- **Skill aliases:** New aliases in `aliasMap` in `skills/aliases.go`. Universal skills in `universalSkillsForKind`.
- **Task roles:** New `TaskKind` instructions in `taskRoleDescriptions` in `agent/roles.go`.
- **Model tiering:** New `TaskKind` defaults in `defaultTierForKind` in `orchestrator/tier.go`.

---

## 9. Specification Reference

`system-declaration-menu.md` is the canonical spec for all field options. Cross-reference when adding/modifying editor fields.

Dependency graph: Description > Data > Backend > Contracts > Frontend > Infra > Cross-Cutting > Realize. Empty references show as "unlinked" placeholders — UI allows editing in any order.
