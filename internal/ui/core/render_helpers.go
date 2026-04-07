package core

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// renderFormFields renders a list of Fields into display lines using the
// shared vim-style form layout. It is the canonical rendering helper for all
// editors and can be called from any sub-editor's view methods.
//
// When ddOpen is true, an inline scrollable dropdown is injected below the
// active KindSelect/KindMultiSelect field; ddOptIdx is the highlighted index.
// Pass ddOpen=false and ddOptIdx=0 for plain form rendering without dropdowns.
func RenderFormFields(w int, fields []Field, activeIdx int, insertMode bool, input textinput.Model, ddOpen bool, ddOptIdx int) []string {
	if len(fields) == 0 {
		return nil
	}
	const labelW = 14
	const eqW = 3
	const lineNumW = 4
	const ddIndent = lineNumW + labelW + eqW // 21 spaces to align with value column
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	lines := make([]string, 0, len(fields))
	for i, f := range fields {
		isCur := i == activeIdx

		// Capture the cursor-row background once per row so every individually-
		// styled component (line number, label, separator, value) carries it.
		// Without this the ANSI reset at the end of each styled segment clears the
		// background set by the outer row wrapper, leaving only bare spaces lit up.
		var curBG lipgloss.TerminalColor
		if isCur {
			curBG = ActiveCurLineStyle().GetBackground()
		}

		var lineNo string
		if isCur {
			lineNo = StyleCurLineNum.Background(curBG).Render(fmt.Sprintf("%3d ", i+1))
		} else {
			lineNo = StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}

		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Background(curBG).Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		var eq string
		if isCur {
			eq = StyleEquals.Background(curBG).Render(" = ")
		} else {
			eq = StyleEquals.Render(" = ")
		}

		var valStr string
		switch {
		case insertMode && isCur && f.CanEditAsText():
			valStr = input.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			// Reserve 2 chars for the " ▾/▴" arrow to prevent line wrapping.
			if valW > 3 && len(val) > valW-2 {
				val = val[:valW-3] + "…"
			}
			if isCur {
				val = StyleFieldValActive.Background(curBG).Render(val)
				arrow := StyleSelectArrow.Background(curBG)
				if ddOpen {
					valStr = val + arrow.Render(" ▴")
				} else {
					valStr = val + arrow.Render(" ▾")
				}
			} else {
				val = StyleFieldVal.Render(val)
				if ddOpen {
					valStr = val + StyleSelectArrow.Render(" ▴")
				} else {
					valStr = val + StyleSelectArrow.Render(" ▾")
				}
			}
		case f.Kind == KindMultiSelect && f.ColorSwatch:
			if isCur {
				arrow := StyleSelectArrow.Background(curBG)
				arrowStr := arrow.Render(" ▾")
				if ddOpen {
					arrowStr = arrow.Render(" ▴")
				}
				if len(f.SelectedIdxs) == 0 {
					valStr = StyleFieldValActive.Background(curBG).Render("(none)") + arrowStr
				} else {
					var pieces []string
					for _, idx := range f.SelectedIdxs {
						if idx >= 0 && idx < len(f.Options) {
							hex := f.Options[idx]
							if IsCustomOption(hex) {
								hex = f.CustomText
							}
							if strings.HasPrefix(hex, "#") {
								swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Background(curBG).Bold(true).Render("■")
								pieces = append(pieces, swatch+StyleFieldVal.Background(curBG).Render(" "+hex))
							} else {
								pieces = append(pieces, StyleSectionDesc.Background(curBG).Render("■")+StyleFieldVal.Background(curBG).Render(" custom"))
							}
						}
					}
					val := strings.Join(pieces, StyleSectionDesc.Background(curBG).Render(" · "))
					if lipgloss.Width(val) > valW-2 {
						val = StyleFieldVal.Background(curBG).Render(fmt.Sprintf("%d colors", len(f.SelectedIdxs)))
					}
					valStr = val + arrowStr
				}
			} else {
				arrow := StyleSelectArrow.Render(" ▾")
				if ddOpen {
					arrow = StyleSelectArrow.Render(" ▴")
				}
				if len(f.SelectedIdxs) == 0 {
					valStr = StyleFieldVal.Render("(none)") + arrow
				} else {
					var pieces []string
					for _, idx := range f.SelectedIdxs {
						if idx >= 0 && idx < len(f.Options) {
							hex := f.Options[idx]
							if IsCustomOption(hex) {
								hex = f.CustomText
							}
							if strings.HasPrefix(hex, "#") {
								swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true).Render("■")
								pieces = append(pieces, swatch+StyleFieldVal.Render(" "+hex))
							} else {
								pieces = append(pieces, StyleSectionDesc.Render("■")+StyleFieldVal.Render(" custom"))
							}
						}
					}
					val := strings.Join(pieces, StyleSectionDesc.Render(" · "))
					if lipgloss.Width(val) > valW-2 {
						val = StyleFieldVal.Render(fmt.Sprintf("%d colors", len(f.SelectedIdxs)))
					}
					valStr = val + arrow
				}
			}
		case f.Kind == KindMultiSelect:
			val := f.DisplayValue()
			if val == "" {
				val = "(none)"
			}
			// Reserve 2 chars for the " ▾/▴" arrow to prevent line wrapping.
			if valW > 3 && len(val) > valW-2 {
				val = val[:valW-3] + "…"
			}
			if isCur {
				val = StyleFieldValActive.Background(curBG).Render(val)
				arrow := StyleSelectArrow.Background(curBG)
				if ddOpen {
					valStr = val + arrow.Render(" ▴")
				} else {
					valStr = val + arrow.Render(" ▾")
				}
			} else {
				val = StyleFieldVal.Render(val)
				if ddOpen {
					valStr = val + StyleSelectArrow.Render(" ▴")
				} else {
					valStr = val + StyleSelectArrow.Render(" ▾")
				}
			}
		default:
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" {
				if isCur {
					valStr = StyleSectionDesc.Background(curBG).Render("_")
				} else {
					valStr = StyleSectionDesc.Render("_")
				}
			} else if isCur {
				valStr = StyleFieldValActive.Background(curBG).Render(dv)
			} else {
				valStr = StyleFieldVal.Render(dv)
			}
		}

		row := lineNo + keyStr + eq + valStr
		if isCur {
			raw := lipgloss.Width(row)
			if raw < w {
				row += strings.Repeat(" ", w-raw)
			}
			row = ActiveCurLineStyle().Render(row)
		}
		lines = append(lines, row)

		// When typing a custom hex for a ColorSwatch field, show the currently
		// selected colors as swatches on a hint line below the text input.
		if insertMode && isCur && f.ColorSwatch && len(f.SelectedIdxs) > 0 {
			indent := strings.Repeat(" ", ddIndent)
			var pieces []string
			for _, idx := range f.SelectedIdxs {
				if idx < 0 || idx >= len(f.Options) {
					continue
				}
				hex := f.Options[idx]
				if IsCustomOption(hex) {
					hex = f.CustomText
				}
				if strings.HasPrefix(hex, "#") {
					swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true).Render("■")
					pieces = append(pieces, swatch+" "+StyleSectionDesc.Render(hex))
				}
			}
			if len(pieces) > 0 {
				lines = append(lines, indent+StyleSectionDesc.Render("selected: ")+strings.Join(pieces, StyleSectionDesc.Render(" · ")))
			}
		}

		// Inject scrollable dropdown options below the active select/multiselect field.
		// Skip dropdown when in insert mode for a custom-option field (text input is active).
		if isCur && ddOpen && (f.Kind == KindSelect || f.Kind == KindMultiSelect) && !(insertMode && f.CanEditAsText()) {
			const ddMaxVisible = 8
			indent := strings.Repeat(" ", ddIndent)
			total := len(f.Options)

			// Compute scroll window centered around the highlighted option.
			start := ddOptIdx - ddMaxVisible/2
			if start+ddMaxVisible > total {
				start = total - ddMaxVisible
			}
			if start < 0 {
				start = 0
			}
			end := start + ddMaxVisible
			if end > total {
				end = total
			}

			if start > 0 {
				lines = append(lines, StyleSectionDesc.Render(fmt.Sprintf("%s↑ %d more", indent, start)))
			}

			for j := start; j < end; j++ {
				opt := f.Options[j]
				isHL := j == ddOptIdx
				var optRow string
				if f.ColorSwatch && IsCustomOption(opt) {
					// Custom hex entry row: show a live preview swatch of the typed hex.
					previewHex := f.CustomText
					var swatch string
					if strings.HasPrefix(previewHex, "#") {
						swatch = lipgloss.NewStyle().Foreground(lipgloss.Color(previewHex)).Bold(true).Render("■")
					} else {
						swatch = StyleSectionDesc.Render("■")
					}
					check := "  "
					if f.IsMultiSelected(j) {
						check = StyleNeonGreen.Render("✓") + " "
					}
					label := opt
					if previewHex != "" {
						label = opt + " " + previewHex
					} else {
						label = opt + " (enter hex)"
					}
					if isHL {
						optRow = indent + StyleFieldValActive.Render("► ") + check + swatch + " " + StyleFieldValActive.Render(label)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = ActiveCurLineStyle().Render(optRow)
					} else {
						optRow = indent + "   " + check + swatch + " " + StyleFieldVal.Render(label)
					}
				} else if f.ColorSwatch && strings.HasPrefix(opt, "#") {
					// Render a colored swatch block (foreground-only, survives cursor highlight).
					swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(opt)).Bold(true).Render("■")
					check := "  "
					if f.IsMultiSelected(j) {
						check = StyleNeonGreen.Render("✓") + " "
					}
					if isHL {
						optRow = indent + StyleFieldValActive.Render("► ") + check + swatch + " " + StyleFieldValActive.Render(opt)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = ActiveCurLineStyle().Render(optRow)
					} else {
						optRow = indent + "   " + check + swatch + " " + StyleFieldVal.Render(opt)
					}
				} else if f.Kind == KindMultiSelect {
					check := "  "
					if f.IsMultiSelected(j) {
						check = StyleNeonGreen.Render("✓") + " "
					}
					if isHL {
						optRow = indent + StyleFieldValActive.Render("► "+check+opt)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = ActiveCurLineStyle().Render(optRow)
					} else {
						optRow = indent + StyleFieldVal.Render("  "+check+opt)
					}
				} else {
					if isHL {
						optRow = indent + StyleFieldValActive.Render("► "+opt)
						rw := lipgloss.Width(optRow)
						if rw < w {
							optRow += strings.Repeat(" ", w-rw)
						}
						optRow = ActiveCurLineStyle().Render(optRow)
					} else {
						optRow = indent + StyleFieldVal.Render("  "+opt)
					}
				}
				lines = append(lines, optRow)
			}

			if end < total {
				lines = append(lines, StyleSectionDesc.Render(fmt.Sprintf("%s↓ %d more", indent, total-end)))
			}
		}
	}
	return lines
}

// renderSubTabBar renders a horizontal sub-tab bar scaled to fit within w columns.
// Three levels of fidelity:
//  1. Full labels — all tabs with their text labels, expanded to fill w.
//  2. Compact — active tab keeps its label; others shrink to their 1-based index, expanded to fill w.
//  3. Minimal — "[N/total] LABEL" single-item indicator.
func RenderSubTabBar(labels []string, active, w int) string {
	sep := StyleTabSep.Render("│")
	n := len(labels)
	const margin = 2 // leading "  " prefix

	// build renders lbls as a tab bar. If the natural width fits within w,
	// extra space is distributed evenly among tabs so the bar fills the full width.
	// Returns (rendered, fits).
	build := func(lbls []string) (string, bool) {
		var parts []string
		for i, lbl := range lbls {
			if i == active {
				parts = append(parts, StyleTabActive.Render(" "+lbl+" "))
			} else {
				parts = append(parts, StyleTabInactive.Render(" "+lbl+" "))
			}
		}
		naturalW := margin + lipgloss.Width(strings.Join(parts, sep))
		if w > 0 && naturalW > w {
			return "", false
		}
		if w <= 0 || naturalW >= w {
			return strings.Repeat(" ", margin) + strings.Join(parts, sep), true
		}
		// Distribute extra space inside each tab's right padding.
		extra := w - naturalW
		perTab := extra / n
		rem := extra % n
		var expanded []string
		for i, lbl := range lbls {
			pad := perTab
			if i < rem {
				pad++
			}
			padded := " " + lbl + strings.Repeat(" ", 1+pad)
			if i == active {
				expanded = append(expanded, StyleTabActive.Render(padded))
			} else {
				expanded = append(expanded, StyleTabInactive.Render(padded))
			}
		}
		return strings.Repeat(" ", margin) + strings.Join(expanded, sep), true
	}

	// Level 1: full labels.
	if result, ok := build(labels); ok {
		return result
	}

	// Level 2: active tab shows label, others collapse to index number.
	compact := make([]string, n)
	for i := range labels {
		if i == active {
			compact[i] = labels[i]
		} else {
			compact[i] = fmt.Sprintf("%d", i+1)
		}
	}
	if result, ok := build(compact); ok {
		return result
	}

	// Level 3: single "[N/total] LABEL" indicator — always fits.
	pos := fmt.Sprintf("[%d/%d]", active+1, n)
	lbl := labels[active]
	maxLblW := w - lipgloss.Width("  "+pos+" ") - 4
	if maxLblW > 0 && len([]rune(lbl)) > maxLblW {
		lbl = string([]rune(lbl)[:maxLblW-1]) + "…"
	}
	return "  " + StyleTabInactive.Render(" "+pos+" ") + StyleTabActive.Render(" "+lbl+" ")
}

// renderListItem renders one row in a list view.
func RenderListItem(w int, isCur bool, arrow, name, extra string) string {
	var arrowStr, nameStr string
	if isCur {
		arrowStr = StyleCurLineNum.Render(arrow)
		nameStr = StyleFieldKeyActive.Render(name)
	} else {
		arrowStr = StyleFgDimStyle.Render("  ► ")
		nameStr = StyleFieldKey.Render(name)
	}
	row := arrowStr + nameStr
	if extra != "" {
		row += StyleSectionDesc.Render("  " + extra)
	}
	// Truncate row to terminal width to prevent horizontal overflow.
	if w > 0 {
		if rw := lipgloss.Width(row); rw > w {
			row = lipgloss.NewStyle().MaxWidth(w).Render(row)
		}
	}
	if isCur {
		raw := lipgloss.Width(row)
		if raw < w {
			row += strings.Repeat(" ", w-raw)
		}
		row = ActiveCurLineStyle().Render(row)
	}
	return row
}

// StyleFgDimStyle is an inline style for dim inactive list arrows.
var StyleFgDimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

// hintBar builds a hint line from key-description pairs.
func HintBar(pairs ...string) string {
	if len(pairs)%2 != 0 {
		return ""
	}
	var hints []string
	for i := 0; i+1 < len(pairs); i += 2 {
		hints = append(hints, StyleHelpKey.Render(pairs[i])+StyleHelpDesc.Render(" "+pairs[i+1]))
	}
	sep := StyleHelpDesc.Render("  │  ")
	return "  " + strings.Join(hints, sep)
}

// withDescriptionPanel lays out left content and a pre-styled description panel
// side-by-side. It does NOT rune-slice or re-style the right lines, so ANSI
// escape sequences in descLines are preserved intact.
func WithDescriptionPanel(left string, descLines []string, leftW, descW, h int) string {
	leftLines := strings.Split(strings.TrimRight(left, "\n"), "\n")
	sep := StyleArtSep.Render("│")

	var sb strings.Builder
	for i := 0; i < h; i++ {
		var leftLine string
		if i < len(leftLines) {
			leftLine = leftLines[i]
		}
		lw := lipgloss.Width(leftLine)
		if lw < leftW {
			leftLine += strings.Repeat(" ", leftW-lw)
		}

		var rightLine string
		if i < len(descLines) {
			rightLine = descLines[i]
		}

		sb.WriteString(leftLine)
		sb.WriteString(sep)
		sb.WriteString(rightLine)
		if i < h-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
	return sb.String()
}
