package verify

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ── Go import fixer ───────────────────────────────────────────────────────────
//
// fixGoImports runs goimports on all .go files in the directory.
// goimports adds missing stdlib/module imports and removes unused ones.
// Falls back to removeUnusedGoImports when goimports is not installed.
func fixGoImports(dir string, files []string) string {
	// Check if goimports is available.
	goimportsPath, err := exec.LookPath("goimports")
	if err != nil {
		// goimports not installed — fall back to the simpler unused-import remover.
		return removeUnusedGoImports(dir, files)
	}

	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		before, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Run goimports in the file's own directory so it can resolve module imports.
		cmd := exec.Command(goimportsPath, "-w", path)
		cmd.Dir = dir
		_ = cmd.Run()
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("goimports fixed imports in %d file(s)", fixed)
}

// removeUnusedGoImports is a fallback for when goimports is not installed.
// It parses "go build" errors for "imported and not used" lines and removes
// the offending import from the named file. This handles the most common
// case where the LLM imports a package it doesn't actually use.
func removeUnusedGoImports(dir string, files []string) string {
	if len(files) == 0 {
		return ""
	}

	// Find the go.mod root so we can run go build in the right module directory.
	// We need the nearest go.mod directory, not dir itself.
	goModDir := findGoModDir(dir, files)
	if goModDir == "" {
		return ""
	}

	// Run go build to find unused import errors.
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = goModDir
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		return ""
	}

	// Parse "imported and not used" errors.
	// Format: <file>:<line>:<col>: "<pkg>" imported and not used
	unusedImportRe := regexp.MustCompile(`^(.+\.go):(\d+):\d+: "([^"]+)" imported and not used`)
	type fix struct {
		file string
		line int
		pkg  string
	}
	var fixes []fix
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		m := unusedImportRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		relFile, lineStr, pkg := m[1], m[2], m[3]
		key := relFile + ":" + lineStr
		if seen[key] {
			continue
		}
		seen[key] = true
		lineNum := 0
		fmt.Sscanf(lineStr, "%d", &lineNum)
		fixes = append(fixes, fix{file: relFile, line: lineNum, pkg: pkg})
	}
	if len(fixes) == 0 {
		return ""
	}

	// Group by file and remove the import lines.
	byFile := make(map[string][]fix)
	for _, fx := range fixes {
		byFile[fx.file] = append(byFile[fx.file], fx)
	}

	applied := 0
	for relFile, fileFixes := range byFile {
		// Try both the path as reported by go build and relative to goModDir.
		path := relFile
		if !filepath.IsAbs(path) {
			path = filepath.Join(goModDir, relFile)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for _, fx := range fileFixes {
			if fx.line < 1 || fx.line > len(lines) {
				continue
			}
			lineIdx := fx.line - 1
			trimmed := strings.TrimSpace(lines[lineIdx])
			// Match the import line: \t"pkg" or \t_ "pkg" or \talias "pkg"
			if strings.Contains(trimmed, `"`+fx.pkg+`"`) {
				lines[lineIdx] = ""
				changed = true
			}
		}
		if !changed {
			continue
		}
		_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		applied++
	}
	if applied == 0 {
		return ""
	}
	return fmt.Sprintf("removed unused imports in %d file(s) (goimports fallback)", applied)
}

// findGoModDir returns the directory containing go.mod that is a parent of
// the given dir, walking up the tree. Returns "" when no go.mod is found.
func findGoModDir(dir string, files []string) string {
	// First check if any of the listed files are under a go.mod directory.
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		abs := filepath.Join(dir, f)
		current := filepath.Dir(abs)
		for {
			if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
				return current
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}
	// Fall back to walking up from dir itself.
	current := dir
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

// ── Placeholder import path fix ──────────────────────────────────────────────
//
// LLMs sometimes use placeholder organisation names in import paths
// (e.g. "github.com/your-org/app-name/internal/domain") instead of the actual
// Go module path declared in go.mod. This function reads go.mod from the temp
// directory, extracts the module path, and rewrites any placeholder-org imports
// to use the correct path — without requiring an LLM retry.

// placeholderOrgRe matches quoted import paths whose second path component is a
// known placeholder organisation name, e.g. "github.com/your-org/repo/internal/...".
var placeholderOrgRe = regexp.MustCompile(
	`"github\.com/(?:your-org|your-company|yourcompany|your_company|` +
		`mycompany|my-company|example-org|my-org|acme|acme-corp|` +
		`your-app|myapp|company|org-name|team-name)/[^"]+"`,
)

func fixPlaceholderImportPaths(dir string, files []string) string {
	modulePath := extractModulePathFromDir(dir)
	if modulePath == "" {
		return ""
	}

	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		newContent := placeholderOrgRe.ReplaceAllStringFunc(content, func(match string) string {
			// match is e.g. `"github.com/your-org/auth-api/internal/domain"`
			inner := match[1 : len(match)-1] // strip surrounding quotes
			// Split: ["github.com", "your-org", "auth-api", "internal", "domain", ...]
			parts := strings.SplitN(inner, "/", 4)
			if len(parts) < 3 {
				return match // unexpected shape — leave unchanged
			}
			if len(parts) == 4 {
				// Has sub-path: replace org+repo with modulePath, keep sub-path.
				return `"` + modulePath + "/" + parts[3] + `"`
			}
			// Import is the module root itself (e.g. "github.com/your-org/auth-api").
			return `"` + modulePath + `"`
		})
		if newContent != content {
			_ = os.WriteFile(path, []byte(newContent), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("rewrote placeholder import paths in %d file(s) (module: %s)", fixed, modulePath)
}

// extractModulePathFromDir reads the module path from the go.mod file located
// in dir or any of its parent directories, returning "" when none is found.
func extractModulePathFromDir(dir string) string {
	current := dir
	for {
		data, err := os.ReadFile(filepath.Join(current, "go.mod"))
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module "))
				}
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

// ── Escape sequence fix ──────────────────────────────────────────────────────

func fixGoEscapeSequences(dir string, files []string) string {
	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		result := rewriteInvalidEscapes(content)
		if result != content {
			_ = os.WriteFile(path, []byte(result), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed escape sequences in %d file(s)", fixed)
}

func rewriteInvalidEscapes(src string) string {
	var out strings.Builder
	i := 0
	for i < len(src) {
		// Skip raw strings.
		if src[i] == '`' {
			end := strings.IndexByte(src[i+1:], '`')
			if end >= 0 {
				out.WriteString(src[i : i+end+2])
				i += end + 2
			} else {
				out.WriteByte(src[i])
				i++
			}
			continue
		}
		// Skip // comments.
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			end := strings.IndexByte(src[i:], '\n')
			if end >= 0 {
				out.WriteString(src[i : i+end])
				i += end
			} else {
				out.WriteString(src[i:])
				i = len(src)
			}
			continue
		}
		// Skip /* */ comments.
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			end := strings.Index(src[i+2:], "*/")
			if end >= 0 {
				out.WriteString(src[i : i+end+4])
				i += end + 4
			} else {
				out.WriteString(src[i:])
				i = len(src)
			}
			continue
		}
		// Process double-quoted strings.
		if src[i] == '"' {
			strEnd := findQuoteEnd(src, i+1)
			if strEnd < 0 {
				out.WriteByte(src[i])
				i++
				continue
			}
			inner := src[i+1 : strEnd]
			if hasInvalidGoEscape(inner) && !strings.Contains(inner, "`") && !strings.Contains(inner, "\n") {
				rawInner := interpretedToRaw(inner)
				out.WriteByte('`')
				out.WriteString(rawInner)
				out.WriteByte('`')
			} else {
				out.WriteString(src[i : strEnd+1])
			}
			i = strEnd + 1
			continue
		}
		out.WriteByte(src[i])
		i++
	}
	return out.String()
}

func findQuoteEnd(s string, start int) int {
	escaped := false
	for i := start; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		if s[i] == '\\' {
			escaped = true
			continue
		}
		if s[i] == '"' {
			return i
		}
		if s[i] == '\n' {
			return -1
		}
	}
	return -1
}

func hasInvalidGoEscape(inner string) bool {
	for i := 0; i < len(inner)-1; i++ {
		if inner[i] == '\\' {
			next := inner[i+1]
			switch next {
			case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"', '\'',
				'0', '1', '2', '3', '4', '5', '6', '7',
				'x', 'u', 'U':
				// valid escape sequence
			default:
				return true
			}
			i++
		}
	}
	return false
}

func interpretedToRaw(inner string) string {
	var out strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			next := inner[i+1]
			switch next {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case '\\':
				out.WriteByte('\\')
			case '"':
				out.WriteByte('"')
			default:
				out.WriteByte('\\')
				out.WriteByte(next)
			}
			i++
		} else {
			out.WriteByte(inner[i])
		}
	}
	return out.String()
}

// ── Duplicate type fix ───────────────────────────────────────────────────────

func fixDuplicateTypes(dir string, files []string) string {
	byDir := make(map[string][]string)
	for _, f := range files {
		if filepath.Ext(f) != ".go" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		byDir[filepath.Dir(f)] = append(byDir[filepath.Dir(f)], f)
	}
	fixed := 0
	for _, goFiles := range byDir {
		if len(goFiles) >= 2 && removeDuplicateDecls(dir, goFiles) {
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("fixed duplicate types in %d package(s)", fixed)
}

func removeDuplicateDecls(baseDir string, files []string) bool {
	re := regexp.MustCompile(`(?m)^type\s+(\w+)\s+`)

	typesByFile := make(map[string][]string)
	allTypes := make(map[string][]string)

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(baseDir, f))
		if err != nil {
			continue
		}
		var types []string
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			types = append(types, m[1])
			allTypes[m[1]] = append(allTypes[m[1]], f)
		}
		typesByFile[f] = types
	}

	// Find duplicated type names.
	duplicates := make(map[string][]string)
	for name, declFiles := range allTypes {
		if len(declFiles) > 1 {
			duplicates[name] = declFiles
		}
	}
	if len(duplicates) == 0 {
		return false
	}

	// Keep each type in the file with the most declarations; remove from others.
	filesToFix := make(map[string]map[string]bool)
	for typeName, declFiles := range duplicates {
		bestFile, bestCount := declFiles[0], len(typesByFile[declFiles[0]])
		for _, f := range declFiles[1:] {
			if len(typesByFile[f]) > bestCount {
				bestFile, bestCount = f, len(typesByFile[f])
			}
		}
		for _, f := range declFiles {
			if f != bestFile {
				if filesToFix[f] == nil {
					filesToFix[f] = make(map[string]bool)
				}
				filesToFix[f][typeName] = true
			}
		}
	}

	for f, typesToRemove := range filesToFix {
		data, err := os.ReadFile(filepath.Join(baseDir, f))
		if err != nil {
			continue
		}
		content := string(data)
		for typeName := range typesToRemove {
			typeRe := regexp.MustCompile(
				fmt.Sprintf(`(?ms)^type %s\s+(?:struct|interface)\s*\{[^}]*\}\s*\n?`,
					regexp.QuoteMeta(typeName)))
			content = typeRe.ReplaceAllString(content, "")
		}
		_ = os.WriteFile(filepath.Join(baseDir, f), []byte(content), 0644)
	}
	return true
}

// ── Misplaced import fix ──────────────────────────────────────────────────────
//
// Go does not allow import statements inside function bodies. LLMs sometimes
// generate `import (...)` blocks inside functions (often as commentary stubs).
// These cause "syntax error: unexpected keyword import" compile errors.
// This fix detects indented import blocks inside function bodies and removes them.

var indentedImportRe = regexp.MustCompile(`(?m)^[ \t]+import\s*(?:\([\s\S]*?\)|\".+?\")[ \t]*\n?`)

func fixMisplacedImports(dir string, files []string) string {
	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		newContent := indentedImportRe.ReplaceAll(data, nil)
		if len(newContent) != len(data) {
			_ = os.WriteFile(path, newContent, 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("removed misplaced import statement(s) in %d file(s)", fixed)
}

// ── Orphaned interface fragment fix ──────────────────────────────────────────
//
// When an LLM response is truncated mid-way through a type declaration, the file
// can contain a top-level line that starts with ", " — the tail of a return-type
// expression whose opening was cut off. For example:
//
//   // PgxPool is the interface for pgxpool.Pool operations.
//   , error)                             ← truncated: "type PgxPool interface {\n\tExec(..., error)" was lost
//       Query(ctx context.Context, ...) (pgx.Rows, error)
//       Close()
//   }
//
// Go rejects `, error)` at package scope with "non-declaration statement outside
// function body". This fix detects the pattern and reconstructs the missing
// `type X interface {` declaration so the file at least parses. The resulting
// interface will be incomplete (missing the first method), but syntactically valid
// — gofmt accepts it and the LLM retry can fill in the rest.

// fixOrphanedInterfaceFragments scans Go files for top-level orphaned return
// fragments and reconstructs the missing interface declaration header.
func fixOrphanedInterfaceFragments(dir string, files []string) string {
	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		patched, changed := repairOrphanedFragments(string(data))
		if changed {
			_ = os.WriteFile(path, []byte(patched), 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("repaired truncated interface declaration(s) in %d file(s)", fixed)
}

// repairOrphanedFragments is the pure-function core of fixOrphanedInterfaceFragments.
// It scans src for lines at brace depth 0 that start with ", " (orphaned return
// fragments), extracts the interface name from the preceding doc comment, removes
// the fragment, and inserts "type <Name> interface {" in its place.
func repairOrphanedFragments(src string) (string, bool) {
	lines := strings.Split(src, "\n")
	depth := 0
	changed := false

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		// Track brace depth using naive counting.
		// This is safe here: we only act on depth==0 lines with the ", " prefix,
		// which cannot appear in valid Go at package scope.
		depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")

		// Only act on lines at package scope (depth 0) that look like an orphaned
		// return-type tail: starts with ", " followed by a non-space character.
		if depth != 0 {
			continue
		}
		if len(trimmed) < 2 || trimmed[0] != ',' || trimmed[1] != ' ' {
			continue
		}

		// Found an orphan. Determine the interface name from the preceding comment.
		name := extractInterfaceNameFromComment(lines, i)

		// Remove the orphaned line.
		lines = append(lines[:i], lines[i+1:]...)
		changed = true

		// Insert "type <Name> interface {" at position i (before the remaining body).
		header := "type " + name + " interface {"
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:i]...)
		newLines = append(newLines, header)
		newLines = append(newLines, lines[i:]...)
		lines = newLines

		// The inserted "{" increases depth to 1. Skip past the inserted header.
		depth = 1
		i++
	}

	if !changed {
		return src, false
	}
	return strings.Join(lines, "\n"), true
}

// extractInterfaceNameFromComment scans backwards from lineIdx looking for the
// nearest doc comment whose first word is an exported identifier (uppercase).
// Returns "UnknownInterface" when no qualifying comment is found.
func extractInterfaceNameFromComment(lines []string, lineIdx int) string {
	for j := lineIdx - 1; j >= 0; j-- {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "//") {
			break // hit a non-comment line — stop
		}
		comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
		if comment == "" {
			continue
		}
		word := strings.Fields(comment)[0]
		// Must be an exported identifier: starts with an uppercase ASCII letter.
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
			return word
		}
	}
	return "UnknownInterface"
}

// ── pgxpool v5 invalid field fix ─────────────────────────────────────────────
//
// pgxpool.Config in github.com/jackc/pgx/v5 removed the ConnectTimeout field.
// LLMs trained on older documentation frequently generate config.ConnectTimeout = ...
// which causes a compile error. This fix removes those lines deterministically.

var pgxpoolInvalidFieldRe = regexp.MustCompile(`(?m)^\s*config\.ConnectTimeout\s*=.*\n?`)

func fixInvalidPgxpoolConfig(dir string, files []string) string {
	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		newContent := pgxpoolInvalidFieldRe.ReplaceAll(data, nil)
		if len(newContent) != len(data) {
			_ = os.WriteFile(path, newContent, 0644)
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("removed invalid pgxpool v5 fields in %d file(s)", fixed)
}

// ── gofmt fix ────────────────────────────────────────────────────────────────

func fixGofmt(dir string, files []string) string {
	fixed := 0
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			continue
		}
		path := filepath.Join(dir, f)
		before, _ := os.ReadFile(path)
		_ = exec.Command("gofmt", "-w", path).Run()
		after, _ := os.ReadFile(path)
		if !bytes.Equal(before, after) {
			fixed++
		}
	}
	if fixed == 0 {
		return ""
	}
	return fmt.Sprintf("gofmt fixed %d file(s)", fixed)
}
