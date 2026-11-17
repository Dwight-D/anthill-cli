package e2e_test

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// newID creates an item via `backlog new` and returns its key (stdout).
func newID(t *testing.T, root, title string) string {
	t.Helper()
	r := runIn(t, root, "backlog", "new", "--title", title, "--value", "v")
	wantExit(t, r, 0)
	return strings.TrimSpace(r.stdout)
}

// TestIdSequentialKeys covers allocation (ITEM_KEYS_SPEC.md §2.2): keys are
// PREFIX-N in creation order, the filename is NNNN-<slug>.md with the title's
// kebab slug (≤ 40 chars; "item" when empty), and a removed number is never
// reused because `sequence` remembers the high-water mark.
func TestIdSequentialKeys(t *testing.T) {
	root := mkTree(t)
	if got := newID(t, root, "Fix the Thing!"); got != key(1) {
		t.Fatalf("first key = %q, want %s", got, key(1))
	}
	if got := newID(t, root, "  Weird   spacing  "); got != key(2) {
		t.Fatalf("second key = %q, want %s", got, key(2))
	}
	wantFilePresent(t, itemPath(root, "intake", 1, "fix-the-thing"))
	wantFilePresent(t, itemPath(root, "intake", 2, "weird-spacing"))
	if got := strings.TrimSpace(readAll(t, filepath.Join(backlogDir(root), "sequence"))); got != "2" {
		t.Fatalf("sequence = %q, want 2", got)
	}

	wantExit(t, runIn(t, root, "backlog", "close", "--remove", key(2)), 0)
	wantFileGone(t, itemPath(root, "intake", 2, "weird-spacing"))
	if got := newID(t, root, "***"); got != key(3) {
		t.Fatalf("key after remove = %q, want %s (numbers are never reused)", got, key(3))
	}
	wantFilePresent(t, itemPath(root, "intake", 3, "item"))

	long := "This is an extremely long title that goes well beyond the forty character slug budget"
	newID(t, root, long)
	p := findItemPath(t, root, "intake", 4)
	slug := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(p), "0004-"), ".md")
	if len(slug) > 40 || strings.HasSuffix(slug, "-") || slug == "" {
		t.Fatalf("slug tail = %q, want <= 40 chars with no trailing hyphen", slug)
	}
}

// TestIdAllocationSpansDirs covers step 1 of §2.2: the next number is one
// above the highest in ANY item directory (archives included), and a
// sequence file above them all wins.
func TestIdAllocationSpansDirs(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 7, "in-cli", readyFields("In cli", "cli"), "")
	writeAtStatus(t, root, 12, "archived", "done")
	if got := newID(t, root, "Next"); got != key(13) {
		t.Fatalf("key = %q, want %s", got, key(13))
	}
	writeRaw(t, filepath.Join(backlogDir(root), "sequence"), "40\n")
	if got := newID(t, root, "After sequence"); got != key(41) {
		t.Fatalf("key = %q, want %s", got, key(41))
	}
}

// TestIdResolution covers the accepted input forms (§1): the canonical key,
// case-insensitive, the bare number, and — for a migrated item — the former
// id; every form echoes the canonical key.
func TestIdResolution(t *testing.T) {
	root := mkTree(t)
	fields := newFields("Resolvable")
	fields["former-id"] = "old-slug-name"
	writeItem(t, root, "intake", 2, "resolvable", fields, "")
	for _, ref := range []string{"ANT-2", "ant-2", "Ant-2", "2", "old-slug-name"} {
		r := runIn(t, root, "--json", "backlog", "show", ref)
		wantExit(t, r, 0)
		if obj := jsonObj(t, r.stdout); obj["id"] != "ANT-2" || obj["former-id"] != "old-slug-name" {
			t.Fatalf("show %s → id %v former-id %v", ref, obj["id"], obj["former-id"])
		}
	}
	// Transitions echo the canonical key whatever form was given.
	r := runIn(t, root, "backlog", "park", "2", "later")
	wantExit(t, r, 0)
	if strings.TrimSpace(r.stdout) != "ANT-2" {
		t.Fatalf("park stdout = %q, want ANT-2", r.stdout)
	}
	// Unknown but well-formed references are not found (exit 4).
	for _, ref := range []string{"ANT-9", "9", "never-filed"} {
		wantExit(t, runIn(t, root, "backlog", "show", ref), 4)
	}
	// Malformed references are usage errors (exit 2) and never touch a path.
	for _, ref := range []string{"../x", "ANT-0", "ANT-074", "A-1", "ABCDEF-1", "BH-2", "Foo", "a/b"} {
		r := runIn(t, root, "backlog", "show", ref)
		wantExit(t, r, 2)
		wantContains(t, r.stderr, "invalid item id", "show "+ref)
	}
}

// TestIdParallelAllocation covers the exclusive create of §2.2: eight
// concurrent `new` processes get eight distinct keys and no file is lost.
func TestIdParallelAllocation(t *testing.T) {
	root := mkTree(t)
	const n = 8
	results := make([]result, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = runIn(t, root, "backlog", "new", "--title", "Parallel "+itoa(i), "--value", "v")
		}(i)
	}
	wg.Wait()
	seen := map[string]bool{}
	for i, r := range results {
		if r.exit != 0 {
			t.Fatalf("new #%d exit %d: %s", i, r.exit, r.stderr)
		}
		seen[strings.TrimSpace(r.stdout)] = true
	}
	for i := 1; i <= n; i++ {
		if !seen[key(i)] {
			t.Fatalf("keys allocated = %v, want %s..%s", seen, key(1), key(n))
		}
	}
	matches, _ := filepath.Glob(filepath.Join(backlogDir(root), "intake", "*.md"))
	if len(matches) != n {
		t.Fatalf("%d item files, want %d", len(matches), n)
	}
}

// TestRenumber covers `backlog renumber` (§3.3): the file moves to the new
// number, sequence follows, the sessions ledger and escalation records that
// named the old key are rewritten, a Log line is appended, and a taken
// number is a conflict.
func TestRenumber(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "colliding", claimedFields("Colliding", "cli"), "## Log\n")
	writeItem(t, root, "cli", 2, "neighbour", readyFields("Neighbour", "cli"), "")
	wantExit(t, runIn(t, root, "session", "link", "--session", "sess-1", "--item", "1"), 0)
	r := runIn(t, root, "escalation", "raise", "--to", "user", "--from", "worker", "--item", "1", "--question", "Which number?")
	wantExit(t, r, 0)
	escID := strings.TrimSpace(r.stdout)

	r = runIn(t, root, "backlog", "renumber", "1", "--to", "5")
	wantExit(t, r, 0)
	if strings.TrimSpace(r.stdout) != key(5) {
		t.Fatalf("renumber stdout = %q, want %s", r.stdout, key(5))
	}
	wantFileGone(t, itemPath(root, "cli", 1, "colliding"))
	p := itemPath(root, "cli", 5, "colliding")
	wantFilePresent(t, p)
	wantContains(t, readAll(t, p), "renumber ANT-1 → ANT-5", "Log line")
	if got := strings.TrimSpace(readAll(t, filepath.Join(backlogDir(root), "sequence"))); got != "5" {
		t.Fatalf("sequence = %q, want 5", got)
	}
	wantContains(t, readAll(t, filepath.Join(root, ".anthill", "sessions", "sess-1.json")), `"item": "ANT-5"`, "ledger rewritten")
	wantContains(t, readAll(t, filepath.Join(escalDir(root), escID+".md")), "item: ANT-5\n", "escalation rewritten")
	wantExit(t, runIn(t, root, "backlog", "show", "1"), 4)

	// A taken number conflicts; the default is the next free number.
	wantExit(t, runIn(t, root, "backlog", "renumber", "5", "--to", "2"), 5)
	r = runIn(t, root, "backlog", "renumber", "2")
	wantExit(t, r, 0)
	if strings.TrimSpace(r.stdout) != key(6) {
		t.Fatalf("default renumber = %q, want %s", r.stdout, key(6))
	}
	wantFilePresent(t, itemPath(root, "cli", 6, "neighbour"))
	wantExit(t, runIn(t, root, "backlog", "validate"), 0)
}

// TestNewRefusesUnmigratedTree: a tree holding items under an earlier schema
// must be migrated before new keys are allocated (exit 6), and a template
// tree whose key-prefix is still the placeholder names the fix.
func TestNewRefusesUnmigratedTree(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 2\nsweep-order: bugs, cli, dev, process\n---\n")
	writeRaw(t, filepath.Join(backlogDir(root), "intake", "old-slug.md"), "---\ntitle: Old\nvalue: v\nstatus: new\n---\n")
	r := runIn(t, root, "backlog", "new", "--title", "x", "--value", "v")
	wantExit(t, r, 6)
	wantContains(t, r.stderr, "migrate", "needs migration")

	root = mkTree(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 3\nkey-prefix: <PREFIX>   # 2-5 uppercase letters; anthill init fills it\nsweep-order: bugs, cli, dev, process\n---\n")
	r = runIn(t, root, "backlog", "new", "--title", "x", "--value", "v")
	wantExit(t, r, 6)
	wantContains(t, r.stderr, "key-prefix: XXX", "placeholder counts as missing")
}
