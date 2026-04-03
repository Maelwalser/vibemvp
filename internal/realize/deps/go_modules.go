package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var WellKnownGoModules = map[string]ModuleInfo{
	// ── Web frameworks ─────────────────────────────────────────────
	"Fiber": {
		Module: "github.com/gofiber/fiber/v2", Version: "v2.52.5",
		TestDeps: []ModuleDep{
			{Module: "github.com/stretchr/testify", Version: "v1.9.0"},
		},
	},
	"Gin":  {Module: "github.com/gin-gonic/gin", Version: "v1.10.0"},
	"Echo": {Module: "github.com/labstack/echo/v4", Version: "v4.12.0"},
	"Chi":  {Module: "github.com/go-chi/chi/v5", Version: "v5.1.0"},

	// ── Database drivers ───────────────────────────────────────────
	"pgx": {
		Module: "github.com/jackc/pgx/v5", Version: "v5.7.2",
		TestDeps: []ModuleDep{
			{Module: "github.com/pashagolub/pgxmock/v4", Version: "v4.4.0"},
		},
	},
	"PostgreSQL": {Module: "github.com/jackc/pgx/v5", Version: "v5.7.2"},
	"MySQL":      {Module: "github.com/go-sql-driver/mysql", Version: "v1.8.1"},
	"SQLite":     {Module: "modernc.org/sqlite", Version: "v1.34.4"},
	"MongoDB":    {Module: "go.mongodb.org/mongo-driver", Version: "v1.17.1"},

	// ── Auth ───────────────────────────────────────────────────────
	"JWT":    {Module: "github.com/golang-jwt/jwt/v5", Version: "v5.2.1"},
	"bcrypt": {Module: "golang.org/x/crypto", Version: "v0.31.0"},

	// ── Testing ────────────────────────────────────────────────────
	"testify": {Module: "github.com/stretchr/testify", Version: "v1.9.0"},
	"pgxmock": {Module: "github.com/pashagolub/pgxmock/v4", Version: "v4.4.0"},

	// ── Validation ─────────────────────────────────────────────────
	"validator": {Module: "github.com/go-playground/validator/v10", Version: "v10.22.1"},

	// ── Logging ────────────────────────────────────────────────────
	"zap":     {Module: "go.uber.org/zap", Version: "v1.27.0"},
	"zerolog": {Module: "github.com/rs/zerolog", Version: "v1.33.0"},

	// ── UUID ───────────────────────────────────────────────────────
	"uuid": {Module: "github.com/google/uuid", Version: "v1.6.0"},

	// ── Message brokers ────────────────────────────────────────────
	"NATS": {Module: "github.com/nats-io/nats.go", Version: "v1.37.0"},

	// ── Config ─────────────────────────────────────────────────────
	"envconfig": {Module: "github.com/kelseyhightower/envconfig", Version: "v1.4.0"},
}

// GoDevTool describes a Go tool installed in Dockerfiles (not in go.mod).
type GoDevTool struct {
	Name         string // human-readable name
	ModulePath   string // correct module path (may differ from historical path)
	Version      string // pinned version known to work
	MinGoVersion string // minimum Go version required by this tool version
}

// WellKnownGoDevTools lists dev tools installed via `go install` in Dockerfiles.
// These are NOT added to go.mod — they are installed in the Docker image layer only.
// The Version field is a fallback; InfraPromptContext resolves the actual latest
// compatible version from proxy.golang.org at runtime.
// IMPORTANT: github.com/cosmtrek/air was renamed to github.com/air-verse/air at v1.52.x.
// Never use the old module path.
var WellKnownGoDevTools = []GoDevTool{
	{
		Name:         "air",
		ModulePath:   "github.com/air-verse/air",
		Version:      "v1.61.5", // fallback; resolved dynamically at runtime
		MinGoVersion: "1.23",
	},
}

// WellKnownNpmPackages maps npm package names to fallback versions used when the
// npm registry is unreachable. At runtime, InfraPromptContext resolves the actual
// latest stable versions from registry.npmjs.org and only falls back to these.
func GoModForService(modulePath, framework string, technologies []string) string {
	var requires []string
	seen := make(map[string]bool)

	addModule := func(info ModuleInfo) {
		if !seen[info.Module] {
			seen[info.Module] = true
			requires = append(requires, fmt.Sprintf("\t%s %s", info.Module, info.Version))
		}
		for _, td := range info.TestDeps {
			if !seen[td.Module] {
				seen[td.Module] = true
				requires = append(requires, fmt.Sprintf("\t%s %s", td.Module, td.Version))
			}
		}
	}

	if info, ok := WellKnownGoModules[framework]; ok {
		addModule(info)
	}
	for _, tech := range technologies {
		if info, ok := WellKnownGoModules[tech]; ok {
			addModule(info)
		}
	}
	for _, key := range []string{"testify", "uuid"} {
		if info, ok := WellKnownGoModules[key]; ok {
			addModule(info)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("module %s\n\ngo 1.23\n\n", modulePath))
	if len(requires) > 0 {
		b.WriteString("require (\n")
		for _, r := range requires {
			b.WriteString(r + "\n")
		}
		b.WriteString(")\n")
	}
	return b.String()
}

// resolveNpmVersion fetches the latest stable version of a package from the npm registry.
// Falls back to fallback on any error (network failure, registry unavailable, etc.).
func ValidateGoMod(ctx context.Context, goModContent string, goFiles map[string]string) (*ResolvedDeps, error) {
	tmpDir, err := os.MkdirTemp("", "deps-resolve-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return nil, fmt.Errorf("write go.mod: %w", err)
	}
	for path, content := range goFiles {
		full := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(full), 0755)
		os.WriteFile(full, []byte(content), 0644)
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go mod tidy: %s\n%s", err, string(out))
	}

	modData, _ := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	sumData, _ := os.ReadFile(filepath.Join(tmpDir, "go.sum"))

	return &ResolvedDeps{
		GoMod:    string(modData),
		GoSum:    string(sumData),
		Versions: parseRequires(string(modData)),
	}, nil
}

func parseRequires(gomod string) map[string]string {
	versions := make(map[string]string)
	inRequire := false
	for _, line := range strings.Split(gomod, "\n") {
		line = strings.TrimSpace(line)
		if line == "require (" {
			inRequire = true
			continue
		}
		if line == ")" {
			inRequire = false
			continue
		}
		if inRequire {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				versions[parts[0]] = parts[1]
			}
		}
	}
	return versions
}

// SaveResolvedDeps persists resolved deps for downstream tasks.
func resolveGoDevToolVersion(ctx context.Context, t GoDevTool) GoDevTool {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	encoded := goModProxyPath(t.ModulePath)

	// Step 1: resolve @latest version tag.
	latestURL := "https://proxy.golang.org/" + encoded + "/@latest"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, latestURL, nil)
	if err != nil {
		return t
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return t
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return t
	}
	var latest struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil || latest.Version == "" {
		return t
	}

	resolved := t
	resolved.Version = latest.Version

	// Step 2: fetch the go.mod for that version to read the `go` directive,
	// which tells us the minimum Go version the tool requires.
	modURL := "https://proxy.golang.org/" + encoded + "/@v/" + latest.Version + ".mod"
	modReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, modURL, nil)
	if err != nil {
		return resolved
	}
	modResp, err := http.DefaultClient.Do(modReq)
	if err != nil || modResp.StatusCode != http.StatusOK {
		if modResp != nil {
			modResp.Body.Close()
		}
		return resolved
	}
	defer modResp.Body.Close()

	// Parse "go X.YY" from the go.mod content.
	if data, err := io.ReadAll(modResp.Body); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "go ") {
				if ver := strings.TrimPrefix(line, "go "); ver != "" {
					resolved.MinGoVersion = ver
				}
				break
			}
		}
	}

	return resolved
}

// goModProxyPath encodes a module path for the Go module proxy API by replacing
// each uppercase letter with "!" followed by the lowercase equivalent.
func goModProxyPath(module string) string {
	var b strings.Builder
	for _, r := range module {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// resolveAllGoDevTools resolves the latest version for each dev tool concurrently.
func resolveAllGoDevTools(ctx context.Context) []GoDevTool {
	resolved := make([]GoDevTool, len(WellKnownGoDevTools))
	var wg sync.WaitGroup
	for i, t := range WellKnownGoDevTools {
		wg.Add(1)
		go func(idx int, tool GoDevTool) {
			defer wg.Done()
			resolved[idx] = resolveGoDevToolVersion(ctx, tool)
		}(i, t)
	}
	wg.Wait()
	return resolved
}

// InfraPromptContext generates the dependency & API reference context for infrastructure
// and frontend tasks. Versions are resolved dynamically from the npm registry and Go
// module proxy; the values in WellKnownNpmPackages and WellKnownGoDevTools are used as
// fallbacks only when the registries are unreachable.
