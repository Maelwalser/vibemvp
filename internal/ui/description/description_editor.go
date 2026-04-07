package description

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/ui/core"
)

// Editor is the first tab — a large free-text area for describing
// the project in natural language before filling in the structured pillars.
type Editor struct {
	ta   textarea.Model
	mode core.Mode
}

func NewEditor() Editor {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = "  "
	ta.Placeholder = "Describe your project…\n\nWhat kind of system are you building?\nWhat are the main goals and constraints?\nWhat users does it serve?"
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(core.ClrBgHL))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(lipgloss.Color(core.ClrBgHL))
	ta.FocusedStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(core.ClrFg))
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(core.ClrBgHL))
	ta.BlurredStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(core.ClrFgDim))
	return Editor{ta: ta}
}

// Mode implements core.Editor.
func (e Editor) Mode() core.Mode { return e.mode }

// HintLine implements core.Editor.
func (e Editor) HintLine() string {
	if e.mode == core.ModeInsert {
		return core.StyleInsertMode.Render(" ▷ INSERT ◁ ") +
			core.StyleHelpDesc.Render("  Esc: normal mode  │  Type freely to describe your project")
	}
	hints := []string{
		core.StyleHelpKey.Render("i") + core.StyleHelpDesc.Render(" edit"),
		core.StyleHelpKey.Render("Tab") + core.StyleHelpDesc.Render(" next section"),
		core.StyleHelpKey.Render(":w") + core.StyleHelpDesc.Render(" save"),
		core.StyleHelpKey.Render(":q") + core.StyleHelpDesc.Render(" quit"),
	}
	sep := core.StyleHelpDesc.Render("  │  ")
	return "  " + strings.Join(hints, sep)
}

// View implements core.Editor.
func (e Editor) View(w, h int) string {
	// Textarea height: content area minus header line and blank separator.
	taH := h - 2
	if taH < 3 {
		taH = 3
	}

	// Use a local copy so we don't mutate the receiver in View.
	ta := e.ta
	ta.SetWidth(w)
	ta.SetHeight(taH)

	// Split textarea output into individual lines.
	taLines := strings.Split(ta.View(), "\n")
	// Trim to taH lines in case the widget emits extras.
	if len(taLines) > taH {
		taLines = taLines[:taH]
	}

	// Collect all output lines: header, blank, textarea rows.
	// The textarea prompt ("  ") already provides left-side spacing, so no
	// extra indent is needed. Setting width=w ensures the textarea background
	// covers the full content area with no gaps on either side.
	lines := make([]string, 0, h)
	lines = append(lines, core.StyleSectionDesc.Render("  # Describe your project — what are you building?"))
	lines = append(lines, "")
	lines = append(lines, taLines...)

	// fillTildes ensures exactly h lines so the tab bar is never pushed off-screen.
	return core.FillTildes(lines, h)
}

// Update handles keyboard input for the description editor.
func (e Editor) Update(msg tea.Msg) (Editor, tea.Cmd) {
	if wsz, ok := msg.(tea.WindowSizeMsg); ok {
		e.ta.SetWidth(wsz.Width)
		e.ta.SetHeight(wsz.Height - 7)
		return e, nil
	}

	key, ok := msg.(tea.KeyMsg)
	if ok {
		switch e.mode {
		case core.ModeNormal:
			switch key.String() {
			case "i", "a", "enter":
				e.mode = core.ModeInsert
				return e, e.ta.Focus()
			}
			return e, nil
		case core.ModeInsert:
			if key.String() == "esc" {
				e.mode = core.ModeNormal
				e.ta.Blur()
				return e, nil
			}
		}
	}

	if e.mode == core.ModeInsert {
		var cmd tea.Cmd
		e.ta, cmd = e.ta.Update(msg)
		return e, cmd
	}
	return e, nil
}

// Value returns the current description text.
func (e Editor) Value() string { return e.ta.Value() }

// SetValue sets the description text (used when loading a manifest).
func (e *Editor) SetValue(v string) { e.ta.SetValue(v) }
