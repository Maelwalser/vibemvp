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
	inp.Width = 46
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
			openIdx := 0
			newIdx := 1
			if w.recentPath != "" {
				openIdx = 1
				newIdx = 2
			}
			switch {
			case w.recentPath != "" && w.cursor == 0:
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

// vibeBanner — editorial block-letter wordmark, 3 lines × 28 chars.
var vibeBanner = [3]string{
	"╦  ╦╦╔╗ ╔═╗╔╦╗╔═╗╔╗╔╦ ╦",
	"╚╗╔╝║╠╩╗║╣ ║║║║╣ ║║║║ ║",
	" ╚╝ ╩╚═╝╚═╝╩ ╩╚═╝╝╚╝╚═╝",
}

// welcomeArtPanel is shown to the right of the menu inside the welcome modal.
var welcomeArtPanel = []string{
	`                   `,
	`   ┌─────────────┐ `,
	`   │ 01 BACKEND  │ `,
	`   │ 02 DATA     │ `,
	`   │ 03 CONTRACTS│ `,
	`   │ 04 FRONTEND │ `,
	`   │ 05 INFRA    │ `,
	`   │ 06 CROSSCUT │ `,
	`   │ 07 REALIZE  │ `,
	`   └─────────────┘ `,
	`                   `,
	`   ░░░░░░░░░░░░░░  `,
	`   ░ DECLARE   ░  `,
	`   ░ MANIFEST  ░  `,
	`   ░░░░░░░░░░░░░░  `,
	`                   `,
}

// View satisfies tea.Model.
func (w WelcomeModel) View() string {
	if w.width == 0 {
		return "Loading…"
	}

	modalBg := lipgloss.Color(clrBg2)
	dimColor := lipgloss.Color(clrFgDim)

	// Column widths: menu panel + spacer + art panel.
	menuW := 52
	artW := 21
	totalInner := menuW + artW
	boxWidth := totalInner + 4 // 2 padding each side

	innerStyle := lipgloss.NewStyle().Width(menuW).Background(modalBg)
	centerStyle := lipgloss.NewStyle().Width(menuW).Align(lipgloss.Center).Background(modalBg)

	// ── Left (menu) panel ────────────────────────────────────────────────────
	var leftB strings.Builder

	// Banner — warm amber gradient across 3 lines.
	bannerColors := [3]string{clrYellow, clrYellow, clrOrange}
	for i, line := range vibeBanner {
		styled := lipgloss.NewStyle().
			Foreground(lipgloss.Color(bannerColors[i])).
			Bold(true).
			Background(modalBg).
			Render(line)
		leftB.WriteString(centerStyle.Render(styled))
		leftB.WriteString("\n")
	}
	leftB.WriteString("\n")

	// Subtitle.
	subtitle := lipgloss.NewStyle().Foreground(dimColor).Background(modalBg).
		Render("DECLARATIVE  SYSTEM  ARCHITECTURE")
	leftB.WriteString(centerStyle.Render(subtitle))
	leftB.WriteString("\n")

	// Divider.
	div := lipgloss.NewStyle().Foreground(lipgloss.Color(clrComment)).Background(modalBg).
		Render(strings.Repeat("─", menuW-2))
	leftB.WriteString(div)
	leftB.WriteString("\n\n")

	switch w.phase {
	case welcomePhaseMenu:
		var options []string
		menuIcons := []string{}
		if w.recentPath != "" {
			options = append(options, "CONTINUE  "+shortPath(w.recentPath))
			menuIcons = append(menuIcons, "▶")
		}
		options = append(options, "OPEN EXISTING MANIFEST", "NEW PROJECT")
		menuIcons = append(menuIcons, "◈", "+")

		for i, opt := range options {
			icon := menuIcons[i]
			if i == w.cursor {
				indicator := lipgloss.NewStyle().
					Foreground(lipgloss.Color(clrYellow)).
					Bold(true).
					Background(modalBg).Render("❯ " + icon + " ")
				label := lipgloss.NewStyle().
					Foreground(lipgloss.Color(clrFg)).
					Bold(true).
					Background(modalBg).Render(opt)
				leftB.WriteString(innerStyle.Render(indicator + label))
			} else {
				indicator := lipgloss.NewStyle().Foreground(dimColor).Background(modalBg).Render("  " + icon + " ")
				label := lipgloss.NewStyle().Foreground(dimColor).Background(modalBg).Render(opt)
				leftB.WriteString(innerStyle.Render(indicator + label))
			}
			leftB.WriteString("\n")
		}
		if w.errMsg != "" {
			leftB.WriteString("\n")
			leftB.WriteString(StyleMsgErr.Background(modalBg).Render("✗ " + w.errMsg))
			leftB.WriteString("\n")
		}
		leftB.WriteString("\n")
		hints := lipgloss.NewStyle().Foreground(dimColor).Background(modalBg).
			Render("j/k  navigate   Enter  select   q  quit")
		leftB.WriteString(hints)

	case welcomePhaseOpenPath:
		leftB.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(modalBg).Render("◈  OPEN EXISTING MANIFEST"))
		leftB.WriteString("\n\n")
		leftB.WriteString(StyleFieldKeyActive.Background(modalBg).Render("Path ") +
			StyleEquals.Background(modalBg).Render(" = ") +
			w.input.View())
		if w.errMsg != "" {
			leftB.WriteString("\n\n")
			leftB.WriteString(StyleMsgErr.Background(modalBg).Render("✗ " + w.errMsg))
		}
		leftB.WriteString("\n\n")
		leftB.WriteString(lipgloss.NewStyle().Foreground(dimColor).Background(modalBg).
			Render("Enter  load     Esc  back"))

	case welcomePhaseNewName:
		leftB.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(clrYellow)).Bold(true).Background(modalBg).Render("+  NEW PROJECT"))
		leftB.WriteString("\n\n")
		leftB.WriteString(StyleFieldKeyActive.Background(modalBg).Render("Name ") +
			StyleEquals.Background(modalBg).Render(" = ") +
			w.input.View())
		if w.errMsg != "" {
			leftB.WriteString("\n\n")
			leftB.WriteString(StyleMsgErr.Background(modalBg).Render("✗ " + w.errMsg))
		}
		leftB.WriteString("\n\n")
		leftB.WriteString(lipgloss.NewStyle().Foreground(dimColor).Background(modalBg).
			Render("Enter  create   Esc  back"))
	}

	// ── Right (art) panel ───────────────────────────────────────────────────
	artStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment)).
		Background(modalBg)
	var artB strings.Builder
	for _, line := range welcomeArtPanel {
		artB.WriteString(artStyle.Render(line))
		artB.WriteString("\n")
	}

	// ── Combine left + right side by side ───────────────────────────────────
	leftLines := strings.Split(strings.TrimRight(leftB.String(), "\n"), "\n")
	artLines := strings.Split(strings.TrimRight(artB.String(), "\n"), "\n")

	maxLines := len(leftLines)
	if len(artLines) > maxLines {
		maxLines = len(artLines)
	}

	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment)).
		Background(modalBg)

	var combined strings.Builder
	for i := 0; i < maxLines; i++ {
		var left, right string
		if i < len(leftLines) {
			left = leftLines[i]
		}
		if i < len(artLines) {
			right = artLines[i]
		}
		lw := lipgloss.Width(left)
		if lw < menuW {
			left += lipgloss.NewStyle().Background(modalBg).Render(strings.Repeat(" ", menuW-lw))
		}
		combined.WriteString(left)
		combined.WriteString(sepStyle.Render("│"))
		combined.WriteString(right)
		if i < maxLines-1 {
			combined.WriteString("\n")
		}
	}

	box := lipgloss.NewStyle().
		Border(sharpBorder).
		BorderForeground(lipgloss.Color(clrComment)).
		Background(modalBg).
		Width(boxWidth).
		Padding(0, 1).
		Render(combined.String())

	return lipgloss.Place(w.width, w.height, lipgloss.Center, lipgloss.Center, box)
}
