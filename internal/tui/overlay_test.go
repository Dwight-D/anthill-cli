package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func ctrlS() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyCtrlS} }
func escKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEsc} }

// inputStub returns a stub mode with one free-text action on [i], recording
// calls into got.
func inputStub(got *[]string) *stubMode {
	mode := &stubMode{name: "A", rows: []string{"r1", "r2"}}
	mode.acts = []Action{
		{Key: "i", Label: "note", Input: true, DoInput: func(i int, text string) (string, error) {
			*got = append(*got, mode.rows[i]+":"+text)
			return "noted", nil
		}},
	}
	return mode
}

func TestOverlayOpenTypeSubmit(t *testing.T) {
	var got []string
	mode := inputStub(&got)
	m := NewWithModes([]Mode{mode})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)

	// [i] opens the overlay; the hint line swaps to the overlay's keys.
	next, _ = m.Update(key("i"))
	m = next.(Model)
	if m.overlay == nil {
		t.Fatal("overlay did not open")
	}
	if h := m.renderHeader(); !strings.Contains(h, "ctrl+s") || !strings.Contains(h, "note r1") {
		t.Errorf("header hints while open: %q", h)
	}

	// Typed keys land in the textarea, not the global keymap: 'q' must not
	// quit and 'i' must not re-open.
	for _, s := range []string{"q", "i", "hello"} {
		next, _ = m.Update(key(s))
		m = next.(Model)
	}
	if m.overlay == nil {
		t.Fatal("overlay closed by typed text")
	}

	// ctrl+s submits: DoInput gets the trimmed text, the overlay closes, the
	// mode reloads.
	next, _ = m.Update(ctrlS())
	m = next.(Model)
	if m.overlay != nil {
		t.Fatal("overlay still open after submit")
	}
	if len(got) != 1 || got[0] != "r1:qihello" {
		t.Fatalf("DoInput calls = %v", got)
	}
	if mode.reloads != 1 {
		t.Errorf("reloads = %d, want 1", mode.reloads)
	}
	if m.status != "noted" || m.statusErr {
		t.Errorf("status = %q (err=%v)", m.status, m.statusErr)
	}
}

func TestOverlayEscCancels(t *testing.T) {
	var got []string
	m := NewWithModes([]Mode{inputStub(&got)})
	next, _ := m.Update(key("i"))
	m = next.(Model)
	next, _ = m.Update(key("draft text"))
	m = next.(Model)
	next, _ = m.Update(escKey())
	m = next.(Model)
	if m.overlay != nil {
		t.Fatal("overlay still open after esc")
	}
	if len(got) != 0 {
		t.Fatalf("cancelled overlay ran the action: %v", got)
	}
	if m.status != "cancelled" {
		t.Errorf("status = %q, want cancelled", m.status)
	}
}

func TestOverlayRejectsEmptySubmit(t *testing.T) {
	var got []string
	m := NewWithModes([]Mode{inputStub(&got)})
	next, _ := m.Update(key("i"))
	m = next.(Model)

	// Whitespace-only text is not a submission.
	next, _ = m.Update(key("   "))
	m = next.(Model)
	next, cmd := m.Update(ctrlS())
	m = next.(Model)
	if m.overlay == nil {
		t.Fatal("overlay closed on empty submit")
	}
	if len(got) != 0 {
		t.Fatalf("empty submit ran the action: %v", got)
	}
	// The rejection travels as a statusMsg command into the status line.
	if cmd == nil {
		t.Fatal("no status command returned")
	}
	next, _ = m.Update(cmd())
	m = next.(Model)
	if !m.statusErr || !strings.Contains(m.status, "required") {
		t.Errorf("status = %q (err=%v), want required-error", m.status, m.statusErr)
	}
}

// TestAnswerOverlayEndToEnd drives the root model over a real escalation
// fixture: open the overlay on the open record, type a decision, submit, and
// see the answered state land on disk and in the detail pane.
func TestAnswerOverlayEndToEnd(t *testing.T) {
	root := escalationFixtureRoot(t)
	if err := os.MkdirAll(filepath.Join(root, ".anthill", "backlog", "intake"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := New(root)
	next, _ := m.Update(reloadMsg{})
	m = next.(Model)
	next, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = next.(Model)

	// Escalations mode, selection on the open record (row 1 after sort).
	next, _ = m.Update(key("2"))
	m = next.(Model)
	next, _ = m.Update(key("j"))
	m = next.(Model)
	if id := m.mode().RowID(m.selected); id != "2026-08-29-open-q" {
		t.Fatalf("selected %s, want the open record", id)
	}

	next, _ = m.Update(key("a"))
	m = next.(Model)
	if m.overlay == nil {
		t.Fatal("[a] did not open the answer overlay")
	}
	next, _ = m.Update(key("Go with option A."))
	m = next.(Model)
	next, _ = m.Update(ctrlS())
	m = next.(Model)
	if m.overlay != nil {
		t.Fatal("overlay still open after submit")
	}
	if m.statusErr || !strings.Contains(m.status, "answered 2026-08-29-open-q") {
		t.Fatalf("status = %q (err=%v)", m.status, m.statusErr)
	}

	// The reloaded detail shows the answered state and the decision text.
	if id := m.mode().RowID(m.selected); id != "2026-08-29-open-q" {
		t.Fatalf("selection moved to %s", id)
	}
	d := m.mode().Detail(m.selected)
	for _, want := range []string{"status:  answered", "## Decision", "Go with option A."} {
		if !strings.Contains(d, want) {
			t.Errorf("detail missing %q:\n%s", want, d)
		}
	}

	// And on disk, via the store the CLI verb reads.
	data, err := os.ReadFile(filepath.Join(root, ".anthill", "escalations", "2026-08-29-open-q.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "status: answered") || !strings.Contains(string(data), "Go with option A.") {
		t.Errorf("record file after answer:\n%s", data)
	}
}
