package core

// DropdownState holds the open/cursor state for a dropdown overlay.
// It is embedded by name in every sub-editor struct so that all dropdown
// rendering and navigation code can share the same field paths.
type DropdownState struct {
	Open   bool
	OptIdx int
}

// SubView represents the list/form drill-down state used by editors that
// follow the list+form pattern (contracts, frontend, etc.).
type SubView int

const (
	ViewList    SubView = iota
	ViewForm            // top-level form
	ViewSubList         // sub-list (e.g., DTO fields, endpoint error responses)
	ViewSubForm         // sub-item form
)
