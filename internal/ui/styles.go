package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Macchiato palette — soft contrast with vivid accents
const (
	clrBg      = "#24273a" // base
	clrBg2     = "#1e2030" // mantle
	clrCrust   = "#181926" // crust — deepest background
	clrBgHL    = "#363a4f" // surface0 — active selection
	clrBgHL2   = "#494d64" // surface1 — pulse-frame selection
	clrFg      = "#cad3f5" // text
	clrFgDim   = "#6e738d" // overlay0
	clrBlue    = "#8aadf4" // blue
	clrCyan    = "#91d7e3" // sky
	clrTeal    = "#8bd5ca" // teal
	clrGreen   = "#a6da95" // green
	clrYellow  = "#eed49f" // yellow
	clrRed     = "#ed8796" // red
	clrMagenta = "#c6a0f6" // mauve
	clrPink    = "#f5bde6" // pink
	clrComment = "#5b6078" // surface2
	clrSel     = "#363a4f" // surface0
	clrTabBg   = "#1e2030" // mantle
	clrViolet  = "#b7bdf8" // lavender
	clrOrange  = "#f5a97f" // peach
)

// heavyRoundBorder mixes rounded corners with bold horizontal/vertical bars.
var heavyRoundBorder = lipgloss.Border{
	Top:         "─",
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "╰",
	BottomRight: "╯",
}

// panelBorderActive uses bright lavender for the active pane border.
var panelBorderActive = lipgloss.Border{
	Top:         "━",
	Bottom:      "━",
	Left:        "┃",
	Right:       "┃",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "╰",
	BottomRight: "╯",
}

var (
	StyleNormalMode = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCyan)).
			Foreground(lipgloss.Color(clrBg)).
			Bold(true).
			Padding(0, 1)

	StyleInsertMode = lipgloss.NewStyle().
			Background(lipgloss.Color(clrGreen)).
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
				Foreground(lipgloss.Color(clrComment))

	StyleTabActive = lipgloss.NewStyle().
			Background(lipgloss.Color(clrViolet)).
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

	StyleLineNum = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	StyleCurLineNum = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrCyan)).
				Bold(true)

	StyleCurLine = lipgloss.NewStyle().
			Background(lipgloss.Color(clrBgHL))

	StyleCurLinePulse = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBgHL2))

	StyleTilde = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	StyleFieldKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrBlue))

	StyleFieldKeyActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrCyan)).
				Bold(true)

	StyleEquals = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	StyleFieldVal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFg))

	StyleFieldValActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrFg)).
				Bold(true)

	StyleSelectArrow = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrViolet))

	StyleSectionTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrCyan)).
				Bold(true)

	StyleSectionDesc = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrComment)).
				Italic(true)

	StyleHeaderBar = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCrust)).
			Foreground(lipgloss.Color(clrFg))

	StyleHeaderTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrCyan)).
				Bold(true)

	StyleHeaderMod = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrOrange)).
			Bold(true)

	StyleCmdLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFg))

	StyleMsgOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrGreen)).
			Bold(true)

	StyleMsgErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrRed)).
			Bold(true)

	StyleHelpKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrCyan)).
			Bold(true)

	StyleHelpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	StyleTextAreaLabel = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrViolet)).
				Bold(true)

	StyleTextAreaBorder = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(clrViolet))

	StyleCursor = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCyan)).
			Foreground(lipgloss.Color(clrBg))

	// StyleModalBorder: active modal with heavy rounded border.
	StyleModalBorder = lipgloss.NewStyle().
				Border(panelBorderActive).
				BorderForeground(lipgloss.Color(clrViolet)).
				Background(lipgloss.Color(clrBg2)).
				Padding(0, 1)

	// StylePanelActive: high-contrast border for the active content pane.
	StylePanelActive = lipgloss.NewStyle().
				Border(panelBorderActive).
				BorderForeground(lipgloss.Color(clrViolet)).
				Background(lipgloss.Color(clrBg))

	// StylePanelInactive: dim border for inactive / background panes.
	StylePanelInactive = lipgloss.NewStyle().
				Border(heavyRoundBorder).
				BorderForeground(lipgloss.Color(clrComment)).
				Background(lipgloss.Color(clrBg2))

	// StyleDivider: thin horizontal rule between sections.
	StyleDivider = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrBgHL2))

	// Neon accent styles used in headers and decorations.
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

	// Powerline segment styles for statusline.
	StyleStatusSegmentMode = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBg2)).
				Foreground(lipgloss.Color(clrComment))

	StyleStatusSegmentPos = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBg2)).
				Foreground(lipgloss.Color(clrBlue))
)

// activeCurLineStyle returns the appropriate highlighted-row style based on the
// current animation frame, producing a subtle breathing/pulse effect.
func activeCurLineStyle() lipgloss.Style {
	if AnimFrame == 1 {
		return StyleCurLinePulse
	}
	return StyleCurLine
}
