package frontend

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

func (fe FrontendEditor) updatePages(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.pageSubView == core.ViewList {
		return fe.updatePageList(key)
	}
	return fe.updatePageForm(key)
}

func (fe FrontendEditor) updatePageList(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	n := len(fe.pages)
	switch key.String() {
	case "j", "down":
		if n > 0 && fe.pageIdx < n-1 {
			fe.pageIdx++
		}
	case "k", "up":
		if fe.pageIdx > 0 {
			fe.pageIdx--
		}
	case "u":
		if snap, ok := fe.pagesUndo.Pop(); ok {
			fe.pages = snap
			if fe.pageIdx >= len(fe.pages) && fe.pageIdx > 0 {
				fe.pageIdx = len(fe.pages) - 1
			}
		}
	case "a":
		fe.pagesUndo.Push(core.CopySlice(fe.pages))
		fe.pages = append(fe.pages, manifest.PageDef{})
		fe.pageIdx = len(fe.pages) - 1
		fe.pageForm = defaultPageFormFields(core.FieldGet(fe.techFields, "meta_framework"), fe.availableAuthRoles, fe.pageRoutes(), fe.assetNames(), fe.componentNames())
		existing := make([]string, 0, len(fe.pages)-1)
		for i, p := range fe.pages {
			if i != fe.pageIdx {
				existing = append(existing, p.Name)
			}
		}
		name := core.UniqueName("page", existing)
		fe.pageForm = core.SetFieldValue(fe.pageForm, "name", name)
		fe.pageForm = core.SetFieldValue(fe.pageForm, "route", "/"+name)
		fe.pageFormIdx = 0
		fe.pageSubView = core.ViewForm
		return fe.tryEnterInsert()
	case "d":
		if n > 0 {
			fe.pagesUndo.Push(core.CopySlice(fe.pages))
			fe.pages = append(fe.pages[:fe.pageIdx], fe.pages[fe.pageIdx+1:]...)
			if fe.pageIdx > 0 && fe.pageIdx >= len(fe.pages) {
				fe.pageIdx = len(fe.pages) - 1
			}
		}
	case "enter":
		if n > 0 {
			p := fe.pages[fe.pageIdx]
			// Exclude current page's route from linked_pages options
			otherRoutes := make([]string, 0, len(fe.pages))
			for i, pg := range fe.pages {
				if i != fe.pageIdx && pg.Route != "" {
					otherRoutes = append(otherRoutes, pg.Route)
				}
			}
			fe.pageForm = defaultPageFormFields(core.FieldGet(fe.techFields, "meta_framework"), fe.availableAuthRoles, otherRoutes, fe.assetNames(), fe.componentNames())
			fe.pageForm = core.SetFieldValue(fe.pageForm, "name", p.Name)
			fe.pageForm = core.SetFieldValue(fe.pageForm, "route", p.Route)
			if p.Purpose != "" {
				fe.pageForm = core.SetFieldValue(fe.pageForm, "purpose", p.Purpose)
			}
			fe.pageForm = core.SetFieldValue(fe.pageForm, "auth_required", p.AuthRequired)
			if p.Layout != "" {
				fe.pageForm = core.SetFieldValue(fe.pageForm, "layout", p.Layout)
			}
			fe.pageForm = core.SetFieldValue(fe.pageForm, "description", p.Description)
			fe.pageForm = core.SetFieldValue(fe.pageForm, "core_actions", p.CoreActions)
			if p.Loading != "" {
				fe.pageForm = core.SetFieldValue(fe.pageForm, "loading", p.Loading)
			}
			if p.ErrorHandling != "" {
				fe.pageForm = core.SetFieldValue(fe.pageForm, "error_handling", p.ErrorHandling)
			}
			restoreMultiField(fe.pageForm, "auth_roles", p.AuthRoles)
			restoreMultiField(fe.pageForm, "linked_pages", p.LinkedPages)
			restoreMultiField(fe.pageForm, "assets", p.Assets)
			restoreMultiField(fe.pageForm, "component_refs", p.ComponentRefs)
			fe.pageFormIdx = 0
			fe.pageSubView = core.ViewForm
		}
	}
	return fe, nil
}

// restoreMultiField restores a comma-separated saved value into a KindMultiSelect field.
func restoreMultiField(fields []core.Field, key, saved string) {
	if saved == "" {
		return
	}
	for i := range fields {
		if fields[i].Key == key {
			for _, sel := range strings.Split(saved, ", ") {
				for j, opt := range fields[i].Options {
					if opt == strings.TrimSpace(sel) {
						fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
					}
				}
			}
			return
		}
	}
}

func (fe FrontendEditor) updatePageForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.dd.Open {
		return fe.updatePageFormDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.pageFormIdx < len(fe.pageForm)-1 {
			fe.pageFormIdx++
		}
	case "k", "up":
		if fe.pageFormIdx > 0 {
			fe.pageFormIdx--
		}
	case "enter", " ":
		f := &fe.pageForm[fe.pageFormIdx]
		if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
			fe.dd.Open = true
			if f.Kind == core.KindSelect {
				fe.dd.OptIdx = f.SelIdx
			} else {
				fe.dd.OptIdx = f.DDCursor
			}
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.pageForm[fe.pageFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if fe.pageForm[fe.pageFormIdx].CanEditAsText() {
			return fe.tryEnterInsert()
		}
	case "b", "esc":
		fe.savePageForm()
		fe.pageSubView = core.ViewList
	}
	fe.savePageForm()
	return fe, nil
}

func (fe FrontendEditor) updatePageFormDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.pageFormIdx >= len(fe.pageForm) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.pageForm[fe.pageFormIdx]
	fe.dd.OptIdx = core.NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == core.KindMultiSelect {
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
		} else if f.Kind == core.KindSelect {
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
		if f.Kind == core.KindMultiSelect {
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
		} else if f.Kind == core.KindSelect {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			fe.dd.Open = false
			if f.PrepareCustomEntry() {
				fe.savePageForm()
				return fe.tryEnterInsert()
			}
		}
	case "esc", "b":
		if f.Kind == core.KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	fe.savePageForm()
	return fe, nil
}

func (fe *FrontendEditor) savePageForm() {
	if fe.pageIdx >= len(fe.pages) {
		return
	}
	p := &fe.pages[fe.pageIdx]
	p.Name = core.FieldGet(fe.pageForm, "name")
	p.Route = core.FieldGet(fe.pageForm, "route")
	p.Purpose = core.FieldGet(fe.pageForm, "purpose")
	p.AuthRequired = core.FieldGet(fe.pageForm, "auth_required")
	p.Layout = core.FieldGet(fe.pageForm, "layout")
	p.Description = core.FieldGet(fe.pageForm, "description")
	p.CoreActions = core.FieldGet(fe.pageForm, "core_actions")
	p.Loading = core.FieldGet(fe.pageForm, "loading")
	p.ErrorHandling = core.FieldGet(fe.pageForm, "error_handling")
	p.AuthRoles = core.FieldGetMulti(fe.pageForm, "auth_roles")
	p.LinkedPages = core.FieldGetMulti(fe.pageForm, "linked_pages")
	p.Assets = core.FieldGetMulti(fe.pageForm, "assets")
	p.ComponentRefs = core.FieldGetMulti(fe.pageForm, "component_refs")
}
