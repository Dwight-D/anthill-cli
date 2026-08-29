package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// stubMode is a minimal in-memory Mode for exercising the root model.
type stubMode struct {
	name     string
	rows     []string
	nextRows []string // if non-nil, Reload swaps these in (simulating disk change)
	reloads  int
	acts     []Action
	detail   string // if non-empty, Detail returns this for every row
}

func (s *stubMode) Name() string { return s.name }
func (s *stubMode) Reload() error {
	s.reloads++
	if s.nextRows != nil {
		s.rows, s.nextRows = s.nextRows, nil
	}
	return nil
}
func (s *stubMode) Len() int            { return len(s.rows) }
func (s *stubMode) RowID(i int) string  { return s.rows[i] }
func (s *stubMode) Row(i, w int) string { return s.rows[i] }
func (s *stubMode) Detail(i int) string {
	if s.detail != "" {
		return s.detail
	}
	return "detail:" + s.rows[i]
}
func (s *stubMode) Actions() []Action { return s.acts }

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestLayoutFloors(t *testing.T) {
	m := NewWithModes([]Mode{&stubMode{name: "A"}})
	m.width, m.height = 5, 3 // absurdly small terminal
	for name, got := range map[string]int{
		"leftInnerW":  m.leftInnerW(),
		"rightInnerW": m.rightInnerW(),
		"innerH":      m.innerH(),
	} {
		if got < 4 {
			t.Errorf("%s = %d, want >= 4", name, got)
		}
	}
}

func TestLayoutSplit(t *testing.T) {
	m := NewWithModes([]Mode{&stubMode{name: "A"}})
	m.width, m.height = 100, 30
	if got := m.leftInnerW(); got != 100*3/5-2 {
		t.Errorf("leftInnerW = %d, want %d", got, 100*3/5-2)
	}
	// left + right inner widths + 4 border cells fill the terminal exactly.
	if total := m.leftInnerW() + m.rightInnerW() + 4; total != 100 {
		t.Errorf("pane widths sum to %d, want 100", total)
	}
	if got := m.innerH(); got != 30-headerHeight-statusHeight-2 {
		t.Errorf("innerH = %d, want %d", got, 30-headerHeight-statusHeight-2)
	}
}

func TestModeRegistrySwitching(t *testing.T) {
	a := &stubMode{name: "Alpha", rows: []string{"a1"}}
	b := &stubMode{name: "Beta", rows: []string{"b1"}}
	m := NewWithModes([]Mode{a, b})

	// [m] cycles to the next mode and reloads it.
	next, _ := m.Update(key("m"))
	m = next.(Model)
	if m.mode().Name() != "Beta" {
		t.Fatalf("after [m], mode = %s, want Beta", m.mode().Name())
	}
	if b.reloads != 1 {
		t.Errorf("Beta reloads = %d, want 1", b.reloads)
	}

	// Number keys select a mode directly.
	next, _ = m.Update(key("1"))
	m = next.(Model)
	if m.mode().Name() != "Alpha" {
		t.Fatalf("after [1], mode = %s, want Alpha", m.mode().Name())
	}

	// Header shows the active mode.
	m.width, m.height, m.sizeReady = 80, 24, true
	if !strings.Contains(m.renderHeader(), "1:Alpha") || !strings.Contains(m.renderHeader(), "2:Beta") {
		t.Errorf("header missing mode tabs: %q", m.renderHeader())
	}
}

func TestActionDispatchAndConfirm(t *testing.T) {
	var did []string
	mode := &stubMode{name: "A", rows: []string{"r1", "r2"}}
	mode.acts = []Action{
		{Key: "a", Label: "plain", Do: func(i int) (string, error) {
			did = append(did, "plain:"+mode.rows[i])
			return "ok", nil
		}},
		{Key: "X", Label: "danger", Confirm: true, Do: func(i int) (string, error) {
			did = append(did, "danger:"+mode.rows[i])
			return "ok", nil
		}},
	}
	m := NewWithModes([]Mode{mode})

	// Plain action fires immediately and triggers a reload.
	next, _ := m.Update(key("a"))
	m = next.(Model)
	if len(did) != 1 || did[0] != "plain:r1" {
		t.Fatalf("plain action: did = %v", did)
	}
	if mode.reloads != 1 {
		t.Errorf("reloads after action = %d, want 1", mode.reloads)
	}

	// Confirm-gated action arms a prompt; a non-y key cancels.
	next, _ = m.Update(key("X"))
	m = next.(Model)
	if m.confirm == nil {
		t.Fatal("confirm action did not arm a prompt")
	}
	next, _ = m.Update(key("n"))
	m = next.(Model)
	if m.confirm != nil || len(did) != 1 {
		t.Fatalf("cancel failed: confirm=%v did=%v", m.confirm, did)
	}

	// Armed again, y runs it.
	next, _ = m.Update(key("X"))
	m = next.(Model)
	next, _ = m.Update(key("y"))
	m = next.(Model)
	if len(did) != 2 || did[1] != "danger:r1" {
		t.Fatalf("confirm run: did = %v", did)
	}
}

func TestSelectionPreservedAcrossReload(t *testing.T) {
	mode := &stubMode{name: "A", rows: []string{"r1", "r2", "r3"}}
	m := NewWithModes([]Mode{mode})
	m.selected = 1

	// Row survives: selection follows its id even if rows shift.
	mode.nextRows = []string{"r0", "r1", "r2", "r3"}
	(&m).reload()
	if m.selected != 2 || m.mode().RowID(m.selected) != "r2" {
		t.Errorf("selection = %d (%s), want 2 (r2)", m.selected, m.mode().RowID(m.selected))
	}

	// Row gone at the tail: index clamps.
	m.selected = 3
	mode.nextRows = []string{"r0", "r1"}
	(&m).reload()
	if m.selected != 1 {
		t.Errorf("clamped selection = %d, want 1", m.selected)
	}
}

func TestTextHelpers(t *testing.T) {
	if got := truncateCells("hello", 10); got != "hello" {
		t.Errorf("truncateCells short = %q", got)
	}
	if got := truncateCells("hello world", 5); got != "hell…" {
		t.Errorf("truncateCells cut = %q", got)
	}
	if got := truncateCells("x", 0); got != "" {
		t.Errorf("truncateCells zero = %q", got)
	}
	if got := padCells("ab", 4); got != "ab  " {
		t.Errorf("padCells = %q", got)
	}
	if got := padCells("abcdef", 4); got != "abc…" {
		t.Errorf("padCells over = %q", got)
	}
}
