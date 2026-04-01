package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibe-mvp/internal/realize/dag"
)

// aliasMap normalizes manifest technology strings to skill file base names.
var aliasMap = map[string]string{
	// Go frameworks
	"Fiber":   "go-fiber",
	"Gin":     "go-gin",
	"Echo":    "go-echo",
	"Chi":     "go-chi",
	// TypeScript/Node frameworks
	"Express":  "typescript-express",
	"Fastify":  "typescript-fastify",
	"NestJS":   "typescript-nestjs",
	"Hono":     "typescript-hono",
	// Python frameworks
	"FastAPI":  "python-fastapi",
	"Django":   "python-django",
	"Flask":    "python-flask",
	// Java / Kotlin
	"Spring Boot": "java-spring-boot",
	"Ktor":        "kotlin-ktor",
	// Rust
	"Axum":       "rust-axum",
	"Actix-web":  "rust-actix",
	// Frontend frameworks
	"React":        "react",
	"Next.js":      "react-nextjs",
	"Vue":          "vue",
	"Nuxt.js":      "vue-nuxt",
	"Svelte":       "svelte",
	"SvelteKit":    "svelte-kit",
	"Angular":      "angular",
	"Flutter":      "flutter",
	"React Native": "react-native",
	// Databases
	"PostgreSQL": "postgresql",
	"MySQL":      "mysql",
	"MongoDB":    "mongodb",
	"Redis":      "redis",
	"DynamoDB":   "dynamodb",
	"SQLite":     "sqlite",
	// Message brokers
	"Kafka":      "kafka",
	"RabbitMQ":   "rabbitmq",
	"NATS":       "nats",
	"AWS SQS/SNS": "aws-sqs-sns",
	// Styling
	"Tailwind CSS": "tailwind",
	"Tailwind":     "tailwind",
	// Infrastructure
	"Terraform":       "terraform",
	"Terraform (AWS)": "terraform-aws",
	"Pulumi":          "pulumi",
	"GitHub Actions":  "github-actions",
	"GitLab CI":       "gitlab-ci",
	// Auth
	"Auth0":   "auth0",
	"Clerk":   "clerk",
	"Cognito": "aws-cognito",
}

// FileRegistry implements Registry by reading skill markdown files from a directory.
type FileRegistry struct {
	skillsDir string
	index     map[string]string // normalized key → content
}

// Load reads all *.md files from skillsDir and returns a FileRegistry.
// If skillsDir does not exist, an empty registry is returned without error.
func Load(skillsDir string) (*FileRegistry, error) {
	r := &FileRegistry{
		skillsDir: skillsDir,
		index:     make(map[string]string),
	}

	entries, err := os.ReadDir(skillsDir)
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read skills dir %s: %w", skillsDir, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		key := strings.TrimSuffix(e.Name(), ".md")
		data, err := os.ReadFile(filepath.Join(skillsDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read skill file %s: %w", e.Name(), err)
		}
		r.index[key] = string(data)
	}
	return r, nil
}

// Lookup returns the content for the given technology string, or ("", false).
func (r *FileRegistry) Lookup(technology string) (string, bool) {
	key := normalize(technology)
	if content, ok := r.index[key]; ok {
		return content, true
	}
	return "", false
}

// LookupAll returns all skill docs relevant to a task kind and technology list.
func (r *FileRegistry) LookupAll(kind dag.TaskKind, technologies []string) []Doc {
	seen := make(map[string]bool)
	docs := make([]Doc, 0)

	for _, tech := range technologies {
		if tech == "" {
			continue
		}
		key := normalize(tech)
		if seen[key] {
			continue
		}
		seen[key] = true
		content, ok := r.index[key]
		if !ok {
			continue
		}
		docs = append(docs, Doc{Technology: tech, Content: content})
	}
	return docs
}

// normalize maps a technology name to its skill file base name.
func normalize(tech string) string {
	if alias, ok := aliasMap[tech]; ok {
		return alias
	}
	// Fallback: lowercase with spaces replaced by hyphens.
	return strings.ToLower(strings.ReplaceAll(tech, " ", "-"))
}
