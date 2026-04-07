package core

import (
	"math"
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// AnimFrame is the current global animation frame index (0 or 1).
// It is toggled by the root model's ticker and read by render helpers
// to produce pulse / breathing effects on the active selection.
var AnimFrame int

// uiTickMsg is sent on each animation tick.
type UITickMsg struct{}

// uiTick returns a command that sends uiTickMsg after the animation interval.
func UITick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return UITickMsg{}
	})
}

// modeSpinFrames are the two-frame Nerd Font powerline decorators flanking the mode badge.
// Frame 0 uses half-circle powerline glyphs; frame 1 uses solid-arrow glyphs.
// Requires a Nerd Font patched terminal font (e.g. JetBrainsMono Nerd Font).
var ModeSpinFrames = [2][2]string{
	{"\ue0b6", "\ue0b4"}, //  … round left / round right (Nerd Font PUA)
	{"\ue0b2", "\ue0b0"}, //  … solid arrow left / right
}

// matrixChars is the character pool used for matrix-rain and glitch decorations.
// Mixes half-width katakana, digits, and box-drawing elements.
const matrixChars = "ｦｧｨｩｪｫｬｭｮｯｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜﾝ0123456789░▒▓│╮╭╰╯"

// MatrixSpinnerFrames is a 12-frame braille spinner styled for matrix output screens.
var MatrixSpinnerFrames = [12]string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷", "⡿", "⣟", "⣯", "⣷"}

// sineWaveFrames holds 16 pre-computed frames of a block-element sine-wave bar.
// Each frame is 8 chars wide, suitable for inline progress / loader display.
var SineWaveFrames [16]string

func init() {
	blockLevels := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}
	n := len(blockLevels)
	for f := 0; f < 16; f++ {
		var s string
		for i := 0; i < 8; i++ {
			v := (math.Sin(float64(i+f)*math.Pi/4) + 1) / 2
			idx := int(math.Round(v * float64(n-1)))
			if idx < 0 {
				idx = 0
			}
			if idx >= n {
				idx = n - 1
			}
			s += blockLevels[idx]
		}
		SineWaveFrames[f] = s
	}
}

// RandomMatrixChar returns a random character from the matrix character pool.
func RandomMatrixChar() string {
	runes := []rune(matrixChars)
	return string(runes[rand.Intn(len(runes))])
}
