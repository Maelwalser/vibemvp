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
	"Go":    "golang-patterns",
	"Fiber": "go-fiber",
	"Gin":   "go-gin",
	"Echo":  "go-echo",
	"Chi":   "go-chi",
	// TypeScript/Node frameworks
	"TypeScript": "coding-standards",
	"JavaScript": "coding-standards",
	"Express":    "typescript-express",
	"Fastify":    "typescript-fastify",
	"NestJS":     "typescript-nestjs",
	"Hono":       "typescript-hono",
	// Python frameworks
	"Python":  "python-patterns",
	"FastAPI": "python-fastapi",
	"Django":  "python-django",
	"Flask":   "python-flask",
	// Java / Spring Boot
	"Java":        "java-coding-standards",
	"Spring Boot": "java-spring-boot",
	"JPA":         "jpa-patterns",
	// Kotlin / Android / KMP
	"Kotlin":               "kotlin-patterns",
	"Ktor":                 "kotlin-ktor",
	"Android":              "android-clean-architecture",
	"Kotlin Multiplatform": "compose-multiplatform",
	// Swift / iOS
	"Swift":   "swiftui",
	"SwiftUI": "swiftui",
	"iOS":     "swiftui",
	// Native
	"Perl": "perl-patterns",
	"C++":  "cpp-coding-standards",
	"C":    "cpp-coding-standards",
	// Rust
	"Axum":      "rust-axum",
	"Actix-web": "rust-actix",
	// Frontend frameworks
	"React":        "frontend-patterns",
	"Next.js":      "react-nextjs",
	"Vue":          "frontend-patterns",
	"Nuxt.js":      "vue-nuxt",
	"Svelte":       "frontend-patterns",
	"SvelteKit":    "svelte-kit",
	"Angular":      "frontend-patterns",
	"Flutter":      "flutter",
	"React Native": "react-native",
	// Databases
	"PostgreSQL": "postgres-patterns",
	"Postgres":   "postgres-patterns",
	"MySQL":      "mysql",
	"MongoDB":    "mongodb",
	"Redis":      "redis",
	"DynamoDB":   "dynamodb",
	"SQLite":     "sqlite",
	// Message brokers
	"Kafka":       "kafka",
	"RabbitMQ":    "rabbitmq",
	"NATS":        "nats",
	"AWS SQS/SNS": "aws-sqs-sns",
	// Styling
	"Tailwind CSS": "tailwind",
	"Tailwind":     "tailwind",
	// Infrastructure
	"Docker":          "docker-patterns",
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

// universalSkillsForKind lists skill keys always injected for these task kinds,
// regardless of which specific technologies appear in the manifest payload.
var universalSkillsForKind = map[dag.TaskKind][]string{
	// Backend service: quality patterns + security (language-specific skills come via aliasMap)
	dag.TaskKindService: {
		"backend-patterns", "security-review", "coding-standards",
	},
	// Auth: always security-first
	dag.TaskKindAuth: {
		"security-review", "coding-standards", "security-scan",
	},
	// Data schemas: migrations guide
	dag.TaskKindDataSchemas: {
		"database-migrations",
	},
	// Migrations: migrations guide + postgres patterns (covers SQL DDL for any SQL DB)
	dag.TaskKindDataMigrations: {
		"database-migrations", "postgres-patterns",
	},
	// Messaging: backend patterns for broker wiring
	dag.TaskKindMessaging: {
		"backend-patterns", "coding-standards",
	},
	// Gateway: security + backend patterns
	dag.TaskKindGateway: {
		"backend-patterns", "security-review", "api-design",
	},
	// Contracts: API design is the primary skill here
	dag.TaskKindContracts: {
		"api-design", "coding-standards",
	},
	// Frontend: frontend patterns + coding standards
	dag.TaskKindFrontend: {
		"frontend-patterns", "coding-standards",
	},
	// Docker: docker patterns + deployment
	dag.TaskKindInfraDocker: {
		"docker-patterns", "deployment-patterns",
	},
	// Terraform/IaC: deployment patterns
	dag.TaskKindInfraTerraform: {
		"deployment-patterns",
	},
	// CI/CD: deployment + verification loop
	dag.TaskKindInfraCI: {
		"deployment-patterns", "verification-loop",
	},
	// Testing: full TDD suite + e2e + verification
	dag.TaskKindCrossCutTesting: {
		"tdd-workflow", "e2e-testing", "coding-standards", "verification-loop",
	},
	// Docs: API design standards
	dag.TaskKindCrossCutDocs: {
		"api-design", "coding-standards",
	},
}

// LookupAll returns all skill docs relevant to a task kind and technology list.
// Technology-specific skills are looked up first; universal quality skills for the
// task kind are appended afterwards (deduplication ensures no double-injection).
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

	// Inject universal quality skills for this task kind.
	for _, key := range universalSkillsForKind[kind] {
		if seen[key] {
			continue
		}
		seen[key] = true
		content, ok := r.index[key]
		if !ok {
			continue
		}
		docs = append(docs, Doc{Technology: key, Content: content})
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
