package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/ui/core"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	if m.width < minTermWidth || m.height < minTermHeight {
		msg := fmt.Sprintf(" Terminal too small (%d×%d). Resize to at least %d×%d. ",
			m.width, m.height, minTermWidth, minTermHeight)
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			core.StyleMsgErr.Render(msg))
	}

	base := m.renderBaseView()

	if m.modal.open {
		modal := m.modal.menu.View()
		modalLines := strings.Split(modal, "\n")
		modalH := len(modalLines)
		modalW := 0
		for _, l := range modalLines {
			if w := lipgloss.Width(l); w > modalW {
				modalW = w
			}
		}
		x := (m.width - modalW) / 2
		y := (m.height - modalH) / 2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		return core.PlaceOverlay(base, modal, x, y)
	}

	return base
}

func (m Model) renderBaseView() string {
	if m.realize.show {
		return m.renderRealizeFullScreen()
	}
	if m.arch.show {
		return m.renderArchFullScreen()
	}
	var b strings.Builder
	w := m.width
	b.WriteString(m.renderHeader(w))
	b.WriteString("\n")
	b.WriteString(m.renderContent(w))
	b.WriteString(m.renderTabBar(w))
	b.WriteString("\n")
	b.WriteString(m.renderStatusLine(w))
	b.WriteString("\n")
	b.WriteString(m.renderCmdLine(w))
	return b.String()
}

func (m Model) renderRealizeFullScreen() string {
	w, h := m.width, m.height
	contentH := h - 1
	if contentH < 1 {
		contentH = 1
	}
	content := m.realize.screen.View(w, contentH)
	hint := m.realize.screen.HintLine()
	return content + "\n" + hint
}

func (m Model) renderArchFullScreen() string {
	w, h := m.width, m.height
	header := m.renderArchHeader(w)
	contentH := h - 2 // -1 for the header, -1 for the hint line
	if contentH < 1 {
		contentH = 1
	}
	content := m.arch.screen.View(w, contentH)
	hint := m.arch.screen.HintLine()
	return header + "\n" + content + "\n" + hint
}

// renderArchHeader renders the top bar for the architecture overview overlay.
// Mirrors renderHeader's layout: wordmark ── SECTION ──────────── LABEL
func (m Model) renderArchHeader(w int) string {
	wordmark := core.StyleHeaderTitle.Render("VIBEMENU")
	divL := core.StyleHeaderDeco.Render("  ──  ")
	sectionLabel := core.StyleSectionTitle.Render("  ARCHITECTURE.OVERVIEW")
	leftSeg := " " + wordmark + divL + sectionLabel

	rightSeg := core.StyleHeaderTitle.Render("OVERVIEW") + " "

	leftW := lipgloss.Width(leftSeg)
	rightW := lipgloss.Width(rightSeg)
	gap := w - leftW - rightW - 2
	if gap < 2 {
		gap = 2
	}
	mid := core.StyleHeaderDeco.Render("  " + strings.Repeat("─", gap))
	line := leftSeg + mid + rightSeg
	return core.StyleHeaderBar.Width(w).Render(line)
}

// renderHeader renders the top application bar with editorial typography.
//
//	VIBEMENU  ── 01  SECTION.MANIFEST  ──────────────────  02/08
func (m Model) renderHeader(w int) string {
	sec := m.sections[m.activeSection]

	// Left: VIBEMENU wordmark + section label.
	wordmark := core.StyleHeaderTitle.Render("VIBEMENU")
	divL := core.StyleHeaderDeco.Render("  ──  ")
	idx := core.StyleHeaderDeco.Render(fmt.Sprintf("%02d", m.activeSection+1))
	sectionLabel := core.StyleSectionTitle.Render("  " + strings.ToUpper(sec.ID) + ".MANIFEST")
	modMark := ""
	if m.modified {
		modMark = core.StyleHeaderMod.Render(" [+]")
	}
	leftSeg := " " + wordmark + divL + idx + sectionLabel + modMark

	// Right: position counter.
	pos := core.StyleHeaderTitle.Render(fmt.Sprintf("%02d/%02d", m.activeSection+1, len(m.sections)))
	rightSeg := pos + " "

	leftW := lipgloss.Width(leftSeg)
	rightW := lipgloss.Width(rightSeg)
	gap := w - leftW - rightW - 2
	if gap < 2 {
		gap = 2
	}
	mid := core.StyleHeaderDeco.Render("  " + strings.Repeat("─", gap))

	line := leftSeg + mid + rightSeg
	return core.StyleHeaderBar.Width(w).Render(line)
}

// renderContent renders the main editor area with a description side panel.
//
// Panel sizing strategy:
//   - Always show a 44-column panel when the terminal is ≥ 120 columns wide.
//   - When a KindSelect field is highlighted and a description exists, show
//     the field/option description panel.
//   - Otherwise show a section overview panel.
//   - Below 120 columns the panel is hidden entirely.
func (m Model) renderContent(w int) string {
	ch := m.contentHeight()

	const minFormW = 72   // minimum columns to keep the form usable
	const descPanelW = 44 // fixed width for the right panel
	const minTermW = 120  // minimum terminal width to attempt the panel

	if w >= minTermW && (w-minFormW-1) >= descPanelW {
		contentW := w - descPanelW - 1
		label, value, desc := m.getActiveFieldDescription()
		var panelLines []string
		if desc != "" {
			panelLines = core.FormatDescriptionPanel(label, value, desc, descPanelW, ch)
		} else {
			panelLines = core.FormatSectionPanel(m.activeSection, descPanelW, ch)
		}
		return core.WithDescriptionPanel(m.renderLeft(contentW, ch), panelLines, contentW, descPanelW, ch)
	}

	// Narrow terminal — render the editor at full width.
	return m.renderLeft(w, ch)
}

// renderLeft renders the left-side content area (editor or field list).
func (m Model) renderLeft(w, h int) string {
	if e := m.activeEditor(); e != nil {
		return e.View(w, h)
	}
	sec := m.sections[m.activeSection]
	return m.renderFieldList(w, h, sec)
}

// getActiveFieldDescription returns display info for the currently highlighted
// KindSelect field. Returns ("", "", "") when no description is registered.
func (m Model) getActiveFieldDescription() (label, value, desc string) {
	var f *core.Field
	switch m.activeSection {
	case 1:
		f = m.backendEditor.CurrentField()
	case 2:
		f = m.dataTabEditor.CurrentField()
	case 3:
		f = m.contractsEditor.CurrentField()
	case 4:
		f = m.frontendEditor.CurrentField()
	case 5:
		f = m.infraEditor.CurrentField()
	case 6:
		f = m.crossCutEditor.CurrentField()
	case 7:
		f = m.realizeEditor.CurrentField()
	}
	if f == nil || (f.Kind != core.KindSelect && f.Kind != core.KindMultiSelect) {
		return "", "", ""
	}
	d := core.GetOptionDescription(f.Key, f.Value)
	if d == "" {
		return "", "", ""
	}
	return strings.TrimSpace(f.Label), f.Value, d
}

func (m Model) renderFieldList(w, h int, sec core.Section) string {
	const lineNumW = 4
	const labelW = 14
	const eqW = 3
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	var lines []string
	descLine := core.StyleSectionDesc.Render(fmt.Sprintf("  # %s", sec.Desc))
	lines = append(lines, descLine, "")

	for i, f := range sec.Fields {
		lineNo := i + 1
		isCur := i == m.activeField

		var numStr string
		if isCur {
			numStr = core.StyleCurLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		} else {
			numStr = core.StyleLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		}

		var keyStr string
		if isCur {
			keyStr = core.StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = core.StyleFieldKey.Render(f.Label)
		}

		eq := core.StyleEquals.Render(" = ")

		var valStr string
		if m.mode == core.ModeInsert && isCur && f.Kind == core.KindText {
			valStr = m.textInput.View()
		} else if f.Kind == core.KindSelect {
			arrow := core.StyleSelectArrow.Render(" ▾")
			val := f.DisplayValue()
			if isCur {
				val = core.StyleFieldValActive.Render(val)
			} else {
				val = core.StyleFieldVal.Render(val)
			}
			valStr = val + arrow
		} else {
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" && !isCur {
				dv = core.StyleFieldVal.Foreground(lipgloss.Color(core.ClrFgDim)).Render("_")
			} else if isCur {
				valStr = core.StyleFieldValActive.Render(dv)
			} else {
				valStr = core.StyleFieldVal.Render(dv)
			}
			if valStr == "" {
				valStr = core.StyleFieldVal.Render(dv)
			}
		}

		row := numStr + keyStr + eq + valStr
		if isCur {
			rawW := lipgloss.Width(row)
			if rawW < w {
				row += strings.Repeat(" ", w-rawW)
			}
			row = core.ActiveCurLineStyle().Render(row)
		}
		lines = append(lines, row)
	}

	return core.FillTildes(lines, h)
}

// renderTabBar renders the main section tab bar at the bottom of the content area.
func (m Model) renderTabBar(w int) string {
	sep := core.StyleTabSep.Render("│")
	sepW := lipgloss.Width(sep)
	n := len(m.sections)

	buildTabs := func(labels []string) (string, bool) {
		var parts []string
		for i, lbl := range labels {
			if i == m.activeSection {
				parts = append(parts, core.StyleTabActive.Render(" "+lbl+" "))
			} else {
				parts = append(parts, core.StyleTabInactive.Render(" "+lbl+" "))
			}
		}
		naturalW := lipgloss.Width(strings.Join(parts, sep))
		if naturalW > w {
			return "", false
		}
		extra := w - naturalW
		if extra == 0 {
			return strings.Join(parts, sep), true
		}
		perTab := extra / n
		rem := extra % n
		_ = sepW
		var expanded []string
		for i, lbl := range labels {
			pad := perTab
			if i < rem {
				pad++
			}
			padded := " " + lbl + strings.Repeat(" ", 1+pad)
			if i == m.activeSection {
				expanded = append(expanded, core.StyleTabActive.Render(padded))
			} else {
				expanded = append(expanded, core.StyleTabInactive.Render(padded))
			}
		}
		return strings.Join(expanded, sep), true
	}

	// Level 1: full Abbr labels.
	fullLabels := make([]string, n)
	for i, s := range m.sections {
		fullLabels[i] = s.Abbr
	}
	if tabs, ok := buildTabs(fullLabels); ok {
		return tabs
	}

	// Level 2: icon only (first word of Abbr).
	iconLabels := make([]string, n)
	for i, s := range m.sections {
		parts := strings.Fields(s.Abbr)
		if len(parts) > 0 {
			iconLabels[i] = parts[0]
		} else {
			iconLabels[i] = fmt.Sprintf("%d", i+1)
		}
	}
	if tabs, ok := buildTabs(iconLabels); ok {
		return tabs
	}

	// Level 3: bare index numbers.
	numLabels := make([]string, n)
	for i := range m.sections {
		numLabels[i] = fmt.Sprintf("%d", i+1)
	}
	tabs, _ := buildTabs(numLabels)
	return tabs
}

// renderStatusLine renders the vim-style status bar.
//
//	[NRM]  SECTION  ──────────────────────────────────────  02/08
func (m Model) renderStatusLine(w int) string {
	glyphs := core.ModeSpinFrames[0]

	var modeStyle lipgloss.Style
	var modeText string
	switch m.activeMode() {
	case core.ModeNormal:
		modeStyle = core.StyleNormalMode
		modeText = "NRM"
	case core.ModeInsert:
		modeStyle = core.StyleInsertMode
		modeText = "INS"
	case core.ModeCommand:
		modeStyle = core.StyleCommandMode
		modeText = "CMD"
	}
	modeLabel := modeStyle.Render(glyphs[0] + " " + modeText + " " + glyphs[1])

	// Right segment: section + position counter.
	sec := m.sections[m.activeSection]
	sectionName := strings.ToUpper(sec.ID)
	pos := fmt.Sprintf("%02d/%02d", m.activeSection+1, len(m.sections))
	rightSeg := core.StyleStatusRight.Render("  "+sectionName+"  ") +
		core.StyleStatusSegmentPos.Render(pos+" ")

	// Centre: status message.
	msg := ""
	if m.cmd.status != "" {
		if m.cmd.isErr {
			msg = core.StyleMsgErr.Render(" ✗ " + m.cmd.status + " ")
		} else {
			msg = core.StyleMsgOK.Render(" ✓ " + m.cmd.status + " ")
		}
	}

	leftW := lipgloss.Width(modeLabel)
	rightW := lipgloss.Width(rightSeg)
	msgW := lipgloss.Width(msg)
	gapW := w - leftW - rightW - msgW
	if gapW < 1 {
		gapW = 1
	}

	line := modeLabel + core.StyleStatusLine.Render(strings.Repeat(" ", gapW/2)) + msg +
		core.StyleStatusLine.Render(strings.Repeat(" ", gapW-gapW/2)) + rightSeg
	return line
}

func (m Model) renderCmdLine(w int) string {
	if m.mode == core.ModeCommand {
		cursor := core.StyleCursor.Render(" ")
		return core.StyleCmdLine.Render(":"+m.cmd.buffer) + cursor
	}

	var line string
	if e := m.activeEditor(); e != nil {
		line = e.HintLine()
	} else {
		switch m.mode {
		case core.ModeNormal:
			hints := []string{
				core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" navigate"),
				core.StyleHelpKey.Render("i") + core.StyleHelpDesc.Render(" insert"),
				core.StyleHelpKey.Render("Tab") + core.StyleHelpDesc.Render(" section"),
				core.StyleHelpKey.Render("Enter") + core.StyleHelpDesc.Render(" cycle"),
				core.StyleHelpKey.Render(":w") + core.StyleHelpDesc.Render(" save"),
				core.StyleHelpKey.Render(":q") + core.StyleHelpDesc.Render(" quit"),
			}
			sep := core.StyleHelpDesc.Render("  │  ")
			line = "  " + strings.Join(hints, sep)
		case core.ModeInsert:
			line = core.StyleInsertMode.Render(" ── INSERT ── ") + core.StyleHelpDesc.Render("  Esc: normal  │  Tab: next field")
		}
	}

	if lipgloss.Width(line) > w {
		line = lipgloss.NewStyle().MaxWidth(w).Render(line)
	}
	return line
}
