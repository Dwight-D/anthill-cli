package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
)

// fixtureRoot builds a throwaway .anthill tree:
//
//	intake/idea-one          (status new)
//	cli/claimed-item         (validation, delivered)
//	cli/ready-item           (triaged, approved, auto + verify)
//	bugs/bug-item            (triaged, proposed)
//
// with sweep-order "cli, bugs".
func fixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".anthill", "backlog")
	for _, d := range []string{"intake", "cli", "bugs"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(rel, content string) {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("workstreams.md", "---\nschema: 2\nsweep-order: cli, bugs\n---\n")
	write("intake/idea-one.md", "---\ntitle: Idea one\nvalue: v\nstatus: new\ndispatch: proposed\ncreated-at: \"2026-11-01T00:00:00Z\"\n---\nIdea body.\n")
	write("cli/claimed-item.md", "---\nworkstream: cli\ntitle: Claimed item\nvalue: v\nstatus: validation\ndispatch: approved\nverify: go test\ncreated-at: \"2026-11-01T00:00:00Z\"\n---\n")
	write("cli/ready-item.md", "---\nworkstream: cli\ntitle: Ready item\nvalue: v\nstatus: triaged\ndispatch: approved\ndisposition: auto\nverify: go test\ncreated-at: \"2026-11-02T00:00:00Z\"\n---\n")
	write("bugs/bug-item.md", "---\nworkstream: bugs\ntitle: Bug item\nvalue: v\nstatus: triaged\ndispatch: proposed\ncreated-at: \"2026-11-01T00:00:00Z\"\n---\n")
	return root
}

func loadedMode(t *testing.T, root string) *BacklogMode {
	t.Helper()
	b := NewBacklogMode(backlog.NewStore(root))
	if err := b.Reload(); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBacklogModeListRankOrder(t *testing.T) {
	b := loadedMode(t, fixtureRoot(t))
	want := []string{"claimed-item", "ready-item", "bug-item", "idea-one"}
	if b.Len() != len(want) {
		t.Fatalf("Len = %d, want %d", b.Len(), len(want))
	}
	for i, id := range want {
		if b.RowID(i) != id {
			t.Errorf("row %d = %s, want %s", i, b.RowID(i), id)
		}
	}
	// Row text carries id, workstream, status; intake items show "intake".
	if r := b.Row(0, 60); !strings.Contains(r, "claimed-item") || !strings.Contains(r, "cli") || !strings.Contains(r, "validation") {
		t.Errorf("row 0 = %q", r)
	}
	if r := b.Row(3, 60); !strings.Contains(r, "intake") {
		t.Errorf("intake row = %q", r)
	}
}

func TestBacklogModeDetail(t *testing.T) {
	b := loadedMode(t, fixtureRoot(t))
	d := b.Detail(3) // idea-one
	for _, want := range []string{"id:", "idea-one", "title:", "Idea one", "status:", "new", "dispatch:", "proposed", "Idea body."} {
		if !strings.Contains(d, want) {
			t.Errorf("detail missing %q in:\n%s", want, d)
		}
	}
}

func indexOf(t *testing.T, b *BacklogMode, id string) int {
	t.Helper()
	for i := 0; i < b.Len(); i++ {
		if b.RowID(i) == id {
			return i
		}
	}
	t.Fatalf("no row %s", id)
	return -1
}

func action(t *testing.T, b *BacklogMode, keyStr string) Action {
	t.Helper()
	for _, a := range b.Actions() {
		if a.Key == keyStr {
			return a
		}
	}
	t.Fatalf("no action with key %s", keyStr)
	return Action{}
}

func TestBacklogModeActionKeys(t *testing.T) {
	b := NewBacklogMode(nil)
	acts := b.Actions()
	if len(acts) != 4 {
		t.Fatalf("actions = %d, want 4", len(acts))
	}
	for _, tc := range []struct {
		key, label string
		confirm    bool
	}{
		{"a", "approve", false},
		{"d", "done", false},
		{"x", "discard", true},
		{"X", "remove", true},
	} {
		a := action(t, b, tc.key)
		if a.Label != tc.label || a.Confirm != tc.confirm {
			t.Errorf("[%s] = %s confirm=%v, want %s confirm=%v", tc.key, a.Label, a.Confirm, tc.label, tc.confirm)
		}
	}
}

func TestBacklogModeApprove(t *testing.T) {
	root := fixtureRoot(t)
	b := loadedMode(t, root)
	if _, err := action(t, b, "a").Do(indexOf(t, b, "idea-one")); err != nil {
		t.Fatal(err)
	}
	it, err := backlog.NewStore(root).Find("idea-one")
	if err != nil {
		t.Fatal(err)
	}
	if it.Dispatch != "approved" || it.Status != "new" {
		t.Errorf("dispatch/status = %s/%s, want approved/new", it.Dispatch, it.Status)
	}
}

func TestBacklogModeCloseDone(t *testing.T) {
	root := fixtureRoot(t)
	b := loadedMode(t, root)

	// done requires validation.
	if _, err := action(t, b, "d").Do(indexOf(t, b, "ready-item")); err == nil {
		t.Error("done on triaged item: want precondition error")
	}
	if _, err := action(t, b, "d").Do(indexOf(t, b, "claimed-item")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".anthill", "backlog", "cli", "claimed-item.md")); !os.IsNotExist(err) {
		t.Error("closed item file still in its workstream dir")
	}
	if _, err := os.Stat(filepath.Join(root, ".anthill", "backlog", "done", "claimed-item.md")); err != nil {
		t.Errorf("closed item not archived in done/: %v", err)
	}
	log, err := os.ReadFile(filepath.Join(root, ".anthill", "backlog", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "claimed-item — done") {
		t.Errorf("changelog missing done line:\n%s", log)
	}
}

func TestBacklogModeDiscardAndRemove(t *testing.T) {
	root := fixtureRoot(t)
	b := loadedMode(t, root)

	if _, err := action(t, b, "x").Do(indexOf(t, b, "ready-item")); err != nil {
		t.Fatal(err)
	}
	if _, err := action(t, b, "X").Do(indexOf(t, b, "bug-item")); err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile(filepath.Join(root, ".anthill", "backlog", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ready-item — discarded", "bug-item — removed"} {
		if !strings.Contains(string(log), want) {
			t.Errorf("changelog missing %q:\n%s", want, log)
		}
	}
}

// TestRootModelOverFixture wires the real backlog mode into the root model and
// drives one destructive action end to end (arm, confirm, reload, selection).
func TestRootModelOverFixture(t *testing.T) {
	root := fixtureRoot(t)
	m := New(root)
	next, _ := m.Update(reloadMsg{})
	m = next.(Model)
	if m.mode().Len() != 4 {
		t.Fatalf("loaded %d rows, want 4", m.mode().Len())
	}

	// Discard the selected (first) row: [x] then [y].
	next, _ = m.Update(key("x"))
	m = next.(Model)
	if m.confirm == nil {
		t.Fatal("no confirm prompt armed")
	}
	next, _ = m.Update(key("y"))
	m = next.(Model)
	if m.mode().Len() != 3 {
		t.Fatalf("after discard: %d rows, want 3", m.mode().Len())
	}
	if m.selected != 0 {
		t.Errorf("selection = %d, want 0 (next item in place)", m.selected)
	}
	if m.statusErr {
		t.Errorf("status is an error: %s", m.status)
	}
}
