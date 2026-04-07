package provider

import (
	"os/exec"
	"runtime"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// pmFocus tracks which column owns keyboard input.
type pmFocus int

const (
	pmFocusProviders pmFocus = iota
	pmFocusAuth
	pmFocusCredential // API key / OAuth token input step
)

// ProviderSelection holds a confirmed provider/auth/credential triple.
type ProviderSelection struct {
	Provider   string
	Model      string
	Version    string
	Auth       string
	Credential string // API key or OAuth token
}

// IsSet reports whether this selection is fully configured.
func (p ProviderSelection) IsSet() bool {
	return p.Provider != "" && p.Auth != "" && p.Credential != ""
}

// Short returns a compact display string like "Claude · API Key".
func (p ProviderSelection) Short() string {
	if !p.IsSet() {
		return ""
	}
	return p.Provider + " · " + p.Auth
}

// providerEntry defines an AI provider and its supported auth methods.
type providerEntry struct {
	label       string
	authMethods []string
}

// ProviderMenu is the centered modal for configuring multiple providers.
// Each provider (Claude, Gemini, etc.) can be independently configured with
// its own auth method and credential. The resulting registry is used by the
// realize tab for per-section model assignment.
type ProviderMenu struct {
	// Confirmed provider configurations, keyed by provider label.
	Configured map[string]ProviderSelection

	// Provider/auth columns
	providers    []providerEntry
	cursor       int     // hovered row in provider list
	authCursor   int     // hovered row in auth list
	focus        pmFocus // column that owns input
	selectedProv int     // provider currently being edited (-1 = none)
	selectedAuth int     // -1 = none confirmed in current edit

	// Credential input step
	credInput             textinput.Model
	oauthStatus           string // non-empty while an OAuth flow is in progress or errored
	oauthClientID         string // client ID entered or resolved for the active OAuth flow
	oauthAwaitingClientID bool   // true when credInput is collecting the OAuth client ID
}

// NewMenu creates a new ProviderMenu with persisted credentials loaded.
func NewMenu() ProviderMenu {
	ci := textinput.New()
	ci.Prompt = ""
	ci.Width = pmBoxW - 10
	ci.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFg))
	ci.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFgDim))

	// Load persisted credentials from OS keyring / config file.
	persisted := LoadAllProviderCredentials()
	if persisted == nil {
		persisted = make(map[string]ProviderSelection)
	}

	pm := ProviderMenu{
		Configured: persisted,
		credInput:  ci,
		providers: []providerEntry{
			{label: "Claude", authMethods: []string{"API Key"}},
			{label: "ChatGPT", authMethods: []string{"API Key"}},
			{label: "Gemini", authMethods: []string{"API Key", "OAuth"}},
			{label: "Mistral", authMethods: []string{"API Key"}},
			{label: "Llama", authMethods: []string{"API Key"}},
			{label: "Custom", authMethods: []string{"API Key"}},
		},
		selectedProv: -1,
		selectedAuth: -1,
	}
	return pm
}

// IsCredentialFocused reports whether the credential input currently owns keyboard focus.
func (p ProviderMenu) IsCredentialFocused() bool {
	return p.focus == pmFocusCredential
}

// IsProviderFocused reports whether the provider list currently owns keyboard focus.
func (p ProviderMenu) IsProviderFocused() bool {
	return p.focus == pmFocusProviders
}

// GetConfiguredProviders returns all confirmed provider selections.
func (p ProviderMenu) GetConfiguredProviders() map[string]ProviderSelection {
	return p.Configured
}

// ToManifestConfiguredProviders converts all confirmed configurations to manifest types.
func (p ProviderMenu) ToManifestConfiguredProviders() manifest.ProviderAssignments {
	if len(p.Configured) == 0 {
		return nil
	}
	result := make(manifest.ProviderAssignments, len(p.Configured))
	for label, sel := range p.Configured {
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

// loadStateForProvider restores auth cursor position from the existing
// configuration for the given provider label (if any).
func (p ProviderMenu) loadStateForProvider(label string) ProviderMenu {
	sel, ok := p.Configured[label]
	if !ok || !sel.IsSet() {
		p.selectedAuth = -1
		p.authCursor = 0
		p.credInput.SetValue("")
		return p
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
	if p.selectedProv < 0 || p.selectedAuth < 0 {
		return p
	}

	prov := p.providers[p.selectedProv]
	sel := ProviderSelection{
		Provider:   prov.label,
		Auth:       prov.authMethods[p.selectedAuth],
		Credential: p.credInput.Value(),
	}

	// Copy map (immutable pattern).
	newConfigured := make(map[string]ProviderSelection, len(p.Configured)+1)
	for k, v := range p.Configured {
		newConfigured[k] = v
	}
	newConfigured[prov.label] = sel
	p.Configured = newConfigured
	// Persist to OS keyring / config file.
	_ = SaveProviderCredential(sel.Provider, sel.Auth, sel.Credential)
	return p
}

// clearCurrentProvider removes the configuration for the currently hovered provider.
func (p ProviderMenu) clearCurrentProvider() ProviderMenu {
	provLabel := p.providers[p.cursor].label
	newConfigured := make(map[string]ProviderSelection, len(p.Configured))
	for k, v := range p.Configured {
		if k != provLabel {
			newConfigured[k] = v
		}
	}
	p.Configured = newConfigured
	// Remove from OS keyring / config file.
	DeleteProviderCredential(provLabel)
	// Reset edit state
	p.selectedProv = -1
	p.selectedAuth = -1
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
