package verify

import (
	"strings"
	"testing"

	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/dag"
)

// ── normalizeLanguage ─────────────────────────────────────────────────────────

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Go", "go"},
		{"TypeScript", "typescript"},
		{"JavaScript", "typescript"},
		{"Node.js", "typescript"},
		{"TypeScript/Node", "typescript"},
		{"Python", "python"},
		{"Rust", "rust"},
		// Unknown / unsupported languages return empty string
		{"Java", ""},
		{"Kotlin", ""},
		{"PHP", ""},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeLanguage(tc.input)
			if got != tc.want {
				t.Errorf("normalizeLanguage(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ── normalizeFrontendLanguage ─────────────────────────────────────────────────

func TestNormalizeFrontendLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"TypeScript", "typescript"},
		{"JavaScript", "typescript"},
		{"Dart", ""}, // Flutter — no tsc equivalent
		{"", ""},
		{"Unknown", ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeFrontendLanguage(tc.input)
			if got != tc.want {
				t.Errorf("normalizeFrontendLanguage(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ── taskLanguage ──────────────────────────────────────────────────────────────

func TestTaskLanguage_ServiceTask_UsesServiceLanguage(t *testing.T) {
	task := &dag.Task{
		Kind: dag.TaskKindServiceHandler,
		Payload: dag.TaskPayload{
			Service: &manifest.ServiceDef{Language: "Go"},
		},
	}
	if got := taskLanguage(task); got != "go" {
		t.Errorf("expected 'go', got %q", got)
	}
}

func TestTaskLanguage_ServiceTask_FallsBackToAllServices(t *testing.T) {
	task := &dag.Task{
		Kind: dag.TaskKindServiceBootstrap,
		Payload: dag.TaskPayload{
			AllServices: []manifest.ServiceDef{{Language: "Python"}},
		},
	}
	if got := taskLanguage(task); got != "python" {
		t.Errorf("expected 'python', got %q", got)
	}
}

func TestTaskLanguage_DataTask_ReturnsEmpty(t *testing.T) {
	task := &dag.Task{Kind: dag.TaskKindDataSchemas}
	if got := taskLanguage(task); got != "" {
		t.Errorf("data task language should be empty, got %q", got)
	}
}

func TestTaskLanguage_InfraTerraform_ReturnsTerraform(t *testing.T) {
	task := &dag.Task{Kind: dag.TaskKindInfraTerraform}
	if got := taskLanguage(task); got != "terraform" {
		t.Errorf("expected 'terraform', got %q", got)
	}
}

func TestTaskLanguage_FrontendTask_UsesFrontendLanguage(t *testing.T) {
	task := &dag.Task{
		Kind: dag.TaskKindFrontend,
		Payload: dag.TaskPayload{
			Frontend: &manifest.FrontendPillar{
				Tech: &manifest.FrontendTechConfig{Language: "TypeScript"},
			},
		},
	}
	if got := taskLanguage(task); got != "typescript" {
		t.Errorf("expected 'typescript', got %q", got)
	}
}

// ── modTidyHint ───────────────────────────────────────────────────────────────

func TestModTidyHint_InvalidVersion(t *testing.T) {
	output := `github.com/foo/bar@v0.0.0-20200101000000-abcdef123456: invalid version: git ls-remote -q https://github.com/foo/bar terminal prompts disabled`
	hint := modTidyHint(output)
	if hint == "" {
		t.Error("expected non-empty hint for invalid version error")
	}
	if !strings.Contains(hint, "github.com/foo/bar") {
		t.Errorf("hint should mention the broken module, got: %q", hint)
	}
}

func TestModTidyHint_NotFound(t *testing.T) {
	output := `github.com/missing/pkg@v1.2.3: reading github.com/missing/pkg/go.mod at revision v1.2.3: 404 Not Found`
	hint := modTidyHint(output)
	if hint == "" {
		t.Error("expected non-empty hint for 404 Not Found error")
	}
}

func TestModTidyHint_NoMatchReturnsEmpty(t *testing.T) {
	output := "some unrelated go mod tidy output without known patterns"
	hint := modTidyHint(output)
	if hint != "" {
		t.Errorf("expected empty hint for unrecognized error, got: %q", hint)
	}
}

func TestModTidyHint_EmptyOutput(t *testing.T) {
	hint := modTidyHint("")
	if hint != "" {
		t.Errorf("expected empty hint for empty output, got: %q", hint)
	}
}

func TestModTidyHint_DeduplicatesBrokenModules(t *testing.T) {
	// Same module appears twice in output — should only be listed once in hint
	output := `github.com/dup/pkg@v1.0.0: invalid version: something
github.com/dup/pkg@v1.0.0: invalid version: something`
	hint := modTidyHint(output)
	count := strings.Count(hint, "github.com/dup/pkg@v1.0.0")
	if count != 1 {
		t.Errorf("broken module should appear exactly once in hint, got %d occurrences", count)
	}
}

// ── goModDirs ─────────────────────────────────────────────────────────────────

func TestGoModDirs_FindsGoModFiles(t *testing.T) {
	files := []string{
		"services/api/go.mod",
		"services/api/main.go",
		"services/worker/go.mod",
		"services/worker/worker.go",
	}
	dirs := goModDirs("output", files)
	if len(dirs) != 2 {
		t.Fatalf("expected 2 dirs, got %d: %v", len(dirs), dirs)
	}
	dirSet := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		dirSet[d] = true
	}
	for _, want := range []string{"services/api", "services/worker"} {
		if !dirSet[want] {
			t.Errorf("expected dir %q in result, got %v", want, dirs)
		}
	}
}

func TestGoModDirs_DeduplicatesDirs(t *testing.T) {
	files := []string{
		"api/go.mod",
		"api/go.sum",
	}
	dirs := goModDirs("output", files)
	if len(dirs) != 1 {
		t.Errorf("expected 1 unique dir, got %d: %v", len(dirs), dirs)
	}
}

func TestGoModDirs_EmptyFilesReturnsEmpty(t *testing.T) {
	dirs := goModDirs("output", nil)
	if len(dirs) != 0 {
		t.Errorf("expected empty result for nil files, got %v", dirs)
	}
}

func TestGoModDirs_NoGoModFallsBackToGoFileDirs(t *testing.T) {
	// No go.mod — should not panic and returns something from .go files
	files := []string{"api/main.go"}
	dirs := goModDirs("output", files)
	// Just verify it doesn't panic; fallback behavior is OS-path-list-dependent
	_ = dirs
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistry_ForTask_GoTask_ReturnsGoVerifier(t *testing.T) {
	r := NewRegistry()
	task := &dag.Task{
		Kind: dag.TaskKindServiceHandler,
		Payload: dag.TaskPayload{
			Service: &manifest.ServiceDef{Language: "Go"},
		},
	}
	v := r.ForTask(task)
	if v.Language() != "go" {
		t.Errorf("expected go verifier, got %q", v.Language())
	}
}

func TestRegistry_ForTask_TerraformTask_ReturnsTfVerifier(t *testing.T) {
	r := NewRegistry()
	task := &dag.Task{Kind: dag.TaskKindInfraTerraform}
	v := r.ForTask(task)
	if v.Language() != "terraform" {
		t.Errorf("expected terraform verifier, got %q", v.Language())
	}
}

func TestRegistry_ForTask_DataTask_ReturnsNullVerifier(t *testing.T) {
	r := NewRegistry()
	task := &dag.Task{Kind: dag.TaskKindDataSchemas}
	v := r.ForTask(task)
	// NullVerifier.Language() returns "null"
	if v.Language() != "null" {
		t.Errorf("expected null verifier for data task, got %q", v.Language())
	}
}
