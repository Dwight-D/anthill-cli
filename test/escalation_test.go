package e2e_test

import (
	"strings"
	"testing"
)

// raiseOne raises an escalation and returns its id (the <date>-<slug> stem),
// read from the --json record's path/id.
func raiseOne(t *testing.T, root, question string, extra ...string) string {
	t.Helper()
	args := append([]string{"--json", "escalation", "raise",
		"--to", "supervisor", "--from", "dispatcher", "--question", question}, extra...)
	r := runIn(t, root, args...)
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if id, ok := obj["id"].(string); ok && id != "" {
		return id
	}
	// Fall back to deriving the stem from the path.
	if p, ok := obj["path"].(string); ok {
		base := p
		if i := strings.LastIndexAny(base, "/\\"); i >= 0 {
			base = base[i+1:]
		}
		return strings.TrimSuffix(base, ".md")
	}
	t.Fatalf("raise --json missing id/path: %v", obj)
	return ""
}

// TestEscalationRaise covers `escalation raise` (spec §3.9): creates a record
// file under escalations/ with status: open.
func TestEscalationRaise(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "--json", "escalation", "raise",
		"--to", "supervisor", "--from", "dispatcher",
		"--question", "Should we do the thing?")
	wantExit(t, r, 0)

	obj := jsonObj(t, r.stdout)
	if obj["to"] != "supervisor" {
		t.Fatalf("record to = %v, want supervisor", obj["to"])
	}
	if obj["status"] != "open" {
		t.Fatalf("record status = %v, want open", obj["status"])
	}
	p, ok := obj["path"].(string)
	if !ok || p == "" {
		t.Fatalf("raise --json missing path: %v", obj)
	}
	wantFilePresent(t, p)
}

// TestEscalationRaiseMissingQuestion covers exit 2 for a missing required flag.
func TestEscalationRaiseMissingQuestion(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "escalation", "raise", "--to", "supervisor", "--from", "dispatcher")
	wantExit(t, r, 2)
}

// TestEscalationRaiseBadTo covers exit 3 for an illegal --to value (spec §3.9).
func TestEscalationRaiseBadTo(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "escalation", "raise",
		"--to", "nobody", "--from", "dispatcher", "--question", "q")
	wantExit(t, r, 3)
}

// TestEscalationList covers `escalation list --json` returning an array.
func TestEscalationList(t *testing.T) {
	root := mkTree(t)
	raiseOne(t, root, "First question")
	raiseOne(t, root, "Second question")

	r := runIn(t, root, "--json", "escalation", "list")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 2 {
		t.Fatalf("escalation list length = %d, want 2\n%s", len(arr), r.stdout)
	}
}

// TestEscalationListFilterStatus covers the --status filter.
func TestEscalationListFilterStatus(t *testing.T) {
	root := mkTree(t)
	raiseOne(t, root, "Open one")
	r := runIn(t, root, "--json", "escalation", "list", "--status", "open")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 1 {
		t.Fatalf("--status open should return 1, got %d", len(arr))
	}
}

// TestEscalationShow covers `escalation show <id> --json`.
func TestEscalationShow(t *testing.T) {
	root := mkTree(t)
	id := raiseOne(t, root, "Verbatim question text")
	r := runIn(t, root, "--json", "escalation", "show", id)
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["status"] != "open" {
		t.Fatalf("show status = %v, want open", obj["status"])
	}
}

// TestEscalationShowNotFound covers exit 4.
func TestEscalationShowNotFound(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "escalation", "show", "2000-01-01-nope")
	wantExit(t, r, 4)
}

// TestEscalationAnswer covers `escalation answer` setting status answered and
// appending a ## Decision section (spec §3.12).
func TestEscalationAnswer(t *testing.T) {
	root := mkTree(t)
	id := raiseOne(t, root, "Answer me")

	r := runIn(t, root, "escalation", "answer", id, "--decision", "Yes, proceed.")
	wantExit(t, r, 0)

	// Read the still-present record and verify state.
	show := runIn(t, root, "--json", "escalation", "show", id)
	wantExit(t, show, 0)
	obj := jsonObj(t, show.stdout)
	if obj["status"] != "answered" {
		t.Fatalf("status after answer = %v, want answered", obj["status"])
	}
	// The decision text should be present in the record body somewhere.
	wantContains(t, show.stdout, "Yes, proceed.", "answered record")
}

// TestEscalationAnswerNotFound covers exit 4.
func TestEscalationAnswerNotFound(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "escalation", "answer", "2000-01-01-nope", "--decision", "x")
	wantExit(t, r, 4)
}

// TestEscalationAnswerNotOpen covers exit 6: answering a record that is not open
// (already answered) (spec §3.12).
func TestEscalationAnswerNotOpen(t *testing.T) {
	root := mkTree(t)
	id := raiseOne(t, root, "Double answer")
	runIn(t, root, "escalation", "answer", id, "--decision", "first")
	r := runIn(t, root, "escalation", "answer", id, "--decision", "second")
	wantExit(t, r, 6)
}

// TestEscalationApply covers `escalation apply` (spec §3.13): appends ## Applied,
// logs one line to LOG.md, then DELETES the record file.
func TestEscalationApply(t *testing.T) {
	root := mkTree(t)
	id := raiseOne(t, root, "Apply me")
	runIn(t, root, "escalation", "answer", id, "--decision", "go")

	before := readAll(t, escalLogPath(root))
	r := runIn(t, root, "--json", "escalation", "apply", id)
	wantExit(t, r, 0)

	obj := jsonObj(t, r.stdout)
	if obj["applied"] != true {
		t.Fatalf("json applied = %v, want true", obj["applied"])
	}
	if obj["logged"] != true {
		t.Fatalf("json logged = %v, want true", obj["logged"])
	}

	// Record file deleted.
	wantFileGone(t, escalDir(root)+"/"+id+".md")
	// LOG.md grew.
	after := readAll(t, escalLogPath(root))
	if after == before {
		t.Fatalf("apply must append a line to LOG.md; content unchanged")
	}
	wantContains(t, after, id, "LOG.md after apply")
}

// TestEscalationApplyNotFound covers exit 4.
func TestEscalationApplyNotFound(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "escalation", "apply", "2000-01-01-nope")
	wantExit(t, r, 4)
}

// TestEscalationApplyNotAnswered covers exit 6: applying an open (not answered)
// record (spec §3.13).
func TestEscalationApplyNotAnswered(t *testing.T) {
	root := mkTree(t)
	id := raiseOne(t, root, "Still open")
	r := runIn(t, root, "escalation", "apply", id)
	wantExit(t, r, 6)
}

// TestEscalationLifecycle exercises the full raise → list → show → answer →
// apply lifecycle end to end (spec §3.9–3.13).
func TestEscalationLifecycle(t *testing.T) {
	root := mkTree(t)
	id := raiseOne(t, root, "Full lifecycle question", "--item", "some-item")

	// list shows it open
	l := runIn(t, root, "--json", "escalation", "list", "--status", "open")
	if len(jsonArr(t, l.stdout)) != 1 {
		t.Fatalf("expected 1 open record")
	}

	// answer
	wantExit(t, runIn(t, root, "escalation", "answer", id, "--decision", "decided"), 0)
	// apply
	wantExit(t, runIn(t, root, "escalation", "apply", id), 0)
	// gone
	wantFileGone(t, escalDir(root)+"/"+id+".md")
}
