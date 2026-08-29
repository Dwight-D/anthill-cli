package escalation

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// storeFixture builds a throwaway .anthill/escalations tree with one open and
// one answered record, returning the store.
func storeFixture(t *testing.T) *Store {
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
	write("2026-08-29-open-question.md",
		"---\nto: supervisor\nfrom: dispatcher\nstatus: open\nopened: 2026-08-29\n---\n"+
			"## Question\nShould we do the thing?\n")
	write("2026-08-29-answered-question.md",
		"---\nto: user\nfrom: supervisor\nstatus: answered\nopened: 2026-08-29\n---\n"+
			"## Question\nWhich option?\n## Decision\nOption B.\n")
	return NewStore(root)
}

func readLog(t *testing.T, s *Store) string {
	t.Helper()
	data, err := os.ReadFile(s.LogPath())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCloseOpenRecord(t *testing.T) {
	s := storeFixture(t)
	r, err := s.Close("2026-08-29-open-question", "went stale")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.Path); !os.IsNotExist(err) {
		t.Error("closed record file still exists")
	}
	log := readLog(t, s)
	if !strings.Contains(log, "2026-08-29-open-question — supervisor/closed: went stale") {
		t.Errorf("LOG.md missing close line:\n%s", log)
	}
}

func TestCloseDefaultsReasonToQuestion(t *testing.T) {
	s := storeFixture(t)
	if _, err := s.Close("2026-08-29-open-question", ""); err != nil {
		t.Fatal(err)
	}
	log := readLog(t, s)
	if !strings.Contains(log, "closed: Should we do the thing?") {
		t.Errorf("LOG.md close line should fall back to the question:\n%s", log)
	}
}

func TestCloseAnsweredRecordIsPrecondition(t *testing.T) {
	s := storeFixture(t)
	_, err := s.Close("2026-08-29-answered-question", "")
	var pe *PreconditionError
	if err == nil {
		t.Fatal("closing an answered record: want precondition error")
	}
	if !errors.As(err, &pe) {
		t.Fatalf("closing an answered record: got %T (%v), want *PreconditionError", err, err)
	}
	// The record must survive a refused close.
	if _, err := s.Find("2026-08-29-answered-question"); err != nil {
		t.Errorf("record gone after refused close: %v", err)
	}
}

func TestCloseNotFound(t *testing.T) {
	s := storeFixture(t)
	if _, err := s.Close("2000-01-01-nope", ""); err == nil {
		t.Fatal("closing a missing record: want error")
	}
}

func TestRemoveAnyStatus(t *testing.T) {
	s := storeFixture(t)
	for _, id := range []string{"2026-08-29-open-question", "2026-08-29-answered-question"} {
		r, err := s.Remove(id, "stale")
		if err != nil {
			t.Fatalf("remove %s: %v", id, err)
		}
		if _, err := os.Stat(r.Path); !os.IsNotExist(err) {
			t.Errorf("removed record %s still exists", id)
		}
	}
	log := readLog(t, s)
	for _, want := range []string{
		"2026-08-29-open-question — supervisor/removed: stale",
		"2026-08-29-answered-question — user/removed: stale",
	} {
		if !strings.Contains(log, want) {
			t.Errorf("LOG.md missing %q:\n%s", want, log)
		}
	}
}

func TestRemoveNotFound(t *testing.T) {
	s := storeFixture(t)
	if _, err := s.Remove("2000-01-01-nope", ""); err == nil {
		t.Fatal("removing a missing record: want error")
	}
}
