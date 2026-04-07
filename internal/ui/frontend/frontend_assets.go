package frontend

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Asset form fields ─────────────────────────────────────────────────────────

func newAssetForm(a manifest.AssetDef) []core.Field {
	usageIdx := 0
	if a.Usage == manifest.AssetUsageInspiration {
		usageIdx = 1
	}
	assetTypeIdx := 0
	assetTypeOpts := []string{"image", "icon", "font", "video", "mockup", "moodboard"}
	for i, opt := range assetTypeOpts {
		if opt == a.AssetType {
			assetTypeIdx = i
			break
		}
	}
	formatIdx := 0
	formatOpts := []string{"png", "jpg", "svg", "gif", "mp4", "pdf", "figma", "sketch", "other"}
	for i, opt := range formatOpts {
		if opt == a.Format {
			formatIdx = i
			break
		}
	}
	return []core.Field{
		{
			Key: "name", Label: "name          ", Kind: core.KindText,
			Value: a.Name,
		},
		{
			Key: "path", Label: "path          ", Kind: core.KindText,
			Value: a.Path,
		},
		{
			Key: "asset_type", Label: "asset_type    ", Kind: core.KindSelect,
			Options: assetTypeOpts, Value: assetTypeOpts[assetTypeIdx], SelIdx: assetTypeIdx,
		},
		{
			Key: "format", Label: "format        ", Kind: core.KindSelect,
			Options: formatOpts, Value: formatOpts[formatIdx], SelIdx: formatIdx,
		},
		{
			Key: "usage", Label: "usage         ", Kind: core.KindSelect,
			Options: []string{"project", "inspiration"}, Value: []string{"project", "inspiration"}[usageIdx], SelIdx: usageIdx,
		},
		{
			Key: "description", Label: "description   ", Kind: core.KindText,
			Value: a.Description,
		},
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) updateAssets(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.assetSubView == core.ViewList {
		return fe.updateAssetList(key)
	}
	return fe.updateAssetForm(key)
}

func (fe FrontendEditor) updateAssetList(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	n := len(fe.assets)
	switch key.String() {
	case "j", "down":
		if n > 0 && fe.assetIdx < n-1 {
			fe.assetIdx++
		}
	case "k", "up":
		if fe.assetIdx > 0 {
			fe.assetIdx--
		}
	case "u":
		if snap, ok := fe.assetsUndo.Pop(); ok {
			fe.assets = snap
			if fe.assetIdx >= len(fe.assets) && fe.assetIdx > 0 {
				fe.assetIdx = len(fe.assets) - 1
			}
		}
	case "a":
		fe.assetsUndo.Push(core.CopySlice(fe.assets))
		fe.assets = append(fe.assets, manifest.AssetDef{})
		fe.assetIdx = len(fe.assets) - 1
		fe.assetForm = newAssetForm(manifest.AssetDef{})
		fe.assetFormIdx = 0
		fe.assetSubView = core.ViewForm
		return fe.tryEnterInsert()
	case "d":
		if n > 0 {
			fe.assetsUndo.Push(core.CopySlice(fe.assets))
			fe.assets = append(fe.assets[:fe.assetIdx], fe.assets[fe.assetIdx+1:]...)
			if fe.assetIdx > 0 && fe.assetIdx >= len(fe.assets) {
				fe.assetIdx = len(fe.assets) - 1
			}
		}
	case "enter":
		if n > 0 {
			a := fe.assets[fe.assetIdx]
			fe.assetForm = newAssetForm(a)
			fe.assetFormIdx = 0
			fe.assetSubView = core.ViewForm
		}
	}
	return fe, nil
}

func (fe FrontendEditor) updateAssetForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.dd.Open {
		return fe.updateAssetFormDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.assetFormIdx < len(fe.assetForm)-1 {
			fe.assetFormIdx++
		}
	case "k", "up":
		if fe.assetFormIdx > 0 {
			fe.assetFormIdx--
		}
	case "enter", " ":
		f := &fe.assetForm[fe.assetFormIdx]
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
		f := &fe.assetForm[fe.assetFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if fe.assetForm[fe.assetFormIdx].CanEditAsText() {
			return fe.tryEnterInsert()
		}
	case "b", "esc":
		fe.saveAssetForm()
		fe.assetSubView = core.ViewList
	}
	fe.saveAssetForm()
	return fe, nil
}

func (fe FrontendEditor) updateAssetFormDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.assetFormIdx >= len(fe.assetForm) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.assetForm[fe.assetFormIdx]
	fe.dd.OptIdx = core.NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == core.KindMultiSelect {
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
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
			f.DDCursor = fe.dd.OptIdx
		} else {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			if f.PrepareCustomEntry() {
				return fe.tryEnterInsert()
			}
		}
		fe.dd.Open = false
	case "esc", "b":
		if f.Kind == core.KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	fe.saveAssetForm()
	return fe, nil
}

func (fe *FrontendEditor) saveAssetForm() {
	if fe.assetIdx >= len(fe.assets) {
		return
	}
	a := &fe.assets[fe.assetIdx]
	a.Name = core.FieldGet(fe.assetForm, "name")
	a.Path = core.FieldGet(fe.assetForm, "path")
	a.AssetType = core.FieldGet(fe.assetForm, "asset_type")
	a.Format = core.FieldGet(fe.assetForm, "format")
	a.Usage = manifest.AssetUsage(core.FieldGet(fe.assetForm, "usage"))
	a.Description = core.FieldGet(fe.assetForm, "description")
}

// ── View ──────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) viewAssets(w int) []string {
	switch fe.assetSubView {
	case core.ViewList:
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # Assets — a: add  d: delete  Enter: edit"), "")
		if len(fe.assets) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no assets yet — press 'a' to add)"))
		} else {
			for i, a := range fe.assets {
				name := a.Name
				if name == "" {
					name = fmt.Sprintf("(asset #%d)", i+1)
				}
				badge := string(a.Usage)
				if badge == "" {
					badge = "project"
				}
				lines = append(lines, core.RenderListItem(w, i == fe.assetIdx, "  ▶ ", name, "["+badge+"] "+a.Path))
			}
		}
		return lines

	case core.ViewForm:
		name := core.FieldGet(fe.assetForm, "name")
		if name == "" {
			name = "(new asset)"
		}
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
		lines = append(lines, core.RenderFormFields(w, fe.assetForm, fe.assetFormIdx, fe.internalMode == core.ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)...)
		return lines
	}
	return nil
}
