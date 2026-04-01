package verify

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// TsVerifier runs `tsc --noEmit` on generated TypeScript projects.
// Degrades gracefully if `tsc` is not installed.
type TsVerifier struct{}

func NewTsVerifier() *TsVerifier { return &TsVerifier{} }

func (t *TsVerifier) Language() string { return "typescript" }

func (t *TsVerifier) Verify(ctx context.Context, outputDir string, files []string) (*Result, error) {
	tscPath, err := exec.LookPath("tsc")
	if err != nil {
		return &Result{Passed: true, Output: "tsc not found in PATH — skipping TypeScript verification"}, nil
	}

	// Find directories with tsconfig.json.
	dirs := tsConfigDirs(outputDir, files)
	if len(dirs) == 0 {
		return &Result{Passed: true, Output: "no tsconfig.json found — skipping TypeScript verification"}, nil
	}

	var combined bytes.Buffer
	allPassed := true

	for _, dir := range dirs {
		absDir := filepath.Join(outputDir, dir)
		out, err := runCmd(ctx, absDir, tscPath, "--noEmit")
		combined.WriteString(fmt.Sprintf("=== tsc --noEmit in %s ===\n%s\n", dir, out))
		if err != nil {
			allPassed = false
		}
	}

	return &Result{Passed: allPassed, Output: combined.String()}, nil
}

func tsConfigDirs(outputDir string, files []string) []string {
	seen := make(map[string]bool)
	dirs := []string{}
	for _, f := range files {
		if filepath.Base(f) == "tsconfig.json" {
			dir := filepath.Dir(f)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs
}
