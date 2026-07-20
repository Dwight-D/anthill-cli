package e2e_test

import (
	"os"
	"strings"
	"testing"
)

// TestBootstrapPrintsEntrypoint covers `anthill bootstrap` (spec §4.1): a pure
// redirect that prints the canonical BOOTSTRAP.md URL plus an agent-directed
// preamble, exit 0, in any directory.
func TestBootstrapPrintsEntrypoint(t *testing.T) {
	// Run in a bare (non-git) temp dir: bootstrap is documented safe anywhere,
	// in or out of a repo.
	dir := t.TempDir()
	r := runInDir(t, dir, "bootstrap")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "BOOTSTRAP.md", "bootstrap stdout")
}

// TestBootstrapJSON covers the --json shape: { entrypoint, preamble } with the
// entrypoint pointing at BOOTSTRAP.md and a non-empty preamble (spec §4.1).
func TestBootstrapJSON(t *testing.T) {
	dir := t.TempDir()
	r := runInDir(t, dir, "--json", "bootstrap")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)

	entrypoint, _ := obj["entrypoint"].(string)
	if !strings.Contains(entrypoint, "BOOTSTRAP.md") {
		t.Fatalf("entrypoint = %q, want it to contain BOOTSTRAP.md", entrypoint)
	}
	preamble, ok := obj["preamble"].(string)
	if !ok || strings.TrimSpace(preamble) == "" {
		t.Fatalf("preamble missing or empty: %v", obj["preamble"])
	}
}

// TestBootstrapNoSideEffects covers the zero-side-effects guarantee (spec §4.1):
// bootstrap runs fine outside a git repo and creates nothing on disk.
func TestBootstrapNoSideEffects(t *testing.T) {
	dir := t.TempDir()
	r := runInDir(t, dir, "bootstrap")
	wantExit(t, r, 0)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir %s: %v", dir, err)
	}
	if len(entries) != 0 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("bootstrap created files in a clean dir: %v", names)
	}
}
