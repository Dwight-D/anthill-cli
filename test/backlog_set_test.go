package e2e_test

import (
	"path/filepath"
	"testing"
)

// TestSetMutateInPlace covers `backlog set` updating a free key in place.
func TestSetMutateInPlace(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "some-item", readyFields("Some item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", key(1), "priority=high"), 0)
	if fm := readFrontmatter(t, itemPath(root, "cli", 1, "some-item")); fm["priority"] != "high" {
		t.Fatalf("priority = %q, want high", fm["priority"])
	}
}

// TestSetFreeFields covers every free key in one atomic write (§6.5).
func TestSetFreeFields(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "free", readyFields("Free", "cli"), "body")
	r := runIn(t, root, "backlog", "set", key(1), "summary=new summary", "note=a note", "priority=low",
		"disposition=design", "change-type=doc", "risk=reversible", "verify=none", "source=chat")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "free"))
	for k, v := range map[string]string{"summary": "new summary", "note": "a note", "priority": "low",
		"disposition": "design", "change-type": "doc", "risk": "reversible", "verify": "none", "source": "chat"} {
		if fm[k] != v {
			t.Fatalf("%s = %q, want %q", k, fm[k], v)
		}
	}
}

// TestSetWorkstreamMovesFile covers reassigning a triaged item: the file moves
// into backlog/<ws>/ with id/filename unchanged, and the move is logged.
func TestSetWorkstreamMovesFile(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "move-me", readyFields("Move me", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", key(1), "workstream=dev"), 0)
	wantFileGone(t, itemPath(root, "cli", 1, "move-me"))
	moved := itemPath(root, "dev", 1, "move-me")
	if fm := readFrontmatter(t, moved); fm["workstream"] != "dev" {
		t.Fatalf("workstream = %q, want dev", fm["workstream"])
	}
	wantContains(t, readAll(t, moved), "move cli → dev", "Log after reassign")
}

// TestSetWorkstreamOnIntakeRefused covers exit 6: an untriaged item gets its
// workstream only through triage.
func TestSetWorkstreamOnIntakeRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", 1, "raw", newFields("Raw"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", key(1), "workstream=cli"), 6)
	wantFilePresent(t, itemPath(root, "intake", 1, "raw"))
}

// TestSetTitleEditKeepsId confirms a title edit does NOT change the id (the
// number): the slug tail follows the title, the item stays number 1.
func TestSetTitleEditKeepsId(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "original-id", readyFields("Original title", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "set", key(1), "title=Completely different now")
	wantExit(t, r, 0)
	if obj := jsonObj(t, r.stdout); obj["id"] != key(1) {
		t.Fatalf("id after title edit = %v, want %s", obj["id"], key(1))
	}
	path := findItemPath(t, root, "cli", 1)
	if path == "" {
		t.Fatal("item 1 vanished from cli/ after a title edit")
	}
	if fm := readFrontmatter(t, path); fm["title"] != "Completely different now" {
		t.Fatalf("title = %q, want the edited value", fm["title"])
	}
	if findItemPath(t, root, "cli", 2) != "" {
		t.Fatal("title edit must not allocate a new number")
	}
}

// TestSetTitleRenamesFile covers `set title=`: the file is renamed to
// NNNN-<new-slug>.md in place (same number, same dir), the frontmatter title
// is updated, and the rename is logged.
func TestSetTitleRenamesFile(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", 1, "old-title", newFields("Old title"), "body")
	oldPath := itemPath(root, "intake", 1, "old-title")
	wantExit(t, runIn(t, root, "backlog", "set", key(1), "title=Brand new title"), 0)

	path := findItemPath(t, root, "intake", 1)
	if filepath.Base(path) != "0001-brand-new-title.md" {
		t.Fatalf("after title edit item 1 is at %q, want 0001-brand-new-title.md", path)
	}
	wantFileGone(t, oldPath)
	if fm := readFrontmatter(t, path); fm["title"] != "Brand new title" {
		t.Fatalf("title = %q, want Brand new title", fm["title"])
	}
	body := readAll(t, path)
	wantContains(t, body, "## Log", "renamed item body")
	wantContains(t, body, "rename 0001-old-title.md → 0001-brand-new-title.md", "Log after rename")
}

// TestSetValueEditable confirms value is editable via set.
func TestSetValueEditable(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "val-item", readyFields("Val item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", key(1), "value=a brand new value"), 0)
	if fm := readFrontmatter(t, itemPath(root, "cli", 1, "val-item")); fm["value"] != "a brand new value" {
		t.Fatalf("value = %q, want edited", fm["value"])
	}
}

// TestSetStatusAndDispatchRefused covers §2: status and dispatch move only
// through their verbs; `set status=` / `set dispatch=` exit 3, file untouched.
func TestSetStatusAndDispatchRefused(t *testing.T) {
	root := mkTree(t)
	f := readyFields("Gate item", "cli")
	f["dispatch"] = "proposed"
	writeItem(t, root, "cli", 1, "gate-item", f, "body")
	for _, pair := range []string{"status=in-progress", "status=done", "status=triaged", "dispatch=approved", "dispatch=fast-tracked"} {
		wantExit(t, runIn(t, root, "backlog", "set", key(1), pair), 3)
	}
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "gate-item"))
	if fm["status"] != "triaged" || fm["dispatch"] != "proposed" {
		t.Fatalf("refused set changed the file: %v", fm)
	}
}

// TestSetCLIOwnedKeysRefused covers exit 3 for stamps, claim fields, reporter,
// and keys outside the schema.
func TestSetCLIOwnedKeysRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "owned", readyFields("Owned", "cli"), "body")
	for _, pair := range []string{"id=" + key(9), "former-id=x", "created-at=x", "claimed-by=x", "session=x", "reporter=worker", "value-verdict=ADVANCE"} {
		wantExit(t, runIn(t, root, "backlog", "set", key(1), pair), 3)
	}
	wantFilePresent(t, itemPath(root, "cli", 1, "owned"))
	if findItemPath(t, root, "cli", 9) != "" {
		t.Fatal("refused set id= must not create item 9")
	}
}

// TestSetIllegalEnum covers exit 3 for an illegal enum value, file untouched.
func TestSetIllegalEnum(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "enum-item", readyFields("Enum item", "cli"), "body")
	for _, pair := range []string{"risk=bogus-value", "priority=urgent", "disposition=AUTO"} {
		wantExit(t, runIn(t, root, "backlog", "set", key(1), pair), 3)
	}
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "enum-item"))
	if fm["risk"] != "additive" || fm["priority"] != "normal" || fm["disposition"] != "auto" {
		t.Fatalf("rejected write must leave file untouched: %v", fm)
	}
}

// TestSetUnknownWorkstream covers exit 3 for a non-existent workstream dir.
func TestSetUnknownWorkstream(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "ws-item", readyFields("Ws item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", key(1), "workstream=ghoststream"), 3)
	wantFilePresent(t, itemPath(root, "cli", 1, "ws-item"))
}

// TestSetNotFound covers exit 4 when the id does not exist (key, bare number,
// and slug-shaped former-id lookups alike).
func TestSetNotFound(t *testing.T) {
	root := mkTree(t)
	wantExit(t, runIn(t, root, "backlog", "set", key(99), "priority=high"), 4)
	wantExit(t, runIn(t, root, "backlog", "set", "99", "priority=high"), 4)
	wantExit(t, runIn(t, root, "backlog", "set", "no-such-item", "priority=high"), 4)
}

// TestSetUsage covers exit 2 for no pair and a malformed pair.
func TestSetUsage(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "u", readyFields("U", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", key(1)), 2)
	wantExit(t, runIn(t, root, "backlog", "set", key(1), "priorityhigh"), 2)
}

// TestSetJSON confirms set returns the updated item object under --json, with
// the canonical key as id even when given the bare number.
func TestSetJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "json-set-item", readyFields("Json set item", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "set", "1", "priority=high")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["priority"] != "high" || obj["id"] != key(1) {
		t.Fatalf("json = %v", obj)
	}
}
