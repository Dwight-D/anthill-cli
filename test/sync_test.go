package e2e_test

import (
	"strings"
	"testing"
)

// TestSyncDryRunWritesNothing covers `anthill sync --dry-run` (spec §4.5) on a
// CLEAN freshly-scaffolded install: every skill is reported unchanged, nothing
// is written, exit 0, and the tree is byte-for-byte unchanged.
func TestSyncDryRunWritesNothing(t *testing.T) {
	dir := scaffoldFresh(t)
	before := snapshotTree(t, dir)

	r := runIn(t, dir, "--json", "sync", "--dry-run")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)

	if updated := jsonStrings(obj["updated"]); len(updated) != 0 {
		t.Fatalf("dry-run on a clean install reports updates: %v", updated)
	}
	if conflicts := jsonStrings(obj["conflicts"]); len(conflicts) != 0 {
		t.Fatalf("dry-run on a clean install reports conflicts: %v", conflicts)
	}
	if unchanged := jsonStrings(obj["unchanged"]); len(unchanged) == 0 {
		t.Fatalf("dry-run reported no unchanged skills on a clean install\n%s", r.stdout)
	}
	if after := snapshotTree(t, dir); before != after {
		t.Fatalf("--dry-run modified the tree")
	}
}

// TestSyncRestoresStaleSkill covers the restore path (spec §4.5). A diverged
// installed skill on an otherwise-current install is a conflict without --force
// (exit 3, left untouched); --force overwrites the local edit, restoring the
// skill byte-identical to the embedded template (exit 0) and bumping
// synced-through to the embedded ref.
func TestSyncRestoresStaleSkill(t *testing.T) {
	dir := scaffoldFresh(t)
	stale := "STALE — replace me on sync\n"
	writeRaw(t, skillPath(dir, "dispatch"), stale)

	// (a) Without --force: a differing skill on a current install is a conflict.
	r := runIn(t, dir, "--json", "sync")
	wantExit(t, r, 3)
	obj := jsonObj(t, r.stdout)
	wantListContains(t, obj, "conflicts", "dispatch")
	if got := readAll(t, skillPath(dir, "dispatch")); got != stale {
		t.Fatalf("sync without --force modified the conflicted skill\n got: %q\nwant: %q", got, stale)
	}

	// (b) With --force: overwrite the local edit, restore verbatim, exit 0.
	rf := runIn(t, dir, "--json", "sync", "--force")
	wantExit(t, rf, 0)
	fobj := jsonObj(t, rf.stdout)
	wantListContains(t, fobj, "updated", "dispatch")

	// Restored byte-identical to the embedded template (compared via an
	// untouched reference scaffold of the same binary).
	ref := scaffoldFresh(t)
	wantSameFile(t, skillPath(dir, "dispatch"), skillPath(ref, "dispatch"))

	// synced-through bumped to the embedded ref.
	if to, _ := fobj["to_ref"].(string); to != templateRef(t) {
		t.Fatalf("sync to_ref = %q, want embedded template_ref %q", fobj["to_ref"], templateRef(t))
	}
	if fw := readAll(t, frameworkPath(dir)); !strings.Contains(fw, templateRef(t)) {
		t.Fatalf("framework.md not stamped with embedded ref after sync\n%s", fw)
	}
}

// TestSyncFlagsAutonomousLocalEdit covers the retired-exemption contract (spec
// §sync): the autonomous skill has no sanctioned regions, so a derived
// "## Proceed freely" edit on a current install is an unexpected local edit —
// reported as a conflict (exit 3) and left untouched without --force, then
// overwritten verbatim with --force like any other skill.
func TestSyncFlagsAutonomousLocalEdit(t *testing.T) {
	dir := scaffoldFresh(t)
	marker := "- Deploy the widget service via ./tools/deploy.sh without asking."
	setProceedList(t, skillPath(dir, "autonomous"), []string{marker})

	// Without --force: an unexpected local edit is a conflict, file untouched.
	r := runIn(t, dir, "--json", "sync")
	wantExit(t, r, 3)
	obj := jsonObj(t, r.stdout)
	wantListContains(t, obj, "conflicts", "autonomous")
	if got := readAll(t, skillPath(dir, "autonomous")); !strings.Contains(got, marker) {
		t.Fatalf("sync without --force modified the conflicted autonomous skill\n%s", got)
	}

	// With --force: overwrite verbatim, restoring the pristine skill (exit 0).
	rf := runIn(t, dir, "--json", "sync", "--force")
	wantExit(t, rf, 0)
	wantListContains(t, jsonObj(t, rf.stdout), "updated", "autonomous")
	ref := scaffoldFresh(t)
	wantSameFile(t, skillPath(dir, "autonomous"), skillPath(ref, "autonomous"))
}

// TestSyncJSONShape covers the --json shape (spec §4.5): { updated, unchanged,
// conflicts, from_ref, to_ref }.
func TestSyncJSONShape(t *testing.T) {
	dir := scaffoldFresh(t)
	r := runIn(t, dir, "--json", "sync")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	for _, k := range []string{"updated", "unchanged", "conflicts", "from_ref", "to_ref"} {
		wantHasKey(t, obj, k)
	}
}
