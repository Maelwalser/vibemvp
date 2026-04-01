package orchestrator

// modelSpec holds the version-specific and default model IDs for one provider tier.
type modelSpec struct {
	// byVersion maps specific version strings to model IDs.
	byVersion map[string]string
	// fallback is used when version is empty or not found in byVersion.
	fallback string
}

// providerModels maps provider name → tier → modelSpec.
// Add new providers or tiers here without touching any other file.
var providerModels = map[string]map[string]modelSpec{
	"Claude": {
		"Haiku":  {fallback: "claude-haiku-4-5-20251001"},
		"Sonnet": {fallback: "claude-sonnet-4-6"},
		"Opus":   {fallback: "claude-opus-4-6"},
	},
	"ChatGPT": {
		"Mini": {byVersion: map[string]string{"o3-mini": "o3-mini"}, fallback: "gpt-4o-mini"},
		"4o":   {byVersion: map[string]string{"4o-2024": "gpt-4o-2024-11-20"}, fallback: "gpt-4o"},
		"o1":   {byVersion: map[string]string{"o1-preview": "o1-preview"}, fallback: "o1"},
	},
	"Gemini": {
		"Flash": {byVersion: map[string]string{"1.5": "gemini-1.5-flash"}, fallback: "gemini-2.0-flash"},
		"Pro":   {byVersion: map[string]string{"1.5": "gemini-1.5-pro"}, fallback: "gemini-2.0-pro-exp"},
		"Ultra": {fallback: "gemini-ultra"},
	},
	"Mistral": {
		"Nemo":  {fallback: "open-mistral-nemo"},
		"Small": {byVersion: map[string]string{"3.0": "mistral-small-2402"}, fallback: "mistral-small-2409"},
		"Large": {byVersion: map[string]string{"2.0": "mistral-large-2407"}, fallback: "mistral-large-2411"},
	},
	"Llama": {
		"8B":   {byVersion: map[string]string{"3.1": "llama-3.1-8b-instant"}, fallback: "llama-3.2-8b-preview"},
		"70B":  {byVersion: map[string]string{"3.1": "llama-3.1-70b-versatile"}, fallback: "llama-3.3-70b-versatile"},
		"405B": {fallback: "llama-3.1-405b-reasoning"},
	},
}

// providerDefaults maps each provider to its default model ID when no tier is matched.
var providerDefaults = map[string]string{
	"Claude":  defaultModel,
	"ChatGPT": "gpt-4o",
	"Gemini":  "gemini-2.0-flash",
	"Mistral": "mistral-large-2411",
	"Llama":   "llama-3.3-70b-versatile",
}

// resolveModelID returns the model ID string for a given provider, tier, and version.
// Falls back to the provider default when the tier or version is not found.
func resolveModelID(provider, tier, version string) string {
	tiers, ok := providerModels[provider]
	if !ok {
		if d, ok := providerDefaults[provider]; ok {
			return d
		}
		return defaultModel
	}
	spec, ok := tiers[tier]
	if !ok {
		if d, ok := providerDefaults[provider]; ok {
			return d
		}
		return defaultModel
	}
	if id, ok := spec.byVersion[version]; ok {
		return id
	}
	return spec.fallback
}
