package e2e_test

import (
	"strings"
	"testing"
)

// TestNewPlain covers `backlog new` (§6.1): creates intake/NNNN-<slug>.md
// with status new, dispatch proposed, created-at, reporter, the body
// skeleton, and no workstream; prints the key on stdout, "created" on stderr.
func TestNewPlain(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--title", "Fix the thing", "--value", "removes pain")
	wantExit(t, r, 0)

	id := strings.TrimSpace(r.stdout)
	if id != key(1) {
		t.Fatalf("stdout id = %q, want %q", id, key(1))
	}
	wantContains(t, r.stderr, "created "+key(1), "new stderr")

	path := itemPath(root, "intake", 1, "fix-the-thing")
	wantFilePresent(t, path)
	fm := readFrontmatter(t, path)
	if fm["status"] != "new" || fm["dispatch"] != "proposed" || fm["priority"] != "normal" ||
		fm["reporter"] != "user" || fm["created-at"] == "" {
		t.Fatalf("new item frontmatter = %v", fm)
	}
	body := readAll(t, path)
	for _, h := range []string{"## Description", "## Acceptance", "## Triage", "## Evidence", "## Log"} {
		wantContains(t, body, h, "new item body skeleton")
	}
	if _, ok := fm["workstream"]; ok {
		t.Fatalf("intake item must not carry a workstream key, got %q", fm["workstream"])
	}
	if _, ok := fm["former-id"]; ok {
		t.Fatalf("new item must not carry former-id, got %q", fm["former-id"])
	}
	if fm["title"] != "Fix the thing" {
		t.Fatalf("title = %q, want %q", fm["title"], "Fix the thing")
	}
}

// TestNewJSON covers the --json object shape for `backlog new`.
func TestNewJSON(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "--json", "backlog", "new", "--title", "Add a widget", "--value", "unlocks X")
	wantExit(t, r, 0)

	obj := jsonObj(t, r.stdout)
	if obj["id"] != key(1) {
		t.Fatalf("json id = %v, want %s", obj["id"], key(1))
	}
	if obj["status"] != "new" || obj["dispatch"] != "proposed" {
		t.Fatalf("json status/dispatch = %v/%v, want new/proposed", obj["status"], obj["dispatch"])
	}
	if obj["title"] != "Add a widget" {
		t.Fatalf("json title = %v, want %q", obj["title"], "Add a widget")
	}
	if p, ok := obj["path"].(string); !ok || !strings.HasSuffix(p, "0001-add-a-widget.md") {
		t.Fatalf("json path = %v, want …/0001-add-a-widget.md", obj["path"])
	}
}

// TestNewMissingTitle covers exit 2 when --title is absent (spec §3.1).
func TestNewMissingTitle(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--value", "v")
	wantExit(t, r, 2)
}

// TestNewMissingValue covers exit 2 when --value is absent.
func TestNewMissingValue(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--title", "t")
	wantExit(t, r, 2)
}

// TestNewHintFlag covers --hint recording the hint field (spec §3.1, §7 Q6).
func TestNewHintFlag(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--title", "Hinted item", "--value", "v", "--hint", "cli")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "intake", 1, "hinted-item"))
	if fm["hint"] != "cli" {
		t.Fatalf("hint = %q, want cli", fm["hint"])
	}
}

// TestNewBacklogAlias covers the hidden --backlog alias setting the same field
// as --hint (spec §7 Q6: "Support both with the hidden alias").
func TestNewBacklogAlias(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--title", "Aliased item", "--value", "v", "--backlog", "dev")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "intake", 1, "aliased-item"))
	if fm["hint"] != "dev" {
		t.Fatalf("--backlog alias set hint = %q, want dev", fm["hint"])
	}
}

// TestNewBacklogAliasHidden confirms --backlog is a HIDDEN alias: it works but
// is not advertised in help (spec §7 Q6).
func TestNewBacklogAliasHidden(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--help")
	wantExit(t, r, 0)
	help := r.stdout + r.stderr
	if strings.Contains(help, "--backlog") {
		t.Fatalf("--backlog should be a hidden alias, but appears in help:\n%s", help)
	}
	wantContains(t, help, "--hint", "new --help")
}

// TestNewSource covers the optional --source field.
func TestNewSource(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--title", "Sourced item", "--value", "v", "--source", "code review")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "intake", 1, "sourced-item"))
	if fm["source"] != "code review" {
		t.Fatalf("source = %q, want %q", fm["source"], "code review")
	}
}

// TestNewStampsEmptyTree covers the first `new` on a tree whose workstreams.md
// has no schema line at all (scaffold output before the template carried one):
// both schema: 3 and a key-prefix derived from the root dir name are stamped.
func TestNewStampsEmptyTree(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, backlogDir(root)+"/workstreams.md", "---\nsweep-order: bugs, cli, dev, process\n---\n")
	r := runIn(t, root, "backlog", "new", "--title", "First", "--value", "v")
	wantExit(t, r, 0)
	id := strings.TrimSpace(r.stdout)
	if !strings.HasSuffix(id, "-1") {
		t.Fatalf("first key = %q, want <derived prefix>-1", id)
	}
	ws := readAll(t, backlogDir(root)+"/workstreams.md")
	wantContains(t, ws, "schema: 3", "schema stamped")
	wantContains(t, ws, "key-prefix: "+strings.TrimSuffix(id, "-1"), "prefix stamped")
}
