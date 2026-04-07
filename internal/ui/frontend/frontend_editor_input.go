package frontend

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Update ────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) Update(msg tea.Msg) (FrontendEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		fe.width = wsz.Width
		fe.formInput.Width = wsz.Width - 22
		return fe, nil
	}
	if fe.internalMode == core.ModeInsert {
		return fe.updateInsert(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return fe, nil
	}

	// Sub-tab switching — blocked while inside a component or action form.
	switch key.String() {
	case "h", "left", "l", "right":
		inCompForm := fe.activeTab == feTabComponents && (fe.compSubView == core.ViewForm || fe.inCompAction)
		if !inCompForm {
			// Auto-save any open form before switching tabs.
			switch fe.activeTab {
			case feTabPages:
				if fe.pageSubView == core.ViewForm {
					fe.savePageForm()
				}
			case feTabAssets:
				if fe.assetSubView == core.ViewForm {
					fe.saveAssetForm()
				}
			}
			fe.activeTab = feTabIdx(core.NavigateTab(key.String(), int(fe.activeTab), len(feTabLabels)))
		}
		return fe, nil
	}

	// cc detection: clear field and enter insert mode
	if !fe.dd.Open && !fe.inTextArea {
		if key.String() == "c" {
			if fe.cBuf {
				fe.cBuf = false
				return fe.clearAndEnterInsert()
			}
			fe.cBuf = true
			return fe, nil
		}
		fe.cBuf = false
	}

	switch fe.activeTab {
	case feTabTech:
		return fe.updateTech(key)
	case feTabTheme:
		return fe.updateTheme(key)
	case feTabPages:
		return fe.updatePages(key)
	case feTabComponents:
		return fe.updateComponents(key)
	case feTabNav:
		return fe.updateNav(key)
	case feTabI18n:
		return fe.updateI18n(key)
	case feTabA11ySEO:
		return fe.updateA11ySEO(key)
	case feTabAssets:
		return fe.updateAssets(key)
	}
	return fe, nil
}

func (fe FrontendEditor) updateInsert(msg tea.Msg) (FrontendEditor, tea.Cmd) {
	if fe.inTextArea {
		key, ok := msg.(tea.KeyMsg)
		if ok && key.String() == "esc" {
			fe.saveInput()
			fe.internalMode = core.ModeNormal
			fe.inTextArea = false
			fe.formTextArea.Blur()
			return fe, nil
		}
		var cmd tea.Cmd
		fe.formTextArea, cmd = fe.formTextArea.Update(msg)
		return fe, cmd
	}
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc", "enter":
			fe.saveInput()
			fe.internalMode = core.ModeNormal
			fe.formInput.Blur()
			return fe, nil
		case "tab":
			fe.saveInput()
			fe.advanceField(1)
			return fe.tryEnterInsert()
		case "shift+tab":
			fe.saveInput()
			fe.advanceField(-1)
			return fe.tryEnterInsert()
		}
	}
	var cmd tea.Cmd
	fe.formInput, cmd = fe.formInput.Update(msg)
	return fe, cmd
}

func (fe *FrontendEditor) advanceField(delta int) {
	switch fe.activeTab {
	case feTabTech:
		n := len(fe.visibleTechFields())
		if n > 0 {
			fe.techFormIdx = (fe.techFormIdx + delta + n) % n
		}
	case feTabTheme:
		n := len(fe.themeFields)
		if n > 0 {
			fe.themeFormIdx = (fe.themeFormIdx + delta + n) % n
		}
	case feTabComponents:
		if fe.inCompAction && fe.actionSubView == core.ViewForm {
			if delta > 0 {
				fe.actionFormIdx = nextActionFormIdx(fe.actionForm, fe.actionFormIdx)
			} else {
				fe.actionFormIdx = prevActionFormIdx(fe.actionForm, fe.actionFormIdx)
			}
		} else if fe.compSubView == core.ViewForm {
			n := len(fe.compForm)
			if n > 0 {
				fe.compFormIdx = (fe.compFormIdx + delta + n) % n
			}
		}
	case feTabPages:
		if fe.pageSubView == core.ViewForm {
			n := len(fe.pageForm)
			if n > 0 {
				fe.pageFormIdx = (fe.pageFormIdx + delta + n) % n
			}
		}
	case feTabNav:
		n := len(fe.navFields)
		if n > 0 {
			fe.navFormIdx = (fe.navFormIdx + delta + n) % n
		}
	case feTabI18n:
		n := len(fe.i18nFields)
		if n > 0 {
			fe.i18nFormIdx = (fe.i18nFormIdx + delta + n) % n
		}
	case feTabA11ySEO:
		n := len(fe.a11yFields)
		if n > 0 {
			fe.a11yFormIdx = (fe.a11yFormIdx + delta + n) % n
		}
	case feTabAssets:
		if fe.assetSubView == core.ViewForm {
			n := len(fe.assetForm)
			if n > 0 {
				fe.assetFormIdx = (fe.assetFormIdx + delta + n) % n
			}
		}
	}
}

func (fe *FrontendEditor) saveInput() {
	if fe.inTextArea {
		val := fe.formTextArea.Value()
		if fe.activeTab == feTabTheme && fe.themeFormIdx < len(fe.themeFields) {
			fe.themeFields[fe.themeFormIdx].Value = val
		}
		return
	}
	val := fe.formInput.Value()
	switch fe.activeTab {
	case feTabTech:
		visible := fe.visibleTechFields()
		if fe.techFormIdx < len(visible) {
			f := fe.techFieldByKey(visible[fe.techFormIdx].Key)
			if f != nil && f.CanEditAsText() {
				f.SaveTextInput(val)
			}
		}
	case feTabTheme:
		if fe.themeFormIdx < len(fe.themeFields) {
			f := &fe.themeFields[fe.themeFormIdx]
			if f.CanEditAsText() {
				if f.ColorSwatch {
					hex := strings.TrimSpace(val)
					if !strings.HasPrefix(hex, "#") && len(hex) > 0 {
						hex = "#" + hex // auto-prepend # if omitted
					}
					if !f.AddCustomHexColor(hex) {
						f.DeselectCustom() // invalid input: undo Custom selection
					}
				} else {
					f.SaveTextInput(val)
				}
			}
		}
	case feTabComponents:
		if fe.inCompAction && fe.actionSubView == core.ViewForm && fe.actionFormIdx < len(fe.actionForm) && fe.actionForm[fe.actionFormIdx].CanEditAsText() {
			fe.actionForm[fe.actionFormIdx].SaveTextInput(val)
			fe.saveActionForm()
			fe.saveActionsToComp()
		} else if fe.compSubView == core.ViewForm && fe.compFormIdx < len(fe.compForm) && fe.compForm[fe.compFormIdx].CanEditAsText() {
			fe.compForm[fe.compFormIdx].SaveTextInput(val)
			fe.saveCompForm()
		}
	case feTabPages:
		if fe.pageSubView == core.ViewForm && fe.pageFormIdx < len(fe.pageForm) && fe.pageForm[fe.pageFormIdx].CanEditAsText() {
			fe.pageForm[fe.pageFormIdx].SaveTextInput(val)
			fe.savePageForm()
		}
	case feTabNav:
		if fe.navFormIdx < len(fe.navFields) && fe.navFields[fe.navFormIdx].CanEditAsText() {
			fe.navFields[fe.navFormIdx].SaveTextInput(val)
		}
	case feTabI18n:
		if fe.i18nFormIdx < len(fe.i18nFields) && fe.i18nFields[fe.i18nFormIdx].CanEditAsText() {
			fe.i18nFields[fe.i18nFormIdx].SaveTextInput(val)
		}
	case feTabA11ySEO:
		if fe.a11yFormIdx < len(fe.a11yFields) && fe.a11yFields[fe.a11yFormIdx].CanEditAsText() {
			fe.a11yFields[fe.a11yFormIdx].SaveTextInput(val)
		}
	case feTabAssets:
		if fe.assetSubView == core.ViewForm && fe.assetFormIdx < len(fe.assetForm) && fe.assetForm[fe.assetFormIdx].CanEditAsText() {
			fe.assetForm[fe.assetFormIdx].SaveTextInput(val)
			fe.saveAssetForm()
		}
	}
}

func (fe FrontendEditor) clearAndEnterInsert() (FrontendEditor, tea.Cmd) {
	fe, cmd := fe.tryEnterInsert()
	if fe.internalMode == core.ModeInsert {
		fe.formInput.SetValue("")
	}
	return fe, cmd
}

func (fe FrontendEditor) tryEnterInsert() (FrontendEditor, tea.Cmd) {
	n := 0
	switch fe.activeTab {
	case feTabTech:
		n = len(fe.visibleTechFields())
	case feTabTheme:
		n = len(fe.themeFields)
	case feTabComponents:
		if fe.inCompAction && fe.actionSubView == core.ViewForm {
			n = len(fe.actionForm)
		} else if fe.compSubView == core.ViewForm {
			n = len(fe.compForm)
		}
	case feTabPages:
		if fe.pageSubView == core.ViewForm {
			n = len(fe.pageForm)
		}
	case feTabNav:
		n = len(fe.navFields)
	case feTabI18n:
		n = len(fe.i18nFields)
	case feTabA11ySEO:
		n = len(fe.a11yFields)
	case feTabAssets:
		if fe.assetSubView == core.ViewForm {
			n = len(fe.assetForm)
		}
	}
	for range n {
		var f *core.Field
		switch fe.activeTab {
		case feTabTech:
			visible := fe.visibleTechFields()
			if fe.techFormIdx < len(visible) {
				f = fe.techFieldByKey(visible[fe.techFormIdx].Key)
			}
		case feTabTheme:
			if fe.themeFormIdx < len(fe.themeFields) {
				f = &fe.themeFields[fe.themeFormIdx]
			}
		case feTabComponents:
			if fe.inCompAction && fe.actionSubView == core.ViewForm && fe.actionFormIdx < len(fe.actionForm) {
				f = &fe.actionForm[fe.actionFormIdx]
			} else if fe.compSubView == core.ViewForm && fe.compFormIdx < len(fe.compForm) {
				f = &fe.compForm[fe.compFormIdx]
			}
		case feTabPages:
			if fe.pageSubView == core.ViewForm && fe.pageFormIdx < len(fe.pageForm) {
				f = &fe.pageForm[fe.pageFormIdx]
			}
		case feTabNav:
			if fe.navFormIdx < len(fe.navFields) {
				f = &fe.navFields[fe.navFormIdx]
			}
		case feTabI18n:
			if fe.i18nFormIdx < len(fe.i18nFields) {
				f = &fe.i18nFields[fe.i18nFormIdx]
			}
		case feTabA11ySEO:
			if fe.a11yFormIdx < len(fe.a11yFields) {
				f = &fe.a11yFields[fe.a11yFormIdx]
			}
		case feTabAssets:
			if fe.assetSubView == core.ViewForm && fe.assetFormIdx < len(fe.assetForm) {
				f = &fe.assetForm[fe.assetFormIdx]
			}
		}
		if f == nil {
			break
		}
		if f.CanEditAsText() {
			fe.internalMode = core.ModeInsert
			if f.Kind == core.KindTextArea {
				fe.inTextArea = true
				fe.formTextArea.SetValue(f.Value)
				fe.formTextArea.SetWidth(fe.width - 4)
				return fe, fe.formTextArea.Focus()
			}
			fe.formInput.SetValue(f.TextInputValue())
			fe.formInput.Width = fe.width - 22
			fe.formInput.CursorEnd()
			return fe, fe.formInput.Focus()
		}
		fe.advanceField(1)
	}
	return fe, nil
}

// CurrentField returns the currently highlighted form field for the description panel.
// Returns nil when in list view or when no field can be resolved.
func (fe *FrontendEditor) CurrentField() *core.Field {
	switch fe.activeTab {
	case feTabTech:
		if !fe.techEnabled {
			return nil
		}
		visible := fe.visibleTechFields()
		if fe.techFormIdx >= 0 && fe.techFormIdx < len(visible) {
			return fe.techFieldByKey(visible[fe.techFormIdx].Key)
		}
	case feTabTheme:
		if !fe.themeEnabled {
			return nil
		}
		if fe.themeFormIdx >= 0 && fe.themeFormIdx < len(fe.themeFields) {
			return &fe.themeFields[fe.themeFormIdx]
		}
	case feTabComponents:
		if fe.inCompAction && fe.actionSubView == core.ViewForm && fe.actionFormIdx >= 0 && fe.actionFormIdx < len(fe.actionForm) {
			return &fe.actionForm[fe.actionFormIdx]
		} else if fe.compSubView == core.ViewForm && fe.compFormIdx >= 0 && fe.compFormIdx < len(fe.compForm) {
			return &fe.compForm[fe.compFormIdx]
		}
	case feTabPages:
		if fe.pageSubView == core.ViewForm && fe.pageFormIdx >= 0 && fe.pageFormIdx < len(fe.pageForm) {
			return &fe.pageForm[fe.pageFormIdx]
		}
	case feTabNav:
		if !fe.navEnabled {
			return nil
		}
		if fe.navFormIdx >= 0 && fe.navFormIdx < len(fe.navFields) {
			return &fe.navFields[fe.navFormIdx]
		}
	case feTabI18n:
		if !fe.i18nEnabled {
			return nil
		}
		if fe.i18nFormIdx >= 0 && fe.i18nFormIdx < len(fe.i18nFields) {
			return &fe.i18nFields[fe.i18nFormIdx]
		}
	case feTabA11ySEO:
		if !fe.a11yEnabled {
			return nil
		}
		if fe.a11yFormIdx >= 0 && fe.a11yFormIdx < len(fe.a11yFields) {
			return &fe.a11yFields[fe.a11yFormIdx]
		}
	case feTabAssets:
		if fe.assetSubView == core.ViewForm && fe.assetFormIdx >= 0 && fe.assetFormIdx < len(fe.assetForm) {
			return &fe.assetForm[fe.assetFormIdx]
		}
	}
	return nil
}
