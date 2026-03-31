package ui

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
// Parameters:
//   - w         : total available width
//   - fields    : the slice of Field values to render
//   - activeIdx : which field index is currently focused
//   - insertMode: true when the active field is being edited
//   - input     : the live textinput widget (used only when insertMode && KindText)
func renderFormFields(w int, fields []Field, activeIdx int, insertMode bool, input textinput.Model) []string {
	if len(fields) == 0 {
		return nil
	}
	const labelW = 14
	const eqW = 3
	const lineNumW = 4
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	lines := make([]string, 0, len(fields))
	for i, f := range fields {
		isCur := i == activeIdx

		// Line number
		var lineNo string
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		} else {
			lineNo = StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}

		// Key label
		var keyStr string
		if isCur {
			keyStr = StyleFieldKeyActive.Render(f.Label)
		} else {
			keyStr = StyleFieldKey.Render(f.Label)
		}

		// Equals
		eq := StyleEquals.Render(" = ")

		// Value
		var valStr string
		switch {
		case insertMode && isCur && f.Kind == KindText:
			valStr = input.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			valStr = val + StyleSelectArrow.Render(" ▾")
		default:
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" {
				valStr = StyleSectionDesc.Render("_")
			} else if isCur {
				valStr = StyleFieldValActive.Render(dv)
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
			row = StyleCurLine.Render(row)
		}
		lines = append(lines, row)
	}
	return lines
}

// renderFormFieldsWithDisabled is like renderFormFields but also renders
// disabled (grayed-out) fields with a dash value and no highlighting.
func renderFormFieldsWithDisabled(w int, fields []Field, activeIdx int, insertMode bool, input textinput.Model, isDisabled func([]Field, int) bool) []string {
	if len(fields) == 0 {
		return nil
	}
	const labelW = 14
	const eqW = 3
	const lineNumW = 4
	valW := w - lineNumW - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	lines := make([]string, 0, len(fields))
	for i, f := range fields {
		isCur := i == activeIdx
		disabled := isDisabled(fields, i)

		var lineNo string
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
		} else {
			lineNo = StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		}

		var keyStr string
		switch {
		case disabled:
			keyStr = StyleSectionDesc.Render(f.Label)
		case isCur:
			keyStr = StyleFieldKeyActive.Render(f.Label)
		default:
			keyStr = StyleFieldKey.Render(f.Label)
		}

		eq := StyleEquals.Render(" = ")

		var valStr string
		switch {
		case disabled:
			valStr = StyleSectionDesc.Render("—")
		case insertMode && isCur && f.Kind == KindText:
			valStr = input.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			valStr = val + StyleSelectArrow.Render(" ▾")
		default:
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" {
				valStr = StyleSectionDesc.Render("_")
			} else if isCur {
				valStr = StyleFieldValActive.Render(dv)
			} else {
				valStr = StyleFieldVal.Render(dv)
			}
		}

		row := lineNo + keyStr + eq + valStr
		if isCur && !disabled {
			raw := lipgloss.Width(row)
			if raw < w {
				row += strings.Repeat(" ", w-raw)
			}
			row = StyleCurLine.Render(row)
		}
		lines = append(lines, row)
	}
	return lines
}

// renderSubTabBar renders a horizontal sub-tab bar and returns the string.
func renderSubTabBar(labels []string, active int) string {
	var parts []string
	for i, lbl := range labels {
		if i == active {
			parts = append(parts, StyleTabActive.Render(" "+lbl+" "))
		} else {
			parts = append(parts, StyleTabInactive.Render(" "+lbl+" "))
		}
	}
	return "  " + strings.Join(parts, "")
}

// renderListItem renders one row in a list view.
func renderListItem(w int, isCur bool, arrow, name, extra string) string {
	var arrowStr, nameStr string
	if isCur {
		arrowStr = StyleCurLineNum.Render(arrow)
		nameStr = StyleFieldKeyActive.Render(name)
	} else {
		arrowStr = "  ▸ "
		nameStr = StyleFieldKey.Render(name)
	}
	row := arrowStr + nameStr
	if extra != "" {
		row += StyleSectionDesc.Render("  "+extra)
	}
	if isCur {
		raw := lipgloss.Width(row)
		if raw < w {
			row += strings.Repeat(" ", w-raw)
		}
		row = StyleCurLine.Render(row)
	}
	return row
}

// hintBar builds a hint line from key-description pairs.
func hintBar(pairs ...string) string {
	if len(pairs)%2 != 0 {
		return ""
	}
	var hints []string
	for i := 0; i < len(pairs); i += 2 {
		hints = append(hints, StyleHelpKey.Render(pairs[i])+StyleHelpDesc.Render(" "+pairs[i+1]))
	}
	return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
}

// fieldGet returns the DisplayValue for the field with the given key in a slice.
func fieldGet(fields []Field, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return f.DisplayValue()
		}
	}
	return ""
}

// setFieldValue sets the value (and SelIdx for select fields) for the field
// with the given key in a slice, returning the modified slice.
func setFieldValue(fields []Field, key, val string) []Field {
	for i := range fields {
		if fields[i].Key != key {
			continue
		}
		fields[i].Value = val
		if fields[i].Kind == KindSelect {
			for j, opt := range fields[i].Options {
				if opt == val {
					fields[i].SelIdx = j
					break
				}
			}
		}
		return fields
	}
	return fields
}

// newFormInput creates a standard textinput for use in form editors.
func newFormInput() textinput.Model {
	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	fi.CursorStyle = StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))
	return fi
}
