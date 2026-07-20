package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMergeGitignore covers the append-merge that lets scaffold coexist with an
// install target's pre-existing .gitignore: the consumer's rules are preserved,
// the framework block is added under sentinel markers, and a second merge is a
// no-op (idempotent via the marker).
func TestMergeGitignore(t *testing.T) {
	tmpl := []byte("# OS cruft\n.DS_Store\n*.swp\n")

	// Absent/empty target → just the wrapped block.
	fresh := mergeGitignore(nil, tmpl)
	if !strings.Contains(string(fresh), gitignoreMarkerStart) || !strings.Contains(string(fresh), ".DS_Store") {
		t.Fatalf("merge into empty did not produce the framework block:\n%s", fresh)
	}

	// Existing rules preserved AND framework block appended.
	existing := []byte("/node_modules\n*.log\n")
	merged := mergeGitignore(existing, tmpl)
	for _, want := range []string{"/node_modules", "*.log", gitignoreMarkerStart, ".DS_Store", gitignoreMarkerEnd} {
		if !strings.Contains(string(merged), want) {
			t.Fatalf("merged .gitignore missing %q:\n%s", want, merged)
		}
	}

	// Idempotent: merging again (block already present) returns it unchanged.
	if again := mergeGitignore(merged, tmpl); string(again) != string(merged) {
		t.Fatalf("second merge was not a no-op:\n%s", again)
	}
}

// TestScaffoldPartialApply covers the per-file (non-terminal) refusal: a single
// differing file is refused while every other payload file is still written, and
// the result lists reflect exactly what landed on disk.
func TestScaffoldPartialApply(t *testing.T) {
	dir := t.TempDir()
	// Plant a differing file that must be refused.
	derivedRel := filepath.FromSlash(".anthill/backlog/workstreams.md")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, derivedRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, derivedRel), []byte("derived\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Scaffold(dir, false, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Refused) != 1 || res.Refused[0] != ".anthill/backlog/workstreams.md" {
		t.Fatalf("want exactly the derived file refused, got %v", res.Refused)
	}
	if len(res.Written) == 0 {
		t.Fatalf("partial apply wrote nothing; expected the safe payload to install")
	}
	// Every reported-written path must actually exist on disk (the false-report bug).
	for _, p := range res.Written {
		if _, statErr := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); statErr != nil {
			t.Fatalf("reported %q as written but it is not on disk: %v", p, statErr)
		}
	}
	// The refused file is untouched.
	got, _ := os.ReadFile(filepath.Join(dir, derivedRel))
	if string(got) != "derived\n" {
		t.Fatalf("refused file was modified: %q", got)
	}
}
