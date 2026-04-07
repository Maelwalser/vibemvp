package frontend

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

func (fe FrontendEditor) updateTech(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.techEnabled {
		if key.String() == "a" {
			fe.techEnabled = true
			fe.techFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateTechDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.techFormIdx < len(fe.visibleTechFields())-1 {
			fe.techFormIdx++
		}
	case "k", "up":
		if fe.techFormIdx > 0 {
			fe.techFormIdx--
		}
	case "enter", " ":
		visible := fe.visibleTechFields()
		if fe.techFormIdx >= len(visible) {
			return fe, nil
		}
		f := fe.techFieldByKey(visible[fe.techFormIdx].Key)
		if f == nil {
			return fe, nil
		}
		if f.Kind == core.KindSelect && len(f.Options) > 0 {
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		visible := fe.visibleTechFields()
		if fe.techFormIdx >= len(visible) {
			return fe, nil
		}
		f := fe.techFieldByKey(visible[fe.techFormIdx].Key)
		if f != nil && f.Kind == core.KindSelect {
			f.CyclePrev()
			if f.Key == "language" || f.Key == "platform" || f.Key == "framework" || f.Key == "language_version" {
				fe.updateFEDependentOptions()
			}
		}
	case "D":
		fe.techEnabled = false
		fe.techFields = defaultFETechFields()
		fe.techFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateTechDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	visible := fe.visibleTechFields()
	if fe.techFormIdx >= len(visible) {
		fe.dd.Open = false
		return fe, nil
	}
	f := fe.techFieldByKey(visible[fe.techFormIdx].Key)
	if f == nil {
		fe.dd.Open = false
		return fe, nil
	}
	fe.dd.OptIdx = core.NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = fe.dd.OptIdx
		if fe.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[fe.dd.OptIdx]
		}
		fe.dd.Open = false
		if f.Key == "language" || f.Key == "platform" || f.Key == "framework" || f.Key == "language_version" {
			fe.updateFEDependentOptions()
		}
		if f.PrepareCustomEntry() {
			return fe.tryEnterInsert()
		}
	case "esc", "b":
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe FrontendEditor) updateTheme(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.themeEnabled {
		if key.String() == "a" {
			fe.themeEnabled = true
			fe.themeFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateThemeDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.themeFormIdx < len(fe.themeFields)-1 {
			fe.themeFormIdx++
		}
	case "k", "up":
		if fe.themeFormIdx > 0 {
			fe.themeFormIdx--
		}
	case "enter", " ":
		f := &fe.themeFields[fe.themeFormIdx]
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
		f := &fe.themeFields[fe.themeFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "D":
		fe.themeEnabled = false
		fe.themeFields = defaultFEThemeFields()
		fe.themeFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateThemeDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.themeFormIdx >= len(fe.themeFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.themeFields[fe.themeFormIdx]
	fe.dd.OptIdx = core.NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == core.KindMultiSelect {
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
			// If this is the Custom hex option and it was just selected, enter insert mode.
			if f.ColorSwatch && fe.dd.OptIdx < len(f.Options) && core.IsCustomOption(f.Options[fe.dd.OptIdx]) && f.IsMultiSelected(fe.dd.OptIdx) {
				fe.dd.Open = false
				return fe.tryEnterInsert()
			}
		} else {
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
			if f.ColorSwatch && fe.dd.OptIdx < len(f.Options) && core.IsCustomOption(f.Options[fe.dd.OptIdx]) && f.IsMultiSelected(fe.dd.OptIdx) {
				fe.dd.Open = false
				return fe.tryEnterInsert()
			}
		} else {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			if f.PrepareCustomEntry() {
				return fe.tryEnterInsert()
			}
			fe.dd.Open = false
		}
	case "esc", "b":
		if f.Kind == core.KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	return fe, nil
}

func (fe *FrontendEditor) saveCompForm() {
	if fe.compIdx >= len(fe.components) {
		return
	}
	c := &fe.components[fe.compIdx]
	c.Name = core.FieldGet(fe.compForm, "name")
	c.ComponentType = core.FieldGet(fe.compForm, "comp_type")
	c.Description = core.FieldGet(fe.compForm, "description")
}

func (fe *FrontendEditor) saveActionsToComp() {
	if fe.compIdx >= len(fe.components) {
		return
	}
	acts := make([]manifest.ComponentActionDef, len(fe.compActions))
	copy(acts, fe.compActions)
	fe.components[fe.compIdx].Actions = acts
}

func (fe *FrontendEditor) saveActionForm() {
	if fe.actionIdx >= len(fe.compActions) {
		return
	}
	a := &fe.compActions[fe.actionIdx]
	a.Trigger = core.FieldGet(fe.actionForm, "trigger")
	a.ActionType = core.FieldGet(fe.actionForm, "action_type")

	ep := core.FieldGet(fe.actionForm, "endpoint")
	if ep == "None" {
		ep = ""
	}
	a.Endpoint = ep
	a.HttpMethod = core.FieldGet(fe.actionForm, "http_method")
	a.RequestBody = core.FieldGet(fe.actionForm, "request_body")
	a.SuccessAction = core.FieldGet(fe.actionForm, "success_action")
	a.ErrorAction = core.FieldGet(fe.actionForm, "error_action")

	ft := core.FieldGet(fe.actionForm, "form_target")
	if ft == "(none)" {
		ft = ""
	}
	a.FormTarget = ft

	mt := core.FieldGet(fe.actionForm, "modal_target")
	if mt == "(none)" {
		mt = ""
	}
	a.ModalTarget = mt

	tp := core.FieldGet(fe.actionForm, "target_page")
	if tp == "(none)" {
		tp = ""
	}
	a.TargetPage = tp

	a.ToastMessage = core.FieldGet(fe.actionForm, "toast_message")
	a.ToastType = core.FieldGet(fe.actionForm, "toast_type")
	a.ConfirmDialog = core.FieldGet(fe.actionForm, "confirm_dialog")
	a.StateKey = core.FieldGet(fe.actionForm, "state_key")
	a.StateValue = core.FieldGet(fe.actionForm, "state_value")
	a.CustomHandler = core.FieldGet(fe.actionForm, "custom_handler")
	a.Description = core.FieldGet(fe.actionForm, "description")
}

func (fe FrontendEditor) updateCompForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.dd.Open {
		return fe.updateCompFormDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.compFormIdx < len(fe.compForm)-1 {
			fe.compFormIdx++
		}
	case "k", "up":
		if fe.compFormIdx > 0 {
			fe.compFormIdx--
		}
	case "enter", " ":
		f := &fe.compForm[fe.compFormIdx]
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
		f := &fe.compForm[fe.compFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i":
		if fe.compForm[fe.compFormIdx].CanEditAsText() {
			return fe.tryEnterInsert()
		}
	case "a", "A":
		fe.saveCompForm()
		fe.currentCompType = core.FieldGet(fe.compForm, "comp_type")
		if fe.compIdx < len(fe.components) {
			acts := fe.components[fe.compIdx].Actions
			fe.compActions = make([]manifest.ComponentActionDef, len(acts))
			copy(fe.compActions, acts)
		} else {
			fe.compActions = nil
		}
		fe.inCompAction = true
		fe.actionSubView = core.ViewList
		fe.actionIdx = 0
	case "b", "esc":
		fe.saveCompForm()
		fe.compSubView = core.ViewList
	}
	fe.saveCompForm()
	return fe, nil
}

func (fe FrontendEditor) updateCompFormDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.compFormIdx >= len(fe.compForm) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.compForm[fe.compFormIdx]
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
		}
	case "esc", "b":
		if f.Kind == core.KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	fe.saveCompForm()
	return fe, nil
}

// ── COMPONENTS tab ────────────────────────────────────────────────────────────

func (fe FrontendEditor) updateComponents(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.inCompAction {
		if fe.actionSubView == core.ViewList {
			return fe.updateCompActionList(key)
		}
		return fe.updateCompActionForm(key)
	}
	if fe.compSubView == core.ViewForm {
		return fe.updateCompForm(key)
	}
	return fe.updateCompLibList(key)
}

func (fe FrontendEditor) updateCompLibList(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	n := len(fe.components)
	switch key.String() {
	case "j", "down":
		if n > 0 && fe.compIdx < n-1 {
			fe.compIdx++
		}
	case "k", "up":
		if fe.compIdx > 0 {
			fe.compIdx--
		}
	case "u":
		if snap, ok := fe.compsUndo.Pop(); ok {
			fe.components = snap
			if fe.compIdx >= len(fe.components) && fe.compIdx > 0 {
				fe.compIdx = len(fe.components) - 1
			}
		}
	case "a":
		existing := make([]string, 0, n)
		for _, c := range fe.components {
			existing = append(existing, c.Name)
		}
		fe.compsUndo.Push(core.CopySlice(fe.components))
		fe.components = append(fe.components, manifest.PageComponentDef{})
		fe.compIdx = len(fe.components) - 1
		fe.compForm = defaultComponentFormFields()
		fe.compForm = core.SetFieldValue(fe.compForm, "name", core.UniqueName("component", existing))
		fe.compFormIdx = 0
		fe.compSubView = core.ViewForm
		return fe.tryEnterInsert()
	case "d":
		if n > 0 {
			fe.compsUndo.Push(core.CopySlice(fe.components))
			fe.components = append(fe.components[:fe.compIdx], fe.components[fe.compIdx+1:]...)
			if fe.compIdx > 0 && fe.compIdx >= len(fe.components) {
				fe.compIdx = len(fe.components) - 1
			}
		}
	case "enter":
		if n > 0 {
			c := fe.components[fe.compIdx]
			fe.compForm = defaultComponentFormFields()
			fe.compForm = core.SetFieldValue(fe.compForm, "name", c.Name)
			fe.compForm = core.SetFieldValue(fe.compForm, "comp_type", c.ComponentType)
			fe.compForm = core.SetFieldValue(fe.compForm, "description", c.Description)
			fe.compFormIdx = 0
			fe.inCompAction = false
			fe.compSubView = core.ViewForm
		}
	}
	return fe, nil
}

func (fe FrontendEditor) updateCompActionList(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	n := len(fe.compActions)
	switch key.String() {
	case "j", "down":
		if n > 0 && fe.actionIdx < n-1 {
			fe.actionIdx++
		}
	case "k", "up":
		if fe.actionIdx > 0 {
			fe.actionIdx--
		}
	case "a":
		fe.compActions = append(fe.compActions, manifest.ComponentActionDef{})
		fe.actionIdx = len(fe.compActions) - 1
		fe.actionForm = defaultActionFormFields(fe.currentCompType, fe.availableEndpoints, fe.pageRoutes(), fe.formComponentNames(), fe.modalComponentNames())
		fe.actionFormIdx = 0
		fe.actionSubView = core.ViewForm
		return fe.tryEnterInsert()
	case "d":
		if n > 0 {
			fe.compActions = append(fe.compActions[:fe.actionIdx], fe.compActions[fe.actionIdx+1:]...)
			if fe.actionIdx > 0 && fe.actionIdx >= len(fe.compActions) {
				fe.actionIdx = len(fe.compActions) - 1
			}
			fe.saveActionsToComp()
		}
	case "enter":
		if n > 0 {
			fe.actionForm = defaultActionFormFields(fe.currentCompType, fe.availableEndpoints, fe.pageRoutes(), fe.formComponentNames(), fe.modalComponentNames())
			fe.restoreActionForm(fe.compActions[fe.actionIdx])
			fe.actionFormIdx = 0
			fe.actionSubView = core.ViewForm
		}
	case "b", "esc":
		fe.saveActionsToComp()
		fe.inCompAction = false
	}
	return fe, nil
}

func (fe FrontendEditor) updateCompActionForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.dd.Open {
		return fe.updateCompActionFormDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		fe.actionFormIdx = nextActionFormIdx(fe.actionForm, fe.actionFormIdx)
	case "k", "up":
		fe.actionFormIdx = prevActionFormIdx(fe.actionForm, fe.actionFormIdx)
	case "enter", " ":
		f := &fe.actionForm[fe.actionFormIdx]
		if f.Kind == core.KindSelect && len(f.Options) > 0 {
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.actionForm[fe.actionFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
			// Reposition cursor if it landed on a now-hidden field.
			if isActionFieldHidden(fe.actionForm, fe.actionFormIdx) {
				fe.actionFormIdx = nextActionFormIdx(fe.actionForm, fe.actionFormIdx)
			}
		}
	case "i", "a":
		if fe.actionForm[fe.actionFormIdx].CanEditAsText() {
			return fe.tryEnterInsert()
		}
	case "b", "esc":
		fe.saveActionForm()
		fe.saveActionsToComp()
		fe.actionSubView = core.ViewList
	}
	fe.saveActionForm()
	fe.saveActionsToComp()
	return fe, nil
}

func (fe FrontendEditor) updateCompActionFormDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.actionFormIdx >= len(fe.actionForm) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.actionForm[fe.actionFormIdx]
	fe.dd.OptIdx = core.NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = fe.dd.OptIdx
		if fe.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[fe.dd.OptIdx]
		}
		fe.dd.Open = false
		// If action_type changed, current cursor may now be on a hidden field.
		if f.Key == "action_type" && isActionFieldHidden(fe.actionForm, fe.actionFormIdx) {
			fe.actionFormIdx = nextActionFormIdx(fe.actionForm, fe.actionFormIdx)
		}
		if f.PrepareCustomEntry() {
			return fe.tryEnterInsert()
		}
	case "esc", "b":
		fe.dd.Open = false
	}
	fe.saveActionForm()
	fe.saveActionsToComp()
	return fe, nil
}

func (fe FrontendEditor) updateNav(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if !fe.navEnabled {
		if key.String() == "a" {
			fe.navEnabled = true
			fe.navFormIdx = 0
		}
		return fe, nil
	}
	if fe.dd.Open {
		return fe.updateNavDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.navFormIdx < len(fe.navFields)-1 {
			fe.navFormIdx++
		}
	case "k", "up":
		if fe.navFormIdx > 0 {
			fe.navFormIdx--
		}
	case "enter", " ":
		f := &fe.navFields[fe.navFormIdx]
		if f.Kind == core.KindSelect && len(f.Options) > 0 {
			fe.dd.Open = true
			fe.dd.OptIdx = f.SelIdx
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.navFields[fe.navFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "D":
		fe.navEnabled = false
		fe.navFields = defaultNavFields()
		fe.navFormIdx = 0
	case "i", "a":
		return fe.tryEnterInsert()
	}
	return fe, nil
}

func (fe FrontendEditor) updateNavDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.navFormIdx >= len(fe.navFields) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.navFields[fe.navFormIdx]
	fe.dd.OptIdx = core.NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
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

// ── View ──────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) View(w, h int) string {
	fe.width = w
	fe.formInput.Width = w - 22
	var lines []string
	lines = append(lines,
		core.StyleSectionDesc.Render("  # Frontend — technologies, theming, pages, and navigation"),
		"",
		core.RenderSubTabBar(feTabLabels, int(fe.activeTab), w),
		"",
	)
	const feHeaderH = 4

	switch fe.activeTab {
	case feTabTech:
		if fe.techEnabled {
			fl := core.RenderFormFields(w, fe.visibleTechFields(), fe.techFormIdx, fe.internalMode == core.ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, fe.techFormIdx, h-feHeaderH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabTheme:
		if fe.themeEnabled {
			if fe.inTextArea && fe.internalMode == core.ModeInsert {
				// Render all fields in normal mode, then show the textarea expanded below.
				lines = append(lines, core.RenderFormFields(w, fe.themeFields, fe.themeFormIdx, false, fe.formInput, false, 0)...)
				lines = append(lines, "")
				lines = append(lines, core.StyleFieldKeyActive.Render("  ── description ─────────────────────────────────────"))
				taHeight := h - feHeaderH - len(fe.themeFields) - 3
				if taHeight < 4 {
					taHeight = 4
				}
				fe.formTextArea.SetHeight(taHeight)
				fe.formTextArea.SetWidth(w - 4)
				lines = append(lines, fe.formTextArea.View())
			} else {
				fl := core.RenderFormFields(w, fe.themeFields, fe.themeFormIdx, fe.internalMode == core.ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
				lines = append(lines, core.AppendViewport(fl, 0, fe.themeFormIdx, h-feHeaderH)...)
			}
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabPages:
		pageLines := fe.viewPages(w)
		if fe.pageSubView == core.ViewList {
			pageLines = core.AppendViewport(pageLines, 2, fe.pageIdx, h-feHeaderH)
		}
		lines = append(lines, pageLines...)
	case feTabComponents:
		compLines := fe.viewComponents(w)
		if fe.compSubView == core.ViewList && !fe.inCompAction {
			compLines = core.AppendViewport(compLines, 2, fe.compIdx, h-feHeaderH)
		} else if fe.inCompAction && fe.actionSubView == core.ViewList {
			compLines = core.AppendViewport(compLines, 2, fe.actionIdx, h-feHeaderH)
		}
		lines = append(lines, compLines...)
	case feTabNav:
		if fe.navEnabled {
			fl := core.RenderFormFields(w, fe.navFields, fe.navFormIdx, fe.internalMode == core.ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, fe.navFormIdx, h-feHeaderH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabI18n:
		if fe.i18nEnabled {
			fl := core.RenderFormFields(w, fe.i18nFields, fe.i18nFormIdx, fe.internalMode == core.ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, fe.i18nFormIdx, h-feHeaderH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabA11ySEO:
		if fe.a11yEnabled {
			fl := core.RenderFormFields(w, fe.a11yFields, fe.a11yFormIdx, fe.internalMode == core.ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)
			lines = append(lines, core.AppendViewport(fl, 0, fe.a11yFormIdx, h-feHeaderH)...)
		} else {
			lines = append(lines, core.StyleSectionDesc.Render("  (not configured — press 'a' to configure)"))
		}
	case feTabAssets:
		assetLines := fe.viewAssets(w)
		if fe.assetSubView == core.ViewList {
			assetLines = core.AppendViewport(assetLines, 2, fe.assetIdx, h-feHeaderH)
		}
		lines = append(lines, assetLines...)
	}

	return core.FillTildes(lines, h)
}
