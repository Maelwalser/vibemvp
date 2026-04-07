package core

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Editorial Dark palette — exported so sub-packages (arch, provider) can use them.
const (
	ClrBg      = "#0C0C0C"
	ClrBg2     = "#131313"
	ClrCrust   = "#060606"
	ClrBgHL    = "#1C1C1C"
	ClrBgHL2   = "#252525"
	ClrFg      = "#E0E0E0"
	ClrFgDim   = "#525252"
	ClrBlue    = "#7AACCF"
	ClrCyan    = "#78B4B4"
	ClrTeal    = "#5A9690"
	ClrGreen   = "#6A9E6A"
	ClrYellow  = "#C8A05A"
	ClrRed     = "#A86060"
	ClrMagenta = "#8A78A8"
	ClrPink    = "#A87888"
	ClrComment = "#282828"
	ClrSel     = "#1C1C1C"
	ClrTabBg   = "#080808"
	ClrViolet  = "#7A6EA0"
	ClrOrange  = "#B08058"
)

// Package-internal aliases for convenience within core.
const (
	clrBg      = ClrBg
	clrBg2     = ClrBg2
	clrCrust   = ClrCrust
	clrBgHL    = ClrBgHL
	clrBgHL2   = ClrBgHL2
	clrFg      = ClrFg
	clrFgDim   = ClrFgDim
	clrBlue    = ClrBlue
	clrCyan    = ClrCyan
	clrTeal    = ClrTeal
	clrGreen   = ClrGreen
	clrYellow  = ClrYellow
	clrRed     = ClrRed
	clrMagenta = ClrMagenta
	clrPink    = ClrPink
	clrComment = ClrComment
	clrSel     = ClrSel
	clrTabBg   = ClrTabBg
	clrViolet  = ClrViolet
	clrOrange  = ClrOrange
)

// SharpBorder is the rectangular border used throughout the UI.
var SharpBorder = lipgloss.Border{
	Top:         "─",
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	TopLeft:     "┌",
	TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
}

// ActiveBorder — heavy weight for the active / focused pane.
var ActiveBorder = lipgloss.Border{
	Top:         "━",
	Bottom:      "━",
	Left:        "┃",
	Right:       "┃",
	TopLeft:     "┌",
	TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
}

var (
	// Mode indicator pills — high-contrast badges.
	StyleNormalMode = lipgloss.NewStyle().
			Background(lipgloss.Color(clrYellow)).
			Foreground(lipgloss.Color(clrBg)).
			Bold(true).
			Padding(0, 1)

	StyleInsertMode = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCyan)).
			Foreground(lipgloss.Color(clrBg)).
			Bold(true).
			Padding(0, 1)

	StyleCommandMode = lipgloss.NewStyle().
				Background(lipgloss.Color(clrMagenta)).
				Foreground(lipgloss.Color(clrBg)).
				Bold(true).
				Padding(0, 1)

	StyleStatusLine = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCrust)).
			Foreground(lipgloss.Color(clrFg))

	StyleStatusRight = lipgloss.NewStyle().
				Background(lipgloss.Color(clrCrust)).
				Foreground(lipgloss.Color(clrFgDim))

	// Tab bar — active tab uses warm amber, inactive uses near-black.
	StyleTabActive = lipgloss.NewStyle().
			Background(lipgloss.Color(clrYellow)).
			Foreground(lipgloss.Color(clrBg)).
			Bold(true).
			Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
				Background(lipgloss.Color(clrTabBg)).
				Foreground(lipgloss.Color(clrFgDim)).
				Padding(0, 1)

	StyleTabSep = lipgloss.NewStyle().
			Background(lipgloss.Color(clrTabBg)).
			Foreground(lipgloss.Color(clrComment))

	StyleTabBar = lipgloss.NewStyle().
			Background(lipgloss.Color(clrTabBg))

	// Line numbers — dim for inactive, amber for current.
	StyleLineNum = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFgDim))

	StyleCurLineNum = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrYellow)).
			Bold(true)

	StyleCurLine = lipgloss.NewStyle().
			Background(lipgloss.Color(clrBgHL))

	StyleCurLinePulse = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBgHL2))

	// Empty-line marker (tilde · equivalent).
	StyleTilde = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	// Form field keys — slate blue when inactive, amber when active.
	StyleFieldKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrBlue))

	StyleFieldKeyActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleEquals = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFgDim))

	StyleFieldVal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFg))

	StyleFieldValActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrFg)).
				Bold(true)

	StyleSelectArrow = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrFgDim))

	// Section headers.
	StyleSectionTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleSectionDesc = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrFgDim)).
				Italic(true)

	// Application header bar.
	StyleHeaderBar = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCrust)).
			Foreground(lipgloss.Color(clrFg))

	StyleHeaderTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleHeaderMod = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrRed)).
			Bold(true)

	// Command line.
	StyleCmdLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFg))

	// Status messages.
	StyleMsgOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrGreen)).
			Bold(true)

	StyleMsgErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrRed)).
			Bold(true)

	// Hint bar keys/descriptions.
	StyleHelpKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrYellow)).
			Bold(true)

	StyleHelpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFgDim))

	// Text area styling.
	StyleTextAreaLabel = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleTextAreaBorder = lipgloss.NewStyle().
				BorderStyle(SharpBorder).
				BorderForeground(lipgloss.Color(clrComment))

	StyleCursor = lipgloss.NewStyle().
			Background(lipgloss.Color(clrYellow)).
			Foreground(lipgloss.Color(clrBg))

	// Modal / panel borders.
	StyleModalBorder = lipgloss.NewStyle().
				Border(SharpBorder).
				BorderForeground(lipgloss.Color(clrComment)).
				Background(lipgloss.Color(clrBg2)).
				Padding(0, 1)

	StylePanelActive = lipgloss.NewStyle().
				Border(ActiveBorder).
				BorderForeground(lipgloss.Color(clrYellow)).
				Background(lipgloss.Color(clrBg))

	StylePanelInactive = lipgloss.NewStyle().
				Border(SharpBorder).
				BorderForeground(lipgloss.Color(clrComment)).
				Background(lipgloss.Color(clrBg2))

	StyleDivider = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	// Accent styles — all use the warm amber palette.
	StyleNeonMagenta = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrMagenta)).
				Bold(true)

	StyleNeonCyan = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrCyan)).
			Bold(true)

	StyleNeonGreen = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrGreen)).
			Bold(true)

	StyleNeonViolet = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrViolet)).
			Bold(true)

	StyleNeonOrange = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrOrange)).
			Bold(true)

	StyleNeonTeal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrTeal)).
			Bold(true)

	StyleHeaderDeco = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCrust)).
			Foreground(lipgloss.Color(clrFgDim))

	// Powerline / statusline segment styles.
	StyleStatusSegmentMode = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBg2)).
				Foreground(lipgloss.Color(clrFgDim))

	StyleStatusSegmentPos = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBg2)).
				Foreground(lipgloss.Color(clrBlue))

	// ASCII art side panel styles.
	// The art is rendered dim so it provides architectural visual texture
	// without overpowering the form content on the left.
	StyleArtPanel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#424242"))

	StyleArtPanelAccent = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#565656"))

	StyleArtSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#303030"))
)

// ActiveCurLineStyle returns the highlighted-row style (constant, no pulse animation).
func ActiveCurLineStyle() lipgloss.Style {
	return StyleCurLine
}

// HintBarBg renders a hint bar with explicit background on every segment.
// pairs is a variadic key/description alternating list.
func HintBarBg(bg lipgloss.Color, pairs ...string) string {
	if len(pairs)%2 != 0 {
		return ""
	}
	keyStyle := StyleHelpKey.Background(bg)
	descStyle := StyleHelpDesc.Background(bg)
	var hints []string
	for i := 0; i+1 < len(pairs); i += 2 {
		hints = append(hints, keyStyle.Render(pairs[i])+descStyle.Render(" "+pairs[i+1]))
	}
	sep := descStyle.Render("  │  ")
	prefix := lipgloss.NewStyle().Background(bg).Render("  ")
	return prefix + strings.Join(hints, sep)
}
