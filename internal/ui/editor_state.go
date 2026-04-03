package ui

// DropdownState holds the open/cursor state for a dropdown overlay.
// It is embedded by name in every sub-editor struct so that all dropdown
// rendering and navigation code can share the same field paths.
type DropdownState struct {
	Open   bool
	OptIdx int
}
