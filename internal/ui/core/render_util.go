package core

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// withoutField returns a copy of fields with the entry matching key removed.
func WithoutField(fields []Field, key string) []Field {
	out := make([]Field, 0, len(fields))
	for _, f := range fields {
		if f.Key != key {
			out = append(out, f)
		}
	}
	return out
}

// copyFields makes a deep copy of a field slice, duplicating the Options slice
// so mutations to one copy do not affect others.
func CopyFields(src []Field) []Field {
	dst := make([]Field, len(src))
	for i, f := range src {
		dst[i] = f
		if f.Options != nil {
			dst[i].Options = make([]string, len(f.Options))
			copy(dst[i].Options, f.Options)
		}
	}
	return dst
}

// fieldGet returns the DisplayValue for the field with the given key in a slice.
func FieldGet(fields []Field, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return f.DisplayValue()
		}
	}
	return ""
}

// fieldGetSelectedSlice returns the selected option names for a KindMultiSelect
// field as a []string, or nil if the field is not found or nothing is selected.
func FieldGetSelectedSlice(fields []Field, key string) []string {
	for _, f := range fields {
		if f.Key != key {
			continue
		}
		var out []string
		for _, idx := range f.SelectedIdxs {
			if idx < len(f.Options) {
				out = append(out, f.Options[idx])
			}
		}
		return out
	}
	return nil
}

// fieldGetMulti returns the comma-separated DisplayValue for a KindMultiSelect field,
// or the plain DisplayValue for any other kind.
func FieldGetMulti(fields []Field, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return f.DisplayValue()
		}
	}
	return ""
}

// setFieldValue sets the value (and SelIdx for select fields) for the field
// with the given key in a slice, returning the modified slice.
// For KindSelect fields where val is not found in Options: if a "Custom" or "Other"
// option exists it is selected and val is stored in CustomText (round-trip support).
func SetFieldValue(fields []Field, key, val string) []Field {
	for i := range fields {
		if fields[i].Key != key {
			continue
		}
		fields[i].Value = val
		if fields[i].Kind == KindSelect {
			matched := false
			for j, opt := range fields[i].Options {
				if opt == val {
					fields[i].SelIdx = j
					matched = true
					break
				}
			}
			// Value not in options — try to use a Custom/Other sentinel option.
			if !matched && val != "" {
				for j, opt := range fields[i].Options {
					if IsCustomOption(opt) {
						fields[i].SelIdx = j
						fields[i].CustomText = val
						break
					}
				}
			}
		}
		return fields
	}
	return fields
}

// restoreMultiSelectValue restores SelectedIdxs for a KindMultiSelect field from
// a comma-separated value string (as produced by DisplayValue). Skips unknown tokens.
func RestoreMultiSelectValue(fields []Field, key, val string) []Field {
	if val == "" {
		return fields
	}
	for i := range fields {
		if fields[i].Key != key || fields[i].Kind != KindMultiSelect {
			continue
		}
		fields[i].SelectedIdxs = nil
		for _, part := range strings.Split(val, ", ") {
			part = strings.TrimSpace(part)
			found := false
			for j, opt := range fields[i].Options {
				if opt == part {
					fields[i].SelectedIdxs = append(fields[i].SelectedIdxs, j)
					found = true
					break
				}
			}
			// ColorSwatch fields: custom hexes are not in the static palette,
			// so inject them dynamically (preserving round-trip through manifest).
			if !found && fields[i].ColorSwatch && strings.HasPrefix(part, "#") {
				fields[i].AddCustomHexColor(part)
			}
		}
		fields[i].Value = val
		return fields
	}
	return fields
}

// stringSlicesEqual returns true when a and b have the same length and elements.
func StringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// nextFormIdx returns the next non-disabled field index after cur, wrapping around.
// disabled is a predicate called with the full field slice and a candidate index.
func NextFormIdx(form []Field, cur int, disabled func([]Field, int) bool) int {
	n := len(form)
	if n == 0 {
		return cur
	}
	next := (cur + 1) % n
	for next != cur && disabled(form, next) {
		next = (next + 1) % n
	}
	return next
}

// prevFormIdx returns the previous non-disabled field index before cur, wrapping around.
func PrevFormIdx(form []Field, cur int, disabled func([]Field, int) bool) int {
	n := len(form)
	if n == 0 {
		return cur
	}
	prev := (cur - 1 + n) % n
	for prev != cur && disabled(form, prev) {
		prev = (prev - 1 + n) % n
	}
	return prev
}

// parseVimCount converts a digit buffer (e.g. "3", "12") to an integer count.
// Returns 1 when the buffer is empty. Caps at 999 for sanity.
func ParseVimCount(buf string) int {
	if buf == "" {
		return 1
	}
	n := 0
	for _, c := range buf {
		if c < '0' || c > '9' {
			return 1
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 1
	}
	if n > 999 {
		return 999
	}
	return n
}

// uniqueName returns base if not present in existing, otherwise appends an
// incrementing suffix (base1, base2, …) until a free name is found.
func UniqueName(base string, existing []string) string {
	taken := make(map[string]bool, len(existing))
	for _, n := range existing {
		taken[n] = true
	}
	if !taken[base] {
		return base
	}
	for i := 1; ; i++ {
		if candidate := fmt.Sprintf("%s%d", base, i); !taken[candidate] {
			return candidate
		}
	}
}

// newFormInput creates a standard textinput for use in form editors.
func NewFormInput() textinput.Model {
	fi := textinput.New()
	fi.Prompt = ""
	fi.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFg))
	fi.Cursor.Style = StyleCursor
	fi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(clrFgDim))
	return fi
}

// newFormTextArea creates a styled multi-line textarea for use in form editors.
func NewFormTextArea() textarea.Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.SetHeight(8)
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg)).
		Background(lipgloss.Color(clrBg)).
		BorderStyle(SharpBorder).
		BorderForeground(lipgloss.Color(clrYellow))
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg)).
		Background(lipgloss.Color(clrBg)).
		BorderStyle(SharpBorder).
		BorderForeground(lipgloss.Color(clrComment))
	ta.FocusedStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg)).
		Background(lipgloss.Color(clrBg))
	ta.BlurredStyle.Text = ta.FocusedStyle.Text
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color(clrBgHL))
	ta.CharLimit = 0
	return ta
}

// placeOverlay paints fg on top of bg at position (x, y), where x and y are
// zero-based visible-column and line indices. Lines outside bg bounds are
// skipped. The portion of each bg line to the right of the overlay is
// preserved as plain (un-styled) text so the overlay always looks clean.
func PlaceOverlay(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fgLine := range fgLines {
		idx := y + i
		if idx < 0 || idx >= len(bgLines) {
			continue
		}
		fgW := lipgloss.Width(fgLine)
		bgW := lipgloss.Width(bgLines[idx])

		// Left part: bg up to column x (ANSI-aware truncation).
		left := lipgloss.NewStyle().MaxWidth(x).Render(bgLines[idx])
		leftW := lipgloss.Width(left)
		if leftW < x {
			left += strings.Repeat(" ", x-leftW)
		}

		// Right part: plain text after the overlay's right edge.
		right := ""
		rightStart := x + fgW
		if rightStart < bgW {
			plain := StripANSI(bgLines[idx])
			runes := []rune(plain)
			if rightStart < len(runes) {
				right = string(runes[rightStart:])
			}
		}

		bgLines[idx] = left + fgLine + right
	}

	return strings.Join(bgLines, "\n")
}

// appendViewport applies a scrolling viewport to a list-with-header layout.
// It preserves the first headerH lines unchanged and applies viewportSlice to
// the remaining item lines, keeping the item at itemIdx visible within the
// available height. Returns the combined fixed+scrolled slice.
func AppendViewport(lines []string, headerH, itemIdx, available int) []string {
	if available <= 0 || len(lines) <= available {
		return lines
	}
	if headerH > len(lines) {
		headerH = len(lines)
	}
	header := lines[:headerH]
	items := ViewportSlice(lines[headerH:], itemIdx, available-headerH)
	result := make([]string, 0, len(header)+len(items))
	result = append(result, header...)
	result = append(result, items...)
	return result
}

// viewportSlice returns a height-bounded window of lines keeping activeLine visible.
// The active line is kept roughly centered. Returns lines unchanged if height <= 0
// or len(lines) <= height.
func ViewportSlice(lines []string, activeLine, height int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}
	if activeLine < 0 {
		activeLine = 0
	}
	if activeLine >= len(lines) {
		activeLine = len(lines) - 1
	}
	half := height / 2
	start := activeLine - half
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
		start = end - height
		if start < 0 {
			start = 0
		}
	}
	return lines[start:end]
}

// stripANSI removes ANSI CSI escape sequences from s, returning plain text.
func StripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip CSI sequence: ESC [ ... <terminator>
			i += 2
			for i < len(s) && (s[i] < 0x40 || s[i] > 0x7e) {
				i++
			}
			if i < len(s) {
				i++ // consume terminator
			}
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// splitCSV splits a comma-separated string into trimmed, non-empty parts.
// Both "a, b" and "a,b" are handled identically.
func SplitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NoneToEmpty converts UI sentinel "None" values to empty strings so they are
// omitted from the manifest JSON (all manifest string fields use omitempty).
func NoneToEmpty(s string) string {
	switch s {
	case "None", "none", "(none)":
		return ""
	}
	return s
}

// fillTildes pads lines with vim-style tilde lines to height h.
func FillTildes(lines []string, h int) string {
	for len(lines) < h {
		lines = append(lines, StyleTilde.Render("·"))
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n") + "\n"
}
