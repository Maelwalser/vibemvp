package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (p ProviderMenu) Update(msg tea.Msg) (ProviderMenu, tea.Cmd) {
	// Handle OAuth flow completion.
	if omsg, ok := msg.(oauthTokenMsg); ok {
		if omsg.err != nil {
			p.oauthStatus = "OAuth error: " + omsg.err.Error()
		} else {
			p.credInput.SetValue(omsg.token)
			p.oauthStatus = ""
		}
		return p, nil
	}

	// Delegate to textinput when credential focus is active.
	if p.focus == pmFocusCredential {
		key, ok := msg.(tea.KeyMsg)
		if ok {
			switch key.String() {
			case "enter":
				if p.oauthAwaitingClientID {
					// User just typed their OAuth Client ID — save it and start the flow.
					clientID := strings.TrimSpace(p.credInput.Value())
					if clientID == "" {
						return p, nil
					}
					p.oauthClientID = clientID
					p.oauthAwaitingClientID = false
					p.credInput.SetValue("")
					if p.selectedProv >= 0 {
						p.credInput.Placeholder = "token will appear here after browser authorization"
					}
					p.oauthStatus = "Opening browser…"
					prov := ""
					if p.selectedProv >= 0 {
						prov = p.providers[p.selectedProv].label
					}
					return p, startOAuthCmd(prov, clientID)
				}
				// Normal confirm (API key or OAuth token already filled).
				p = p.confirmCurrentSelection()
				p.focus = pmFocusProviders
				p.credInput.Blur()
				p.oauthAwaitingClientID = false
				return p, nil
			case "esc":
				p.focus = pmFocusAuth
				p.credInput.Blur()
				p.oauthAwaitingClientID = false
				p.oauthStatus = ""
				return p, nil
			case "ctrl+o":
				if p.selectedProv >= 0 {
					prov := p.providers[p.selectedProv].label
					authMethod := ""
					if p.selectedAuth >= 0 {
						authMethod = p.providers[p.selectedProv].authMethods[p.selectedAuth]
					}
					if authMethod == "OAuth" && !p.oauthAwaitingClientID {
						// Re-trigger the OAuth flow with the stored client ID.
						if p.oauthClientID == "" {
							p.oauthAwaitingClientID = true
							p.credInput.SetValue("")
							p.credInput.Placeholder = oauthClientIDPlaceholder(prov)
							return p, nil
						}
						p.oauthStatus = "Opening browser…"
						p.credInput.SetValue("")
						return p, startOAuthCmd(prov, p.oauthClientID)
					}
					// API Key mode: open the key management page.
					if u := oauthURL(prov); u != "" {
						openBrowser(u)
					}
				}
				return p, nil
			}
		}
		var cmd tea.Cmd
		p.credInput, cmd = p.credInput.Update(msg)
		return p, cmd
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}

	switch key.String() {

	// ── Vertical navigation ───────────────────────────────────────────────────
	case "j", "down":
		switch {
		case p.focus == pmFocusModels && p.dropdownOpen:
			vers := p.providers[p.cursor].models[p.modelCursor].versions
			if p.versionCursor < len(vers)-1 {
				p.versionCursor++
			}
		case p.focus == pmFocusProviders:
			if p.cursor < len(p.providers)-1 {
				p.cursor++
			}
			p.modelCursor, p.authCursor = 0, 0
			p.dropdownOpen = false
		case p.focus == pmFocusModels:
			models := p.providers[p.cursor].models
			if p.modelCursor < len(models)-1 {
				p.modelCursor++
			}
		case p.focus == pmFocusAuth:
			auths := p.providers[p.cursor].authMethods
			if p.authCursor < len(auths)-1 {
				p.authCursor++
			}
		}

	case "k", "up":
		switch {
		case p.focus == pmFocusModels && p.dropdownOpen:
			if p.versionCursor > 0 {
				p.versionCursor--
			}
		case p.focus == pmFocusProviders:
			if p.cursor > 0 {
				p.cursor--
			}
			p.modelCursor, p.authCursor = 0, 0
			p.dropdownOpen = false
		case p.focus == pmFocusModels:
			if p.modelCursor > 0 {
				p.modelCursor--
			}
		case p.focus == pmFocusAuth:
			if p.authCursor > 0 {
				p.authCursor--
			}
		}

	// ── Horizontal focus movement (blocked while dropdown open) ───────────────
	case "l", "tab":
		if !p.dropdownOpen {
			switch p.focus {
			case pmFocusProviders:
				p.focus = pmFocusModels
			case pmFocusModels:
				p.focus = pmFocusAuth
			}
		}

	case "h", "shift+tab":
		if !p.dropdownOpen {
			switch p.focus {
			case pmFocusModels:
				p.focus = pmFocusProviders
			case pmFocusAuth:
				p.focus = pmFocusModels
			}
		}

	// ── Clear current provider's configuration ────────────────────────────────
	case "x":
		if p.focus == pmFocusProviders {
			p = p.clearCurrentProvider()
		}

	// ── Confirm / open dropdown ───────────────────────────────────────────────
	case "enter", " ":
		switch p.focus {
		case pmFocusProviders:
			// Start configuring the hovered provider; load existing config.
			p.selectedProv = p.cursor
			p.selectedModel = -1
			p.selectedVersion = -1
			p.selectedAuth = -1
			p.modelCursor = 0
			p.authCursor = 0
			p = p.loadStateForProvider(p.providers[p.cursor].label)
			p.focus = pmFocusModels

		case pmFocusModels:
			if p.dropdownOpen {
				p.selectedModel = p.modelCursor
				p.selectedVersion = p.versionCursor
				p.selectedAuth = -1
				p.dropdownOpen = false
				p.focus = pmFocusAuth
				p.authCursor = 0
			} else {
				p.dropdownOpen = true
				p.versionCursor = 0
				if p.selectedProv == p.cursor && p.selectedModel == p.modelCursor && p.selectedVersion >= 0 {
					p.versionCursor = p.selectedVersion
				}
			}

		case pmFocusAuth:
			p.selectedAuth = p.authCursor
			return p.enterCredentialStep()
		}

	// ── Cancel dropdown / step back ───────────────────────────────────────────
	case "esc":
		if p.dropdownOpen {
			p.dropdownOpen = false
			p.versionCursor = 0
		} else if p.focus != pmFocusProviders {
			switch p.focus {
			case pmFocusAuth:
				p.focus = pmFocusModels
			case pmFocusModels:
				p.focus = pmFocusProviders
				p.selectedProv = -1
			}
		}
	}

	return p, nil
}

