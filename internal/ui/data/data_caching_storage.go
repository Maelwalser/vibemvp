package data

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Caching update ────────────────────────────────────────────────────────────

func (dt DataTabEditor) updateCaching(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.cachingSubView {
	case cachingViewList:
		return dt.updateCachingList(key)
	case cachingViewForm:
		return dt.updateCachingForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateCachingList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.cachings)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.cachingIdx < n-1 {
			dt.cachingIdx++
		}
	case "k", "up":
		if dt.cachingIdx > 0 {
			dt.cachingIdx--
		}
	case "u":
		if snap, ok := dt.cachingUndo.Pop(); ok {
			dt.cachings = snap
			if dt.cachingIdx >= len(dt.cachings) && dt.cachingIdx > 0 {
				dt.cachingIdx = len(dt.cachings) - 1
			}
		}
	case "a":
		dt.cachingUndo.Push(core.CopySlice(dt.cachings))
		dt.cachings = append(dt.cachings, manifest.CachingConfig{})
		dt.cachingIdx = len(dt.cachings) - 1
		dt.cachingForm = defaultCachingFields()
		existing := make([]string, 0, len(dt.cachings)-1)
		for i, c := range dt.cachings {
			if i != dt.cachingIdx {
				existing = append(existing, c.Name)
			}
		}
		dt.cachingForm = core.SetFieldValue(dt.cachingForm, "name", core.UniqueName("caching", existing))
		dt.cachingFormIdx = 0
		dt.cachingSubView = cachingViewForm
	case "d":
		if n > 0 {
			dt.cachingUndo.Push(core.CopySlice(dt.cachings))
			dt.cachings = append(dt.cachings[:dt.cachingIdx], dt.cachings[dt.cachingIdx+1:]...)
			if dt.cachingIdx > 0 && dt.cachingIdx >= len(dt.cachings) {
				dt.cachingIdx = len(dt.cachings) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.cachingForm = cachingFormFromDef(dt.cachings[dt.cachingIdx])
			dt.cachingFormIdx = 0
			dt.cachingSubView = cachingViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateCachingForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	// Refresh dynamic options
	dt = dt.withRefreshedCachingEntities()
	dt = dt.withRefreshedCachingDBs()
	dt = dt.withRefreshedCachingStrategies()
	switch key.String() {
	case "j", "down":
		dt.cachingFormIdx = nextCachingFormIdx(dt.cachingForm, dt.cachingFormIdx)
	case "k", "up":
		dt.cachingFormIdx = prevCachingFormIdx(dt.cachingForm, dt.cachingFormIdx)
	case "enter", " ":
		f := &dt.cachingForm[dt.cachingFormIdx]
		if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
			dt.dd.Open = true
			if f.Kind == core.KindSelect {
				dt.dd.OptIdx = f.SelIdx
			} else {
				dt.dd.OptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.cachingForm[dt.cachingFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.cachingForm[dt.cachingFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.cachingIdx < len(dt.cachings) {
			dt.cachings[dt.cachingIdx] = cachingDefFromForm(dt.cachingForm)
		}
		dt.cachingSubView = cachingViewList
	}
	if dt.cachingIdx < len(dt.cachings) {
		dt.cachings[dt.cachingIdx] = cachingDefFromForm(dt.cachingForm)
	}
	return dt, nil
}

// ── Governance update ─────────────────────────────────────────────────────────

// complianceAutoUpgrade upgrades pii_encryption from "None" to "core.Field-level AES-256"
// when HIPAA, GDPR, or PCI-DSS is selected in compliance_frameworks.
func (dt DataTabEditor) complianceAutoUpgrade() DataTabEditor {
	selected := core.FieldGetSelectedSlice(dt.govForm, "compliance_frameworks")
	sensitive := false
	for _, f := range selected {
		if f == "HIPAA" || f == "GDPR" || f == "PCI-DSS" {
			sensitive = true
			break
		}
	}
	if !sensitive {
		return dt
	}
	pii := core.FieldGet(dt.govForm, "pii_encryption")
	if pii == "None" {
		dt.govForm = core.SetFieldValue(dt.govForm, "pii_encryption", "core.Field-level AES-256")
	}
	return dt
}

func (dt DataTabEditor) updateGovernance(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.govSubView {
	case govViewList:
		return dt.updateGovList(key)
	case govViewForm:
		return dt.updateGovForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateGovList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.governances)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.govIdx < n-1 {
			dt.govIdx++
		}
	case "k", "up":
		if dt.govIdx > 0 {
			dt.govIdx--
		}
	case "u":
		if snap, ok := dt.govUndo.Pop(); ok {
			dt.governances = snap
			if dt.govIdx >= len(dt.governances) && dt.govIdx > 0 {
				dt.govIdx = len(dt.governances) - 1
			}
		}
	case "a":
		dt.govUndo.Push(core.CopySlice(dt.governances))
		dt.governances = append(dt.governances, manifest.DataGovernanceConfig{})
		dt.govIdx = len(dt.governances) - 1
		dbAliases := dt.dbNames()
		dt.govForm = defaultGovFormFields(dbAliases, dt.cloudProvider)
		existing := make([]string, 0, len(dt.governances)-1)
		for i, g := range dt.governances {
			if i != dt.govIdx {
				existing = append(existing, g.Name)
			}
		}
		dt.govForm = core.SetFieldValue(dt.govForm, "name", core.UniqueName("policy", existing))
		dt.govForm = dt.withRefreshedGovOptions(dt.govForm)
		dt.govFormIdx = 0
		dt.govSubView = govViewForm
	case "d":
		if n > 0 {
			dt.govUndo.Push(core.CopySlice(dt.governances))
			dt.governances = append(dt.governances[:dt.govIdx], dt.governances[dt.govIdx+1:]...)
			if dt.govIdx > 0 && dt.govIdx >= len(dt.governances) {
				dt.govIdx = len(dt.governances) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.govForm = govFormFromDef(dt.governances[dt.govIdx], dt.dbNames(), dt.cloudProvider)
			dt.govForm = dt.withRefreshedGovOptions(dt.govForm)
			dt.govFormIdx = 0
			dt.govSubView = govViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateGovForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	// Persist current form state continuously.
	if dt.govIdx < len(dt.governances) {
		dt.governances[dt.govIdx] = govDefFromForm(dt.govForm)
	}

	isDisabled := func(f []core.Field, i int) bool { return dt.isGovFieldDisabled(f, i) }

	switch key.String() {
	case "j", "down":
		dt.govFormIdx = core.NextFormIdx(dt.govForm, dt.govFormIdx, isDisabled)
	case "k", "up":
		dt.govFormIdx = core.PrevFormIdx(dt.govForm, dt.govFormIdx, isDisabled)
	case "enter", " ":
		f := &dt.govForm[dt.govFormIdx]
		if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
			dt.dd.Open = true
			if f.Kind == core.KindSelect {
				dt.dd.OptIdx = f.SelIdx
			} else {
				dt.dd.OptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.govForm[dt.govFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.govForm[dt.govFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.govIdx < len(dt.governances) {
			dt.governances[dt.govIdx] = govDefFromForm(dt.govForm)
		}
		dt.govSubView = govViewList
		return dt, nil
	}

	// Refresh DB-aware options whenever databases selection may have changed.
	if dt.govFormIdx < len(dt.govForm) && dt.govForm[dt.govFormIdx].Key == "databases" {
		dt.govForm = dt.withRefreshedGovOptions(dt.govForm)
	}

	return dt, nil
}

// ── File storage update ───────────────────────────────────────────────────────

func (dt DataTabEditor) updateFileStorage(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.fsSubView {
	case fsViewList:
		return dt.updateFSList(key)
	case fsViewForm:
		return dt.updateFSForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateFSList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.fileStorages)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.fsIdx < n-1 {
			dt.fsIdx++
		}
	case "k", "up":
		if dt.fsIdx > 0 {
			dt.fsIdx--
		}
	case "u":
		if snap, ok := dt.fsUndo.Pop(); ok {
			dt.fileStorages = snap
			if dt.fsIdx >= len(dt.fileStorages) && dt.fsIdx > 0 {
				dt.fsIdx = len(dt.fileStorages) - 1
			}
		}
	case "a":
		dt.fsUndo.Push(core.CopySlice(dt.fileStorages))
		dt.fileStorages = append(dt.fileStorages, manifest.FileStorageDef{})
		dt.fsIdx = len(dt.fileStorages) - 1
		dt.fsForm = defaultFSFormFields(dt.DomainNames(), dt.cloudProvider, dt.environmentNames, dt.serviceNames)
		existing := make([]string, 0, len(dt.fileStorages)-1)
		for i, fs := range dt.fileStorages {
			if i != dt.fsIdx {
				existing = append(existing, fs.Purpose)
			}
		}
		dt.fsForm = core.SetFieldValue(dt.fsForm, "purpose", core.UniqueName("storage", existing))
		dt.fsFormIdx = 0
		dt.fsSubView = fsViewForm
	case "d":
		if n > 0 {
			dt.fsUndo.Push(core.CopySlice(dt.fileStorages))
			dt.fileStorages = append(dt.fileStorages[:dt.fsIdx], dt.fileStorages[dt.fsIdx+1:]...)
			if dt.fsIdx > 0 && dt.fsIdx >= len(dt.fileStorages) {
				dt.fsIdx = len(dt.fileStorages) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.fsForm = fsFormFromDef(dt.fileStorages[dt.fsIdx], dt.DomainNames(), dt.cloudProvider, dt.environmentNames, dt.serviceNames)
			dt.fsFormIdx = 0
			dt.fsSubView = fsViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateFSForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.fsFormIdx < len(dt.fsForm)-1 {
			dt.fsFormIdx++
		}
	case "k", "up":
		if dt.fsFormIdx > 0 {
			dt.fsFormIdx--
		}
	case "enter", " ":
		f := &dt.fsForm[dt.fsFormIdx]
		if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
			dt.dd.Open = true
			if f.Kind == core.KindSelect {
				dt.dd.OptIdx = f.SelIdx
			} else {
				dt.dd.OptIdx = f.DDCursor
			}
		} else {
			return dt.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &dt.fsForm[dt.fsFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.fsForm[dt.fsFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.fsIdx < len(dt.fileStorages) {
			dt.fileStorages[dt.fsIdx] = fsDefFromForm(dt.fsForm)
		}
		dt.fsSubView = fsViewList
	}
	if dt.fsIdx < len(dt.fileStorages) {
		dt.fileStorages[dt.fsIdx] = fsDefFromForm(dt.fsForm)
	}
	return dt, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (dt DataTabEditor) viewCaching(w int) []string {
	var lines []string
	switch dt.cachingSubView {
	case cachingViewList:
		lines = append(lines, core.StyleSectionDesc.Render("  # Caching Strategies — a: add  d: delete  Enter: edit"), "")
		if len(dt.cachings) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no caching strategies yet — press 'a' to add)"))
		} else {
			for i, c := range dt.cachings {
				name := c.Name
				if name == "" {
					name = fmt.Sprintf("(strategy #%d)", i+1)
				}
				detail := c.Layer
				if c.Strategy != "" {
					detail += " / " + c.Strategy
				}
				lines = append(lines, core.RenderListItem(w, i == dt.cachingIdx, "  ▶ ", name, detail))
			}
		}
	case cachingViewForm:
		dt = dt.withRefreshedCachingEntities()
		dt = dt.withRefreshedCachingDBs()
		dt = dt.withRefreshedCachingStrategies()
		name := core.FieldGet(dt.cachingForm, "name")
		if name == "" {
			name = "(new strategy)"
		}
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
		visible := cachingVisibleFields(dt.cachingForm)
		visIdx := cachingVisibleIdx(dt.cachingForm, dt.cachingFormIdx)
		lines = append(lines, core.RenderFormFields(w, visible, visIdx, dt.internalMode == core.ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
	}
	return lines
}

func (dt DataTabEditor) viewFileStorage(w int) []string {
	switch dt.fsSubView {
	case fsViewList:
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # File / Object Storage — a: add  d: delete  Enter: edit"), "")
		if len(dt.fileStorages) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no storage buckets yet — press 'a' to add)"))
		} else {
			for i, fs := range dt.fileStorages {
				tech := fs.Technology
				if tech == "" {
					tech = "?"
				}
				name := fs.Purpose
				if name == "" {
					name = fmt.Sprintf("(storage #%d)", i+1)
				}
				subtitle := tech + " / " + fs.Access
				if fs.UsedByService != "" {
					subtitle += " · svc:" + fs.UsedByService
				}
				lines = append(lines, core.RenderListItem(w, i == dt.fsIdx, "  ▶ ", name, subtitle))
			}
		}
		return lines

	case fsViewForm:
		var lines []string
		tech := core.FieldGet(dt.fsForm, "technology")
		if tech == "" {
			tech = "(new storage)"
		}
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(tech), "")
		lines = append(lines, core.RenderFormFields(w, dt.fsForm, dt.fsFormIdx, dt.internalMode == core.ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
		return lines
	}
	return nil
}

func (dt DataTabEditor) viewGovernance(w int) []string {
	var lines []string
	lines = append(lines, core.StyleSectionDesc.Render("  # Data Governance Policies"), "")
	switch dt.govSubView {
	case govViewList:
		lines = append(lines, core.StyleSectionDesc.Render("  — a: add  d: delete  Enter: edit"), "")
		if len(dt.governances) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no governance policies yet — press 'a' to add)"))
		} else {
			for i, g := range dt.governances {
				name := g.Name
				if name == "" {
					name = fmt.Sprintf("(policy #%d)", i+1)
				}
				var detail string
				if len(g.Databases) > 0 {
					detail = strings.Join(g.Databases, ", ")
				} else {
					detail = "all databases"
				}
				lines = append(lines, core.RenderListItem(w, i == dt.govIdx, "  ▶ ", name, detail))
			}
		}
	case govViewForm:
		refreshed := dt.withRefreshedGovOptions(dt.govForm)
		name := core.FieldGet(refreshed, "name")
		if name == "" {
			name = "(new policy)"
		}
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
		visible := dt.govVisibleFields(refreshed)
		visIdx := dt.govVisibleIdx(refreshed, dt.govFormIdx)
		lines = append(lines, core.RenderFormFields(w, visible, visIdx, dt.internalMode == core.ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
	}
	return lines
}

func (dt DataTabEditor) govVisibleFields(form []core.Field) []core.Field {
	out := make([]core.Field, 0, len(form))
	for i, f := range form {
		if !dt.isGovFieldDisabled(form, i) {
			out = append(out, f)
		}
	}
	return out
}

func (dt DataTabEditor) govVisibleIdx(form []core.Field, fullIdx int) int {
	vis := 0
	for i := range fullIdx {
		if !dt.isGovFieldDisabled(form, i) {
			vis++
		}
	}
	return vis
}

// Expose db sources for syncing into the DataEditor.
func (dt DataTabEditor) DBSources() []manifest.DBSourceDef {
	return dt.dbEditor.Sources
}
