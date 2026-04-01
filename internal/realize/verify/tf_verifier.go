package verify

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// TfVerifier runs `terraform validate` on generated Terraform files.
// Degrades gracefully if `terraform` is not installed.
type TfVerifier struct{}

func NewTfVerifier() *TfVerifier { return &TfVerifier{} }

func (t *TfVerifier) Language() string { return "terraform" }

func (t *TfVerifier) Verify(ctx context.Context, outputDir string, files []string) (*Result, error) {
	tfPath, err := exec.LookPath("terraform")
	if err != nil {
		return &Result{Passed: true, Output: "terraform not found in PATH — skipping Terraform validation"}, nil
	}

	// Find directories with .tf files.
	dirs := tfDirs(outputDir, files)
	if len(dirs) == 0 {
		return &Result{Passed: true, Output: "no Terraform files found"}, nil
	}

	var combined bytes.Buffer
	allPassed := true

	for _, dir := range dirs {
		absDir := filepath.Join(outputDir, dir)

		// terraform init is required before validate.
		initOut, _ := runCmd(ctx, absDir, tfPath, "init", "-backend=false", "-input=false")
		combined.WriteString(fmt.Sprintf("=== terraform init in %s ===\n%s\n", dir, initOut))

		valOut, err := runCmd(ctx, absDir, tfPath, "validate")
		combined.WriteString(fmt.Sprintf("=== terraform validate in %s ===\n%s\n", dir, valOut))
		if err != nil {
			allPassed = false
		}
	}

	return &Result{Passed: allPassed, Output: combined.String()}, nil
}

func tfDirs(outputDir string, files []string) []string {
	seen := make(map[string]bool)
	dirs := []string{}
	for _, f := range files {
		if filepath.Ext(f) == ".tf" {
			dir := filepath.Dir(f)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs
}
