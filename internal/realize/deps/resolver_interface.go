package deps

// ModuleSpec describes a single dependency to add to a project.
type ModuleSpec struct {
	// Name is the import path (Go) or package name (npm).
	Name string
	// Version is the pinned version string (e.g. "v1.2.3" or "^1.2.3").
	Version string
	// DevOnly indicates the dependency is only needed during development/testing.
	DevOnly bool
}

// LanguageResolver maps technology names to concrete dependency specs for one
// target language. Implement this interface to add support for a new language.
//
// To register a new resolver add it to ResolverRegistry in the init() of its
// file (e.g. go_modules.go, npm_modules.go).
type LanguageResolver interface {
	// Language returns the canonical language name (e.g. "go", "typescript").
	Language() string
	// ResolveModules returns the runtime module specs for the given technologies.
	ResolveModules(technologies []string) []ModuleSpec
	// ResolveDevTools returns the development/test module specs.
	ResolveDevTools(technologies []string) []ModuleSpec
}

// ResolverRegistry maps language names to their resolver implementation.
// Resolvers register themselves via init().
var ResolverRegistry = map[string]LanguageResolver{}

// RegisterResolver adds a LanguageResolver to the registry.
func RegisterResolver(r LanguageResolver) {
	ResolverRegistry[r.Language()] = r
}
