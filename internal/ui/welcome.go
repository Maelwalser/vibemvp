package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
)

type welcomePhase int

const (
	welcomePhaseMenu     welcomePhase = iota
	welcomePhaseOpenPath              // user entering path to existing manifest
	welcomePhaseNewName               // user entering new project name
)

// WelcomeCompleteMsg is emitted as a Cmd when the welcome flow finishes.
type WelcomeCompleteMsg struct {
	Path     string             // file path to save/load
	IsNew    bool               // true = new project, false = open existing
	Manifest *manifest.Manifest // populated when opening an existing manifest
}

// WelcomeModel is the initial screen presented on startup.
type WelcomeModel struct {
	phase      welcomePhase
	cursor     int    // menu index
	recentPath string // most recently used manifest path, or ""
	input      textinput.Model
	errMsg     string
	width      int
	height     int
}

func newWelcomeModel() WelcomeModel {
	inp := newFormInput()
	inp.Width = 48
	recent := manifest.LoadRecentPaths()
	var recentPath string
	if len(recent) > 0 {
		recentPath = recent[0]
	}
	return WelcomeModel{input: inp, recentPath: recentPath}
}

// numMenuOptions returns how many options appear in the main menu.
func (w WelcomeModel) numMenuOptions() int {
	if w.recentPath != "" {
		return 3
	}
	return 2
}

// shortPath returns up to the last two path components for display.
func shortPath(p string) string {
	dir, file := filepath.Split(filepath.Clean(p))
	parent := filepath.Base(filepath.Clean(dir))
	if parent == "." || parent == "" {
		return file
	}
	return filepath.Join(parent, file)
}

// Init satisfies tea.Model — starts the animation ticker.
func (w WelcomeModel) Init() tea.Cmd {
	return uiTick()
}

// Update satisfies tea.Model.
func (w WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		return w, nil
	case uiTickMsg:
		AnimFrame = (AnimFrame + 1) % 2
		return w, uiTick()
	case tea.KeyMsg:
		return w.handleKey(msg)
	}
	// Forward non-key events to the text input when active.
	if w.phase != welcomePhaseMenu {
		var cmd tea.Cmd
		w.input, cmd = w.input.Update(msg)
		return w, cmd
	}
	return w, nil
}

func (w WelcomeModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch w.phase {
	case welcomePhaseMenu:
		n := w.numMenuOptions()
		switch key.String() {
		case "j", "down":
			w.cursor = (w.cursor + 1) % n
		case "k", "up":
			w.cursor = (w.cursor - 1 + n) % n
		case "enter", " ":
			// When a recent path is shown it occupies index 0;
			// Open Existing and New Project shift down by one.
			openIdx := 0
			newIdx := 1
			if w.recentPath != "" {
				openIdx = 1
				newIdx = 2
			}
			switch {
			case w.recentPath != "" && w.cursor == 0:
				// Load the most recent manifest directly.
				mf, err := manifest.Load(w.recentPath)
				if err != nil {
					w.errMsg = fmt.Sprintf("error: %v", err)
					return w, nil
				}
				path := w.recentPath
				return w, func() tea.Msg {
					return WelcomeCompleteMsg{Path: path, IsNew: false, Manifest: mf}
				}
			case w.cursor == openIdx:
				w.phase = welcomePhaseOpenPath
				w.input.Placeholder = "e.g. ./project/manifest.json"
				w.input.SetValue("")
				w.errMsg = ""
				return w, w.input.Focus()
			case w.cursor == newIdx:
				w.phase = welcomePhaseNewName
				w.input.Placeholder = "e.g. my-app"
				w.input.SetValue("")
				w.errMsg = ""
				return w, w.input.Focus()
			}
		case "ctrl+c", "q":
			return w, tea.Quit
		}

	case welcomePhaseOpenPath:
		switch key.String() {
		case "esc":
			w.phase = welcomePhaseMenu
			w.input.Blur()
			w.errMsg = ""
			return w, nil
		case "enter":
			path := strings.TrimSpace(w.input.Value())
			if path == "" {
				w.errMsg = "path cannot be empty"
				return w, nil
			}
			mf, err := manifest.Load(path)
			if err != nil {
				w.errMsg = fmt.Sprintf("error: %v", err)
				return w, nil
			}
			w.input.Blur()
			return w, func() tea.Msg {
				return WelcomeCompleteMsg{Path: path, IsNew: false, Manifest: mf}
			}
		default:
			var cmd tea.Cmd
			w.input, cmd = w.input.Update(key)
			return w, cmd
		}

	case welcomePhaseNewName:
		switch key.String() {
		case "esc":
			w.phase = welcomePhaseMenu
			w.input.Blur()
			w.errMsg = ""
			return w, nil
		case "enter":
			name := strings.TrimSpace(w.input.Value())
			if name == "" {
				w.errMsg = "project name cannot be empty"
				return w, nil
			}
			path := name + ".json"
			w.input.Blur()
			return w, func() tea.Msg {
				return WelcomeCompleteMsg{Path: path, IsNew: true, Manifest: nil}
			}
		default:
			var cmd tea.Cmd
			w.input, cmd = w.input.Update(key)
			return w, cmd
		}
	}
	return w, nil
}

// vibeBanner is the 3-line ASCII art logo rendered inside the welcome box.
// Width: 28 chars — fits comfortably within a 60-char inner area.
var vibeBanner = [3]string{
	"╦  ╦╦╔╗ ╔═╗╔╦╗╔═╗╔╗╔╦ ╦",
	"╚╗╔╝║╠╩╗║╣ ║║║║╣ ║║║║ ║",
	" ╚╝ ╩╚═╝╚═╝╩ ╩╚═╝╝╚╝╚═╝",
}

// View satisfies tea.Model.
func (w WelcomeModel) View() string {
	if w.width == 0 {
		return "Loading…"
	}

	const boxWidth = 62

	var b strings.Builder
	modalBg := lipgloss.Color(clrBg2)

	innerW := boxWidth - 4
	centerIn := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Background(modalBg)

	// ASCII art banner — gradient from teal → cyan → lavender across lines.
	bannerColors := [3]string{clrTeal, clrCyan, clrViolet}
	for i, line := range vibeBanner {
		styled := lipgloss.NewStyle().
			Foreground(lipgloss.Color(bannerColors[i])).
			Bold(true).
			Background(modalBg).
			Render(line)
		b.WriteString(centerIn.Render(styled))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Subtitle + decorative divider.
	subtitle := StyleHelpDesc.Background(modalBg).Render("declarative system architecture")
	b.WriteString(centerIn.Render(subtitle))
	b.WriteString("\n")
	divider := StyleDivider.Background(modalBg).Render(strings.Repeat("─", innerW))
	b.WriteString(divider)
	b.WriteString("\n\n")

	switch w.phase {
	case welcomePhaseMenu:
		var options []string
		menuIcons := []string{}
		if w.recentPath != "" {
			options = append(options, "Continue  "+shortPath(w.recentPath))
			menuIcons = append(menuIcons, "▶")
		}
		options = append(options, "Open Existing Manifest", "New Project")
		menuIcons = append(menuIcons, "◈", "✦")

		for i, opt := range options {
			icon := menuIcons[i]
			if i == w.cursor {
				indicator := StyleNeonViolet.Background(modalBg).Render("❯ " + icon + " ")
				label := StyleFieldValActive.Background(modalBg).Bold(true).Render(opt)
				b.WriteString(indicator + label)
			} else {
				indicator := StyleHelpDesc.Background(modalBg).Render("  " + icon + " ")
				label := StyleFieldVal.Background(modalBg).Render(opt)
				b.WriteString(indicator + label)
			}
			b.WriteString("\n")
		}
		if w.errMsg != "" {
			b.WriteString("\n")
			b.WriteString(StyleMsgErr.Background(modalBg).Render("✗ " + w.errMsg))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		hints := StyleHelpKey.Background(modalBg).Render("j/k") +
			StyleHelpDesc.Background(modalBg).Render(" navigate  ") +
			StyleHelpKey.Background(modalBg).Render("Enter") +
			StyleHelpDesc.Background(modalBg).Render(" select  ") +
			StyleHelpKey.Background(modalBg).Render("q") +
			StyleHelpDesc.Background(modalBg).Render(" quit")
		b.WriteString(hints)

	case welcomePhaseOpenPath:
		b.WriteString(StyleNeonTeal.Background(modalBg).Render("◈  Open Existing Manifest"))
		b.WriteString("\n\n")
		b.WriteString(StyleFieldKeyActive.Background(modalBg).Render("Path ") +
			StyleEquals.Background(modalBg).Render(" = ") +
			w.input.View())
		if w.errMsg != "" {
			b.WriteString("\n\n")
			b.WriteString(StyleMsgErr.Background(modalBg).Render("✗ " + w.errMsg))
		}
		b.WriteString("\n\n")
		b.WriteString(StyleHelpKey.Background(modalBg).Render("Enter") +
			StyleHelpDesc.Background(modalBg).Render(" load   ") +
			StyleHelpKey.Background(modalBg).Render("Esc") +
			StyleHelpDesc.Background(modalBg).Render(" back"))

	case welcomePhaseNewName:
		b.WriteString(StyleNeonViolet.Background(modalBg).Render("✦  New Project"))
		b.WriteString("\n\n")
		b.WriteString(StyleFieldKeyActive.Background(modalBg).Render("Name ") +
			StyleEquals.Background(modalBg).Render(" = ") +
			w.input.View())
		if w.errMsg != "" {
			b.WriteString("\n\n")
			b.WriteString(StyleMsgErr.Background(modalBg).Render("✗ " + w.errMsg))
		}
		b.WriteString("\n\n")
		b.WriteString(StyleHelpKey.Background(modalBg).Render("Enter") +
			StyleHelpDesc.Background(modalBg).Render(" create   ") +
			StyleHelpKey.Background(modalBg).Render("Esc") +
			StyleHelpDesc.Background(modalBg).Render(" back"))
	}

	box := StyleModalBorder.Width(boxWidth).Render(b.String())
	return lipgloss.Place(w.width, w.height, lipgloss.Center, lipgloss.Center, box)
}
