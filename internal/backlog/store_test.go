package backlog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestStore builds an initialized schema-3 .anthill tree (key prefix ANT)
// in a temp dir.
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
	ws := "---\nschema: 3\nkey-prefix: ANT\nsweep-order: bugs, cli, dev, process\nnever-implicit:\n---\n"
	if err := os.WriteFile(filepath.Join(root, ".anthill", "backlog", "workstreams.md"), []byte(ws), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvActor, "")
	t.Setenv(EnvSession, "")
	t.Setenv(EnvReporter, "")
	t.Setenv(EnvClaude, "")
	return NewStore(root)
}

func TestNewAndFindRoundTrip(t *testing.T) {
	s := newTestStore(t)
	it, err := s.New(NewParams{Title: "Add a JSON flag", Value: "machine output", Source: "smoke", Hint: "cli", Description: "do it"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if it.ID != "ANT-1" {
		t.Fatalf("id = %q, want ANT-1", it.ID)
	}
	if filepath.Base(it.Path) != "0001-add-a-json-flag.md" {
		t.Fatalf("path = %q, want a 0001-add-a-json-flag.md file", it.Path)
	}
	got, err := s.Find(it.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.Title != "Add a JSON flag" || got.Status != StatusNew || got.Dispatch != DispatchProposed ||
		got.Reporter != "user" || got.Priority != "normal" || got.CreatedAt == "" || got.Hint != "cli" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	secs := got.Sections()
	if secs["Description"] != "do it" {
		t.Fatalf("description section = %q", secs["Description"])
	}
	if !strings.Contains(secs["Log"], " new by ") {
		t.Fatalf("log missing new line: %q", secs["Log"])
	}
}

// triaged creates an item and triages it into cli as a ready-able auto item.
func triaged(t *testing.T, s *Store, title string) *Item {
	t.Helper()
	it, err := s.New(NewParams{Title: title, Value: "v", Hint: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	err = s.Triage(it, map[string]string{
		"workstream": "cli", "summary": "s", "change-type": "doc", "risk": "additive",
		"verify": "go test ./...", "disposition": "auto",
	}, "worth it", "triager")
	if err != nil {
		t.Fatalf("Triage: %v", err)
	}
	return it
}

func TestTriageMovesStripsHintAndLogs(t *testing.T) {
	s := newTestStore(t)
	it := triaged(t, s, "Move me")
	got, err := s.Find(it.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Workstream != "cli" || got.Hint != "" || got.Status != StatusTriaged || got.TriagedAt == "" {
		t.Fatalf("triage result: %+v", got)
	}
	name := filepath.Base(it.Path) // a move keeps the NNNN-<slug>.md filename
	if got.Path != filepath.Join(s.wsDir("cli"), name) {
		t.Fatalf("path = %q", got.Path)
	}
	if _, err := os.Stat(filepath.Join(s.intakeDir(), name)); !os.IsNotExist(err) {
		t.Fatal("intake file still present")
	}
	if got.Sections()["Triage"] != "worth it" {
		t.Fatalf("triage section = %q", got.Sections()["Triage"])
	}
}

func TestTriageRequiresFieldSet(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.New(NewParams{Title: "x", Value: "v"})
	err := s.Triage(it, map[string]string{"workstream": "cli"}, "", "")
	var ve *ValidationError
	if !errors.As(err, &ve) || !strings.Contains(ve.Msg, "summary") {
		t.Fatalf("want validation error naming summary, got %v", err)
	}
}

func TestMoveToUnknownWorkstreamRejected(t *testing.T) {
	s := newTestStore(t)
	it, _ := s.New(NewParams{Title: "x", Value: "v"})
	var ve *ValidationError
	if err := s.Move(it, "nope"); !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestLifecycleHappyPath(t *testing.T) {
	s := newTestStore(t)
	it := triaged(t, s, "Work")
	if _, err := s.SetDispatch(it, DispatchApproved, "user"); err != nil {
		t.Fatal(err)
	}
	if !it.Ready() {
		t.Fatal("approved auto item with verify should be ready")
	}
	t.Setenv(EnvSession, "2f1c0000-aaaa")
	c, err := s.Claim(it, "worker-a", "", false)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if c.ClaimedBy != "worker-a" || c.Session != "2f1c0000-aaaa" || c.Status != StatusInProgress {
		t.Fatalf("claim stamps: %+v", c)
	}
	if err := s.Deliver(c, "someone-else", "ran", false); err == nil {
		t.Fatal("non-claimant deliver must be refused")
	}
	if err := s.Deliver(c, "worker-a", "go test exit 0", false); err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if !strings.Contains(c.Sections()["Evidence"], "go test exit 0") {
		t.Fatalf("evidence: %q", c.Sections()["Evidence"])
	}
	if _, err := s.Close(c, CloseDone, "", "ctl", false); err != nil {
		t.Fatalf("Close done: %v", err)
	}
	got, err := s.Find(it.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusDone || got.ClosedAt == "" || filepath.Base(filepath.Dir(got.Path)) != DoneDir {
		t.Fatalf("done item: %+v", got)
	}
	log := got.Sections()["Log"]
	for _, verb := range []string{"new", "triage", "approve", "claim by worker-a (session 2f1c0000…)", "deliver", "close --done"} {
		if !strings.Contains(log, verb) {
			t.Fatalf("log lacks %q:\n%s", verb, log)
		}
	}
}

func TestValidateNeedsMigrationWithoutStamp(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.New(NewParams{Title: "x", Value: "v"}); err != nil {
		t.Fatal(err)
	}
	writeWorkstreamsFrontmatter(t, s, "sweep-order: cli\n")
	if _, err := s.Validate(false); !errors.Is(err, ErrNeedsMigration) {
		t.Fatalf("want ErrNeedsMigration, got %v", err)
	}
}

func TestNewStampsSchemaOnEmptyTree(t *testing.T) {
	s := newTestStore(t)
	writeWorkstreamsFrontmatter(t, s, "sweep-order: cli\n")
	if _, err := s.New(NewParams{Title: "first", Value: "v"}); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.SchemaVersion(); v != CurrentSchema {
		t.Fatalf("schema = %d, want %d", v, CurrentSchema)
	}
	if p, err := s.KeyPrefix(); err != nil || !ValidPrefix(p) {
		t.Fatalf("key-prefix = %q (%v), want a derived valid prefix", p, err)
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

// writeTriagedItem hand-writes a triaged cli/ item NNNN-<slug>.md (id ANT-n)
// with the given change-type and disposition (verify real, so only
// change-type/disposition are at play).
func writeTriagedItem(t *testing.T, s *Store, n int, slug, changeType, disposition string) {
	t.Helper()
	content := "---\ntitle: t\nsummary: s\nvalue: v\nworkstream: cli\nstatus: triaged\ndispatch: approved\n" +
		"disposition: " + disposition + "\npriority: normal\nreporter: user\nchange-type: " + changeType +
		"\nrisk: additive\nverify: go test ./...\ncreated-at: \"2026-11-17T00:00:00Z\"\n---\n"
	if err := os.WriteFile(filepath.Join(s.wsDir("cli"), FormatFilename(n, slug)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateChangeTypeVocabularyWarns(t *testing.T) {
	s := newTestStore(t)
	writeWorkstreamsFrontmatter(t, s, "schema: 3\nkey-prefix: ANT\nsweep-order: bugs, cli, dev, process\nchange-types: doc, tooling\n")
	writeTriagedItem(t, s, 1, "good", "doc", "design")
	writeTriagedItem(t, s, 2, "bad", "wildcard", "design")

	res, err := s.Validate(false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("out-of-vocabulary change-type must not fail validation; violations=%+v", res.Violations)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].ID != "ANT-2" || res.Warnings[0].Rule != "change-type-vocab" {
		t.Fatalf("expected one change-type-vocab warning for ANT-2 (bad), got %+v", res.Warnings)
	}
	res, err = s.Validate(true)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("--strict must promote the vocabulary warning to a violation")
	}
}

func TestValidateNeverAutoFromConfig(t *testing.T) {
	s := newTestStore(t)
	writeWorkstreamsFrontmatter(t, s, "schema: 3\nkey-prefix: ANT\nsweep-order: cli\nchange-types: doc, design\nnever-auto-change-types: design\n")
	writeTriagedItem(t, s, 1, "auto-design", "design", "auto")

	res, err := s.Validate(false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range res.Violations {
		if v.Rule == "disposition-coherence" && v.ID == "ANT-1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected disposition-coherence violation; got %+v", res.Violations)
	}
}

func TestSortRank(t *testing.T) {
	mk := func(id, ws, dispatch, prio, created string) *Item {
		return &Item{ID: id, Workstream: ws, Dispatch: dispatch, Priority: prio, CreatedAt: created}
	}
	order := []string{"bugs", "cli"}
	items := []*Item{
		mk("cli-old-normal", "cli", "approved", "normal", "2026-01-01T00:00:00Z"),
		mk("cli-new-normal", "cli", "approved", "normal", "2026-02-01T00:00:00Z"),
		mk("bugs-normal", "bugs", "approved", "normal", "2026-03-01T00:00:00Z"),
		mk("cli-high", "cli", "approved", "high", "2026-03-01T00:00:00Z"),
		mk("ft-low", "cli", "fast-tracked", "low", "2026-03-01T00:00:00Z"),
		mk("intake-high", "", "proposed", "high", "2026-01-01T00:00:00Z"),
		mk("cli-low", "cli", "approved", "low", "2025-01-01T00:00:00Z"),
		mk("cli-nostamp", "cli", "approved", "", ""),
	}
	SortItems(items, "rank", order, false)
	want := []string{"ft-low", "cli-high", "intake-high", "bugs-normal", "cli-old-normal", "cli-new-normal", "cli-nostamp", "cli-low"}
	for i, id := range want {
		if items[i].ID != id {
			got := make([]string, len(items))
			for j, it := range items {
				got[j] = it.ID
			}
			t.Fatalf("rank order = %v, want %v", got, want)
		}
	}
	SortItems(items, "rank", order, true)
	if items[0].ID != "cli-low" || items[len(items)-1].ID != "ft-low" {
		t.Fatalf("reverse rank: first=%s last=%s", items[0].ID, items[len(items)-1].ID)
	}
	SortItems(items, "age", order, false)
	if items[0].ID != "cli-low" || items[len(items)-1].ID != "cli-nostamp" {
		t.Fatalf("age: first=%s last=%s", items[0].ID, items[len(items)-1].ID)
	}
}
