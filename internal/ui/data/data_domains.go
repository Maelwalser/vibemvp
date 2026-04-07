package data

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Domain update ─────────────────────────────────────────────────────────────

func (dt DataTabEditor) updateDomains(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch dt.domainSubView {
	case domainViewList:
		return dt.updateDomainList(key)
	case domainViewForm:
		return dt.updateDomainForm(key)
	case domainViewAttrs:
		return dt.updateAttrList(key)
	case domainViewAttrForm:
		return dt.updateAttrForm(key)
	case domainViewRels:
		return dt.updateRelList(key)
	case domainViewRelForm:
		return dt.updateRelForm(key)
	}
	return dt, nil
}

func (dt DataTabEditor) updateDomainList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.Domains)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.domainIdx < n-1 {
			dt.domainIdx++
		}
	case "k", "up":
		if dt.domainIdx > 0 {
			dt.domainIdx--
		}
	case "u":
		if snap, ok := dt.domainsUndo.Pop(); ok {
			dt.Domains = snap
			if dt.domainIdx >= len(dt.Domains) && dt.domainIdx > 0 {
				dt.domainIdx = len(dt.Domains) - 1
			}
		}
	case "a":
		dt.domainsUndo.Push(core.CopySlice(dt.Domains))
		dt.Domains = append(dt.Domains, manifest.DomainDef{})
		dt.domainIdx = len(dt.Domains) - 1
		dt.domainForm = defaultDomainFormFields(dt.dbNames())
		existing := make([]string, 0, len(dt.Domains)-1)
		for i, d := range dt.Domains {
			if i != dt.domainIdx {
				existing = append(existing, d.Name)
			}
		}
		dt.domainForm = core.SetFieldValue(dt.domainForm, "name", core.UniqueName("domain", existing))
		dt.domainFormIdx = 0
		dt.attrItems = nil
		dt.relItems = nil
		dt.domainSubView = domainViewForm
		return dt.tryEnterInsert()
	case "d":
		if n > 0 {
			dt.domainsUndo.Push(core.CopySlice(dt.Domains))
			dt.Domains = append(dt.Domains[:dt.domainIdx], dt.Domains[dt.domainIdx+1:]...)
			if dt.domainIdx > 0 && dt.domainIdx >= len(dt.Domains) {
				dt.domainIdx = len(dt.Domains) - 1
			}
		}
	case "enter":
		if n > 0 {
			d := dt.Domains[dt.domainIdx]
			dbOpts := dt.dbNames()
			dt.domainForm = defaultDomainFormFields(dbOpts)
			dt.domainForm = core.SetFieldValue(dt.domainForm, "name", d.Name)
			dt.domainForm = core.SetFieldValue(dt.domainForm, "description", d.Description)
			// Restore multiselect for databases
			if d.Databases != "" {
				for i := range dt.domainForm {
					if dt.domainForm[i].Key == "databases" {
						for _, sel := range strings.Split(d.Databases, ", ") {
							for j, opt := range dt.domainForm[i].Options {
								if opt == sel {
									dt.domainForm[i].SelectedIdxs = append(dt.domainForm[i].SelectedIdxs, j)
								}
							}
						}
						break
					}
				}
			}
			dt.domainFormIdx = 0
			// Rebuild attr items
			attrTypes, _ := attrTypesForSources(d.Databases, dt.dbEditor.Sources)
			dt.attrItems = make([][]core.Field, len(d.Attributes))
			for i, attr := range d.Attributes {
				f := defaultAttrFields(attrTypes)
				f = core.SetFieldValue(f, "name", attr.Name)
				f = core.SetFieldValue(f, "type", attr.Type)
				f = core.RestoreMultiSelectValue(f, "constraints", attr.Constraints)
				f = core.SetFieldValue(f, "default", attr.Default)
				if attr.Sensitive {
					f = core.SetFieldValue(f, "sensitive", "true")
				}
				f = core.RestoreMultiSelectValue(f, "validation", attr.Validation)
				dt.attrItems[i] = f
			}
			// Rebuild rel items
			domOpts := dt.DomainNames()
			dt.relItems = make([][]core.Field, len(d.Relationships))
			for i, rel := range d.Relationships {
				f := defaultRelFields(domOpts)
				if rel.RelatedDomain != "" {
					f = core.SetFieldValue(f, "related_domain", rel.RelatedDomain)
				}
				if rel.RelType != "" {
					f = core.SetFieldValue(f, "rel_type", rel.RelType)
				}
				if rel.Cascade != "" {
					f = core.SetFieldValue(f, "cascade", rel.Cascade)
				}
				dt.relItems[i] = f
			}
			dt.domainSubView = domainViewForm
		}
	}
	return dt, nil
}

func (dt DataTabEditor) updateDomainForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.domainFormIdx < len(dt.domainForm)-1 {
			dt.domainFormIdx++
		}
	case "k", "up":
		if dt.domainFormIdx > 0 {
			dt.domainFormIdx--
		}
	case "enter", " ":
		f := &dt.domainForm[dt.domainFormIdx]
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
		f := &dt.domainForm[dt.domainFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.domainForm[dt.domainFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "A":
		dt.saveDomainForm()
		dt.attrIdx = 0
		dt.domainSubView = domainViewAttrs
	case "R":
		dt.saveDomainForm()
		dt.relIdx = 0
		dt.domainSubView = domainViewRels
	case "b", "esc":
		dt.saveDomainForm()
		dt.domainSubView = domainViewList
	}
	dt.saveDomainForm()
	return dt, nil
}

func (dt *DataTabEditor) saveDomainForm() {
	if dt.domainIdx >= len(dt.Domains) {
		return
	}
	d := &dt.Domains[dt.domainIdx]
	d.Name = core.FieldGet(dt.domainForm, "name")
	d.Description = core.FieldGet(dt.domainForm, "description")
	d.Databases = core.FieldGetMulti(dt.domainForm, "databases")

	// Save attrs
	d.Attributes = make([]manifest.DomainAttribute, len(dt.attrItems))
	for i, item := range dt.attrItems {
		d.Attributes[i] = manifest.DomainAttribute{
			Name:        core.FieldGet(item, "name"),
			Type:        core.FieldGet(item, "type"),
			Constraints: core.FieldGet(item, "constraints"),
			Default:     core.FieldGet(item, "default"),
			Sensitive:   core.FieldGet(item, "sensitive") == "true",
			Validation:  core.FieldGet(item, "validation"),
			Indexed:     core.FieldGet(item, "indexed") == "true",
			Unique:      core.FieldGet(item, "unique") == "true",
		}
	}

	// Save rels (no FK field — auto-inferred from rel_type)
	d.Relationships = make([]manifest.DomainRelationship, len(dt.relItems))
	for i, item := range dt.relItems {
		relType := core.FieldGet(item, "rel_type")
		relDomain := core.FieldGet(item, "related_domain")
		// Auto-generate FK name
		fk := ""
		if relDomain != "" {
			switch relType {
			case "One-to-Many":
				fk = strings.ToLower(relDomain) + "_id"
			case "One-to-One":
				fk = strings.ToLower(relDomain) + "_id"
			case "Many-to-Many":
				fk = "" // junction table; no single FK
			}
		}
		d.Relationships[i] = manifest.DomainRelationship{
			RelatedDomain: relDomain,
			RelType:       relType,
			ForeignKey:    fk,
			Cascade:       core.FieldGet(item, "cascade"),
		}
	}
}

// saveDomainAttrItemsOnly saves attrItems back to the current domain's Attributes
// without touching name/description/databases/attr_names fields.
func (dt *DataTabEditor) saveDomainAttrItemsOnly() {
	if dt.domainIdx >= len(dt.Domains) {
		return
	}
	d := &dt.Domains[dt.domainIdx]
	d.Attributes = make([]manifest.DomainAttribute, len(dt.attrItems))
	for i, item := range dt.attrItems {
		d.Attributes[i] = manifest.DomainAttribute{
			Name:        core.FieldGet(item, "name"),
			Type:        core.FieldGet(item, "type"),
			Constraints: core.FieldGet(item, "constraints"),
			Default:     core.FieldGet(item, "default"),
			Sensitive:   core.FieldGet(item, "sensitive") == "true",
			Validation:  core.FieldGet(item, "validation"),
			Indexed:     core.FieldGet(item, "indexed") == "true",
			Unique:      core.FieldGet(item, "unique") == "true",
		}
	}
}

func (dt DataTabEditor) updateAttrList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.attrItems)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.attrIdx < n-1 {
			dt.attrIdx++
		}
	case "k", "up":
		if dt.attrIdx > 0 {
			dt.attrIdx--
		}
	case "a":
		dt.attrItems = append(dt.attrItems, defaultAttrFields(dt.currentDomainAttrTypes()))
		dt.saveDomainAttrItemsOnly()
		dt.attrIdx = len(dt.attrItems) - 1
		dt.attrForm = core.CopyFields(dt.attrItems[dt.attrIdx])
		existing := make([]string, 0, len(dt.attrItems)-1)
		for i, item := range dt.attrItems {
			if i != dt.attrIdx {
				existing = append(existing, core.FieldGet(item, "name"))
			}
		}
		dt.attrForm = core.SetFieldValue(dt.attrForm, "name", core.UniqueName("attribute", existing))
		dt.attrFormIdx = 0
		dt.domainSubView = domainViewAttrForm
		return dt.tryEnterInsert()
	case "d":
		if n > 0 {
			dt.attrItems = append(dt.attrItems[:dt.attrIdx], dt.attrItems[dt.attrIdx+1:]...)
			if dt.attrIdx > 0 && dt.attrIdx >= len(dt.attrItems) {
				dt.attrIdx = len(dt.attrItems) - 1
			}
			dt.saveDomainAttrItemsOnly()
		}
	case "enter":
		if n > 0 {
			dt.attrForm = core.CopyFields(dt.attrItems[dt.attrIdx])
			dt.attrForm = refreshAttrTypeOptions(dt.attrForm, dt.currentDomainAttrTypes())
			dt.attrFormIdx = 0
			dt.domainSubView = domainViewAttrForm
		}
	case "b", "esc":
		dt.saveDomainAttrItemsOnly()
		dt.domainSubView = domainViewForm
	}
	return dt, nil
}

func (dt DataTabEditor) updateAttrForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.attrFormIdx < len(dt.attrForm)-1 {
			dt.attrFormIdx++
		}
	case "k", "up":
		if dt.attrFormIdx > 0 {
			dt.attrFormIdx--
		}
	case "enter", " ":
		f := &dt.attrForm[dt.attrFormIdx]
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
		f := &dt.attrForm[dt.attrFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.attrForm[dt.attrFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.attrIdx < len(dt.attrItems) {
			dt.attrItems[dt.attrIdx] = core.CopyFields(dt.attrForm)
		}
		dt.saveDomainAttrItemsOnly()
		dt.domainSubView = domainViewAttrs
	}
	if dt.attrIdx < len(dt.attrItems) {
		dt.attrItems[dt.attrIdx] = core.CopyFields(dt.attrForm)
	}
	dt.saveDomainAttrItemsOnly()
	return dt, nil
}

func (dt DataTabEditor) updateRelList(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	n := len(dt.relItems)
	switch key.String() {
	case "j", "down":
		if n > 0 && dt.relIdx < n-1 {
			dt.relIdx++
		}
	case "k", "up":
		if dt.relIdx > 0 {
			dt.relIdx--
		}
	case "a":
		dt.relItems = append(dt.relItems, defaultRelFields(dt.DomainNames()))
		dt.relIdx = len(dt.relItems) - 1
		dt.relForm = core.CopyFields(dt.relItems[dt.relIdx])
		dt.relFormIdx = 0
		dt.domainSubView = domainViewRelForm
	case "d":
		if n > 0 {
			dt.relItems = append(dt.relItems[:dt.relIdx], dt.relItems[dt.relIdx+1:]...)
			if dt.relIdx > 0 && dt.relIdx >= len(dt.relItems) {
				dt.relIdx = len(dt.relItems) - 1
			}
		}
	case "enter":
		if n > 0 {
			dt.relForm = core.CopyFields(dt.relItems[dt.relIdx])
			dt.relFormIdx = 0
			dt.domainSubView = domainViewRelForm
		}
	case "b", "esc":
		dt.domainSubView = domainViewForm
	}
	return dt, nil
}

func (dt DataTabEditor) updateRelForm(key tea.KeyMsg) (DataTabEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		if dt.relFormIdx < len(dt.relForm)-1 {
			dt.relFormIdx++
		}
	case "k", "up":
		if dt.relFormIdx > 0 {
			dt.relFormIdx--
		}
	case "enter", " ":
		f := &dt.relForm[dt.relFormIdx]
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
		f := &dt.relForm[dt.relFormIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if dt.relForm[dt.relFormIdx].CanEditAsText() {
			return dt.tryEnterInsert()
		}
	case "b", "esc":
		if dt.relIdx < len(dt.relItems) {
			dt.relItems[dt.relIdx] = core.CopyFields(dt.relForm)
		}
		dt.domainSubView = domainViewRels
	}
	if dt.relIdx < len(dt.relItems) {
		dt.relItems[dt.relIdx] = core.CopyFields(dt.relForm)
	}
	return dt, nil
}

func (dt DataTabEditor) viewDomains(w int) []string {
	switch dt.domainSubView {
	case domainViewList:
		var lines []string
		lines = append(lines, core.StyleSectionDesc.Render("  # Domains — a: add  d: delete  Enter: edit"), "")
		if len(dt.Domains) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no domains yet — press 'a' to add one)"))
		} else {
			for i, d := range dt.Domains {
				desc := d.Description
				if len(desc) > 40 {
					desc = desc[:39] + "…"
				}
				lines = append(lines, core.RenderListItem(w, i == dt.domainIdx, "  ▶ ", d.Name, desc))
			}
		}
		return lines

	case domainViewForm:
		var lines []string
		name := core.FieldGet(dt.domainForm, "name")
		if name == "" {
			name = "(new domain)"
		}
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(name), "")
		lines = append(lines, core.RenderFormFields(w, dt.domainForm, dt.domainFormIdx, dt.internalMode == core.ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
		lines = append(lines, "", core.StyleSectionDesc.Render("  A: edit fields  R: edit relationships"))
		attrCount := len(dt.attrItems)
		relCount := len(dt.relItems)
		lines = append(lines, core.StyleSectionDesc.Render(fmt.Sprintf("  %d field(s)  %d relationship(s)", attrCount, relCount)))
		return lines

	case domainViewAttrs:
		var lines []string
		if dt.domainIdx < len(dt.Domains) {
			lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(dt.Domains[dt.domainIdx].Name)+core.StyleSectionDesc.Render(" › Fields"), "")
		}
		if len(dt.attrItems) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no fields — press 'a' to add)"))
		} else {
			for i, item := range dt.attrItems {
				attrName := core.FieldGet(item, "name")
				attrType := core.FieldGet(item, "type")
				if attrName == "" {
					attrName = fmt.Sprintf("(attr #%d)", i+1)
				}
				lines = append(lines, core.RenderListItem(w, i == dt.attrIdx, "  ▶ ", attrName, attrType))
			}
		}
		return lines

	case domainViewAttrForm:
		var lines []string
		attrName := core.FieldGet(dt.attrForm, "name")
		if attrName == "" {
			attrName = "(new attribute)"
		}
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(attrName), "")
		lines = append(lines, core.RenderFormFields(w, dt.attrForm, dt.attrFormIdx, dt.internalMode == core.ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
		return lines

	case domainViewRels:
		var lines []string
		if dt.domainIdx < len(dt.Domains) {
			lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(dt.Domains[dt.domainIdx].Name)+core.StyleSectionDesc.Render(" › Relationships"), "")
		}
		if len(dt.relItems) == 0 {
			lines = append(lines, core.StyleSectionDesc.Render("  (no relationships — press 'a' to add)"))
		} else {
			for i, item := range dt.relItems {
				relName := core.FieldGet(item, "related_domain")
				relType := core.FieldGet(item, "rel_type")
				if relName == "" {
					relName = fmt.Sprintf("(rel #%d)", i+1)
				}
				lines = append(lines, core.RenderListItem(w, i == dt.relIdx, "  ▶ ", relName, relType))
			}
		}
		return lines

	case domainViewRelForm:
		var lines []string
		relName := core.FieldGet(dt.relForm, "related_domain")
		if relName == "" {
			relName = "(new relationship)"
		}
		lines = append(lines, core.StyleSectionDesc.Render("  ← ")+core.StyleFieldKey.Render(relName), "")
		lines = append(lines, core.RenderFormFields(w, dt.relForm, dt.relFormIdx, dt.internalMode == core.ModeInsert, dt.formInput, dt.dd.Open, dt.dd.OptIdx)...)
		return lines
	}
	return nil
}
