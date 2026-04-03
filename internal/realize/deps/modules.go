package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)


// ResolvedDeps holds the validated go.mod content and a mapping of
// import path → resolved version for common libraries.
type ResolvedDeps struct {
	GoMod    string            `json:"go_mod"`
	GoSum    string            `json:"go_sum"`
	Versions map[string]string `json:"versions"`
}

// ModuleInfo holds the canonical import path and version for a Go module.
type ModuleInfo struct {
	Module   string
	Version  string
	TestDeps []ModuleDep
}

// ModuleDep is a dependency needed alongside a primary module.
type ModuleDep struct {
	Module  string
	Version string
}

// WellKnownGoModules maps framework/library names used in manifests to their
// actual Go module paths and known-good recent versions.
// This is the SINGLE SOURCE OF TRUTH for dependency versions — agents never
// guess versions; they use these.
//
// To update: change the version string here and re-run.
// To add a new library: add an entry and the pipeline picks it up automatically.
// PromptContext generates text to inject into agent prompts that provides
// exact dependency versions and library API docs for the task's technology stack.
func PromptContext(framework string, technologies []string) string {
	var b strings.Builder

	b.WriteString("\n## Dependency & API Reference\n\n")
	b.WriteString("Use EXACTLY these module paths and versions in go.mod. Do NOT invent versions.\n\n")

	// List relevant modules with their exact versions.
	seen := make(map[string]bool)
	var modules []ModuleInfo

	if info, ok := WellKnownGoModules[framework]; ok && !seen[info.Module] {
		seen[info.Module] = true
		modules = append(modules, info)
	}
	for _, tech := range technologies {
		if info, ok := WellKnownGoModules[tech]; ok && !seen[info.Module] {
			seen[info.Module] = true
			modules = append(modules, info)
		}
	}

	if len(modules) > 0 {
		b.WriteString("### Exact Module Versions\n\n| Module | Version |\n|--------|--------|\n")
		for _, m := range modules {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", m.Module, m.Version))
			for _, td := range m.TestDeps {
				b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", td.Module, td.Version))
			}
		}
		b.WriteString("\n")
	}

	// Inject library API docs for relevant technologies.
	injected := make(map[string]bool)
	allTechs := append([]string{framework}, technologies...)
	for _, tech := range allTechs {
		lower := strings.ToLower(tech)
		for key, doc := range LibraryAPIDocs {
			if injected[key] {
				continue
			}
			if strings.Contains(lower, key) || strings.Contains(key, lower) {
				b.WriteString(doc)
				b.WriteString("\n")
				injected[key] = true
			}
		}
	}

	// Special case: if PostgreSQL is in the stack, always inject pgxmock docs.
	for _, tech := range allTechs {
		if strings.Contains(strings.ToLower(tech), "postgre") || strings.Contains(strings.ToLower(tech), "pgx") {
			if !injected["pgxmock"] {
				b.WriteString(LibraryAPIDocs["pgxmock"])
				injected["pgxmock"] = true
			}
		}
	}

	return b.String()
}

// GoModForService generates a go.mod for a service based on its declared
// framework and database dependencies, using only known-good versions.
func InfraPromptContext(ctx context.Context, hasGoServices bool, hasFrontend bool) string {
	var b strings.Builder
	b.WriteString("\n## Infrastructure & Dependency Reference\n\n")
	b.WriteString("Use EXACTLY these versions. Do NOT invent alternatives.\n\n")

	if hasGoServices {
		// Resolve dev tool versions from the Go module proxy at runtime.
		tools := resolveAllGoDevTools(ctx)

		// Derive the minimum Go version required across all dev tools.
		minGoVersion := "1.23"
		for _, t := range tools {
			if t.MinGoVersion > minGoVersion {
				minGoVersion = t.MinGoVersion
			}
		}

		b.WriteString(fmt.Sprintf("### Go Docker Base Image\n\n```\nFROM golang:%s-alpine\n```\n\n", minGoVersion))
		b.WriteString(fmt.Sprintf("Minimum Go %s required for dev tools.\n\n", minGoVersion))

		b.WriteString("### Go Dev Tools (install via `go install`, NOT in go.mod)\n\n")
		b.WriteString("| Tool | Correct Module Path | Version | Min Go |\n")
		b.WriteString("|------|---------------------|---------|--------|\n")
		for _, t := range tools {
			b.WriteString(fmt.Sprintf("| %s | `%s` | `%s` | %s |\n",
				t.Name, t.ModulePath+"@"+t.Version, t.Version, t.MinGoVersion))
		}
		b.WriteString("\n")

		// Build the Dockerfile example dynamically from resolved tool versions.
		b.WriteString("### Go Dockerfile Rules\n\n")
		b.WriteString(fmt.Sprintf("Go base image: always use `golang:%s-alpine` or newer.\n", minGoVersion))
		for _, t := range tools {
			if t.Name == "air" {
				b.WriteString(fmt.Sprintf("Hot-reload tool (air): CORRECT path is `%s@%s`\n", t.ModulePath, t.Version))
				b.WriteString("  WRONG (old, renamed): `github.com/cosmtrek/air` ← DO NOT USE\n\n")
			}
		}
		b.WriteString("Required Dockerfile layer order (copy go.mod before source for layer caching):\n\n")
		b.WriteString(fmt.Sprintf("```dockerfile\nFROM golang:%s-alpine\nWORKDIR /app\n", minGoVersion))
		for _, t := range tools {
			b.WriteString(fmt.Sprintf("RUN go install %s@%s\n", t.ModulePath, t.Version))
		}
		b.WriteString("COPY go.mod go.sum ./\n")
		b.WriteString("RUN go mod download\n")
		b.WriteString("COPY . .\n")
		b.WriteString("CMD [\"air\", \"-c\", \".air.toml\"]\n```\n\n")
		b.WriteString("Without `go mod download`, air's incremental build will fail with `go: updates to go.mod needed`.\n\n")
	}

	if hasFrontend {
		// Resolve npm package versions from the registry at runtime.
		resolved := resolveAllNpmVersions(ctx)

		b.WriteString("### Node.js Docker Base Image\n\n```dockerfile\nFROM node:20-alpine\n```\n\n")
		b.WriteString("### npm Package Versions\n\n")
		b.WriteString("| Package | Version |\n|---------|--------|\n")
		pkgs := make([]string, 0, len(resolved))
		for pkg := range resolved {
			pkgs = append(pkgs, pkg)
		}
		sort.Strings(pkgs)
		for _, pkg := range pkgs {
			b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", pkg, resolved[pkg]))
		}
		b.WriteString("\n")
		b.WriteString(LibraryAPIDocs["next"])
		b.WriteString("\n")
	}

	return b.String()
}

// ValidateGoMod runs go mod tidy in a temp directory to resolve real versions.
func SaveResolvedDeps(dir, taskID string, d *ResolvedDeps) error {
	depsDir := filepath.Join(dir, ".realize", "deps")
	if err := os.MkdirAll(depsDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(depsDir, taskID+".json"), data, 0644)
}

// LoadResolvedDeps reads previously resolved dependency info.
func LoadResolvedDeps(dir, taskID string) (*ResolvedDeps, error) {
	path := filepath.Join(dir, ".realize", "deps", taskID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d ResolvedDeps
	return &d, json.Unmarshal(data, &d)
}
