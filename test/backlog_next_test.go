package e2e_test

import (
	"strings"
	"testing"
)

// TestNextPicksReady covers `backlog next` selecting a ready item without
// claiming it. The item stays triaged (not in-progress).
func TestNextPicksReady(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "pick-me", readyFields("Pick me", "cli"), "body")

	r := runIn(t, root, "--json", "backlog", "next")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["id"] != key(1) {
		t.Fatalf("next id = %v, want %s", obj["id"], key(1))
	}

	// next must not claim: status remains approved.
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "pick-me"))
	if fm["status"] != "triaged" {
		t.Fatalf("next must not mutate status, got %q", fm["status"])
	}
}

// TestNextEmptyIsNull covers the spec §3.5 judgment: empty ready set exits 0
// with a JSON null (a sweep loop terminates cleanly, not an error).
func TestNextEmptyIsNull(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "--json", "backlog", "next")
	wantExit(t, r, 0)
	if strings.TrimSpace(r.stdout) != "null" {
		t.Fatalf("empty next --json should be null, got %q", r.stdout)
	}
}

// TestNextEmptyHuman covers the human empty case: "no ready items", exit 0.
func TestNextEmptyHuman(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "next")
	wantExit(t, r, 0)
	wantContains(t, r.stderr, "no ready items", "next empty stderr")
}

// TestNextUnknownWorkstream covers exit 4 for an unknown --workstream.
func TestNextUnknownWorkstream(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "next", "--workstream", "ghoststream")
	wantExit(t, r, 4)
}
