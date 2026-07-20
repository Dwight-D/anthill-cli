package e2e_test

import (
	"strings"
	"testing"
)

// newID creates an item via `backlog new` and returns its generated id.
func newID(t *testing.T, root, title string) string {
	t.Helper()
	r := runIn(t, root, "backlog", "new", "--title", title, "--value", "v")
	wantExit(t, r, 0)
	return strings.TrimSpace(r.stdout)
}

// TestIdKebabSlug covers id generation from a title: lowercase, runs of
// non-alphanumeric collapse to a single hyphen, no leading/trailing hyphen
// (spec §4 id generation).
func TestIdKebabSlug(t *testing.T) {
	root := mkTree(t)
	cases := []struct {
		title string
		want  string
	}{
		{"Fix the Thing!", "fix-the-thing"},
		{"  Weird   spacing  ", "weird-spacing"},
		{"CamelCase Words", "camelcase-words"},
		{"punctuation: a/b, c.d", "punctuation-a-b-c-d"},
		{"trailing symbols ***", "trailing-symbols"},
	}
	for _, c := range cases {
		got := newID(t, root, c.title)
		if got != c.want {
			t.Errorf("slug(%q) = %q, want %q", c.title, got, c.want)
		}
	}
}

// TestIdCollisionSuffix covers numeric-suffix disambiguation: a second item
// whose title slugs to an already-used id gets -2 (spec §4 collision).
func TestIdCollisionSuffix(t *testing.T) {
	root := mkTree(t)
	first := newID(t, root, "Duplicate title")
	if first != "duplicate-title" {
		t.Fatalf("first id = %q, want duplicate-title", first)
	}
	second := newID(t, root, "Duplicate title")
	if second != "duplicate-title-2" {
		t.Fatalf("second id = %q, want duplicate-title-2", second)
	}
	third := newID(t, root, "Duplicate title")
	if third != "duplicate-title-3" {
		t.Fatalf("third id = %q, want duplicate-title-3", third)
	}

	wantFilePresent(t, itemPath(root, "intake", "duplicate-title"))
	wantFilePresent(t, itemPath(root, "intake", "duplicate-title-2"))
	wantFilePresent(t, itemPath(root, "intake", "duplicate-title-3"))
}

// TestIdCollisionAcrossDirs covers collision detection across intake +
// workstream dirs, not just within one directory (spec §4).
func TestIdCollisionAcrossDirs(t *testing.T) {
	root := mkTree(t)
	// Pre-seed a workstream item with the id the new title would slug to.
	writeItem(t, root, "cli", "shared-slug", approvedFields("Shared slug", "cli"), "body")

	got := newID(t, root, "Shared slug")
	if got != "shared-slug-2" {
		t.Fatalf("id colliding with a workstream item = %q, want shared-slug-2", got)
	}
}

// TestIdTruncation covers the ≤50-char budget with no trailing hyphen (spec §4).
func TestIdTruncation(t *testing.T) {
	root := mkTree(t)
	long := "This is an extremely long title that goes well beyond the fifty character id budget for slugs"
	got := newID(t, root, long)
	if len(got) > 50 {
		t.Fatalf("id length = %d (%q), want <= 50", len(got), got)
	}
	if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") {
		t.Fatalf("id must not have leading/trailing hyphen, got %q", got)
	}
}

// TestNewEmptySlugValidation covers exit 3 when a title slugifies to empty
// (spec §3.1: title slugifies to empty → validation).
func TestNewEmptySlugValidation(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--title", "***", "--value", "v")
	wantExit(t, r, 3)
}
