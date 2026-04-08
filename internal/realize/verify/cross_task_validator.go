package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ConsistencyError describes a semantic mismatch between generated files that
// the compiler cannot catch (e.g. wrong DB name in docker-compose vs backend code).
type ConsistencyError struct {
	Category string // "db_name", "env_var", "port", "migration"
	Message  string
	File1    string // first file involved
	File2    string // second file involved
}

// ValidateCrossTaskConsistency scans the output directory for semantic mismatches
// between docker-compose.yml and backend application code. Returns nil when
// everything is consistent.
func ValidateCrossTaskConsistency(outputDir string) []ConsistencyError {
	var issues []ConsistencyError

	// Find docker-compose file.
	composePath := findComposeFile(outputDir)
	if composePath == "" {
		return nil
	}
	composeContent, err := os.ReadFile(composePath)
	if err != nil {
		return nil
	}
	composeStr := string(composeContent)
	composeRel, _ := filepath.Rel(outputDir, composePath)

	// Extract POSTGRES_DB from docker-compose.
	postgresDB := extractYAMLEnvValue(composeStr, "POSTGRES_DB")

	// Extract DATABASE_URL from docker-compose backend service.
	composeDBURL := extractYAMLEnvValue(composeStr, "DATABASE_URL")

	// Scan Go files for os.Getenv calls and DATABASE_URL references.
	goEnvVars := make(map[string]string) // var name → file where used
	var goFiles []string
	_ = filepath.Walk(outputDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			if info != nil && info.IsDir() {
				name := info.Name()
				if name == ".tmp" || name == "node_modules" || name == ".next" || name == "vendor" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})

	getenvRe := regexp.MustCompile(`os\.Getenv\("([^"]+)"\)`)
	for _, goFile := range goFiles {
		content, err := os.ReadFile(goFile)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(outputDir, goFile)
		for _, match := range getenvRe.FindAllStringSubmatch(string(content), -1) {
			goEnvVars[match[1]] = rel
		}
	}

	// Check 1: DB name consistency.
	if postgresDB != "" && composeDBURL != "" {
		dbNameFromURL := extractDBNameFromURL(composeDBURL)
		if dbNameFromURL != "" && dbNameFromURL != postgresDB {
			issues = append(issues, ConsistencyError{
				Category: "db_name",
				Message:  fmt.Sprintf("POSTGRES_DB=%q but DATABASE_URL references database %q", postgresDB, dbNameFromURL),
				File1:    composeRel,
				File2:    composeRel,
			})
		}
	}

	// Check 2: Backend code expects DATABASE_URL but docker-compose doesn't set it.
	if _, ok := goEnvVars["DATABASE_URL"]; ok && composeDBURL == "" {
		issues = append(issues, ConsistencyError{
			Category: "env_var",
			Message:  "backend code reads DATABASE_URL but docker-compose does not set it for the backend service",
			File1:    goEnvVars["DATABASE_URL"],
			File2:    composeRel,
		})
	}

	// Check 3: Required env vars missing from docker-compose.
	criticalVars := []string{"JWT_SECRET", "DATABASE_URL"}
	for _, v := range criticalVars {
		goFile, usedInCode := goEnvVars[v]
		if usedInCode && !strings.Contains(composeStr, v) {
			issues = append(issues, ConsistencyError{
				Category: "env_var",
				Message:  fmt.Sprintf("backend code reads %s (in %s) but docker-compose does not define it", v, goFile),
				File1:    goFile,
				File2:    composeRel,
			})
		}
	}

	// Check 4: Postgres healthcheck should verify the specific database exists.
	// pg_isready without -d only checks that postgres is accepting connections,
	// NOT that the target database exists — leading to backends that crash with
	// "database does not exist" even after the healthcheck passes.
	if postgresDB != "" && strings.Contains(composeStr, "pg_isready") {
		if !strings.Contains(composeStr, "-d") {
			issues = append(issues, ConsistencyError{
				Category: "healthcheck",
				Message:  fmt.Sprintf("pg_isready healthcheck does not verify database %q exists (missing -d flag) — backend may crash with 'database does not exist' on stale volumes", postgresDB),
				File1:    composeRel,
				File2:    composeRel,
			})
		}
	}

	// Check 5: Init script idempotency.
	// docker-entrypoint-initdb.d scripts only run on first volume init. If the
	// init script uses .sql (not .sh), it can't use $POSTGRES_DB to create the
	// database idempotently. Check for non-idempotent patterns.
	if strings.Contains(composeStr, "docker-entrypoint-initdb.d") {
		initSQLPath := filepath.Join(outputDir, "scripts", "init-db.sql")
		if content, err := os.ReadFile(initSQLPath); err == nil {
			contentStr := string(content)
			// SQL init scripts that use CREATE TABLE without IF NOT EXISTS are fragile.
			if strings.Contains(contentStr, "CREATE TABLE") && !strings.Contains(contentStr, "IF NOT EXISTS") {
				issues = append(issues, ConsistencyError{
					Category: "idempotency",
					Message:  "scripts/init-db.sql uses CREATE TABLE without IF NOT EXISTS — will fail on container restart if volume persists",
					File1:    "scripts/init-db.sql",
					File2:    composeRel,
				})
			}
		}
	}

	// Check 6: Migration path conflict.
	hasMigrationDir := false
	_ = filepath.Walk(outputDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() && info.Name() == "migrations" {
			parent := filepath.Base(filepath.Dir(path))
			if parent == "db" {
				hasMigrationDir = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	if hasMigrationDir && strings.Contains(composeStr, "docker-entrypoint-initdb.d") {
		issues = append(issues, ConsistencyError{
			Category: "migration",
			Message:  "db/migrations/ directory exists but docker-compose mounts init-db.sql via docker-entrypoint-initdb.d — this creates a dual-path conflict; backend should run migrations at startup instead",
			File1:    "db/migrations/",
			File2:    composeRel,
		})
	}

	return issues
}

// findComposeFile returns the path to docker-compose.yml or docker-compose.yaml.
func findComposeFile(outputDir string) string {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml"} {
		p := filepath.Join(outputDir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// extractYAMLEnvValue extracts the value of a YAML environment variable by name
// using simple string matching. Handles both "KEY: value" and "KEY=value" forms.
func extractYAMLEnvValue(yamlContent, key string) string {
	// Match "KEY: value" form.
	re1 := regexp.MustCompile(key + `:\s*(.+)`)
	if m := re1.FindStringSubmatch(yamlContent); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	// Match "KEY=value" form (in environment arrays).
	re2 := regexp.MustCompile(key + `=(.+)`)
	if m := re2.FindStringSubmatch(yamlContent); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// extractDBNameFromURL extracts the database name from a PostgreSQL connection URL.
// E.g. "postgres://user:pass@host:5432/mydb?sslmode=disable" → "mydb"
func extractDBNameFromURL(url string) string {
	// Strip query params.
	if idx := strings.Index(url, "?"); idx >= 0 {
		url = url[:idx]
	}
	// Find last slash — DB name follows it.
	if idx := strings.LastIndex(url, "/"); idx >= 0 && idx < len(url)-1 {
		return url[idx+1:]
	}
	return ""
}
