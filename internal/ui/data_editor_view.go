package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

// ── View ──────────────────────────────────────────────────────────────────────

// View renders the editor into a w×h content block.
func (de DataEditor) View(w, h int) string {
	de.width = w
	de.formInput.Width = w - 22
	switch de.view {
	case deViewEntities:
		return de.viewEntities(w, h)
	case deViewEntitySettings:
		return de.viewEntitySettings(w, h)
	case deViewColumns:
		return de.viewColumns(w, h)
	case deViewColForm:
		return de.viewColForm(w, h)
	}
	return ""
}

func (de DataEditor) viewEntities(w, h int) string {
	const entListHeaderH = 2
	var header []string
	header = append(header,
		StyleSectionDesc.Render("  # Entities — a: add  d: delete  Enter: settings & columns"),
		"",
	)
	var lines []string

	if len(de.Entities) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no entities yet — press 'a' to add one)"))
	} else {
		for i, ent := range de.Entities {
			isCur := i == de.entityIdx
			nCols := len(ent.Columns)
			colLabel := fmt.Sprintf("%d col", nCols)

			arrow := "  ▸ "
			nameStr := ent.Name
			if isCur {
				arrow = StyleCurLineNum.Render("  ▶ ")
				nameStr = StyleFieldKeyActive.Render(nameStr)
			} else {
				nameStr = StyleFieldKey.Render(nameStr)
			}

			// Database badge
			dbBadge := ""
			if ent.Database != "" {
				dbBadge = StyleSectionTitle.Render("[" + ent.Database + "]")
			} else {
				dbBadge = StyleSectionDesc.Render("[?]")
			}

			// Cache badge
			cacheBadge := ""
			if ent.Cached {
				cs := ent.CacheStore
				if cs == "" {
					cs = "cache"
				}
				ttl := ""
				if ent.CacheTTL != "" {
					ttl = " " + ent.CacheTTL
				}
				cacheBadge = " " + StyleMsgOK.Render("⚡"+cs+ttl)
			}

			pad := max(1, 22-len(ent.Name))
			cols := StyleSectionDesc.Render(colLabel)
			row := arrow + nameStr + strings.Repeat(" ", pad) + dbBadge + cacheBadge + "  " + cols

			if isCur {
				raw := lipgloss.Width(row)
				if raw < w {
					row += strings.Repeat(" ", w-raw)
				}
				row = activeCurLineStyle().Render(row)
			}
			lines = append(lines, row)
		}
	}

	lines = viewportSlice(lines, de.entityIdx, h-entListHeaderH)
	all := append(header, lines...)
	if de.internalMode == deNaming && de.nameTarget == "entity" {
		all = append(all, "")
		all = append(all, StyleTextAreaLabel.Render("  New entity: ")+de.nameInput.View())
	}

	return fillTildes(all, h)
}

func (de DataEditor) viewEntitySettings(w, h int) string {
	if de.entityIdx >= len(de.Entities) {
		return fillTildes(nil, h)
	}
	ent := de.Entities[de.entityIdx]
	var lines []string

	breadcrumb := StyleSectionDesc.Render("  ← ") + StyleSectionTitle.Render(ent.Name) +
		StyleSectionDesc.Render("  (c: columns  b: back)")
	lines = append(lines, breadcrumb, "")

	const labelW = 14
	const eqW = 3
	valW := w - 4 - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	for i, f := range de.entForm {
		isCur := i == de.entFormIdx
		disabled := isEntFormFieldDisabled(de.entForm, i)

		lineNo := StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
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
		case de.internalMode == deInsert && isCur && f.Kind == KindText:
			valStr = de.formInput.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			if isCur && de.dd.Open {
				valStr = val + StyleSelectArrow.Render(" ▴")
			} else {
				valStr = val + StyleSelectArrow.Render(" ▾")
			}
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
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)

		// Inject dropdown options below the active KindSelect field
		if isCur && de.dd.Open && !disabled && f.Kind == KindSelect {
			const ddIndent = 4 + 14 + 3 // lineNumW + labelW + eqW
			indent := strings.Repeat(" ", ddIndent)
			for j, opt := range f.Options {
				isHL := j == de.dd.OptIdx
				var optRow string
				if isHL {
					optRow = indent + StyleFieldValActive.Render("► "+opt)
					rw := lipgloss.Width(optRow)
					if rw < w {
						optRow += strings.Repeat(" ", w-rw)
					}
					optRow = activeCurLineStyle().Render(optRow)
				} else {
					optRow = indent + StyleFieldVal.Render("  "+opt)
				}
				lines = append(lines, optRow)
			}
		}
	}

	return fillTildes(lines, h)
}

func (de DataEditor) viewColumns(w, h int) string {
	if de.entityIdx >= len(de.Entities) {
		return fillTildes(nil, h)
	}
	ent := de.Entities[de.entityIdx]
	var lines []string

	dbLabel := ""
	if ent.Database != "" {
		dbLabel = "  " + StyleSectionTitle.Render("["+ent.Database+"]")
	}
	breadcrumb := StyleSectionDesc.Render("  ← ") + StyleSectionTitle.Render(ent.Name) + dbLabel
	lines = append(lines, breadcrumb, "")

	if len(ent.Columns) == 0 {
		lines = append(lines, StyleSectionDesc.Render("  (no columns yet — press 'a' to add one)"))
	} else {
		for i, col := range ent.Columns {
			isCur := i == de.columnIdx

			numStr := fmt.Sprintf("%3d ", i+1)
			if isCur {
				numStr = StyleCurLineNum.Render(numStr)
			} else {
				numStr = StyleLineNum.Render(numStr)
			}

			typeStr := string(col.Type)
			if col.Length != "" {
				typeStr += "(" + col.Length + ")"
			}

			var badges []string
			if col.PrimaryKey {
				badges = append(badges, StyleSelectArrow.Render("PK"))
			}
			if !col.Nullable {
				badges = append(badges, StyleSectionDesc.Render("NOT NULL"))
			}
			if col.Unique {
				badges = append(badges, StyleMsgOK.Render("UNIQUE"))
			}
			if col.ForeignKey != nil {
				ref := fmt.Sprintf("FK→%s.%s", col.ForeignKey.RefEntity, col.ForeignKey.RefColumn)
				onDel := ""
				if col.ForeignKey.OnDelete != "" && col.ForeignKey.OnDelete != manifest.CascadeNoAction {
					onDel = " " + string(col.ForeignKey.OnDelete)
				}
				badges = append(badges, StyleSectionTitle.Render(ref+onDel))
			}
			if col.Index {
				idxType := string(col.IndexType)
				if idxType == "" {
					idxType = "idx"
				}
				badges = append(badges, StyleHelpKey.Render(idxType))
			}

			badgeStr := ""
			if len(badges) > 0 {
				badgeStr = "  " + strings.Join(badges, " ")
			}

			colName := col.Name
			if isCur {
				colName = StyleFieldKeyActive.Render(colName)
			} else {
				colName = StyleFieldKey.Render(colName)
			}

			pad := max(1, 20-len(col.Name))
			typeRendered := StyleFieldVal.Render(fmt.Sprintf("%-14s", typeStr))
			row := numStr + colName + strings.Repeat(" ", pad) + typeRendered + badgeStr

			if isCur {
				raw := lipgloss.Width(row)
				if raw < w {
					row += strings.Repeat(" ", w-raw)
				}
				row = activeCurLineStyle().Render(row)
			}
			lines = append(lines, row)
		}
	}

	if de.internalMode == deNaming && de.nameTarget == "column" {
		lines = append(lines, "")
		lines = append(lines, StyleTextAreaLabel.Render("  New column: ")+de.nameInput.View())
	}

	return fillTildes(lines, h)
}

func (de DataEditor) viewColForm(w, h int) string {
	if de.entityIdx >= len(de.Entities) {
		return fillTildes(nil, h)
	}
	ent := de.Entities[de.entityIdx]

	colLabel := "(new column)"
	if de.columnIdx < len(ent.Columns) {
		colLabel = ent.Columns[de.columnIdx].Name
	}

	var lines []string
	breadcrumb := StyleSectionDesc.Render("  ← ") +
		StyleSectionTitle.Render(ent.Name) +
		StyleSectionDesc.Render(" → ") +
		StyleFieldKey.Render(colLabel)
	lines = append(lines, breadcrumb, "")

	const labelW = 14
	const eqW = 3
	valW := w - 4 - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	for i, f := range de.colForm {
		isCur := i == de.colFormIdx
		disabled := isColFormFieldDisabled(de.colForm, i)

		lineNo := StyleLineNum.Render(fmt.Sprintf("%3d ", i+1))
		if isCur {
			lineNo = StyleCurLineNum.Render(fmt.Sprintf("%3d ", i+1))
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
		case de.internalMode == deInsert && isCur && f.Kind == KindText:
			valStr = de.formInput.View()
		case f.Kind == KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = StyleFieldValActive.Render(val)
			} else {
				val = StyleFieldVal.Render(val)
			}
			if isCur && de.dd.Open {
				valStr = val + StyleSelectArrow.Render(" ▴")
			} else {
				valStr = val + StyleSelectArrow.Render(" ▾")
			}
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
			row = activeCurLineStyle().Render(row)
		}
		lines = append(lines, row)

		// Inject dropdown options below the active KindSelect field
		if isCur && de.dd.Open && !disabled && f.Kind == KindSelect {
			const ddIndent = 4 + 14 + 3 // lineNumW + labelW + eqW
			indent := strings.Repeat(" ", ddIndent)
			for j, opt := range f.Options {
				isHL := j == de.dd.OptIdx
				var optRow string
				if isHL {
					optRow = indent + StyleFieldValActive.Render("► "+opt)
					rw := lipgloss.Width(optRow)
					if rw < w {
						optRow += strings.Repeat(" ", w-rw)
					}
					optRow = activeCurLineStyle().Render(optRow)
				} else {
					optRow = indent + StyleFieldVal.Render("  "+opt)
				}
				lines = append(lines, optRow)
			}
		}
	}

	return fillTildes(lines, h)
}

