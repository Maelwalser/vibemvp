package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ApplyDeterministicFixes applies mechanical, always-correct transformations to
// generated code before running the language verifier. Returns a description
// of fixes applied, or "" if none were needed.
//
// language must match the verifier's Language() string ("go", "typescript",
// "python", "terraform", or "" for unknown). Only fixes relevant to that
// language are executed — no cross-language tool invocations are attempted.
//
// Run this BEFORE every verification attempt — not just on retries — so that
// first-attempt code gets the same cleanup benefit without consuming a retry slot.
func ApplyDeterministicFixes(dir string, files []string, language string) string {
	var fixes []string

	switch language {
	case "go", "":
		// goimports adds missing stdlib/module imports and removes unused ones.
		// Run first so placeholder-path and gofmt steps see a clean import block.
		if f := fixGoImports(dir, files); f != "" {
			fixes = append(fixes, f)
		}
		// Fix placeholder import paths — rewriting imports may introduce temporarily
		// un-gofmt'd lines, so run gofmt after.
		if f := fixPlaceholderImportPaths(dir, files); f != "" {
			fixes = append(fixes, f)
		}
		if f := fixGoEscapeSequences(dir, files); f != "" {
			fixes = append(fixes, f)
		}
		if f := fixDuplicateTypes(dir, files); f != "" {
			fixes = append(fixes, f)
		}
		// Remove invalid pgxpool v5 fields before gofmt so the result is clean.
		if f := fixInvalidPgxpoolConfig(dir, files); f != "" {
			fixes = append(fixes, f)
		}
		// Remove import statements that appear inside function bodies — always a bug.
		if f := fixMisplacedImports(dir, files); f != "" {
			fixes = append(fixes, f)
		}
		// Repair orphaned return-type fragments left by truncated LLM responses, e.g.:
		//   // PgxPool is the interface for ...
		//   , error)          ← truncated — type PgxPool interface { and Exec method were cut off
		//       Query(...)
		//   }
		if f := fixOrphanedInterfaceFragments(dir, files); f != "" {
			fixes = append(fixes, f)
		}
		if f := fixGofmt(dir, files); f != "" {
			fixes = append(fixes, f)
		}

	case "typescript":
		if f := fixTypeScript(dir, files); f != "" {
			fixes = append(fixes, f)
		}

	case "python":
		// isort re-orders imports; run before ruff/black so they see a consistent
		// import block and don't re-report ordering violations as unfixed.
		if f := fixPythonImports(dir, files); f != "" {
			fixes = append(fixes, f)
		}
		if f := fixPython(dir, files); f != "" {
			fixes = append(fixes, f)
		}

		// terraform and other verifier languages have no deterministic fixes yet.
	}

	if len(fixes) == 0 {
		return ""
	}
	return strings.Join(fixes, "; ")
}

// ── Type-as-string conversion fix ────────────────────────────────────────────
//
// LLMs frequently define response struct fields as string when the domain type
// is bool, time.Time, or uuid.UUID, then assign the typed value without conversion.
// This causes "cannot use X (variable of type T) as string value" compile errors.
//
// This fix detects those patterns and rewrites the offending expression with the
// idiomatic Go conversion: .String() for UUID, strconv.FormatBool for bool,
// .Format(time.RFC3339) for time.Time.
//
// The same fix also handles struct literal context ("as string value in struct literal")
// and function argument context ("as string value in argument to").

var typeAsStringErrRe = regexp.MustCompile(
	`^(.+\.go):(\d+):\d+: cannot use (\S+) \(variable of (?:array |struct )?type ([^)]+)\) as string`)

// ApplyUUIDToStringFixes reads go compiler output, finds type-as-string errors for
// well-known types (uuid.UUID, bool, time.Time), and patches source files in dir.
// The name is kept for backward-compatibility with callers.
func ApplyUUIDToStringFixes(dir string, verifyOutput string) string {
	type fix struct {
		file    string
		line    int
		varName string
		srcType string
	}
	var fixes []fix
	seen := make(map[string]bool)
	for _, line := range strings.Split(verifyOutput, "\n") {
		m := typeAsStringErrRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		relFile, lineStr, varName, srcType := m[1], m[2], m[3], m[4]
		key := relFile + ":" + lineStr
		if seen[key] {
			continue
		}
		seen[key] = true
		lineNum := 0
		fmt.Sscanf(lineStr, "%d", &lineNum)
		fixes = append(fixes, fix{file: relFile, line: lineNum, varName: varName, srcType: srcType})
	}
	if len(fixes) == 0 {
		return ""
	}
	// Group fixes by file so we apply all line patches and then a single import pass.
	type fileFix struct {
		lineIdx     int
		varName     string
		replacement string
		needImport  string // package name to ensure is imported, or ""
	}
	byFile := make(map[string][]fileFix)
	for _, fx := range fixes {
		replacement := typeToStringExpr(fx.varName, fx.srcType)
		if replacement == "" {
			continue // unsupported type — let LLM handle on retry
		}
		needImport := requiredImport(fx.srcType)
		path := filepath.Join(dir, fx.file)
		if _, err := os.Stat(path); err != nil {
			path = fx.file
		}
		byFile[path] = append(byFile[path], fileFix{
			lineIdx:     fx.line - 1,
			varName:     fx.varName,
			replacement: replacement,
			needImport:  needImport,
		})
	}
	applied := 0
	for path, fileFixes := range byFile {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		fileLines := strings.Split(string(data), "\n")
		changed := false
		for _, ff := range fileFixes {
			if ff.lineIdx < 0 || ff.lineIdx >= len(fileLines) {
				continue
			}
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(ff.varName) + `\b`)
			patched := re.ReplaceAllString(fileLines[ff.lineIdx], ff.replacement)
			if patched != fileLines[ff.lineIdx] {
				fileLines[ff.lineIdx] = patched
				changed = true
			}
			// Ensure the required package is imported.
			if ff.needImport != "" {
				fileLines = ensureGoImport(fileLines, ff.needImport)
			}
		}
		if !changed {
			continue
		}
		_ = os.WriteFile(path, []byte(strings.Join(fileLines, "\n")), 0644)
		applied++
	}
	if applied == 0 {
		return ""
	}
	return fmt.Sprintf("applied type→string conversions to %d file(s)", applied)
}

// typeToStringExpr returns the idiomatic Go expression that converts varName of
// srcType to string, or "" when the conversion is not known / safe to automate.
func typeToStringExpr(varName, srcType string) string {
	switch {
	case srcType == "uuid.UUID":
		return varName + ".String()"
	case srcType == "bool":
		return "strconv.FormatBool(" + varName + ")"
	case srcType == `"time".Time`, srcType == "time.Time":
		return varName + ".Format(time.RFC3339)"
	case srcType == "int", srcType == "int64", srcType == "int32":
		return "strconv.Itoa(int(" + varName + "))"
	default:
		return ""
	}
}

// requiredImport returns the Go standard-library package that must be imported
// when the given type conversion is applied, or "" when no extra import is needed.
func requiredImport(srcType string) string {
	switch srcType {
	case "bool", "int", "int64", "int32":
		return "strconv"
	default:
		return ""
	}
}

// ensureGoImport ensures that importPkg is present in the import block of the
// given file lines. If it is not present, it is added. Returns the (possibly
// modified) line slice.
func ensureGoImport(lines []string, importPkg string) []string {
	quoted := `"` + importPkg + `"`
	for _, l := range lines {
		if strings.Contains(l, quoted) {
			return lines // already imported
		}
	}
	// Find the import block or single-line import and add the package.
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "import (" {
			// Insert before the closing paren.
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == ")" {
					newLines := make([]string, 0, len(lines)+1)
					newLines = append(newLines, lines[:j]...)
					newLines = append(newLines, "\t"+quoted)
					newLines = append(newLines, lines[j:]...)
					return newLines
				}
			}
		}
		// Single-line import: insert a new import block after it.
		if strings.HasPrefix(trimmed, `import "`) {
			newLines := make([]string, 0, len(lines)+3)
			newLines = append(newLines, lines[:i+1]...)
			newLines = append(newLines, "import "+quoted)
			newLines = append(newLines, lines[i+1:]...)
			return newLines
		}
	}
	return lines
}
