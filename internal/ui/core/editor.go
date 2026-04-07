package core

// Mode represents the vim editing mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeCommand
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeCommand:
		return "COMMAND"
	}
	return ""
}

// Editor is the common read interface satisfied by all section sub-editors.
// Each of the 7 editors implements these three methods, enabling model.go to
// dispatch Mode, View, and HintLine polymorphically via activeEditor() without
// a separate 7-way switch statement per operation.
//
// Update is intentionally excluded: each editor's Update returns its own
// concrete type (value-receiver pattern required by bubbletea), so delegateUpdate
// continues to handle it with a single typed dispatch.
type Editor interface {
	// Mode returns the editor's current vim editing mode.
	Mode() Mode

	// HintLine returns the context-sensitive key-binding hint shown at the
	// bottom of the terminal.
	HintLine() string

	// View renders the editor content into a string of exactly w columns.
	// h is the available content height in lines.
	View(w, h int) string
}
