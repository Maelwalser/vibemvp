package dag

import (
	"sort"
	"testing"
)

// newTestDAG builds a DAG from a slice of tasks, calling build() internally.
// The test is failed immediately if build() returns an error.
func newTestDAG(t *testing.T, tasks []*Task) *DAG {
	t.Helper()
	d := &DAG{Tasks: make(map[string]*Task)}
	for _, task := range tasks {
		d.Tasks[task.ID] = task
	}
	if err := d.build(); err != nil {
		t.Fatalf("build() unexpected error: %v", err)
	}
	return d
}

func TestLevels_EmptyDAG(t *testing.T) {
	d := &DAG{Tasks: make(map[string]*Task)}
	if err := d.build(); err != nil {
		t.Fatalf("build() unexpected error: %v", err)
	}
	if got := d.Levels(); len(got) != 0 {
		t.Errorf("expected 0 levels for empty DAG, got %d: %v", len(got), got)
	}
}

func TestLevels_SingleTask(t *testing.T) {
	d := newTestDAG(t, []*Task{
		{ID: "a", Kind: TaskKindContracts, Dependencies: nil},
	})
	levels := d.Levels()
	if len(levels) != 1 {
		t.Fatalf("expected 1 level, got %d: %v", len(levels), levels)
	}
	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Errorf("wave 0 should be [a], got %v", levels[0])
	}
}

func TestLevels_LinearChain(t *testing.T) {
	// A → B → C: each must occupy its own wave
	d := newTestDAG(t, []*Task{
		{ID: "a", Kind: TaskKindContracts, Dependencies: nil},
		{ID: "b", Kind: TaskKindContracts, Dependencies: []string{"a"}},
		{ID: "c", Kind: TaskKindContracts, Dependencies: []string{"b"}},
	})
	levels := d.Levels()
	if len(levels) != 3 {
		t.Fatalf("expected 3 levels for A→B→C, got %d: %v", len(levels), levels)
	}
	for i, want := range []string{"a", "b", "c"} {
		if len(levels[i]) != 1 || levels[i][0] != want {
			t.Errorf("wave %d should be [%s], got %v", i, want, levels[i])
		}
	}
}

func TestLevels_DiamondDependency(t *testing.T) {
	// A→B, A→C, B→D, C→D: D must be at level 2 (not 1)
	d := newTestDAG(t, []*Task{
		{ID: "a", Kind: TaskKindContracts, Dependencies: nil},
		{ID: "b", Kind: TaskKindContracts, Dependencies: []string{"a"}},
		{ID: "c", Kind: TaskKindContracts, Dependencies: []string{"a"}},
		{ID: "d", Kind: TaskKindContracts, Dependencies: []string{"b", "c"}},
	})
	levels := d.Levels()
	if len(levels) != 3 {
		t.Fatalf("expected 3 waves for diamond, got %d: %v", len(levels), levels)
	}
	if len(levels[0]) != 1 || levels[0][0] != "a" {
		t.Errorf("wave 0 should be [a], got %v", levels[0])
	}
	wave1 := append([]string{}, levels[1]...)
	sort.Strings(wave1)
	if len(wave1) != 2 || wave1[0] != "b" || wave1[1] != "c" {
		t.Errorf("wave 1 should be [b c] (sorted), got %v", wave1)
	}
	if len(levels[2]) != 1 || levels[2][0] != "d" {
		t.Errorf("wave 2 should be [d], got %v", levels[2])
	}
}

func TestLevels_IndependentTasks(t *testing.T) {
	// Three tasks with no dependencies should all land in wave 0
	d := newTestDAG(t, []*Task{
		{ID: "a", Kind: TaskKindContracts, Dependencies: nil},
		{ID: "b", Kind: TaskKindContracts, Dependencies: nil},
		{ID: "c", Kind: TaskKindContracts, Dependencies: nil},
	})
	levels := d.Levels()
	if len(levels) != 1 {
		t.Fatalf("expected 1 level for independent tasks, got %d", len(levels))
	}
	if len(levels[0]) != 3 {
		t.Errorf("wave 0 should contain 3 tasks, got %v", levels[0])
	}
}

func TestOrder_DependenciesPrecedeDependents(t *testing.T) {
	d := newTestDAG(t, []*Task{
		{ID: "a", Kind: TaskKindContracts, Dependencies: nil},
		{ID: "b", Kind: TaskKindContracts, Dependencies: []string{"a"}},
		{ID: "c", Kind: TaskKindContracts, Dependencies: []string{"b"}},
	})
	order := d.Order()
	if len(order) != 3 {
		t.Fatalf("expected 3 tasks in order, got %d: %v", len(order), order)
	}
	pos := make(map[string]int, len(order))
	for i, id := range order {
		pos[id] = i
	}
	if pos["a"] >= pos["b"] {
		t.Errorf("a must precede b in order: %v", order)
	}
	if pos["b"] >= pos["c"] {
		t.Errorf("b must precede c in order: %v", order)
	}
}

func TestOrder_DiamondRespectsDependencies(t *testing.T) {
	d := newTestDAG(t, []*Task{
		{ID: "a", Kind: TaskKindContracts, Dependencies: nil},
		{ID: "b", Kind: TaskKindContracts, Dependencies: []string{"a"}},
		{ID: "c", Kind: TaskKindContracts, Dependencies: []string{"a"}},
		{ID: "d", Kind: TaskKindContracts, Dependencies: []string{"b", "c"}},
	})
	pos := make(map[string]int)
	for i, id := range d.Order() {
		pos[id] = i
	}
	if pos["a"] >= pos["b"] || pos["a"] >= pos["c"] {
		t.Errorf("a must precede b and c: positions=%v", pos)
	}
	if pos["b"] >= pos["d"] || pos["c"] >= pos["d"] {
		t.Errorf("b and c must precede d: positions=%v", pos)
	}
}

func TestBuild_UnknownDependency(t *testing.T) {
	d := &DAG{Tasks: make(map[string]*Task)}
	d.Tasks["a"] = &Task{ID: "a", Kind: TaskKindContracts, Dependencies: []string{"nonexistent"}}
	if err := d.build(); err == nil {
		t.Error("expected error for unknown dependency, got nil")
	}
}

func TestBuild_Cycle_TwoNodes(t *testing.T) {
	d := &DAG{Tasks: make(map[string]*Task)}
	d.Tasks["a"] = &Task{ID: "a", Kind: TaskKindContracts, Dependencies: []string{"b"}}
	d.Tasks["b"] = &Task{ID: "b", Kind: TaskKindContracts, Dependencies: []string{"a"}}
	if err := d.build(); err == nil {
		t.Error("expected cycle error for A↔B, got nil")
	}
}

func TestBuild_Cycle_ThreeNodes(t *testing.T) {
	d := &DAG{Tasks: make(map[string]*Task)}
	d.Tasks["a"] = &Task{ID: "a", Kind: TaskKindContracts, Dependencies: []string{"c"}}
	d.Tasks["b"] = &Task{ID: "b", Kind: TaskKindContracts, Dependencies: []string{"a"}}
	d.Tasks["c"] = &Task{ID: "c", Kind: TaskKindContracts, Dependencies: []string{"b"}}
	if err := d.build(); err == nil {
		t.Error("expected cycle error for A→C→B→A, got nil")
	}
}

func TestBuild_SelfLoop(t *testing.T) {
	d := &DAG{Tasks: make(map[string]*Task)}
	d.Tasks["a"] = &Task{ID: "a", Kind: TaskKindContracts, Dependencies: []string{"a"}}
	if err := d.build(); err == nil {
		t.Error("expected error for self-referencing task, got nil")
	}
}
