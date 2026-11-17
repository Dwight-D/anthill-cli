package e2e_test

import "testing"

// TestSetMutateInPlace covers `backlog set` updating a free key in place.
func TestSetMutateInPlace(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "some-item", readyFields("Some item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", "some-item", "priority=high"), 0)
	if fm := readFrontmatter(t, itemPath(root, "cli", "some-item")); fm["priority"] != "high" {
		t.Fatalf("priority = %q, want high", fm["priority"])
	}
}

// TestSetFreeFields covers every free key in one atomic write (§6.5).
func TestSetFreeFields(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "free", readyFields("Free", "cli"), "body")
	r := runIn(t, root, "backlog", "set", "free", "summary=new summary", "note=a note", "priority=low",
		"disposition=design", "change-type=doc", "risk=reversible", "verify=none", "source=chat")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", "free"))
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
	writeItem(t, root, "cli", "move-me", readyFields("Move me", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", "move-me", "workstream=dev"), 0)
	wantFileGone(t, itemPath(root, "cli", "move-me"))
	moved := itemPath(root, "dev", "move-me")
	if fm := readFrontmatter(t, moved); fm["workstream"] != "dev" {
		t.Fatalf("workstream = %q, want dev", fm["workstream"])
	}
	wantContains(t, readAll(t, moved), "move cli → dev", "Log after reassign")
}

// TestSetWorkstreamOnIntakeRefused covers exit 6: an untriaged item gets its
// workstream only through triage.
func TestSetWorkstreamOnIntakeRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", "raw", newFields("Raw"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", "raw", "workstream=cli"), 6)
	wantFilePresent(t, itemPath(root, "intake", "raw"))
}

// TestSetTitleEditKeepsId confirms a title edit does NOT change the id/filename.
func TestSetTitleEditKeepsId(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "original-id", readyFields("Original title", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", "original-id", "title=Completely different now"), 0)
	path := itemPath(root, "cli", "original-id")
	if fm := readFrontmatter(t, path); fm["title"] != "Completely different now" {
		t.Fatalf("title = %q, want the edited value", fm["title"])
	}
	wantFileGone(t, itemPath(root, "cli", "completely-different-now"))
}

// TestSetValueEditable confirms value is editable via set.
func TestSetValueEditable(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "val-item", readyFields("Val item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", "val-item", "value=a brand new value"), 0)
	if fm := readFrontmatter(t, itemPath(root, "cli", "val-item")); fm["value"] != "a brand new value" {
		t.Fatalf("value = %q, want edited", fm["value"])
	}
}

// TestSetStatusAndDispatchRefused covers §2: status and dispatch move only
// through their verbs; `set status=` / `set dispatch=` exit 3, file untouched.
func TestSetStatusAndDispatchRefused(t *testing.T) {
	root := mkTree(t)
	f := readyFields("Gate item", "cli")
	f["dispatch"] = "proposed"
	writeItem(t, root, "cli", "gate-item", f, "body")
	for _, pair := range []string{"status=in-progress", "status=done", "status=triaged", "dispatch=approved", "dispatch=fast-tracked"} {
		wantExit(t, runIn(t, root, "backlog", "set", "gate-item", pair), 3)
	}
	fm := readFrontmatter(t, itemPath(root, "cli", "gate-item"))
	if fm["status"] != "triaged" || fm["dispatch"] != "proposed" {
		t.Fatalf("refused set changed the file: %v", fm)
	}
}

// TestSetCLIOwnedKeysRefused covers exit 3 for stamps, claim fields, reporter,
// and keys outside the schema.
func TestSetCLIOwnedKeysRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "owned", readyFields("Owned", "cli"), "body")
	for _, pair := range []string{"id=new-id", "created-at=x", "claimed-by=x", "session=x", "reporter=worker", "value-verdict=ADVANCE"} {
		wantExit(t, runIn(t, root, "backlog", "set", "owned", pair), 3)
	}
	wantFilePresent(t, itemPath(root, "cli", "owned"))
	wantFileGone(t, itemPath(root, "cli", "new-id"))
}

// TestSetIllegalEnum covers exit 3 for an illegal enum value, file untouched.
func TestSetIllegalEnum(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "enum-item", readyFields("Enum item", "cli"), "body")
	for _, pair := range []string{"risk=bogus-value", "priority=urgent", "disposition=AUTO"} {
		wantExit(t, runIn(t, root, "backlog", "set", "enum-item", pair), 3)
	}
	fm := readFrontmatter(t, itemPath(root, "cli", "enum-item"))
	if fm["risk"] != "additive" || fm["priority"] != "normal" || fm["disposition"] != "auto" {
		t.Fatalf("rejected write must leave file untouched: %v", fm)
	}
}

// TestSetUnknownWorkstream covers exit 3 for a non-existent workstream dir.
func TestSetUnknownWorkstream(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "ws-item", readyFields("Ws item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", "ws-item", "workstream=ghoststream"), 3)
	wantFilePresent(t, itemPath(root, "cli", "ws-item"))
}

// TestSetNotFound covers exit 4 when the id does not exist.
func TestSetNotFound(t *testing.T) {
	root := mkTree(t)
	wantExit(t, runIn(t, root, "backlog", "set", "no-such-item", "priority=high"), 4)
}

// TestSetUsage covers exit 2 for no pair and a malformed pair.
func TestSetUsage(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "u", readyFields("U", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "set", "u"), 2)
	wantExit(t, runIn(t, root, "backlog", "set", "u", "priorityhigh"), 2)
}

// TestSetJSON confirms set returns the updated item object under --json.
func TestSetJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "json-set-item", readyFields("Json set item", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "set", "json-set-item", "priority=high")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["priority"] != "high" || obj["id"] != "json-set-item" {
		t.Fatalf("json = %v", obj)
	}
}
