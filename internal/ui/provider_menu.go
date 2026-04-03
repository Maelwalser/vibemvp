package ui

import (
	"os/exec"
	"runtime"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

// pmFocus tracks which column owns keyboard input.
type pmFocus int

const (
	pmFocusProviders pmFocus = iota
	pmFocusModels
	pmFocusAuth
	pmFocusCredential // API key / OAuth token input step
)

// ProviderSelection holds a confirmed provider/model/auth/credential quad.
type ProviderSelection struct {
	Provider   string
	Model      string
	Version    string
	Auth       string
	Credential string // API key or OAuth token
}

// IsSet reports whether this selection is fully configured.
func (p ProviderSelection) IsSet() bool {
	return p.Provider != "" && p.Model != "" && p.Auth != "" && p.Credential != ""
}

// Short returns a compact display string like "Claude · Sonnet 4.5".
func (p ProviderSelection) Short() string {
	if !p.IsSet() {
		return ""
	}
	model := p.Model
	if p.Version != "" {
		model += " " + p.Version
	}
	return p.Provider + " · " + model
}

// modelTier represents one tier of a provider (e.g. "Sonnet") with its
// concrete version strings (e.g. "4.5", "4.0", "3.5").
type modelTier struct {
	name     string
	versions []string
}

// providerEntry defines an AI provider, its model tiers, and auth methods.
type providerEntry struct {
	label       string
	models      []modelTier
	authMethods []string
}

// ProviderMenu is the centered modal for configuring multiple providers.
// Each provider (Claude, Gemini, etc.) can be independently configured with
// its own model tier, auth method, and credential. The resulting registry is
// used by the realize tab for per-section model assignment.
type ProviderMenu struct {
	// Confirmed provider configurations, keyed by provider label.
	configured map[string]ProviderSelection

	// Provider/model/auth columns
	providers       []providerEntry
	cursor          int     // hovered row in provider list
	modelCursor     int     // hovered row in model list
	authCursor      int     // hovered row in auth list
	focus           pmFocus // column that owns input
	dropdownOpen    bool    // version dropdown visible
	versionCursor   int     // hovered row inside the dropdown
	selectedProv    int     // provider currently being edited (-1 = none)
	selectedModel   int     // -1 = none confirmed in current edit
	selectedVersion int     // -1 = none confirmed in current edit
	selectedAuth    int     // -1 = none confirmed in current edit

	// Credential input step
	credInput            textinput.Model
	oauthStatus          string // non-empty while an OAuth flow is in progress or errored
	oauthClientID        string // client ID entered or resolved for the active OAuth flow
	oauthAwaitingClientID bool   // true when credInput is collecting the OAuth client ID
}

func newProviderMenu() ProviderMenu {
	ci := textinput.New()
	ci.Prompt = ""
	ci.Width = pmBoxW - 10
	ci.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	ci.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

	pm := ProviderMenu{
		configured: make(map[string]ProviderSelection),
		credInput:  ci,
		providers: []providerEntry{
			{
				label: "Claude",
				models: []modelTier{
					{name: "Haiku", versions: []string{"3.5", "3.0"}},
					{name: "Sonnet", versions: []string{"4.5", "4.0", "3.5"}},
					{name: "Opus", versions: []string{"4.0", "3.0"}},
				},
				authMethods: []string{"API Key"},
			},
			{
				label: "ChatGPT",
				models: []modelTier{
					{name: "Mini", versions: []string{"o3-mini", "4o-mini"}},
					{name: "4o", versions: []string{"4o", "4o-2024"}},
					{name: "o1", versions: []string{"o1", "o1-preview"}},
				},
				authMethods: []string{"API Key"},
			},
			{
				label: "Gemini",
				models: []modelTier{
					{name: "Flash", versions: []string{"2.0", "1.5"}},
					{name: "Pro", versions: []string{"2.0", "1.5"}},
					{name: "Ultra", versions: []string{"1.0"}},
				},
				authMethods: []string{"API Key", "OAuth"},
			},
			{
				label: "Mistral",
				models: []modelTier{
					{name: "Nemo", versions: []string{"latest"}},
					{name: "Small", versions: []string{"3.1", "3.0"}},
					{name: "Large", versions: []string{"2.1", "2.0"}},
				},
				authMethods: []string{"API Key"},
			},
			{
				label: "Llama",
				models: []modelTier{
					{name: "8B", versions: []string{"3.2", "3.1"}},
					{name: "70B", versions: []string{"3.3", "3.1"}},
					{name: "405B", versions: []string{"3.1"}},
				},
				authMethods: []string{"API Key"},
			},
			{
				label: "Custom",
				models: []modelTier{
					{name: "Custom", versions: []string{"endpoint"}},
				},
				authMethods: []string{"API Key"},
			},
		},
		selectedProv:    -1,
		selectedModel:   -1,
		selectedVersion: -1,
		selectedAuth:    -1,
	}
	return pm
}

// GetConfiguredProviders returns all confirmed provider selections.
func (p ProviderMenu) GetConfiguredProviders() map[string]ProviderSelection {
	return p.configured
}

// ToManifestConfiguredProviders converts all confirmed configurations to manifest types.
func (p ProviderMenu) ToManifestConfiguredProviders() manifest.ProviderAssignments {
	if len(p.configured) == 0 {
		return nil
	}
	result := make(manifest.ProviderAssignments, len(p.configured))
	for label, sel := range p.configured {
		if sel.IsSet() {
			result[label] = manifest.ProviderAssignment{
				Provider:   sel.Provider,
				Model:      sel.Model,
				Version:    sel.Version,
				Auth:       sel.Auth,
				Credential: sel.Credential,
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// loadStateForProvider restores model/auth/version cursor positions from the
// existing configuration for the given provider label (if any).
func (p ProviderMenu) loadStateForProvider(label string) ProviderMenu {
	sel, ok := p.configured[label]
	if !ok || !sel.IsSet() {
		p.selectedModel = -1
		p.selectedVersion = -1
		p.selectedAuth = -1
		p.modelCursor = 0
		p.authCursor = 0
		p.credInput.SetValue("")
		return p
	}

	// Restore model + version cursors.
	if p.selectedProv >= 0 {
		models := p.providers[p.selectedProv].models
		for i, tier := range models {
			if tier.name == sel.Model {
				p.modelCursor = i
				p.selectedModel = i
				for j, v := range tier.versions {
					if v == sel.Version {
						p.versionCursor = j
						p.selectedVersion = j
						break
					}
				}
				break
			}
		}
	}

	// Restore auth cursor.
	if p.selectedProv >= 0 {
		auths := p.providers[p.selectedProv].authMethods
		for i, a := range auths {
			if a == sel.Auth {
				p.authCursor = i
				p.selectedAuth = i
				break
			}
		}
	}

	p.credInput.SetValue(sel.Credential)
	return p
}

// confirmCurrentSelection saves the current editing state for the active provider.
func (p ProviderMenu) confirmCurrentSelection() ProviderMenu {
	if p.selectedProv < 0 || p.selectedModel < 0 || p.selectedVersion < 0 || p.selectedAuth < 0 {
		return p
	}

	prov := p.providers[p.selectedProv]
	tier := prov.models[p.selectedModel]

	sel := ProviderSelection{
		Provider:   prov.label,
		Model:      tier.name,
		Version:    tier.versions[p.selectedVersion],
		Auth:       prov.authMethods[p.selectedAuth],
		Credential: p.credInput.Value(),
	}

	// Copy map (immutable pattern).
	newConfigured := make(map[string]ProviderSelection, len(p.configured)+1)
	for k, v := range p.configured {
		newConfigured[k] = v
	}
	newConfigured[prov.label] = sel
	p.configured = newConfigured
	return p
}

// clearCurrentProvider removes the configuration for the currently hovered provider.
func (p ProviderMenu) clearCurrentProvider() ProviderMenu {
	provLabel := p.providers[p.cursor].label
	newConfigured := make(map[string]ProviderSelection, len(p.configured))
	for k, v := range p.configured {
		if k != provLabel {
			newConfigured[k] = v
		}
	}
	p.configured = newConfigured
	// Reset edit state
	p.selectedProv = -1
	p.selectedModel = -1
	p.selectedVersion = -1
	p.selectedAuth = -1
	p.modelCursor = 0
	p.authCursor = 0
	p.credInput.SetValue("")
	return p
}

// oauthURL returns a browser URL for the given provider's authentication page.
func oauthURL(provider string) string {
	switch provider {
	case "Claude":
		return "https://console.anthropic.com/settings/keys"
	case "ChatGPT":
		return "https://platform.openai.com/api-keys"
	case "Gemini":
		return "https://aistudio.google.com/app/apikey"
	case "Mistral":
		return "https://console.mistral.ai/api-keys"
	default:
		return ""
	}
}

// openBrowser attempts to open url in the system browser.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}

