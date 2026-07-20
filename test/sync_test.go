package e2e_test

import (
	"strings"
	"testing"
)

// TestSyncDryRunWritesNothing covers `anthill sync --dry-run` (spec §4.5): show
// the skill-level diff without applying — the stale installed skill is left
// unchanged, exit 0.
func TestSyncDryRunWritesNothing(t *testing.T) {
	dir := scaffoldFresh(t)
	// Simulate a stale install: overwrite an installed skill with altered text.
	stale := "STALE INSTALLED SKILL — differs from embedded\n"
	writeRaw(t, skillPath(dir, "dispatch"), stale)

	r := runIn(t, dir, "sync", "--dry-run")
	wantExit(t, r, 0)

	if got := readAll(t, skillPath(dir, "dispatch")); got != stale {
		t.Fatalf("--dry-run modified the installed skill\n got: %q\nwant: %q", got, stale)
	}
}

// TestSyncRestoresStaleSkill covers the restore path (spec §4.5): sync re-copies
// a diverged general-tier skill verbatim from the embedded template
// (byte-identical), exit 0, and bumps synced-through to the embedded ref.
func TestSyncRestoresStaleSkill(t *testing.T) {
	dir := scaffoldFresh(t)
	writeRaw(t, skillPath(dir, "dispatch"), "STALE — replace me on sync\n")

	r := runIn(t, dir, "--json", "sync")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	wantListContains(t, obj, "updated", "dispatch")

	// Restored byte-identical to the embedded template (compared via an
	// untouched reference scaffold of the same binary).
	ref := scaffoldFresh(t)
	wantSameFile(t, skillPath(dir, "dispatch"), skillPath(ref, "dispatch"))

	// synced-through bumped to the embedded ref.
	if to, _ := obj["to_ref"].(string); to != templateRef(t) {
		t.Fatalf("sync to_ref = %q, want embedded template_ref %q", obj["to_ref"], templateRef(t))
	}
	if fw := readAll(t, frameworkPath(dir)); !strings.Contains(fw, templateRef(t)) {
		t.Fatalf("framework.md not stamped with embedded ref after sync\n%s", fw)
	}
}

// TestSyncPreservesAutonomousAdaptation covers the sanctioned-region guarantee
// (spec §4.5): a derived autonomous "## Proceed freely" edit survives a sync
// (never clobbered), exit 0, no conflict.
func TestSyncPreservesAutonomousAdaptation(t *testing.T) {
	dir := scaffoldFresh(t)
	marker := "- Deploy the widget service via ./tools/deploy.sh without asking."
	setProceedList(t, skillPath(dir, "autonomous"), []string{marker})

	r := runIn(t, dir, "--json", "sync")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)

	if conflicts := jsonStrings(obj["conflicts"]); len(conflicts) != 0 {
		t.Fatalf("sync reported a conflict on a sanctioned adaptation: %v", conflicts)
	}
	if got := readAll(t, skillPath(dir, "autonomous")); !strings.Contains(got, marker) {
		t.Fatalf("sync clobbered the derived proceed-list adaptation\n%s", got)
	}
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
