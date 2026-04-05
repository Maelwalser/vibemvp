package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Layout constants ──────────────────────────────────────────────────────────

const (
	pmCol1W = 14 // provider column visible width
	pmCol2W = 28 // auth column visible width (last, not padded by pmRow)
	// pmBoxW is the Width() argument for StyleModalBorder.
	// StyleModalBorder has Padding(0,1) + sharpBorder, so actual rendered
	// width = pmBoxW + 2 (padding) + 2 (border) = pmBoxW + 4.
	// +1 for the │ column separator inserted by pmRow.
	pmBoxW = pmCol1W + 1 + pmCol2W // 43 → total box ≈ 47 chars
)

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the provider menu as a self-contained bordered box string.
func (p ProviderMenu) View() string {
	var rows []string
	bg := lipgloss.Color(clrBg2)

	// ── Title bar ────────────────────────────────────────────────────────────
	// Minimal amber title — editorial, no decorative noise.
	titleLine := lipgloss.NewStyle().
		Background(bg).
		Width(pmBoxW).
		Align(lipgloss.Center).
		Render(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true).
				Background(bg).
				Render("AI PROVIDERS"),
		)

	rows = append(rows, titleLine)
	rows = append(rows, p.renderDividers())
	rows = append(rows, "")
	rows = append(rows, p.renderHeaders())
	rows = append(rows, p.renderDividers())

	col1 := p.buildProviderCol()
	col2 := p.buildAuthCol()

	h := max(len(col1), len(col2))
	for len(col1) < h {
		col1 = append(col1, "")
	}
	for len(col2) < h {
		col2 = append(col2, "")
	}

	for i := 0; i < h; i++ {
		rows = append(rows, pmRow(col1[i], col2[i]))
	}

	rows = append(rows, "")

	// Credential input step.
	if p.focus == pmFocusCredential {
		rows = append(rows, p.renderCredentialPanel())
		rows = append(rows, "")
	}

	// Show configured providers summary.
	if summary := p.renderConfiguredSummary(); summary != "" {
		rows = append(rows, summary)
		rows = append(rows, "")
	}

	// Context-sensitive hint bar.
	modalBg := lipgloss.Color(clrBg2)
	var hints string
	switch {
	case p.focus == pmFocusCredential:
		authMethod := ""
		if p.selectedProv >= 0 && p.selectedAuth >= 0 {
			authMethod = p.providers[p.selectedProv].authMethods[p.selectedAuth]
		}
		switch {
		case authMethod == "OAuth" && p.oauthAwaitingClientID:
			hints = hintBarBg(modalBg, "Enter", "open browser", "Esc", "back")
		case authMethod == "OAuth":
			hints = hintBarBg(modalBg, "Enter", "confirm", "Ctrl+O", "re-auth", "Esc", "back")
		default:
			hints = hintBarBg(modalBg, "Enter", "confirm", "Esc", "back")
		}
	case p.focus == pmFocusProviders:
		hints = hintBarBg(modalBg, "j/k", "navigate", "Enter", "configure", "x", "clear", "M", "close")
	default:
		hints = hintBarBg(modalBg, "j/k", "nav", "h/l", "col", "Enter", "pick", "Esc", "back")
	}
	rows = append(rows, hints)

	// Width(pmBoxW+2): in lipgloss v1, Width includes padding.
	// Padding(0,1) eats 2 from Width, so content area = (pmBoxW+2)-2 = pmBoxW = 43.
	return StyleModalBorder.Width(pmBoxW + 2).Render(strings.Join(rows, "\n"))
}

// renderConfiguredSummary shows all currently configured providers.
func (p ProviderMenu) renderConfiguredSummary() string {
	if len(p.configured) == 0 {
		return ""
	}

	bg := lipgloss.Color(clrBg2)
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Background(bg)
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Bold(true).Background(bg)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(bg)

	var lines []string
	lines = append(lines, p.renderDividers())

	for _, prov := range p.providers {
		sel, ok := p.configured[prov.label]
		if !ok || !sel.IsSet() {
			continue
		}
		line := checkStyle.Render("  ◈ ") + nameStyle.Render(sel.Provider) + dimStyle.Render("  "+sel.Auth)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// renderCredentialPanel renders the inline API key / OAuth token input.
func (p ProviderMenu) renderCredentialPanel() string {
	authMethod := ""
	provLabel := ""
	if p.selectedProv >= 0 {
		provLabel = p.providers[p.selectedProv].label
		if p.selectedAuth >= 0 {
			authMethod = p.providers[p.selectedProv].authMethods[p.selectedAuth]
		}
	}

	bg := lipgloss.Color(clrBg2)
	amberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(bg)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(bg)

	inputBoxStyle := lipgloss.NewStyle().
		Border(sharpBorder).
		BorderForeground(lipgloss.Color(clrComment)).
		Background(bg).
		Padding(0, 1).
		Width(pmBoxW - 2)

	var lines []string

	if authMethod == "OAuth" {
		if p.oauthAwaitingClientID {
			lines = append(lines, amberStyle.Render("  OAUTH  ─  STEP 1 / 2"))
			lines = append(lines, dimStyle.Render("  Enter your OAuth Client ID, then press Enter."))
			switch provLabel {
			case "Gemini":
				lines = append(lines, dimStyle.Render("  console.cloud.google.com → APIs & Services → Credentials"))
			default:
				lines = append(lines, dimStyle.Render("  Create an OAuth 2.0 Client ID for your application."))
			}
			lines = append(lines, "")

			inputBox := inputBoxStyle.Render(
				amberStyle.Render("CLIENT ID") + "  " + p.credInput.View(),
			)
			lines = append(lines, inputBox)
		} else {
			lines = append(lines, amberStyle.Render("  OAUTH  ─  STEP 2 / 2"))
			if p.oauthStatus != "" {
				lines = append(lines, dimStyle.Render("  Approve access in the browser, then return here."))
			} else {
				lines = append(lines, dimStyle.Render("  Ctrl+O to re-authorize  ·  Enter to confirm"))
			}
			lines = append(lines, "")

			inputBox := inputBoxStyle.Render(
				amberStyle.Render("OAUTH TOKEN") + "  " + p.credInput.View(),
			)
			lines = append(lines, inputBox)
			if p.oauthStatus != "" {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color(clrOrange)).Background(bg).Render("  "+p.oauthStatus))
			}
		}
	} else {
		lines = append(lines, dimStyle.Render(fmt.Sprintf("  %s  ─  API KEY", provLabel)))
		lines = append(lines, "")

		inputBox := inputBoxStyle.Render(
			amberStyle.Render("API KEY") + "  " + p.credInput.View(),
		)
		lines = append(lines, inputBox)
	}
	return strings.Join(lines, "\n")
}

// renderHeaders returns the column header row.
func (p ProviderMenu) renderHeaders() string {
	bg := lipgloss.Color(clrBg2)
	inactive := lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(bg)
	active := lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(bg)

	h1, h2 := inactive, inactive
	switch p.focus {
	case pmFocusProviders:
		h1 = active
	case pmFocusAuth:
		h2 = active
	}
	return pmRow(h1.Render("PROVIDER"), h2.Render("AUTH"))
}

// renderDividers returns the ─── separator row under the headers.
// Each segment fills its full column width so the grid lines are flush.
func (p ProviderMenu) renderDividers() string {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(clrComment)).Background(lipgloss.Color(clrBg2))
	return pmRow(
		s.Render(strings.Repeat("─", pmCol1W)),
		s.Render(strings.Repeat("─", pmCol2W)),
	)
}

// buildProviderCol returns one string per row for the provider column.
// ◈ is shown for any provider that is already configured.
func (p ProviderMenu) buildProviderCol() []string {
	lines := make([]string, 0, len(p.providers))
	for i, prov := range p.providers {
		isCur := i == p.cursor
		isConfigured := p.configured[prov.label].IsSet()
		isHL := isCur && p.focus == pmFocusProviders

		rowBg := lipgloss.Color(clrBg2)
		if isHL {
			rowBg = lipgloss.Color(clrBgHL)
		}

		arrow := lipgloss.NewStyle().Background(rowBg).Render("  ")
		if isCur {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Background(rowBg).Render("▶ ")
		}

		var label string
		switch {
		case isConfigured && isHL:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Bold(true).Background(rowBg).Render("◈ " + prov.label)
		case isConfigured:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Bold(true).Background(lipgloss.Color(clrBg2)).Render("◈ " + prov.label)
		case isHL:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(rowBg).Render(prov.label)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Background(lipgloss.Color(clrBg2)).Render(prov.label)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(lipgloss.Color(clrBg2)).Render(prov.label)
		}

		cell := arrow + label
		if isHL {
			cell = pmHighlight(cell, pmCol1W)
		}
		lines = append(lines, cell)
	}
	return lines
}

// buildAuthCol returns one string per row for the auth method column.
// Shows the auth methods for the currently hovered provider.
func (p ProviderMenu) buildAuthCol() []string {
	auths := p.providers[p.cursor].authMethods
	lines := make([]string, 0, len(auths))
	for i, a := range auths {
		isCur := i == p.authCursor
		isSel := p.selectedProv == p.cursor && i == p.selectedAuth
		isHL := isCur && p.focus == pmFocusAuth

		rowBg := lipgloss.Color(clrBg2)
		if isHL {
			rowBg = lipgloss.Color(clrBgHL)
		}

		arrow := lipgloss.NewStyle().Background(rowBg).Render("  ")
		if isCur && p.focus == pmFocusAuth {
			arrow = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Background(rowBg).Render("▶ ")
		}

		var label string
		switch {
		case isSel:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrGreen)).Bold(true).Background(lipgloss.Color(clrBg2)).Render("✓ " + a)
		case isHL:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(rowBg).Render(a)
		case isCur:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg)).Background(lipgloss.Color(clrBg2)).Render(a)
		default:
			label = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim)).Background(lipgloss.Color(clrBg2)).Render(a)
		}

		cell := arrow + label
		if isHL {
			cell = pmHighlight(cell, pmCol2W)
		}
		lines = append(lines, cell)
	}
	return lines
}

// ── Layout helpers ────────────────────────────────────────────────────────────

// pmRow assembles two column cells into one display line, with a │ separator.
func pmRow(col1, col2 string) string {
	sep := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFgDim)).
		Background(lipgloss.Color(clrBg2)).
		Render("│")
	return pmPad(col1, pmCol1W) + sep + pmPad(col2, pmCol2W)
}

// pmPad pads s with background-colored spaces until its visible width equals toW.
func pmPad(s string, toW int) string {
	if pad := toW - lipgloss.Width(s); pad > 0 {
		return s + lipgloss.NewStyle().Background(lipgloss.Color(clrBg2)).Render(strings.Repeat(" ", pad))
	}
	return s
}

// pmHighlight pads s to colW with highlight-colored spaces and applies the
// cursor-line background (constant color, no animation).
func pmHighlight(s string, colW int) string {
	curStyle := activeCurLineStyle()
	bg := lipgloss.Color(clrBgHL)
	if pad := colW - lipgloss.Width(s); pad > 0 {
		s = s + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", pad))
	}
	return curStyle.Render(s)
}
