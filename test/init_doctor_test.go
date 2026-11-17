package e2e_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// prefixRe is the key-prefix form: 2 to 5 uppercase ASCII letters.
var prefixRe = regexp.MustCompile(`^[A-Z]{2,5}$`)

func workstreamsPath(root string) string { return filepath.Join(backlogDir(root), "workstreams.md") }

// TestInitScaffolds covers `anthill init` (spec §3.14): scaffolds the .anthill/
// tree with the standard workstream dirs, escalations/, empty runtime files,
// and a workstreams.md stamped `schema: 3` with a derived key-prefix.
func TestInitScaffolds(t *testing.T) {
	dir := t.TempDir()
	r := run(t, "init", "--root", dir)
	wantExit(t, r, 0)

	base := filepath.Join(dir, ".anthill")
	for _, sub := range []string{
		"backlog/intake", "backlog/cli", "backlog/dev", "backlog/process",
		"backlog/bugs", "escalations",
	} {
		wantFilePresent(t, filepath.Join(base, sub))
	}
	wantFilePresent(t, filepath.Join(base, "backlog", "CHANGELOG.md"))
	wantFilePresent(t, filepath.Join(base, "escalations", "LOG.md"))

	fm := readFrontmatter(t, workstreamsPath(dir))
	if fm["schema"] != "3" {
		t.Fatalf("init workstreams.md schema = %q, want 3: %v", fm["schema"], fm)
	}
	if !prefixRe.MatchString(fm["key-prefix"]) {
		t.Fatalf("init workstreams.md key-prefix = %q, want 2-5 uppercase letters", fm["key-prefix"])
	}
}

// TestInitKeyPrefixFlag covers --key-prefix: the value is uppercased into
// workstreams.md and the first `new` allocates <PREFIX>-1.
func TestInitKeyPrefixFlag(t *testing.T) {
	dir := t.TempDir()
	wantExit(t, run(t, "init", "--root", dir, "--key-prefix", "tst"), 0)
	if fm := readFrontmatter(t, workstreamsPath(dir)); fm["key-prefix"] != "TST" {
		t.Fatalf("key-prefix = %q, want TST", fm["key-prefix"])
	}
	if id := newID(t, dir, "First"); id != "TST-1" {
		t.Fatalf("first new = %q, want TST-1", id)
	}
}

// TestInitFillsPrefixPlaceholder covers init --force on a tree whose
// workstreams.md still carries the framework template's unfilled
// `key-prefix: <PREFIX>` placeholder: `new` refuses it (exit 6, naming
// key-prefix), init --force --key-prefix fills the line in place, and `new`
// then allocates under the filled prefix.
func TestInitFillsPrefixPlaceholder(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, ".anthill")
	for _, d := range []string{"backlog/intake", "backlog/cli", "escalations"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	writeRaw(t, workstreamsPath(root),
		"---\nschema: 3\nkey-prefix: <PREFIX>   # 2-5 uppercase letters; anthill init fills it\nsweep-order: cli\n---\n")

	r := runIn(t, root, "backlog", "new", "--title", "Too early", "--value", "v")
	wantExit(t, r, 6)
	wantContains(t, r.stderr, "key-prefix", "new stderr")

	wantExit(t, run(t, "init", "--root", root, "--force", "--key-prefix", "TST"), 0)
	if id := newID(t, root, "First"); id != "TST-1" {
		t.Fatalf("first new after init = %q, want TST-1", id)
	}
	ws := strings.ReplaceAll(readAll(t, workstreamsPath(root)), "\r\n", "\n")
	wantContains(t, ws, "\nkey-prefix: TST\n", "workstreams.md")
	if strings.Contains(ws, "<PREFIX>") {
		t.Fatalf("placeholder survived init --force:\n%s", ws)
	}
}

// TestInitAlreadyInitialized covers exit 6: init on an already-initialized tree
// without --force (spec §3.14).
func TestInitAlreadyInitialized(t *testing.T) {
	dir := t.TempDir()
	wantExit(t, run(t, "init", "--root", dir), 0)
	r := run(t, "init", "--root", dir)
	wantExit(t, r, 6)
}

// TestInitForce covers init --force populating an existing dir without error.
func TestInitForce(t *testing.T) {
	dir := t.TempDir()
	wantExit(t, run(t, "init", "--root", dir), 0)
	r := run(t, "init", "--root", dir, "--force")
	wantExit(t, r, 0)
}

// TestInitWorkstream covers seeding an extra workstream directory (spec §3.14).
func TestInitWorkstream(t *testing.T) {
	dir := t.TempDir()
	r := run(t, "init", "--root", dir, "--workstream", "extra")
	wantExit(t, r, 0)
	wantFilePresent(t, filepath.Join(dir, ".anthill", "backlog", "extra"))
}

// TestDoctorSweepOrderNames covers exit 3: workstreams.md sweep-order names a
// directory that does not exist (spec §4.4 Section B integrity check). Built on
// a real scaffolded install so install integrity (Section A) is healthy and the
// bad sweep-order name is the sole failure being isolated.
func TestDoctorSweepOrderNames(t *testing.T) {
	dir := scaffoldFresh(t)
	// Perturb ONLY the Section B sweep-order: name a stream with no directory,
	// leaving skills, structure, and framework.md intact.
	writeRaw(t, workstreamsPath(dir),
		"---\nsweep-order: bugs, product, dev, process, ghoststream\nnever-implicit:\n---\n\n# Workstreams\n")

	r := runIn(t, dir, "doctor")
	wantExit(t, r, 3)
}

// TestDoctorChangeTypeVocabWarnsNotFails covers the soft change-type check: an
// item whose change-type is outside the project's declared `change-types`
// vocabulary makes doctor emit a warn line but exit 0. The vocabulary is the
// project's domain (declared in workstreams.md), not a hard schema rule.
func TestDoctorChangeTypeVocabWarnsNotFails(t *testing.T) {
	dir := scaffoldFresh(t)
	// Declare a narrow vocabulary (streams match the scaffolded dirs so Section B
	// sweep-order stays healthy); the item below uses a change-type outside it.
	writeRaw(t, workstreamsPath(dir),
		"---\nschema: 3\nkey-prefix: "+testPrefix+"\nsweep-order: bugs, product, dev, process\nnever-implicit:\nchange-types: doc, tooling\n---\n\n# Workstreams\n")
	writeItem(t, dir, "dev", 1, "novel-item", withField(readyFields("t", "dev"), "change-type", "new-assembly"), "")

	r := runIn(t, dir, "doctor")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "change-type-vocab", "doctor output")
}

// TestDoctorAnsweredUnapplied covers the failure mode the escalate skill calls
// out: an answered-but-unapplied escalation record (spec §4.4 Section B). The
// escalations-applied check is always-on (not strict-gated); built on a real
// scaffolded install so the answered-but-unapplied record is the sole failure.
func TestDoctorAnsweredUnapplied(t *testing.T) {
	dir := scaffoldFresh(t)
	id := raiseOne(t, dir, "Answered but not applied")
	wantExit(t, runIn(t, dir, "escalation", "answer", id, "--decision", "yes"), 0)

	r := runIn(t, dir, "doctor")
	wantExit(t, r, 3)
}
