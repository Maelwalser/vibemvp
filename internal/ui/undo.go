package ui

import "github.com/vibe-menu/internal/manifest"

const undoMaxDepth = 50

// UndoStack is a bounded last-in-first-out stack of state snapshots.
type UndoStack[T any] struct {
	items []T
}

// Push adds state to the stack. Entries older than undoMaxDepth are evicted.
func (u *UndoStack[T]) Push(state T) {
	u.items = append(u.items, state)
	if len(u.items) > undoMaxDepth {
		u.items = u.items[1:]
	}
}

// Pop removes and returns the most recent snapshot. Returns false when empty.
func (u *UndoStack[T]) Pop() (T, bool) {
	if len(u.items) == 0 {
		var zero T
		return zero, false
	}
	n := len(u.items) - 1
	state := u.items[n]
	u.items = u.items[:n]
	return state, true
}

// Len returns the number of entries in the stack.
func (u *UndoStack[T]) Len() int { return len(u.items) }

// copySlice returns a shallow copy of s, safe to use as an undo snapshot for
// slices of value types (structs without pointer fields).
func copySlice[T any](s []T) []T {
	if s == nil {
		return nil
	}
	out := make([]T, len(s))
	copy(out, s)
	return out
}

// copyFieldItems returns a deep copy of a [][]Field slice — both outer and
// inner slices are freshly allocated so mutations to either do not bleed into
// the snapshot.
func copyFieldItems(items [][]Field) [][]Field {
	out := make([][]Field, len(items))
	for i, row := range items {
		out[i] = copySlice(row)
	}
	return out
}

// ── Snapshot types for editors that keep parallel Field-items + manifest slices

// svcSnapshot captures the service list editor state and the corresponding
// exported manifest slice so both can be restored atomically.
type svcSnapshot struct {
	items    [][]Field
	services []manifest.ServiceDef
}

// commSnapshot captures the comm-link list editor state.
type commSnapshot struct {
	items [][]Field
	comms []manifest.CommLink
}

// eventSnapshot captures the event catalog list editor state.
type eventSnapshot struct {
	items  [][]Field
	events []manifest.EventDef
}
