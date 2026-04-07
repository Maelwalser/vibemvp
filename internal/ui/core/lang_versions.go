package core

// langVersions lists available versions for each language/runtime, newest first.
// These are used to populate the language_version select field.
var LangVersions = map[string][]string{
	// Backend languages
	"Go":              {"1.24", "1.23", "1.22", "1.21", "1.20", "1.19"},
	"TypeScript/Node": {"22", "20", "18"},
	"Python":          {"3.13", "3.12", "3.11", "3.10", "3.9"},
	"Java":            {"21", "17", "11"},
	"Kotlin":          {"2.1", "2.0", "1.9", "1.8"},
	"C#/.NET":         {".NET 9", ".NET 8", ".NET 7", ".NET 6"},
	"Rust":            {"1.82", "1.75", "1.70"},
	"Ruby":            {"3.4", "3.3", "3.2", "3.1"},
	"PHP":             {"8.4", "8.3", "8.2", "8.1"},
	"Elixir":          {"1.17", "1.16", "1.15"},
	"Other":           {"latest"},
	// Frontend languages
	"TypeScript": {"5.7", "5.6", "5.5", "5.4", "5.3", "5.0", "4.9"},
	"JavaScript": {"ES2025", "ES2024", "ES2023", "ES2022"},
	"Dart":       {"3.6", "3.5", "3.4", "3.3"},
	"Swift":      {"6.0", "5.10", "5.9"},
}

// frameworkVersions lists available versions for each framework, newest first.
var frameworkVersions = map[string][]string{
	// Go
	"Fiber":             {"v3.0", "v2.52"},
	"Gin":               {"v1.10", "v1.9"},
	"Echo":              {"v4.13", "v4.12"},
	"Chi":               {"v5.2", "v5.0", "v4.1"},
	"net/http (stdlib)": {"stdlib"},
	"Connect":           {"v0.5", "v0.4"},
	// Node.js
	"Express":      {"v5", "v4"},
	"Fastify":      {"v5", "v4"},
	"NestJS":       {"v11", "v10"},
	"Hono":         {"v4.7", "v4.0"},
	"tRPC":         {"v11", "v10"},
	"Elysia (Bun)": {"v1.2", "v1.0"},
	// Python
	"FastAPI":   {"0.115", "0.112", "0.100"},
	"Django":    {"5.1", "5.0", "4.2"},
	"Flask":     {"3.1", "3.0", "2.3"},
	"Litestar":  {"2.13", "2.10"},
	"Starlette": {"0.41", "0.38"},
	// Java
	"Spring Boot": {"3.4", "3.3", "2.7"},
	"Quarkus":     {"3.17", "3.10", "2.16"},
	"Micronaut":   {"4.7", "4.5"},
	"Jakarta EE":  {"11", "10"},
	// Kotlin (backend)
	"Ktor":                 {"3.0", "2.3"},
	"Spring Boot (Kotlin)": {"3.4", "3.3"},
	"http4k":               {"5.x", "4.x"},
	// .NET
	"ASP.NET Core": {"9.0", "8.0", "7.0", "6.0"},
	"Minimal APIs": {"9.0", "8.0", "7.0"},
	"Carter":       {"8.2", "8.0"},
	// Rust
	"Axum":      {"0.8", "0.7", "0.6"},
	"Actix-web": {"4.9", "4.5"},
	"Rocket":    {"0.5", "0.4"},
	"Warp":      {"0.3"},
	// Ruby
	"Rails":   {"8.0", "7.2", "7.1"},
	"Sinatra": {"4.0", "3.2"},
	"Hanami":  {"2.2", "2.1"},
	"Roda":    {"3.x"},
	// PHP
	"Laravel": {"11", "10"},
	"Symfony": {"7.2", "7.1", "6.4"},
	"Slim":    {"4.x"},
	"Laminas": {"3.x"},
	// Elixir
	"Phoenix": {"1.7", "1.6"},
	"Plug":    {"1.16"},
	"Bandit":  {"1.6"},
	// Frontend - JS/TS frameworks
	"React":   {"19", "18.3", "18", "17"},
	"Vue":     {"3.5", "3.4", "3.3", "2.7"},
	"Svelte":  {"5.x", "4.x"},
	"Angular": {"19", "18", "17"},
	"Solid":   {"1.9", "1.8"},
	"Qwik":    {"1.x"},
	"HTMX":    {"2.x", "1.x"},
	// Meta-frameworks
	"Next.js":   {"15", "14", "13"},
	"Nuxt":      {"3.14", "3.13"},
	"SvelteKit": {"2.x", "1.x"},
	"Remix":     {"2.x"},
	"Astro":     {"5.x", "4.x"},
	// Dart/Flutter
	"Flutter": {"3.27", "3.24", "3.19"},
	// Kotlin (frontend / mobile)
	"Jetpack Compose":             {"1.7", "1.6"},
	"KMP (Compose Multiplatform)": {"1.7", "1.6"},
	// Swift/iOS
	"SwiftUI": {"6.0", "5.0"},
	"UIKit":   {"18.x", "17.x"},
}

// frameworkVersionMinLangVer maps "Framework@version" to the minimum language
// version required. Versions are strings matching entries in langVersions.
// Only non-trivial minimums are listed; unlisted versions have no requirement.
var frameworkVersionMinLangVer = map[string]string{
	// Go frameworks — minimum Go version
	"Fiber@v3.0":   "1.21",
	"Fiber@v2.52":  "1.19",
	"Gin@v1.10":    "1.21",
	"Gin@v1.9":     "1.20",
	"Echo@v4.13":   "1.21",
	"Echo@v4.12":   "1.20",
	"Chi@v5.2":     "1.22",
	"Chi@v5.0":     "1.21",
	"Chi@v4.1":     "1.19",
	"Connect@v0.5": "1.21",
	"Connect@v0.4": "1.20",
	// Node.js frameworks — minimum Node version (LTS)
	"Express@v5":        "18",
	"Fastify@v5":        "20",
	"Fastify@v4":        "18",
	"NestJS@v11":        "20",
	"NestJS@v10":        "18",
	"Hono@v4.7":         "18",
	"tRPC@v11":          "18",
	"Elysia (Bun)@v1.2": "20",
	"Elysia (Bun)@v1.0": "18",
	// Python frameworks — minimum Python version
	"Django@5.1": "3.10",
	"Django@5.0": "3.10",
	"Django@4.2": "3.9",
	"Flask@3.1":  "3.9",
	"Flask@3.0":  "3.9",
	// Java frameworks — minimum Java version
	"Spring Boot@3.4": "17",
	"Spring Boot@3.3": "17",
	"Spring Boot@2.7": "11",
	"Quarkus@3.17":    "17",
	"Quarkus@3.10":    "17",
	"Quarkus@2.16":    "11",
	"Micronaut@4.7":   "17",
	"Micronaut@4.5":   "17",
	"Jakarta EE@11":   "21",
	"Jakarta EE@10":   "11",
	// Kotlin (backend) — minimum Kotlin version
	"Ktor@3.0":                 "1.9",
	"Ktor@2.3":                 "1.8",
	"Spring Boot (Kotlin)@3.4": "1.9",
	"Spring Boot (Kotlin)@3.3": "1.9",
	// .NET — minimum .NET version
	"ASP.NET Core@9.0": ".NET 9",
	"ASP.NET Core@8.0": ".NET 8",
	"ASP.NET Core@7.0": ".NET 7",
	"ASP.NET Core@6.0": ".NET 6",
	"Minimal APIs@9.0": ".NET 9",
	"Minimal APIs@8.0": ".NET 8",
	"Minimal APIs@7.0": ".NET 7",
	"Carter@8.2":       ".NET 8",
	"Carter@8.0":       ".NET 8",
	// Rust — minimum Rust version (MSRV)
	"Axum@0.8":      "1.75",
	"Axum@0.7":      "1.70",
	"Axum@0.6":      "1.70",
	"Actix-web@4.9": "1.75",
	"Actix-web@4.5": "1.70",
	// Ruby — minimum Ruby version
	"Rails@8.0": "3.2",
	"Rails@7.2": "3.1",
	"Rails@7.1": "3.1",
	// PHP — minimum PHP version
	"Laravel@11":  "8.2",
	"Laravel@10":  "8.1",
	"Symfony@7.2": "8.2",
	"Symfony@7.1": "8.2",
	"Symfony@6.4": "8.1",
	// Elixir — minimum Elixir version
	"Phoenix@1.7": "1.15",
	"Phoenix@1.6": "1.14",
	// Frontend React — minimum TypeScript version
	"React@19":   "5.0",
	"React@18.3": "4.9",
	"React@18":   "4.9",
	// Vue — minimum TypeScript version
	"Vue@3.5": "5.0",
	"Vue@3.4": "4.9",
	"Vue@3.3": "4.9",
	// Angular — minimum TypeScript version
	"Angular@19": "5.5",
	"Angular@18": "5.4",
	"Angular@17": "5.2",
	// Flutter — minimum Dart version
	"Flutter@3.27": "3.6",
	"Flutter@3.24": "3.5",
	"Flutter@3.19": "3.3",
	// Kotlin mobile — minimum Kotlin version
	"Jetpack Compose@1.7":             "2.0",
	"Jetpack Compose@1.6":             "1.9",
	"KMP (Compose Multiplatform)@1.7": "2.0",
	"KMP (Compose Multiplatform)@1.6": "1.9",
	// Swift/iOS — minimum Swift version
	"SwiftUI@6.0": "6.0",
	"SwiftUI@5.0": "5.9",
}

// compatibleFrameworkVersions returns the framework versions compatible with
// the given language and its selected version. When langVer is empty, all
// versions for the framework are returned without filtering.
func CompatibleFrameworkVersions(lang, langVer, framework string) []string {
	all, ok := frameworkVersions[framework]
	if !ok {
		return []string{"latest"}
	}
	if langVer == "" {
		return all
	}
	langVers, ok := LangVersions[lang]
	if !ok {
		return all
	}
	// Find the index of the selected language version (0 = newest).
	selectedIdx := -1
	for i, v := range langVers {
		if v == langVer {
			selectedIdx = i
			break
		}
	}
	if selectedIdx == -1 {
		return all
	}

	var compatible []string
	for _, fwVer := range all {
		minLangVer, hasMin := frameworkVersionMinLangVer[framework+"@"+fwVer]
		if !hasMin {
			// No minimum requirement → always compatible.
			compatible = append(compatible, fwVer)
			continue
		}
		// Find the index of the minimum required language version.
		minIdx := -1
		for i, v := range langVers {
			if v == minLangVer {
				minIdx = i
				break
			}
		}
		if minIdx == -1 {
			// Minimum version not in list → assume compatible.
			compatible = append(compatible, fwVer)
			continue
		}
		// selectedIdx <= minIdx means selected version is as new as or newer
		// than the minimum required (lower index = newer).
		if selectedIdx <= minIdx {
			compatible = append(compatible, fwVer)
		}
	}
	if len(compatible) == 0 {
		return all // fallback: show all if nothing passes
	}
	return compatible
}
