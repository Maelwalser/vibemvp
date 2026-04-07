package core

import "testing"

// ── nextEditableIdx / prevEditableIdx ─────────────────────────────────────────

func TestNextEditableIdx_EmptyFields(t *testing.T) {
	if got := NextEditableIdx(nil, 0); got != -1 {
		t.Errorf("expected -1 for nil fields, got %d", got)
	}
	if got := NextEditableIdx([]Field{}, 0); got != -1 {
		t.Errorf("expected -1 for empty fields, got %d", got)
	}
}

func TestNextEditableIdx_NoEditableFields(t *testing.T) {
	fields := []Field{
		{Kind: KindSelect},
		{Kind: KindSelect},
	}
	if got := NextEditableIdx(fields, 0); got != -1 {
		t.Errorf("expected -1 when no editable fields, got %d", got)
	}
}

func TestNextEditableIdx_SingleEditable(t *testing.T) {
	fields := []Field{
		{Kind: KindSelect},
		{Kind: KindText},
		{Kind: KindSelect},
	}
	// From any starting position the only editable field is index 1
	for start := 0; start < len(fields); start++ {
		got := NextEditableIdx(fields, start)
		if start == 1 {
			// From index 1 itself, the next (wrapping) editable is still 1
			if got != 1 {
				t.Errorf("from %d: expected 1, got %d", start, got)
			}
		} else if got != 1 {
			t.Errorf("from %d: expected 1, got %d", start, got)
		}
	}
}

func TestNextEditableIdx_WrapsAroundEnd(t *testing.T) {
	fields := []Field{
		{Kind: KindText},   // 0 — editable
		{Kind: KindSelect}, // 1
		{Kind: KindSelect}, // 2
	}
	// Starting at the last index, should wrap around and find index 0
	if got := NextEditableIdx(fields, 2); got != 0 {
		t.Errorf("expected wrap to index 0, got %d", got)
	}
}

func TestNextEditableIdx_TextAreaIsEditable(t *testing.T) {
	fields := []Field{
		{Kind: KindSelect},
		{Kind: KindTextArea},
	}
	if got := NextEditableIdx(fields, 0); got != 1 {
		t.Errorf("KindTextArea should be editable, got %d", got)
	}
}

func TestPrevEditableIdx_EmptyFields(t *testing.T) {
	if got := PrevEditableIdx(nil, 0); got != -1 {
		t.Errorf("expected -1 for nil fields, got %d", got)
	}
}

func TestPrevEditableIdx_NoEditableFields(t *testing.T) {
	fields := []Field{{Kind: KindSelect}, {Kind: KindSelect}}
	if got := PrevEditableIdx(fields, 1); got != -1 {
		t.Errorf("expected -1 when no editable fields, got %d", got)
	}
}

func TestPrevEditableIdx_WrapsAroundStart(t *testing.T) {
	fields := []Field{
		{Kind: KindSelect}, // 0
		{Kind: KindSelect}, // 1
		{Kind: KindText},   // 2 — editable
	}
	// Starting at index 0, prevEditableIdx should wrap to index 2
	if got := PrevEditableIdx(fields, 0); got != 2 {
		t.Errorf("expected wrap to index 2, got %d", got)
	}
}

func TestPrevEditableIdx_FindsPreviousEditable(t *testing.T) {
	fields := []Field{
		{Kind: KindText},   // 0 — editable
		{Kind: KindSelect}, // 1
		{Kind: KindText},   // 2 — editable
		{Kind: KindSelect}, // 3
	}
	// From index 3, the previous editable is index 2
	if got := PrevEditableIdx(fields, 3); got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
	// From index 2, the previous editable is index 0
	if got := PrevEditableIdx(fields, 2); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

// ── NavigateDropdown ──────────────────────────────────────────────────────────

func TestNavigateDropdown(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		idx     int
		n       int
		wantIdx int
	}{
		{"j moves down", "j", 0, 5, 1},
		{"down moves down", "down", 2, 5, 3},
		{"j at bottom clamps", "j", 4, 5, 4},
		{"k moves up", "k", 3, 5, 2},
		{"up moves up", "up", 1, 5, 0},
		{"k at top clamps", "k", 0, 5, 0},
		{"g goes to first", "g", 3, 5, 0},
		{"G goes to last", "G", 1, 5, 4},
		{"G with n=0 is safe", "G", 0, 0, 0},
		{"unknown key returns unchanged", "x", 2, 5, 2},
		{"enter returns unchanged", "enter", 2, 5, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NavigateDropdown(tc.key, tc.idx, tc.n)
			if got != tc.wantIdx {
				t.Errorf("NavigateDropdown(%q, %d, %d) = %d, want %d",
					tc.key, tc.idx, tc.n, got, tc.wantIdx)
			}
		})
	}
}

// ── NavigateTab ───────────────────────────────────────────────────────────────

func TestNavigateTab(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		active  int
		maxTabs int
		wantIdx int
	}{
		{"h moves left", "h", 2, 5, 1},
		{"left moves left", "left", 3, 5, 2},
		{"h at 0 clamps", "h", 0, 5, 0},
		{"l moves right", "l", 2, 5, 3},
		{"right moves right", "right", 1, 5, 2},
		{"l at last clamps", "l", 4, 5, 4},
		{"unknown key unchanged", "j", 2, 5, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NavigateTab(tc.key, tc.active, tc.maxTabs)
			if got != tc.wantIdx {
				t.Errorf("NavigateTab(%q, %d, %d) = %d, want %d",
					tc.key, tc.active, tc.maxTabs, got, tc.wantIdx)
			}
		})
	}
}

// ── VimNav ────────────────────────────────────────────────────────────────────

func TestVimNav_SingleMoveDown(t *testing.T) {
	var v VimNav
	idx, consumed := v.Handle("j", 2, 10)
	if !consumed {
		t.Error("j should be consumed")
	}
	if idx != 3 {
		t.Errorf("expected idx=3 after j from 2, got %d", idx)
	}
}

func TestVimNav_SingleMoveUp(t *testing.T) {
	var v VimNav
	idx, consumed := v.Handle("k", 5, 10)
	if !consumed {
		t.Error("k should be consumed")
	}
	if idx != 4 {
		t.Errorf("expected idx=4 after k from 5, got %d", idx)
	}
}

func TestVimNav_MoveDownClampsAtBottom(t *testing.T) {
	var v VimNav
	idx, _ := v.Handle("j", 9, 10)
	if idx != 9 {
		t.Errorf("expected idx clamped at 9, got %d", idx)
	}
}

func TestVimNav_MoveUpClampsAtTop(t *testing.T) {
	var v VimNav
	idx, _ := v.Handle("k", 0, 10)
	if idx != 0 {
		t.Errorf("expected idx clamped at 0, got %d", idx)
	}
}

func TestVimNav_CountPrefix_ThreeDown(t *testing.T) {
	var v VimNav
	_, consumed := v.Handle("3", 0, 10)
	if !consumed {
		t.Error("digit prefix should be consumed")
	}
	idx, consumed := v.Handle("j", 0, 10)
	if !consumed {
		t.Error("j after count should be consumed")
	}
	if idx != 3 {
		t.Errorf("expected idx=3 after 3j, got %d", idx)
	}
}

func TestVimNav_CountPrefix_MultiDigit(t *testing.T) {
	var v VimNav
	v.Handle("1", 0, 20)
	v.Handle("0", 0, 20)
	idx, _ := v.Handle("j", 0, 20)
	if idx != 10 {
		t.Errorf("expected idx=10 after 10j, got %d", idx)
	}
}

func TestVimNav_CountPrefix_ClampsAtBottom(t *testing.T) {
	var v VimNav
	v.Handle("1", 0, 5)
	v.Handle("0", 0, 5)
	v.Handle("0", 0, 5)
	idx, _ := v.Handle("j", 0, 5)
	if idx != 4 {
		t.Errorf("expected idx clamped at 4 (n-1), got %d", idx)
	}
}

func TestVimNav_GG_GoesToTop(t *testing.T) {
	var v VimNav
	_, consumed := v.Handle("g", 5, 10)
	if !consumed {
		t.Error("first g should be consumed")
	}
	idx, consumed := v.Handle("g", 5, 10)
	if !consumed {
		t.Error("second g (gg) should be consumed")
	}
	if idx != 0 {
		t.Errorf("expected idx=0 after gg, got %d", idx)
	}
}

func TestVimNav_G_GoesToBottom(t *testing.T) {
	var v VimNav
	idx, consumed := v.Handle("G", 2, 10)
	if !consumed {
		t.Error("G should be consumed")
	}
	if idx != 9 {
		t.Errorf("expected idx=9 after G with n=10, got %d", idx)
	}
}

func TestVimNav_UnknownKey_NotConsumed(t *testing.T) {
	var v VimNav
	idx, consumed := v.Handle("enter", 3, 10)
	if consumed {
		t.Error("enter should not be consumed by VimNav")
	}
	if idx != 3 {
		t.Errorf("idx should be unchanged for unknown key, got %d", idx)
	}
}

func TestVimNav_Reset_ClearsState(t *testing.T) {
	v := VimNav{CountBuf: "5", GBuf: true}
	v.Reset()
	if v.CountBuf != "" {
		t.Errorf("Reset should clear CountBuf, got %q", v.CountBuf)
	}
	if v.GBuf {
		t.Error("Reset should clear GBuf")
	}
}

func TestVimNav_ZeroOnlyValidIfCountBufNonEmpty(t *testing.T) {
	// "0" alone (CountBuf empty) should not be consumed as a digit
	var v VimNav
	_, consumed := v.Handle("0", 5, 10)
	if consumed {
		t.Error("standalone 0 with empty CountBuf should not be consumed as digit")
	}
}

func TestVimNav_ZeroAppendedToExistingCount(t *testing.T) {
	var v VimNav
	v.Handle("1", 0, 20) // CountBuf = "1"
	_, consumed := v.Handle("0", 0, 20)
	if !consumed {
		t.Error("0 after non-empty CountBuf should be consumed")
	}
	if v.CountBuf != "10" {
		t.Errorf("CountBuf should be '10', got %q", v.CountBuf)
	}
}
