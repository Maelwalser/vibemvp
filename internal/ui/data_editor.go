package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

// ── Modes & views ─────────────────────────────────────────────────────────────

type deMode int

const (
	deNormal deMode = iota
	deInsert // typing in a column form text field
	deNaming // typing a new entity or column name
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
	entForm    []Field
	entFormIdx int

	internalMode deMode

	// nameInput is used for typing new entity / column names.
	nameInput  textinput.Model
	nameTarget string // "entity" or "column"

	// formInput is the shared text input for KindText fields in any active form.
	formInput textinput.Model

	// colForm holds the mutable field state for the column currently being edited.
	colForm []Field

	dd DropdownState

	width int
}

// newDataEditor returns an initialised, empty DataEditor.
func newDataEditor() DataEditor {
	ni := textinput.New()
	ni.Prompt = ""
	ni.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	fi.CursorStyle = StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))

	return DataEditor{
		Entities:  []manifest.EntityDef{},
		nameInput: ni,
		formInput: fi,
	}
}

// Mode returns the equivalent app-level Mode for the parent status bar.
func (de DataEditor) Mode() Mode {
	if de.internalMode == deInsert || de.internalMode == deNaming {
		return ModeInsert
	}
	return ModeNormal
}

// HintLine returns context-sensitive key hints for the bottom help bar.
func (de DataEditor) HintLine() string {
	switch de.internalMode {
	case deNaming:
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Enter: confirm  Esc: cancel")
	case deInsert:
		return StyleInsertMode.Render(" -- INSERT -- ") +
			StyleHelpDesc.Render("  Esc: normal  Tab: next field")
	}
	switch de.view {
	case deViewEntities:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("a") + StyleHelpDesc.Render(" add entity"),
			StyleHelpKey.Render("d") + StyleHelpDesc.Render(" delete"),
			StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" settings & columns"),
			StyleHelpKey.Render(":w") + StyleHelpDesc.Render(" save"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	case deViewEntitySettings:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("i") + StyleHelpDesc.Render(" edit"),
			StyleHelpKey.Render("Space") + StyleHelpDesc.Render(" cycle"),
			StyleHelpKey.Render("c") + StyleHelpDesc.Render(" columns"),
			StyleHelpKey.Render("b") + StyleHelpDesc.Render(" back"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	case deViewColumns:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("a") + StyleHelpDesc.Render(" add column"),
			StyleHelpKey.Render("d") + StyleHelpDesc.Render(" delete"),
			StyleHelpKey.Render("Enter") + StyleHelpDesc.Render(" edit"),
			StyleHelpKey.Render("b") + StyleHelpDesc.Render(" back"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	case deViewColForm:
		hints := []string{
			StyleHelpKey.Render("j/k") + StyleHelpDesc.Render(" navigate"),
			StyleHelpKey.Render("i") + StyleHelpDesc.Render(" edit text"),
			StyleHelpKey.Render("Space") + StyleHelpDesc.Render(" cycle option"),
			StyleHelpKey.Render("b/Esc") + StyleHelpDesc.Render(" save & back"),
		}
		return "  " + strings.Join(hints, StyleHelpDesc.Render("  ·  "))
	}
	return ""
}

