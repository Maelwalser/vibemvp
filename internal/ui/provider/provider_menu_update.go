package provider

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
		switch p.focus {
		case pmFocusProviders:
			if p.cursor < len(p.providers)-1 {
				p.cursor++
			}
			p.authCursor = 0
		case pmFocusAuth:
			auths := p.providers[p.cursor].authMethods
			if p.authCursor < len(auths)-1 {
				p.authCursor++
			}
		}

	case "k", "up":
		switch p.focus {
		case pmFocusProviders:
			if p.cursor > 0 {
				p.cursor--
			}
			p.authCursor = 0
		case pmFocusAuth:
			if p.authCursor > 0 {
				p.authCursor--
			}
		}

	// ── Horizontal focus movement ─────────────────────────────────────────────
	case "l", "tab":
		if p.focus == pmFocusProviders {
			p.focus = pmFocusAuth
		}

	case "h", "shift+tab":
		if p.focus == pmFocusAuth {
			p.focus = pmFocusProviders
		}

	// ── Clear current provider's configuration ────────────────────────────────
	case "x":
		if p.focus == pmFocusProviders {
			p = p.clearCurrentProvider()
		}

	// ── Confirm ───────────────────────────────────────────────────────────────
	case "enter", " ":
		switch p.focus {
		case pmFocusProviders:
			// Start configuring the hovered provider; load existing config.
			p.selectedProv = p.cursor
			p.selectedAuth = -1
			p.authCursor = 0
			p = p.loadStateForProvider(p.providers[p.cursor].label)
			p.focus = pmFocusAuth

		case pmFocusAuth:
			p.selectedAuth = p.authCursor
			return p.enterCredentialStep()
		}

	// ── Step back ─────────────────────────────────────────────────────────────
	case "esc":
		if p.focus == pmFocusAuth {
			p.focus = pmFocusProviders
			p.selectedProv = -1
		}
	}

	return p, nil
}
