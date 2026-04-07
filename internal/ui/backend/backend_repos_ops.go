package backend

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── op_type options by DB technology ─────────────────────────────────────────

func opTypesForDBType(dbType string) []string {
	switch dbType {
	case "PostgreSQL", "MySQL", "SQLite":
		return []string{
			"read-one", "read-all", "read-page",
			"create", "update", "delete",
			"count", "exists", "aggregate", "raw-query",
		}
	case "MongoDB":
		return []string{
			"find-one", "find-many", "find-page",
			"insert-one", "insert-many",
			"update-one", "update-many",
			"delete-one", "delete-many",
			"count", "aggregate",
		}
	case "DynamoDB":
		return []string{
			"get-item", "query", "scan",
			"put-item", "update-item", "delete-item",
			"batch-get", "batch-write", "count",
		}
	default: // Redis, other, unknown
		return []string{"read", "read-all", "create", "update", "delete", "count", "aggregate"}
	}
}

var resultShapeOptions = []string{"Single item", "List", "Count", "Boolean", "Void"}
var paginationOptions = []string{"None", "Offset/Limit", "Cursor-based", "Page-based"}

// ── op field constructors ────────────────────────────────────────────────────

func defaultOpFields() []core.Field {
	return []core.Field{
		{Key: "name", Label: "name          ", Kind: core.KindText},
		{
			Key:     "op_type",
			Label:   "op_type       ",
			Kind:    core.KindSelect,
			Options: []string{"read", "read-all", "create", "update", "delete", "count", "aggregate"},
			Value:   "read",
		},
		{Key: "filter_by", Label: "filter_by     ", Kind: core.KindMultiSelect},
		{
			Key:     "sort_by",
			Label:   "sort_by       ",
			Kind:    core.KindSelect,
			Options: []string{"(none)"},
			Value:   "(none)",
		},
		{
			Key:     "result_shape",
			Label:   "result_shape  ",
			Kind:    core.KindSelect,
			Options: resultShapeOptions,
			Value:   "Single item",
		},
		{
			Key:     "pagination",
			Label:   "pagination    ",
			Kind:    core.KindSelect,
			Options: paginationOptions,
			Value:   "None",
		},
		{Key: "query_hint", Label: "query_hint    ", Kind: core.KindText},
		{Key: "description", Label: "description   ", Kind: core.KindText},
	}
}

// ── op manifest converters ───────────────────────────────────────────────────

func opDefFromFields(fields []core.Field) manifest.DataAccessOp {
	var filterBy []string
	for _, f := range fields {
		if f.Key == "filter_by" {
			for _, idx := range f.SelectedIdxs {
				if idx < len(f.Options) {
					filterBy = append(filterBy, f.Options[idx])
				}
			}
			break
		}
	}
	sortBy := core.FieldGet(fields, "sort_by")
	if sortBy == "(none)" {
		sortBy = ""
	}
	pagination := core.FieldGet(fields, "pagination")
	if pagination == "None" {
		pagination = ""
	}
	return manifest.DataAccessOp{
		Name:        core.FieldGet(fields, "name"),
		OpType:      core.FieldGet(fields, "op_type"),
		FilterBy:    filterBy,
		SortBy:      sortBy,
		ResultShape: core.FieldGet(fields, "result_shape"),
		Pagination:  pagination,
		QueryHint:   core.FieldGet(fields, "query_hint"),
		Description: core.FieldGet(fields, "description"),
	}
}

func opFieldsFromDef(op manifest.DataAccessOp) []core.Field {
	f := defaultOpFields()
	f = core.SetFieldValue(f, "name", op.Name)
	if op.OpType != "" {
		f = core.SetFieldValue(f, "op_type", op.OpType)
	}
	// filter_by: store as Value for lazy restoration via withOpRefs.
	if len(op.FilterBy) > 0 {
		for i := range f {
			if f[i].Key == "filter_by" {
				f[i].Value = strings.Join(op.FilterBy, ", ")
				break
			}
		}
	}
	if op.SortBy != "" {
		f = core.SetFieldValue(f, "sort_by", op.SortBy)
	}
	if op.ResultShape != "" {
		f = core.SetFieldValue(f, "result_shape", op.ResultShape)
	}
	if op.Pagination != "" {
		f = core.SetFieldValue(f, "pagination", op.Pagination)
	}
	f = core.SetFieldValue(f, "query_hint", op.QueryHint)
	f = core.SetFieldValue(f, "description", op.Description)
	return f
}

// ── op reference injection ───────────────────────────────────────────────────

// withOpRefs injects filter_by and sort_by options (from the repo's selected
// fields) and op_type options (based on target_db technology) into an op field
// slice.
func (be *BackendEditor) withOpRefs(fields []core.Field) []core.Field {
	f := core.CopyFields(fields)
	// Resolve the parent repo to get selected fields and target_db.
	svcIdx := be.serviceEditor.itemIdx
	repoIdx := be.repoEditor.itemIdx
	var repoFields []string
	var dbType string
	if svcIdx < len(be.Services) && repoIdx < len(be.Services[svcIdx].Repositories) {
		repo := be.Services[svcIdx].Repositories[repoIdx]
		if len(repo.SelectedFields) > 0 {
			repoFields = repo.SelectedFields
		} else if repo.EntityRef != "" {
			repoFields = be.domainAttributes[repo.EntityRef]
		}
		if repo.TargetDB != "" {
			dbType = be.dbSourceTypes[repo.TargetDB]
		}
	}

	// op_type: technology-aware options
	opTypes := opTypesForDBType(dbType)
	for i := range f {
		if f[i].Key != "op_type" {
			continue
		}
		cur := f[i].Value
		f[i].Options = opTypes
		found := false
		for j, o := range opTypes {
			if o == cur {
				f[i].SelIdx = j
				f[i].Value = o
				found = true
				break
			}
		}
		if !found {
			f[i].SelIdx = 0
			f[i].Value = opTypes[0]
		}
		break
	}

	// filter_by and sort_by: from repo's selected fields.
	// When no fields are available, show a descriptive placeholder.
	const entityPlaceholder = "(configure entity_ref in repo first)"
	noFields := len(repoFields) == 0
	sortOpts := append([]string{"(none)"}, repoFields...)
	if noFields {
		sortOpts = []string{entityPlaceholder}
	}
	for i := range f {
		switch f[i].Key {
		case "filter_by":
			if noFields {
				f[i].Options = []string{entityPlaceholder}
				f[i].SelectedIdxs = nil
				f[i].Value = ""
				break
			}
			// Restore previously selected filter fields by name.
			var selectedNames []string
			if len(f[i].SelectedIdxs) > 0 && len(f[i].Options) > 0 {
				for _, idx := range f[i].SelectedIdxs {
					if idx < len(f[i].Options) {
						selectedNames = append(selectedNames, f[i].Options[idx])
					}
				}
			} else if f[i].Value != "" {
				for _, name := range strings.Split(f[i].Value, ", ") {
					name = strings.TrimSpace(name)
					if name != "" {
						selectedNames = append(selectedNames, name)
					}
				}
			}
			f[i].Options = repoFields
			f[i].SelectedIdxs = nil
			f[i].Value = ""
			for _, name := range selectedNames {
				for j, opt := range repoFields {
					if opt == name {
						f[i].SelectedIdxs = append(f[i].SelectedIdxs, j)
						break
					}
				}
			}
		case "sort_by":
			cur := f[i].Value
			f[i].Options = sortOpts
			if noFields {
				f[i].SelIdx = 0
				f[i].Value = entityPlaceholder
				break
			}
			found := false
			for j, o := range sortOpts {
				if o == cur {
					f[i].SelIdx = j
					f[i].Value = o
					found = true
					break
				}
			}
			if !found {
				f[i].SelIdx = 0
				f[i].Value = "(none)"
			}
		}
	}
	return f
}

// ── op data loading ──────────────────────────────────────────────────────────

// loadOpEditorItems populates opEditor.items from the current repo's operations.
func (be *BackendEditor) loadOpEditorItems() {
	svcIdx := be.serviceEditor.itemIdx
	repoIdx := be.repoEditor.itemIdx
	if svcIdx >= len(be.Services) || repoIdx >= len(be.Services[svcIdx].Repositories) {
		be.opEditor.items = nil
		return
	}
	ops := be.Services[svcIdx].Repositories[repoIdx].Operations
	be.opEditor.items = make([][]core.Field, len(ops))
	for i, op := range ops {
		be.opEditor.items[i] = opFieldsFromDef(op)
	}
}

// ── op save helper ───────────────────────────────────────────────────────────

func (be *BackendEditor) saveOpForm() {
	ed := &be.opEditor
	svcIdx := be.serviceEditor.itemIdx
	repoIdx := be.repoEditor.itemIdx
	if ed.itemIdx >= len(ed.items) || svcIdx >= len(be.Services) {
		return
	}
	if repoIdx >= len(be.Services[svcIdx].Repositories) {
		return
	}
	ed.items[ed.itemIdx] = core.CopyFields(ed.form)
	op := opDefFromFields(ed.form)
	ops := be.Services[svcIdx].Repositories[repoIdx].Operations
	if ed.itemIdx < len(ops) {
		be.Services[svcIdx].Repositories[repoIdx].Operations[ed.itemIdx] = op
	}
}

// ── op list update ───────────────────────────────────────────────────────────

func (be BackendEditor) updateOpList(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.opEditor
	svcIdx := be.serviceEditor.itemIdx
	repoIdx := be.repoEditor.itemIdx
	n := len(ed.items)

	validRepo := svcIdx < len(be.Services) && repoIdx < len(be.Services[svcIdx].Repositories)

	switch key.String() {
	case "j", "down":
		if n > 0 && ed.itemIdx < n-1 {
			ed.itemIdx++
		}
	case "k", "up":
		if ed.itemIdx > 0 {
			ed.itemIdx--
		}
	case "a":
		if validRepo {
			existing := make([]string, 0)
			for _, op := range be.Services[svcIdx].Repositories[repoIdx].Operations {
				existing = append(existing, op.Name)
			}
			newOp := manifest.DataAccessOp{Name: core.UniqueName("op", existing), OpType: "read"}
			be.Services[svcIdx].Repositories[repoIdx].Operations = append(
				be.Services[svcIdx].Repositories[repoIdx].Operations, newOp)
			newFields := be.withOpRefs(defaultOpFields())
			newFields = core.SetFieldValue(newFields, "name", newOp.Name)
			ed.items = append(ed.items, newFields)
			ed.itemIdx = len(ed.items) - 1
			ed.form = core.CopyFields(ed.items[ed.itemIdx])
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.repoSubView = beRepoSubViewOpForm
		}
	case "d":
		if n > 0 && validRepo {
			ops := be.Services[svcIdx].Repositories[repoIdx].Operations
			be.Services[svcIdx].Repositories[repoIdx].Operations = append(ops[:ed.itemIdx], ops[ed.itemIdx+1:]...)
			ed.items = append(ed.items[:ed.itemIdx], ed.items[ed.itemIdx+1:]...)
			if ed.itemIdx > 0 && ed.itemIdx >= len(ed.items) {
				ed.itemIdx = len(ed.items) - 1
			}
		}
	case "enter":
		if n > 0 {
			ed.form = be.withOpRefs(core.CopyFields(ed.items[ed.itemIdx]))
			ed.formIdx = 0
			ed.itemView = beListViewForm
			be.repoSubView = beRepoSubViewOpForm
		}
	case "b", "esc":
		ed.itemView = beListViewList
		be.repoEditor.itemView = beListViewForm
		be.repoSubView = beRepoSubViewForm
	}
	return be, nil
}

// ── op form update ───────────────────────────────────────────────────────────

func (be BackendEditor) updateOpForm(key tea.KeyMsg) (BackendEditor, tea.Cmd) {
	ed := &be.opEditor
	switch key.String() {
	case "j", "down":
		if ed.formIdx < len(ed.form)-1 {
			ed.formIdx++
		}
	case "k", "up":
		if ed.formIdx > 0 {
			ed.formIdx--
		}
	case "enter", " ":
		f := &ed.form[ed.formIdx]
		if (f.Kind == core.KindSelect || f.Kind == core.KindMultiSelect) && len(f.Options) > 0 {
			be.dd.Open = true
			if f.Kind == core.KindSelect {
				be.dd.OptIdx = f.SelIdx
			} else {
				be.dd.OptIdx = f.DDCursor
			}
		} else {
			return be.enterOpFormInsert()
		}
	case "H", "shift+left":
		f := &ed.form[ed.formIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if ed.form[ed.formIdx].CanEditAsText() {
			return be.enterOpFormInsert()
		}
	case "b", "esc":
		be.saveOpForm()
		ed.itemView = beListViewList
		be.repoSubView = beRepoSubViewOpList
	}
	be.saveOpForm()
	return be, nil
}

func (be BackendEditor) enterOpFormInsert() (BackendEditor, tea.Cmd) {
	ed := &be.opEditor
	f := ed.form[ed.formIdx]
	if !f.CanEditAsText() {
		return be, nil
	}
	be.internalMode = core.ModeInsert
	be.formInput.SetValue(f.TextInputValue())
	be.formInput.Width = be.width - 22
	be.formInput.CursorEnd()
	return be, be.formInput.Focus()
}
