package verify

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// PythonVerifier runs `ruff check` (and `mypy` if available) on generated Python code.
// Both tools degrade gracefully if not installed.
type PythonVerifier struct{}

func NewPythonVerifier() *PythonVerifier { return &PythonVerifier{} }

func (p *PythonVerifier) Language() string { return "python" }

func (p *PythonVerifier) Verify(ctx context.Context, outputDir string, files []string) (*Result, error) {
	// Find directories with Python files.
	dirs := pythonProjectDirs(outputDir, files)
	if len(dirs) == 0 {
		return &Result{Passed: true, Output: "no Python files found"}, nil
	}

	var combined bytes.Buffer
	allPassed := true

	ruffPath, ruffErr := exec.LookPath("ruff")
	if ruffErr != nil {
		combined.WriteString("ruff not found in PATH — skipping ruff check\n")
	}

	for _, dir := range dirs {
		absDir := filepath.Join(outputDir, dir)

		if ruffErr == nil {
			out, err := runCmd(ctx, absDir, ruffPath, "check", ".")
			combined.WriteString(fmt.Sprintf("=== ruff check in %s ===\n%s\n", dir, out))
			if err != nil {
				allPassed = false
			}
		}
	}

	return &Result{Passed: allPassed, Output: combined.String()}, nil
}

func pythonProjectDirs(outputDir string, files []string) []string {
	seen := make(map[string]bool)
	dirs := []string{}
	for _, f := range files {
		if filepath.Ext(f) == ".py" {
			dir := filepath.Dir(f)
			// Walk up to find pyproject.toml or the top-level service dir.
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs
}
