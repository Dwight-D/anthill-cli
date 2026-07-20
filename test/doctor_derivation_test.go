package e2e_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestDoctorGreenOnDerivedInstall is the end-to-end regression for the bug where
// a fully derived (valid) install failed `doctor` because the derivation-status
// check flagged ANY <angle-bracket> token — including legitimate format
// templates that survive derivation (record filenames like
// <yyyy-mm-dd>-<slug>.md, schema field descriptions, the worker-brief skeleton).
//
// Flow, exactly as specified: scaffold into an empty git repo, replace the
// derive-target files with valid marker-free content, then run `doctor --strict`
// and confirm it is green — even though permanent-format files still contain
// legitimate <...> tokens.
func TestDoctorGreenOnDerivedInstall(t *testing.T) {
	dir := scaffoldFresh(t)
	ref := templateRef(t)

	// Precondition: a fresh scaffold is un-derived, so --strict must fail on the
	// remaining template quote-blocks.
	wantNonZero(t, runInDir(t, dir, "doctor", "--strict"))

	// Replace the seven derive-target files (the ones carrying template
	// quote-blocks) with valid, marker-free derived content. framework.md keeps a
	// parseable synced-through at the embedded ref (sync-status), and
	// workstreams.md keeps a valid sweep-order naming existing dirs (Section B).
	writeRaw(t, filepath.Join(dir, ".anthill", "backlog", "workstreams.md"),
		"---\nsweep-order: bugs, product, dev, process\nnever-implicit:\n---\n\n# Backlog workstreams\n\nDerived for the test project.\n")
	writeRaw(t, filepath.Join(dir, ".anthill", "autonomy.md"),
		"# Autonomy contract config\n\n## Proceed freely (do not ask permission)\n\n- Build and test the project.\n\n## Decisions log\n\n- **Path:** `.anthill/decisions.md`\n")
	writeRaw(t, filepath.Join(dir, ".anthill", "backlog", "bindings.md"),
		"# Backlog bindings\n\nConvention-first schema owner. Derived.\n")
	writeRaw(t, filepath.Join(dir, ".anthill", "resources.md"),
		"# Exclusive resources\n\nGit checkout only; worker cap 2, serial dispatch.\n")
	writeRaw(t, filepath.Join(dir, ".anthill", "supervisor", "bindings.md"),
		"# Supervisor bindings\n\nWorker cap 2, serial dispatch. Derived.\n")
	writeRaw(t, filepath.Join(dir, ".anthill", "supervisor", "agenda.md"),
		"# Agenda\n\n## Standing goals\n- Ship the product.\n")
	writeRaw(t, filepath.Join(dir, ".anthill", "framework.md"),
		"# Anthill framework — provenance & sync state\n\n- **synced-through:** "+ref+" (installed 2026-07-20)\n")

	// A permanent-format file still legitimately holds <...>; the fix must NOT
	// treat that as un-derived.
	esc := readAll(t, filepath.Join(dir, ".anthill", "escalations", "README.md"))
	if !strings.Contains(esc, "<yyyy-mm-dd>") {
		t.Fatalf("precondition: escalations/README.md should retain its <yyyy-mm-dd> record template")
	}

	// The derived install must now pass doctor --strict, green across both sections.
	r := runInDir(t, dir, "--json", "doctor", "--strict")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if ok, _ := obj["ok"].(bool); !ok {
		t.Fatalf("doctor --strict not ok on a valid derived install:\n%s", r.stdout)
	}
}
