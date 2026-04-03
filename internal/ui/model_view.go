package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
			StyleMsgErr.Render(msg))
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
		return placeOverlay(base, modal, x, y)
	}

	return base
}

func (m Model) renderBaseView() string {
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

func (m Model) renderHeader(w int) string {
	sec := m.sections[m.activeSection]
	modMark := ""
	if m.modified {
		modMark = StyleHeaderMod.Render(" [+]")
	}

	deco := StyleHeaderDeco.Render(headerDecoFrames[AnimFrame])
	title := deco + " " + StyleSectionTitle.Render(sec.ID+".manifest") + modMark

	counter := StyleHeaderDeco.Render(headerDecoFrames[1-AnimFrame]) + " " +
		StyleHeaderTitle.Render(fmt.Sprintf("[%02d/%02d]", m.activeSection+1, len(m.sections)))

	titleW := lipgloss.Width(title)
	counterW := lipgloss.Width(counter)
	gap := w - titleW - counterW - 2
	if gap < 1 {
		gap = 1
	}
	line := " " + title + strings.Repeat(" ", gap) + counter + " "
	return StyleHeaderBar.Width(w).Render(line)
}

func (m Model) renderContent(w int) string {
	ch := m.contentHeight()
	if e := m.activeEditor(); e != nil {
		return e.View(w, ch)
	}
	// Fallback: generic field list for sections without a delegated editor.
	sec := m.sections[m.activeSection]
	return m.renderFieldList(w, ch, sec)
}

func (m Model) renderFieldList(w, h int, sec Section) string {
	const lineNumW = 4
	const labelW = 14
	const eqW = 3
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	var lines []string
	descLine := StyleSectionDesc.Render(fmt.Sprintf("  # %s", sec.Desc))
	lines = append(lines, descLine, "")

	for i, f := range sec.Fields {
		lineNo := i + 1
		isCur := i == m.activeField

		var numStr string
		if isCur {
			numStr = StyleCurLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		} else {
			numStr = StyleLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		}

		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		eq := StyleEquals.Render(" = ")

		var valStr string
		if m.mode == ModeInsert && isCur && f.Kind == KindText {
			valStr = m.textInput.View()
		} else if f.Kind == KindSelect {
			arrow := StyleSelectArrow.Render(" ▾")
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			valStr = val + arrow
		} else {
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" && !isCur {
				dv = StyleFieldVal.Foreground(lipgloss.Color(clrFgDim)).Render("_")
			} else if isCur {
				valStr = StyleFieldValActive.Render(dv)
			} else {
				valStr = StyleFieldVal.Render(dv)
			}
			if valStr == "" {
				valStr = StyleFieldVal.Render(dv)
			}
		}

		row := numStr + keyStr + eq + valStr
		if isCur {
			rawW := lipgloss.Width(row)
			if rawW < w {
				row += strings.Repeat(" ", w-rawW)
			}
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)
	}

	return fillTildes(lines, h)
}

func (m Model) renderTabBar(w int) string {
	sep := StyleTabSep.Render("│")
	sepW := lipgloss.Width(sep)
	n := len(m.sections)

	// buildTabs renders labels as tabs. If the natural width fits within w,
	// it distributes any extra space evenly among the tabs so the bar fills
	// the full terminal width. Returns (rendered, fits).
	buildTabs := func(labels []string) (string, bool) {
		var parts []string
		for i, lbl := range labels {
			if i == m.activeSection {
				parts = append(parts, StyleTabActive.Render(" "+lbl+" "))
			} else {
				parts = append(parts, StyleTabInactive.Render(" "+lbl+" "))
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
		// Distribute extra space: add padding inside each tab's right side.
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
				expanded = append(expanded, StyleTabActive.Render(padded))
			} else {
				expanded = append(expanded, StyleTabInactive.Render(padded))
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

	// Level 2: icon only (first word of Abbr, e.g. "⚡").
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


func (m Model) renderStatusLine(w int) string {
	spin := modeSpinFrames[AnimFrame]
	var modeLabel string
	switch m.activeMode() {
	case ModeNormal:
		modeLabel = StyleNormalMode.Render(spin[0] + " NRM " + spin[1])
	case ModeInsert:
		modeLabel = StyleInsertMode.Render(spin[0] + " INS " + spin[1])
	case ModeCommand:
		modeLabel = StyleCommandMode.Render(spin[0] + " CMD " + spin[1])
	}

	sec := m.sections[m.activeSection]
	pos := fmt.Sprintf("%02d/%02d", m.activeSection+1, len(m.sections))
	right := StyleStatusRight.Render(fmt.Sprintf("  %s.manifest  %s  ▪ ", sec.ID, pos))

	msg := ""
	if m.cmd.status != "" {
		if m.cmd.isErr {
			msg = StyleMsgErr.Render("✗ " + m.cmd.status)
		} else {
			msg = StyleMsgOK.Render("✓ " + m.cmd.status)
		}
	}

	leftW := lipgloss.Width(modeLabel)
	rightW := lipgloss.Width(right)
	msgW := lipgloss.Width(msg)
	gapW := w - leftW - rightW - msgW
	if gapW < 1 {
		gapW = 1
	}

	line := modeLabel + strings.Repeat(" ", gapW/2) + msg + StyleStatusLine.Render(strings.Repeat(" ", gapW-gapW/2)) + right
	return line
}

func (m Model) renderCmdLine(w int) string {
	if m.mode == ModeCommand {
		cursor := StyleCursor.Render(" ")
		return StyleCmdLine.Render(":"+m.cmd.buffer) + cursor
	}

	// Delegate hint line to the active sub-editor, with a fallback for the
	// generic field-list renderer (which has no delegated editor).
	var line string
	if e := m.activeEditor(); e != nil {
		line = e.HintLine()
	} else {
		switch m.mode {
		case ModeNormal:
			hints := []string{
				StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
				StyleHelpKey.Render("i") + StyleHelpDesc.Render(" insert"),
				StyleHelpKey.Render("Tab") + StyleHelpDesc.Render(" section"),
				StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" cycle"),
				StyleHelpKey.Render(":w") + StyleHelpDesc.Render(" save"),
				StyleHelpKey.Render(":q") + StyleHelpDesc.Render(" quit"),
			}
			sep := StyleHelpDesc.Render("  │  ")
			line = "  " + strings.Join(hints, sep)
		case ModeInsert:
			line = StyleInsertMode.Render(" ▷ INSERT ◁ ") + StyleHelpDesc.Render("  Esc: normal  │  Tab: next field")
		}
	}

	if lipgloss.Width(line) > w {
		line = line[:w-1]
	}
	return line
}
