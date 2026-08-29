package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dwight-D/anthill-cli/internal/escalation"
)

// escalationFixtureRoot builds a throwaway .anthill tree with two escalation
// records:
//
//	2026-08-29-open-q        (status open)
//	2026-08-29-answered-q    (status answered)
func escalationFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".anthill", "escalations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("LOG.md", "# Escalation log\n")
	write("2026-08-29-answered-q.md",
		"---\nto: user\nfrom: supervisor\nitem: some-item\nstatus: answered\nopened: 2026-08-29\n---\n"+
			"## Question\nWhich option?\n## Decision\nOption B.\n")
	write("2026-08-29-open-q.md",
		"---\nto: supervisor\nfrom: dispatcher\nstatus: open\nopened: 2026-08-29\n---\n"+
			"## Question\nShould we?\n")
	return root
}

func loadedEscalationsMode(t *testing.T, root string) *EscalationsMode {
	t.Helper()
	e := NewEscalationsMode(escalation.NewStore(root))
	if err := e.Reload(); err != nil {
		t.Fatal(err)
	}
	return e
}

func escIndexOf(t *testing.T, e *EscalationsMode, id string) int {
	t.Helper()
	for i := 0; i < e.Len(); i++ {
		if e.RowID(i) == id {
			return i
		}
	}
	t.Fatalf("no row %s", id)
	return -1
}

func escAction(t *testing.T, e *EscalationsMode, keyStr string) Action {
	t.Helper()
	for _, a := range e.Actions() {
		if a.Key == keyStr {
			return a
		}
	}
	t.Fatalf("no action with key %s", keyStr)
	return Action{}
}

func TestEscalationsModeList(t *testing.T) {
	e := loadedEscalationsMode(t, escalationFixtureRoot(t))
	// LoadAll sorts by id.
	want := []string{"2026-08-29-answered-q", "2026-08-29-open-q"}
	if e.Len() != len(want) {
		t.Fatalf("Len = %d, want %d", e.Len(), len(want))
	}
	for i, id := range want {
		if e.RowID(i) != id {
			t.Errorf("row %d = %s, want %s", i, e.RowID(i), id)
		}
	}
	// Row text carries id, addressee, and status.
	if r := e.Row(1, 60); !strings.Contains(r, "open-q") || !strings.Contains(r, "supervisor") || !strings.Contains(r, "open") {
		t.Errorf("row 1 = %q", r)
	}
}

func TestEscalationsModeDetail(t *testing.T) {
	e := loadedEscalationsMode(t, escalationFixtureRoot(t))
	d := e.Detail(0) // answered-q
	for _, want := range []string{
		"id:", "2026-08-29-answered-q", "to:", "user", "item:", "some-item",
		"status:", "answered", "## Question", "Which option?", "## Decision", "Option B.",
	} {
		if !strings.Contains(d, want) {
			t.Errorf("detail missing %q in:\n%s", want, d)
		}
	}
}

func TestEscalationsModeActionKeys(t *testing.T) {
	e := NewEscalationsMode(nil)
	acts := e.Actions()
	if len(acts) != 4 {
		t.Fatalf("actions = %d, want 4", len(acts))
	}
	for _, tc := range []struct {
		key, label string
		confirm    bool
		input      bool
	}{
		{"a", "answer", false, true},
		{"p", "apply", false, false},
		{"x", "close", true, false},
		{"X", "remove", true, false},
	} {
		a := escAction(t, e, tc.key)
		if a.Label != tc.label || a.Confirm != tc.confirm || a.Input != tc.input {
			t.Errorf("[%s] = %s confirm=%v input=%v, want %s confirm=%v input=%v",
				tc.key, a.Label, a.Confirm, a.Input, tc.label, tc.confirm, tc.input)
		}
	}
}

func TestEscalationsModeAnswer(t *testing.T) {
	root := escalationFixtureRoot(t)
	e := loadedEscalationsMode(t, root)
	answer := escAction(t, e, "a")

	// Answering an answered record is a precondition failure.
	if _, err := answer.DoInput(escIndexOf(t, e, "2026-08-29-answered-q"), "again"); err == nil {
		t.Error("answer on answered record: want error")
	}

	// Answering the open record lands the decision via the domain verb.
	msg, err := answer.DoInput(escIndexOf(t, e, "2026-08-29-open-q"), "Yes, proceed.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "answered 2026-08-29-open-q") {
		t.Errorf("status = %q", msg)
	}
	r, err := escalation.NewStore(root).Find("2026-08-29-open-q")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "answered" || r.Section("Decision") != "Yes, proceed." {
		t.Errorf("record after answer: status=%q decision=%q", r.Status, r.Section("Decision"))
	}
}

func TestEscalationsModeApply(t *testing.T) {
	root := escalationFixtureRoot(t)
	e := loadedEscalationsMode(t, root)

	// Applying an open record is a precondition failure.
	if _, err := escAction(t, e, "p").Do(escIndexOf(t, e, "2026-08-29-open-q")); err == nil {
		t.Error("apply on open record: want error")
	}
	if _, err := escAction(t, e, "p").Do(escIndexOf(t, e, "2026-08-29-answered-q")); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, ".anthill", "escalations")
	if _, err := os.Stat(filepath.Join(dir, "2026-08-29-answered-q.md")); !os.IsNotExist(err) {
		t.Error("applied record file still exists")
	}
	log, err := os.ReadFile(filepath.Join(dir, "LOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "2026-08-29-answered-q — user/applied") {
		t.Errorf("LOG.md missing applied line:\n%s", log)
	}
}

func TestEscalationsModeCloseAndRemove(t *testing.T) {
	root := escalationFixtureRoot(t)
	e := loadedEscalationsMode(t, root)

	// Close only fits an open record.
	if _, err := escAction(t, e, "x").Do(escIndexOf(t, e, "2026-08-29-answered-q")); err == nil {
		t.Error("close on answered record: want precondition error")
	}
	if _, err := escAction(t, e, "x").Do(escIndexOf(t, e, "2026-08-29-open-q")); err != nil {
		t.Fatal(err)
	}
	// Remove fits any status.
	if err := e.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, err := escAction(t, e, "X").Do(escIndexOf(t, e, "2026-08-29-answered-q")); err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile(filepath.Join(root, ".anthill", "escalations", "LOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2026-08-29-open-q — supervisor/closed", "2026-08-29-answered-q — user/removed"} {
		if !strings.Contains(string(log), want) {
			t.Errorf("LOG.md missing %q:\n%s", want, log)
		}
	}
}

// TestRootModelEscalationsMode wires the real escalations mode into the root
// model registry and drives mode switching plus one confirm-gated action.
func TestRootModelEscalationsMode(t *testing.T) {
	root := escalationFixtureRoot(t)
	// Give the tree a backlog dir too so mode 1 loads cleanly.
	if err := os.MkdirAll(filepath.Join(root, ".anthill", "backlog", "intake"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := New(root)
	next, _ := m.Update(reloadMsg{})
	m = next.(Model)

	// Switch to the Escalations mode with the number key.
	next, _ = m.Update(key("2"))
	m = next.(Model)
	if m.mode().Name() != "Escalations" {
		t.Fatalf("mode = %s, want Escalations", m.mode().Name())
	}
	if m.mode().Len() != 2 {
		t.Fatalf("loaded %d rows, want 2", m.mode().Len())
	}

	// Remove the selected (first) row: [X] arms a confirm, [y] runs it.
	next, _ = m.Update(key("X"))
	m = next.(Model)
	if m.confirm == nil {
		t.Fatal("no confirm prompt armed")
	}
	next, _ = m.Update(key("y"))
	m = next.(Model)
	if m.mode().Len() != 1 {
		t.Fatalf("after remove: %d rows, want 1", m.mode().Len())
	}
	if m.statusErr {
		t.Errorf("status is an error: %s", m.status)
	}
}
