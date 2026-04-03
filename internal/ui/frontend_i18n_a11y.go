package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (fe FrontendEditor) updateI18n(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.i18nEnabled {
		if key.String() == "a" {
			fe.i18nEnabled = true
			fe.i18nFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateI18nDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.i18nFormIdx < len(fe.i18nFields)-1 {
			fe.i18nFormIdx++
		}
	case "k", "up":
		if fe.i18nFormIdx > 0 {
			fe.i18nFormIdx--
		}
	case "enter", " ":
		f := &fe.i18nFields[fe.i18nFormIdx]
		switch f.Kind {
		case KindSelect:
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		case KindMultiSelect:
			fe.dd.Open = true
			fe.dd.OptIdx = f.DDCursor
		default:
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.i18nFields[fe.i18nFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		fe.i18nEnabled = false
		fe.i18nFields = defaultI18nFields()
		fe.i18nFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateI18nDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.i18nFormIdx >= len(fe.i18nFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.i18nFields[fe.i18nFormIdx]
	fe.dd.OptIdx = NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			fe.dd.Open = false
			if f.PrepareCustomEntry() {
				return fe.tryEnterInsert()
			}
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		} else if f.Kind == KindSelect {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
		}
		fe.dd.Open = false
		if f.Kind == KindSelect && f.PrepareCustomEntry() {
			return fe.tryEnterInsert()
		}
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe FrontendEditor) updateA11ySEO(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.a11yEnabled {
		if key.String() == "a" {
			fe.a11yEnabled = true
			fe.a11yFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateA11ySEODropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.a11yFormIdx < len(fe.a11yFields)-1 {
			fe.a11yFormIdx++
		}
	case "k", "up":
		if fe.a11yFormIdx > 0 {
			fe.a11yFormIdx--
		}
	case "enter", " ":
		f := &fe.a11yFields[fe.a11yFormIdx]
		if f.Kind == KindSelect {
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.a11yFields[fe.a11yFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "D":
		fe.a11yEnabled = false
		fe.a11yFields = defaultA11ySEOFields()
		fe.a11yFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateA11ySEODropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.a11yFormIdx >= len(fe.a11yFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.a11yFields[fe.a11yFormIdx]
	fe.dd.OptIdx = NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = fe.dd.OptIdx
		if fe.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[fe.dd.OptIdx]
		}
		fe.dd.Open = false
	case "esc", "b":
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe FrontendEditor) viewPages(w int) []string {
	switch fe.pageSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Pages — a: add  d: delete  Enter: edit"), "")
		if len(fe.pages) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no pages yet — press 'a' to add)"))
		} else {
			for i, p := range fe.pages {
				name := p.Name
				if name == "" {
					name = fmt.Sprintf("(page #%d)", i+1)
				}
				nComp := len(p.Components)
				suffix := ""
				if nComp == 1 {
					suffix = "1 component"
				} else if nComp > 1 {
					suffix = fmt.Sprintf("%d components", nComp)
				}
				detail := p.Route
				if suffix != "" && detail != "" {
					detail = detail + "  [" + suffix + "]"
				} else if suffix != "" {
					detail = "[" + suffix + "]"
				}
				lines = append(lines, renderListItem(w, i == fe.pageIdx, "  ▶ ", name, detail))
			}
		}
		return lines

	case ceViewForm:
		if fe.inPageComp {
			return fe.viewPageComponents(w)
		}
		name := fieldGet(fe.pageForm, "name")
		if name == "" {
			name = "(new page)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name)+StyleSectionDesc.Render("  [C: components]"), "")
		lines = append(lines, renderFormFields(w, fe.pageForm, fe.pageFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (fe FrontendEditor) viewPageComponents(w int) []string {
	pageName := ""
	if fe.pageIdx < len(fe.pages) {
		pageName = fe.pages[fe.pageIdx].Name
	}
	if pageName == "" {
		pageName = "(page)"
	}

	switch fe.compSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(pageName)+StyleSectionDesc.Render(" › Components — a: add  d: delete  Enter: edit"), "")
		if len(fe.pageComps) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no components yet — press 'a' to add)"))
		} else {
			for i, c := range fe.pageComps {
				name := c.Name
				if name == "" {
					name = fmt.Sprintf("(component #%d)", i+1)
				}
				lines = append(lines, renderListItem(w, i == fe.compIdx, "  ▶ ", name, c.ComponentType))
			}
		}
		return lines

	case ceViewForm:
		compName := fieldGet(fe.compForm, "name")
		if compName == "" {
			compName = "(new component)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(pageName+" › "+compName), "")
		lines = append(lines, renderFormFields(w, fe.compForm, fe.compFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)...)
		return lines
	}
	return nil
}
