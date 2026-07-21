package backlog

import (
	"os"
	"path/filepath"
	"testing"
)

// newTestStore builds an initialized .anthill tree in a temp dir.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"intake", "cli", "dev", "process", "bugs"} {
		if err := os.MkdirAll(filepath.Join(root, ".anthill", "backlog", d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".anthill", "escalations"), 0o755); err != nil {
		t.Fatal(err)
	}
	ws := "---\nsweep-order: bugs, cli, dev, process\nnever-implicit:\n---\n"
	if err := os.WriteFile(filepath.Join(root, ".anthill", "backlog", "workstreams.md"), []byte(ws), 0o644); err != nil {
		t.Fatal(err)
	}
	return NewStore(root)
}

func TestNewAndFindRoundTrip(t *testing.T) {
	s := newTestStore(t)
	it, err := s.New(NewParams{Title: "Add a JSON flag", Value: "machine output", Source: "smoke", Hint: "cli"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if it.ID != "add-a-json-flag" {
		t.Fatalf("id = %q, want add-a-json-flag", it.ID)
	}
	got, err := s.Find(it.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.Title != "Add a JSON flag" || got.Value != "machine output" ||
		got.Hint != "cli" || got.Status != "idea" || got.Source != "smoke" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.Workstream != "" {
		t.Fatalf("intake item should have no workstream, got %q", got.Workstream)
	}
}

func TestIntakeToWorkstreamMoveStripsHint(t *testing.T) {
	s := newTestStore(t)
	it, err := s.New(NewParams{Title: "Move me", Value: "v", Hint: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	oldPath := it.Path
	it.Hint = "" // triage strips hint on the move out of intake
	if err := s.Move(it, "cli"); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old intake file still present at %s", oldPath)
	}
	got, err := s.Find(it.ID)
	if err != nil {
		t.Fatalf("Find after move: %v", err)
	}
	if got.Workstream != "cli" {
		t.Fatalf("workstream = %q, want cli", got.Workstream)
	}
	if got.Hint != "" {
		t.Fatalf("hint should be stripped, got %q", got.Hint)
	}
	wantPath := filepath.Join(s.wsDir("cli"), it.ID+".md")
	if got.Path != wantPath {
		t.Fatalf("path = %q, want %q", got.Path, wantPath)
	}
}

func TestMoveToUnknownWorkstreamRejected(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.New(NewParams{Title: "x", Value: "v"})
	err := s.Move(it, "nope")
	var ve *ValidationError
	if err == nil {
		t.Fatal("expected validation error for unknown workstream")
	}
	if !asValidation(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
}

func asValidation(err error, target **ValidationError) bool {
	if ve, ok := err.(*ValidationError); ok {
		*target = ve
		return true
	}
	return false
}

func TestClaimCAS(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.New(NewParams{Title: "Claimable", Value: "v"})
	it.Hint = ""
	if err := s.Move(it, "cli"); err != nil {
		t.Fatal(err)
	}
	// Bring it to a ready state: approved + verify.
	cur, _ := s.Find(it.ID)
	cur.Status = "approved"
	cur.Verify = "go test ./... exit 0"
	cur.ChangeType = "new-flag"
	cur.Risk = "additive"
	cur.ValueVerdict = "ADVANCE — worth it"
	cur.Disposition = "REVIEW"
	if err := s.Save(cur); err != nil {
		t.Fatalf("Save approved: %v", err)
	}
	if !cur.Ready() {
		t.Fatal("item should be ready")
	}

	// Simulate a claim: read, CAS on status, write in-progress.
	fresh, _ := s.Find(it.ID)
	if fresh.Status != "approved" {
		t.Fatalf("expected approved, got %s", fresh.Status)
	}
	fresh.Status = "in-progress"
	fresh.ClaimedAt = "2026-07-20T00:00:00Z"
	if err := s.Save(fresh); err != nil {
		t.Fatalf("Save claim: %v", err)
	}
	got, _ := s.Find(it.ID)
	if got.Status != "in-progress" || got.ClaimedAt == "" {
		t.Fatalf("claim not persisted: status=%q claimed-at=%q", got.Status, got.ClaimedAt)
	}
}

func TestValidateStrictCatchesReadyInconsistency(t *testing.T) {
	s := newTestStore(t)
	// Hand-write an approved item with an empty verify into cli/.
	content := "---\nworkstream: cli\ntitle: t\nvalue: v\nchange-type: doc\nrisk: additive\n" +
		"verify: none\nvalue-verdict: ADVANCE — x\ndisposition: REVIEW\nstatus: approved\n---\n"
	if err := os.WriteFile(filepath.Join(s.wsDir("cli"), "bad.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := s.Validate(true)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("expected strict validation to flag approved+none-verify")
	}
	found := false
	for _, v := range res.Violations {
		if v.Check == "ready-consistency" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing ready-consistency violation: %+v", res.Violations)
	}
}

// writeWorkstreamsFrontmatter overwrites the store's workstreams.md with the
// given frontmatter body (between the --- fences).
func writeWorkstreamsFrontmatter(t *testing.T, s *Store, fm string) {
	t.Helper()
	content := "---\n" + fm + "---\n\n# Backlog workstreams\n"
	if err := os.WriteFile(filepath.Join(s.dir(), "workstreams.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeApprovedItem hand-writes a ready cli/ item with the given change-type and
// disposition (verify non-none, so only the change-type/disposition are at play).
func writeApprovedItem(t *testing.T, s *Store, id, changeType, disposition string) {
	t.Helper()
	content := "---\nworkstream: cli\ntitle: t\nvalue: v\nchange-type: " + changeType +
		"\nrisk: additive\nverify: go test ./...\nvalue-verdict: ADVANCE — x\ndisposition: " +
		disposition + "\nstatus: approved\n---\n"
	if err := os.WriteFile(filepath.Join(s.wsDir("cli"), id+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestValidateChangeTypeVocabularyWarns: a change-type outside the project's
// declared vocabulary is a warning, not a violation, and never fails validation.
func TestValidateChangeTypeVocabularyWarns(t *testing.T) {
	s := newTestStore(t)
	writeWorkstreamsFrontmatter(t, s, "sweep-order: bugs, cli, dev, process\nnever-implicit:\nchange-types: doc, tooling\n")
	writeApprovedItem(t, s, "good", "doc", "REVIEW")
	writeApprovedItem(t, s, "bad", "wildcard", "REVIEW")

	res, err := s.Validate(true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("out-of-vocabulary change-type must not fail validation; violations=%+v", res.Violations)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].ID != "bad" || res.Warnings[0].Check != "change-type-vocab" {
		t.Fatalf("expected one change-type-vocab warning for 'bad', got %+v", res.Warnings)
	}
}

// TestValidateChangeTypeFreeFormWhenUndeclared: with no declared vocabulary, any
// change-type is accepted with no warning.
func TestValidateChangeTypeFreeFormWhenUndeclared(t *testing.T) {
	s := newTestStore(t) // newTestStore declares no change-types
	writeApprovedItem(t, s, "anything", "some-novel-type", "REVIEW")

	res, err := s.Validate(true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || len(res.Warnings) != 0 {
		t.Fatalf("undeclared vocabulary must be free-form (no warnings); ok=%v warnings=%+v", res.OK, res.Warnings)
	}
}

// TestValidateNeverAutoFromConfig: never-auto-change-types in workstreams.md
// drives the disposition-coherence check (AUTO on a never-auto type is illegal).
func TestValidateNeverAutoFromConfig(t *testing.T) {
	s := newTestStore(t)
	writeWorkstreamsFrontmatter(t, s, "sweep-order: bugs, cli, dev, process\nnever-implicit:\nchange-types: doc, design\nnever-auto-change-types: design\n")
	writeApprovedItem(t, s, "auto-design", "design", "AUTO")

	res, err := s.Validate(true)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range res.Violations {
		if v.Check == "disposition-coherence" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected disposition-coherence violation for AUTO on never-auto type; got %+v", res.Violations)
	}
}
