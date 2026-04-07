package data

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Modes & views ─────────────────────────────────────────────────────────────

type dbeView int

const (
	dbeViewList dbeView = iota
	dbeViewForm
)

// ── DBEditor ─────────────────────────────────────────────────────────────────

// DBEditor manages a list of named database/cache sources in the DATABASES section.
type DBEditor struct {
	Sources []manifest.DBSourceDef

	view   dbeView
	srcIdx int

	internalMode core.Mode

	dbForm    []core.Field
	formIdx   int
	formInput textinput.Model

	dd   core.DropdownState
	cBuf bool

	width    int
	envNames []string // injected from InfraPillar.Environments

	undo core.UndoStack[[]manifest.DBSourceDef]
}

// SetEnvironmentNames injects environment names for the environment selector field
// in the database form. A no-op when unchanged.
func (db *DBEditor) SetEnvironmentNames(names []string) {
	db.envNames = names
	applyEnvNamesToDBForm(db.dbForm, names)
}

func newDBEditor() DBEditor {
	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFg))
	fi.Cursor.Style = core.StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFgDim))

	return DBEditor{
		Sources:   []manifest.DBSourceDef{},
		formInput: fi,
	}
}

// Mode returns the equivalent app-level core.Mode for the parent status bar.
func (db DBEditor) Mode() core.Mode {
	if db.internalMode == core.ModeInsert {
		return core.ModeInsert
	}
	return core.ModeNormal
}

// HintLine returns context-sensitive key hints.
func (db DBEditor) HintLine() string {
	if db.internalMode == core.ModeInsert {
		return core.StyleInsertMode.Render(" -- INSERT -- ") +
			core.StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch db.view {
	case dbeViewList:
		hints := []string{
			core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" navigate"),
			core.StyleHelpKey.Render("a") + core.StyleHelpDesc.Render(" add database"),
			core.StyleHelpKey.Render("d") + core.StyleHelpDesc.Render(" delete"),
			core.StyleHelpKey.Render("u") + core.StyleHelpDesc.Render(" undo"),
			core.StyleHelpKey.Render("Enter") + core.StyleHelpDesc.Render(" edit"),
			core.StyleHelpKey.Render(":w") + core.StyleHelpDesc.Render(" save"),
		}
		return "  " + strings.Join(hints, core.StyleHelpDesc.Render("  ·  "))
	case dbeViewForm:
		if db.dd.Open {
			hints := []string{
				core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" navigate"),
				core.StyleHelpKey.Render("Enter/Space") + core.StyleHelpDesc.Render(" select"),
				core.StyleHelpKey.Render("Esc") + core.StyleHelpDesc.Render(" cancel"),
			}
			return "  " + strings.Join(hints, core.StyleHelpDesc.Render("  ·  "))
		}
		hints := []string{
			core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" navigate"),
			core.StyleHelpKey.Render("i") + core.StyleHelpDesc.Render(" edit text"),
			core.StyleHelpKey.Render("Enter/Space") + core.StyleHelpDesc.Render(" dropdown"),
			core.StyleHelpKey.Render("b/Esc") + core.StyleHelpDesc.Render(" save & back"),
		}
		return "  " + strings.Join(hints, core.StyleHelpDesc.Render("  ·  "))
	}
	return ""
}

// ── Update ────────────────────────────────────────────────────────────────────

func (db DBEditor) Update(msg tea.Msg) (DBEditor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		db.width = wsz.Width
		db.formInput.Width = wsz.Width - 22
		return db, nil
	}
	switch db.internalMode {
	case core.ModeInsert:
		return db.updateInsert(msg)
	default:
		if key, ok := msg.(tea.KeyMsg); ok && !db.dd.Open {
			if key.String() == "c" {
				if db.cBuf {
					db.cBuf = false
					return db.clearDBFormInsert()
				}
				db.cBuf = true
				return db, nil
			}
			db.cBuf = false
		}
		return db.updateNormal(msg)
	}
}

func (db DBEditor) updateInsert(msg tea.Msg) (DBEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch key.String() {
		case "esc":
			db.dbForm[db.formIdx].SaveTextInput(db.formInput.Value())
			db.internalMode = core.ModeNormal
			db.formInput.Blur()
			return db, nil

		case "tab":
			db.dbForm[db.formIdx].SaveTextInput(db.formInput.Value())
			if next := core.NextEditableIdx(db.dbForm, db.formIdx); next >= 0 {
				db.formIdx = next
			} else {
				db.formIdx = nextDBFormIdx(db.dbForm, db.formIdx)
			}
			f := db.dbForm[db.formIdx]
			if !f.CanEditAsText() {
				db.formInput.Blur()
				return db, nil
			}
			db.formInput.SetValue(f.TextInputValue())
			db.formInput.CursorEnd()
			return db, db.formInput.Focus()

		case "shift+tab":
			db.dbForm[db.formIdx].SaveTextInput(db.formInput.Value())
			if prev := core.PrevEditableIdx(db.dbForm, db.formIdx); prev >= 0 {
				db.formIdx = prev
			} else {
				db.formIdx = prevDBFormIdx(db.dbForm, db.formIdx)
			}
			f := db.dbForm[db.formIdx]
			if !f.CanEditAsText() {
				db.formInput.Blur()
				return db, nil
			}
			db.formInput.SetValue(f.TextInputValue())
			db.formInput.CursorEnd()
			return db, db.formInput.Focus()
		}
	}
	var cmd tea.Cmd
	db.formInput, cmd = db.formInput.Update(msg)
	return db, cmd
}

func (db DBEditor) updateNormal(msg tea.Msg) (DBEditor, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return db, nil
	}
	if db.dd.Open && db.view == dbeViewForm {
		return db.updateDBDropdown(key)
	}
	switch db.view {
	case dbeViewList:
		return db.updateNormalList(key)
	case dbeViewForm:
		return db.updateNormalForm(key)
	}
	return db, nil
}

func (db DBEditor) updateNormalList(key tea.KeyMsg) (DBEditor, tea.Cmd) {
	n := len(db.Sources)
	switch key.String() {
	case "j", "down":
		if n > 0 && db.srcIdx < n-1 {
			db.srcIdx++
		}
	case "k", "up":
		if db.srcIdx > 0 {
			db.srcIdx--
		}
	case "g":
		db.srcIdx = 0
	case "G":
		if n > 0 {
			db.srcIdx = n - 1
		}
	case "u":
		if snap, ok := db.undo.Pop(); ok {
			db.Sources = snap
			if db.srcIdx >= len(db.Sources) && db.srcIdx > 0 {
				db.srcIdx = len(db.Sources) - 1
			}
		}
	case "d":
		if n > 0 {
			db.undo.Push(core.CopySlice(db.Sources))
			db.Sources = append(db.Sources[:db.srcIdx], db.Sources[db.srcIdx+1:]...)
			if db.srcIdx >= len(db.Sources) && db.srcIdx > 0 {
				db.srcIdx--
			}
		}
	case "a":
		db.undo.Push(core.CopySlice(db.Sources))
		db.Sources = append(db.Sources, manifest.DBSourceDef{
			Type: manifest.DBPostgres,
		})
		db.srcIdx = len(db.Sources) - 1
		db.dbForm = dbFormFromSourceWithEnvs(db.Sources[db.srcIdx], db.envNames)
		existing := make([]string, 0, len(db.Sources)-1)
		for i, s := range db.Sources {
			if i != db.srcIdx {
				existing = append(existing, s.Alias)
			}
		}
		db.dbForm = core.SetFieldValue(db.dbForm, "alias", core.UniqueName("database", existing))
		db.formIdx = 0
		db.view = dbeViewForm
		// Auto-focus alias field
		return db.enterDBFormInsert()

	case "enter", "l", "right":
		if n > 0 {
			db.dbForm = dbFormFromSourceWithEnvs(db.Sources[db.srcIdx], db.envNames)
			db.formIdx = 0
			db.view = dbeViewForm
		}
	}
	return db, nil
}

func (db DBEditor) updateNormalForm(key tea.KeyMsg) (DBEditor, tea.Cmd) {
	switch key.String() {
	case "j", "down":
		db.formIdx = nextDBFormIdx(db.dbForm, db.formIdx)
	case "k", "up":
		db.formIdx = prevDBFormIdx(db.dbForm, db.formIdx)
	case "g":
		db.formIdx = 0
	case "G":
		db.formIdx = len(db.dbForm) - 1
	case "enter", " ":
		f := &db.dbForm[db.formIdx]
		if f.Kind == core.KindSelect && len(f.Options) > 0 {
			db.dd.Open = true
			db.dd.OptIdx = f.SelIdx
		} else {
			return db.enterDBFormInsert()
		}
	case "H", "shift+left":
		f := &db.dbForm[db.formIdx]
		if f.Kind == core.KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if db.dbForm[db.formIdx].CanEditAsText() {
			return db.enterDBFormInsert()
		}
	case "b", "esc":
		db.saveFormBack()
		db.view = dbeViewList
	}
	db.saveFormBack()
	return db, nil
}

func (db DBEditor) updateDBDropdown(key tea.KeyMsg) (DBEditor, tea.Cmd) {
	if db.formIdx >= len(db.dbForm) {
		db.dd.Open = false
		return db, nil
	}
	f := &db.dbForm[db.formIdx]
	db.dd.OptIdx = core.NavigateDropdown(key.String(), db.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ", "enter":
		f.SelIdx = db.dd.OptIdx
		if db.dd.OptIdx < len(f.Options) {
			f.Value = f.Options[db.dd.OptIdx]
		}
		db.dd.Open = false
		if f.PrepareCustomEntry() {
			return db.enterDBFormInsert()
		}
	case "esc", "b":
		db.dd.Open = false
	}
	db.saveFormBack()
	return db, nil
}

func (db DBEditor) clearDBFormInsert() (DBEditor, tea.Cmd) {
	if db.view != dbeViewForm {
		return db, nil
	}
	db, cmd := db.enterDBFormInsert()
	if db.internalMode == core.ModeInsert {
		db.formInput.SetValue("")
	}
	return db, cmd
}

func (db DBEditor) enterDBFormInsert() (DBEditor, tea.Cmd) {
	f := db.dbForm[db.formIdx]
	if !f.CanEditAsText() {
		return db, nil
	}
	db.internalMode = core.ModeInsert
	db.formInput.SetValue(f.TextInputValue())
	db.formInput.Width = db.width - 22
	db.formInput.CursorEnd()
	return db, db.formInput.Focus()
}

func (db *DBEditor) saveFormBack() {
	if db.srcIdx >= len(db.Sources) {
		return
	}
	db.Sources[db.srcIdx] = dbFormToSource(db.dbForm)
}

// ── View ──────────────────────────────────────────────────────────────────────

// CurrentField returns the currently highlighted form field for the description panel.
// Returns nil when in list view.
func (db *DBEditor) CurrentField() *core.Field {
	if db.view == dbeViewForm && db.formIdx >= 0 && db.formIdx < len(db.dbForm) {
		return &db.dbForm[db.formIdx]
	}
	return nil
}

func (db DBEditor) View(w, h int) string {
	db.width = w
	db.formInput.Width = w - 22
	switch db.view {
	case dbeViewList:
		return db.viewList(w, h)
	case dbeViewForm:
		return db.viewForm(w, h)
	}
	return ""
}

func (db DBEditor) viewList(w, h int) string {
	var lines []string
	lines = append(lines,
		core.StyleSectionDesc.Render("  # Database sources — a: add  d: delete  Enter: configure"),
		"",
	)

	const dbListHeaderH = 2
	var itemLines []string
	if len(db.Sources) == 0 {
		itemLines = append(itemLines, core.StyleSectionDesc.Render("  (no databases yet — press 'a' to add one)"))
	} else {
		for i, src := range db.Sources {
			isCur := i == db.srcIdx

			arrow := "  ▸ "
			alias := src.Alias
			if alias == "" {
				alias = "(unnamed)"
			}
			if isCur {
				arrow = core.StyleCurLineNum.Render("  ▶ ")
				alias = core.StyleFieldKeyActive.Render(alias)
			} else {
				alias = core.StyleFieldKey.Render(alias)
			}

			typeStr := string(src.Type)
			var tags []string
			if src.IsCache {
				tags = append(tags, core.StyleMsgOK.Render("CACHE"))
			}
			if src.Version != "" {
				tags = append(tags, core.StyleSectionDesc.Render("v"+src.Version))
			}
			if src.Namespace != "" {
				tags = append(tags, core.StyleSectionDesc.Render(src.Namespace))
			}
			tagStr := ""
			if len(tags) > 0 {
				tagStr = "  " + strings.Join(tags, " ")
			}

			pad := max(1, 20-len(src.Alias))
			typeRendered := core.StyleFieldVal.Render(fmt.Sprintf("%-14s", typeStr))
			row := arrow + alias + strings.Repeat(" ", pad) + typeRendered + tagStr

			if isCur {
				raw := lipgloss.Width(row)
				if raw < w {
					row += strings.Repeat(" ", w-raw)
				}
				row = core.ActiveCurLineStyle().Render(row)
			}
			itemLines = append(itemLines, row)
		}
	}

	itemLines = core.ViewportSlice(itemLines, db.srcIdx, h-dbListHeaderH)
	lines = append(lines, itemLines...)
	return core.FillTildes(lines, h)
}

func (db DBEditor) viewForm(w, h int) string {
	srcAlias := "(new database)"
	if db.srcIdx < len(db.Sources) && db.Sources[db.srcIdx].Alias != "" {
		srcAlias = db.Sources[db.srcIdx].Alias
	}

	var lines []string
	breadcrumb := core.StyleSectionDesc.Render("  ← ") + core.StyleFieldKey.Render(srcAlias)
	lines = append(lines, breadcrumb, "")

	const labelW = 14
	const eqW = 3
	valW := w - 4 - labelW - eqW - 1
	if valW < 10 {
		valW = 10
	}

	visIdx := 0
	for i, f := range db.dbForm {
		if isDBFormFieldDisabled(db.dbForm, i) {
			continue
		}
		isCur := i == db.formIdx
		visIdx++

		lineNo := core.StyleLineNum.Render(fmt.Sprintf("%3d ", visIdx))
		if isCur {
			lineNo = core.StyleCurLineNum.Render(fmt.Sprintf("%3d ", visIdx))
		}

		var keyStr string
		switch {
		case isCur:
			keyStr = core.StyleFieldKeyActive.Render(f.Label)
		default:
			keyStr = core.StyleFieldKey.Render(f.Label)
		}

		eq := core.StyleEquals.Render(" = ")

		var valStr string
		switch {
		case db.internalMode == core.ModeInsert && isCur && f.Kind == core.KindText:
			valStr = db.formInput.View()
		case f.Kind == core.KindSelect:
			val := f.DisplayValue()
			if isCur {
				val = core.StyleFieldValActive.Render(val)
			} else {
				val = core.StyleFieldVal.Render(val)
			}
			if isCur && db.dd.Open {
				valStr = val + core.StyleSelectArrow.Render(" ▴")
			} else {
				valStr = val + core.StyleSelectArrow.Render(" ▾")
			}
		default:
			dv := f.DisplayValue()
			if len(dv) > valW {
				dv = dv[:valW-1] + "…"
			}
			if dv == "" {
				valStr = core.StyleSectionDesc.Render("_")
			} else if isCur {
				valStr = core.StyleFieldValActive.Render(dv)
			} else {
				valStr = core.StyleFieldVal.Render(dv)
			}
		}

		row := lineNo + keyStr + eq + valStr
		if isCur {
			raw := lipgloss.Width(row)
			if raw < w {
				row += strings.Repeat(" ", w-raw)
			}
			row = core.ActiveCurLineStyle().Render(row)
		}
		lines = append(lines, row)

		// Inject dropdown options below the active core.KindSelect field
		if isCur && db.dd.Open && f.Kind == core.KindSelect {
			const ddIndent = 4 + 14 + 3 // lineNumW + labelW + eqW
			indent := strings.Repeat(" ", ddIndent)
			for j, opt := range f.Options {
				isHL := j == db.dd.OptIdx
				var optRow string
				if isHL {
					optRow = indent + core.StyleFieldValActive.Render("► "+opt)
					rw := lipgloss.Width(optRow)
					if rw < w {
						optRow += strings.Repeat(" ", w-rw)
					}
					optRow = core.ActiveCurLineStyle().Render(optRow)
				} else {
					optRow = indent + core.StyleFieldVal.Render("  "+opt)
				}
				lines = append(lines, optRow)
			}
		}
	}

	return core.FillTildes(lines, h)
}
