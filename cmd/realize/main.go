package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/vibe-menu/internal/bundled"
	"github.com/vibe-menu/internal/realize/orchestrator"
	"github.com/vibe-menu/internal/realize/skills"
)

func main() {
	manifestPath := flag.String("manifest", "manifest.json", "path to manifest.json")
	outputDir := flag.String("output", "output", "directory for generated code")
	skillsDir := flag.String("skills", ".vibemenu/skills", "directory for skill markdown files")
	maxRetries := flag.Int("retries", 3, "max verification retry attempts per task")
	parallelism := flag.Int("parallel", 0, "max concurrent tasks (0 = num CPUs)")
	dryRun := flag.Bool("dry-run", false, "print task plan without running agents")
	verbose := flag.Bool("verbose", false, "print token usage and thinking logs")
	provider := flag.String("provider", "", "default LLM provider: Claude, Gemini, ChatGPT, Mistral, Llama")
	apiKey := flag.String("api-key", "", "API key for the default provider (falls back to env var)")
	flag.Parse()

	// Auto-extract bundled skills when the target directory is absent.
	if _, err := os.Stat(*skillsDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "realize: extracting bundled skills to %s\n", *skillsDir)
		if extractErr := skills.Extract(*skillsDir, bundled.SkillsFS, "skills"); extractErr != nil {
			fmt.Fprintf(os.Stderr, "realize: warning: could not extract bundled skills: %v\n", extractErr)
		}
	}

	p := *parallelism
	if p <= 0 {
		p = 1
	}

	cfg := orchestrator.Config{
		ManifestPath: *manifestPath,
		OutputDir:    *outputDir,
		SkillsDir:    *skillsDir,
		MaxRetries:   *maxRetries,
		Parallelism:  p,
		DryRun:       *dryRun,
		Verbose:      *verbose,
		Provider:     *provider,
		APIKey:       *apiKey,
	}

	if err := orchestrator.New(cfg).Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "realize: %v\n", err)
		os.Exit(1)
	}
}
