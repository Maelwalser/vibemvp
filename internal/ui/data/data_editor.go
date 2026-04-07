package data

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/ui/core"
)

// ── Modes & views ─────────────────────────────────────────────────────────────

type deMode int

const (
	deNormal deMode = iota
	deInsert        // typing in a column form text field
	deNaming        // typing a new entity or column name
)

type deView int

const (
	deViewEntities       deView = iota
	deViewEntitySettings        // entity-level: database assignment, caching
	deViewColumns
	deViewColForm
)

// ── DataEditor ────────────────────────────────────────────────────────────────

// DataEditor is a self-contained entity/column schema editor embedded in the
// DATA section of the manifest TUI.
type DataEditor struct {
	Entities []manifest.EntityDef

	// availableDbs is synced from DBEditor so the entity settings form can
	// present live database/cache selects.
	availableDbs []manifest.DBSourceDef

	view       deView
	entityIdx  int
	columnIdx  int
	colFormIdx int

	// entForm holds mutable field state for the entity-level settings form.
	entForm    []core.Field
	entFormIdx int

	internalMode deMode

	// nameInput is used for typing new entity / column names.
	nameInput  textinput.Model
	nameTarget string // "entity" or "column"

	// formInput is the shared text input for core.KindText fields in any active form.
	formInput textinput.Model

	// colForm holds the mutable field state for the column currently being edited.
	colForm []core.Field

	dd core.DropdownState

	width int
}

// newDataEditor returns an initialised, empty DataEditor.
func newDataEditor() DataEditor {
	ni := textinput.New()
	ni.Prompt = ""
	ni.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFgDim))

	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFg))
	fi.Cursor.Style = core.StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFgDim))

	return DataEditor{
		Entities:  []manifest.EntityDef{},
		nameInput: ni,
		formInput: fi,
	}
}

// Mode returns the equivalent app-level core.Mode for the parent status bar.
func (de DataEditor) Mode() core.Mode {
	if de.internalMode == deInsert || de.internalMode == deNaming {
		return core.ModeInsert
	}
	return core.ModeNormal
}

// HintLine returns context-sensitive key hints for the bottom help bar.
func (de DataEditor) HintLine() string {
	switch de.internalMode {
	case deNaming:
		return core.StyleInsertMode.Render(" -- INSERT -- ") +
			core.StyleHelpDesc.Render("  Enter: confirm  Esc: cancel")
	case deInsert:
		return core.StyleInsertMode.Render(" -- INSERT -- ") +
			core.StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch de.view {
	case deViewEntities:
		hints := []string{
			core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" navigate"),
			core.StyleHelpKey.Render("a") + core.StyleHelpDesc.Render(" add entity"),
			core.StyleHelpKey.Render("d") + core.StyleHelpDesc.Render(" delete"),
			core.StyleHelpKey.Render("Enter") + core.StyleHelpDesc.Render(" settings & columns"),
			core.StyleHelpKey.Render(":w") + core.StyleHelpDesc.Render(" save"),
		}
		return "  " + strings.Join(hints, core.StyleHelpDesc.Render("  ·  "))
	case deViewEntitySettings:
		hints := []string{
			core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" navigate"),
			core.StyleHelpKey.Render("i") + core.StyleHelpDesc.Render(" edit"),
			core.StyleHelpKey.Render("Space") + core.StyleHelpDesc.Render(" cycle"),
			core.StyleHelpKey.Render("c") + core.StyleHelpDesc.Render(" columns"),
			core.StyleHelpKey.Render("b") + core.StyleHelpDesc.Render(" back"),
		}
		return "  " + strings.Join(hints, core.StyleHelpDesc.Render("  ·  "))
	case deViewColumns:
		hints := []string{
			core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" navigate"),
			core.StyleHelpKey.Render("a") + core.StyleHelpDesc.Render(" add column"),
			core.StyleHelpKey.Render("d") + core.StyleHelpDesc.Render(" delete"),
			core.StyleHelpKey.Render("Enter") + core.StyleHelpDesc.Render(" edit"),
			core.StyleHelpKey.Render("b") + core.StyleHelpDesc.Render(" back"),
		}
		return "  " + strings.Join(hints, core.StyleHelpDesc.Render("  ·  "))
	case deViewColForm:
		hints := []string{
			core.StyleHelpKey.Render("j/k") + core.StyleHelpDesc.Render(" navigate"),
			core.StyleHelpKey.Render("i") + core.StyleHelpDesc.Render(" edit text"),
			core.StyleHelpKey.Render("Space") + core.StyleHelpDesc.Render(" cycle option"),
			core.StyleHelpKey.Render("b/Esc") + core.StyleHelpDesc.Render(" save & back"),
		}
		return "  " + strings.Join(hints, core.StyleHelpDesc.Render("  ·  "))
	}
	return ""
}
