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

// TestDoctorHealthy covers `anthill doctor` on a healthy tree (spec §3.15):
// exit 0, json ok=true with a checks array.
func TestDoctorHealthy(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "healthy-item", approvedFields("Healthy item", "cli"), "body")

	r := runIn(t, root, "--json", "doctor")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["ok"] != true {
		t.Fatalf("healthy tree doctor ok = %v, want true\n%s", obj["ok"], r.stdout)
	}
	if _, ok := obj["checks"].([]any); !ok {
		t.Fatalf("doctor --json missing checks array: %v", obj)
	}
}

// TestDoctorSweepOrderNames covers exit 3: workstreams.md sweep-order names a
// directory that does not exist (spec §3.15 integrity check).
func TestDoctorSweepOrderNames(t *testing.T) {
	root := mkTree(t)
	// Rewrite workstreams.md to reference a non-existent stream.
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nsweep-order: bugs, cli, dev, process, ghoststream\nnever-implicit:\n---\n\n# Workstreams\n")

	r := runIn(t, root, "doctor")
	wantExit(t, r, 3)
}

// TestDoctorAnsweredUnapplied covers the failure mode the escalate skill calls
// out: an answered-but-unapplied escalation record (spec §3.15). doctor --strict
// should fail.
func TestDoctorAnsweredUnapplied(t *testing.T) {
	root := mkTree(t)
	id := raiseOne(t, root, "Answered but not applied")
	wantExit(t, runIn(t, root, "escalation", "answer", id, "--decision", "yes"), 0)

	r := runIn(t, root, "doctor", "--strict")
	wantExit(t, r, 3)
}
