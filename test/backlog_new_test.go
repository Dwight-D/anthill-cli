package e2e_test

import (
	"strings"
	"testing"
)

// TestNewPlain covers `backlog new` (spec §3.1): creates intake/<id>.md with
// status idea and no workstream; prints the id on stdout, "created" on stderr.
func TestNewPlain(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--title", "Fix the thing", "--value", "removes pain")
	wantExit(t, r, 0)

	id := strings.TrimSpace(r.stdout)
	if id != "fix-the-thing" {
		t.Fatalf("stdout id = %q, want %q", id, "fix-the-thing")
	}
	wantContains(t, r.stderr, "created", "new stderr")

	path := itemPath(root, "intake", "fix-the-thing")
	wantFilePresent(t, path)
	fm := readFrontmatter(t, path)
	if fm["status"] != "idea" {
		t.Fatalf("status = %q, want idea", fm["status"])
	}
	if _, ok := fm["workstream"]; ok {
		t.Fatalf("intake item must not carry a workstream key, got %q", fm["workstream"])
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
	if obj["id"] != "add-a-widget" {
		t.Fatalf("json id = %v, want add-a-widget", obj["id"])
	}
	if obj["status"] != "idea" {
		t.Fatalf("json status = %v, want idea", obj["status"])
	}
	if obj["title"] != "Add a widget" {
		t.Fatalf("json title = %v, want %q", obj["title"], "Add a widget")
	}
	if _, ok := obj["path"]; !ok {
		t.Fatalf("json object missing path field: %v", obj)
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
	fm := readFrontmatter(t, itemPath(root, "intake", "hinted-item"))
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
	fm := readFrontmatter(t, itemPath(root, "intake", "aliased-item"))
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
	fm := readFrontmatter(t, itemPath(root, "intake", "sourced-item"))
	if fm["source"] != "code review" {
		t.Fatalf("source = %q, want %q", fm["source"], "code review")
	}
}
