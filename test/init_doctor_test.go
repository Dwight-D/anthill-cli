package e2e_test

import (
	"path/filepath"
	"testing"
)

// TestInitScaffolds covers `anthill init` (spec §3.14): scaffolds the .anthill/
// tree with the standard workstream dirs, escalations/, and empty runtime files.
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
	writeRaw(t, filepath.Join(backlogDir(dir), "workstreams.md"),
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
	writeRaw(t, filepath.Join(backlogDir(dir), "workstreams.md"),
		"---\nsweep-order: bugs, product, dev, process\nnever-implicit:\nchange-types: doc, tooling\n---\n\n# Workstreams\n")
	writeItem(t, dir, "dev", "novel-item", map[string]string{
		"workstream":    "dev",
		"title":         "t",
		"value":         "v",
		"change-type":   "new-assembly",
		"risk":          "additive",
		"verify":        "go test ./...",
		"value-verdict": "ADVANCE — x",
		"disposition":   "REVIEW",
		"status":        "approved",
		"priority":      "normal",
	}, "")

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
