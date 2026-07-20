package e2e_test

import "testing"

// TestSetMutateInPlace covers `backlog set` updating a key in place (spec §3.4).
func TestSetMutateInPlace(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "some-item", approvedFields("Some item", "cli"), "body")
	r := runIn(t, root, "backlog", "set", "some-item", "priority=high")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", "some-item"))
	if fm["priority"] != "high" {
		t.Fatalf("priority = %q, want high", fm["priority"])
	}
}

// TestSetWorkstreamMovesFile covers the file-move semantics: setting
// workstream=<ws> moves the file into backlog/<ws>/ with id/filename unchanged,
// and strips any hint key (spec §3.4).
func TestSetWorkstreamMovesFile(t *testing.T) {
	root := mkTree(t)
	fields := ideaFields("Move me")
	fields["hint"] = "dev"
	writeItem(t, root, "intake", "move-me", fields, "body")

	r := runIn(t, root, "backlog", "set", "move-me", "workstream=cli")
	wantExit(t, r, 0)

	wantFileGone(t, itemPath(root, "intake", "move-me"))
	moved := itemPath(root, "cli", "move-me")
	wantFilePresent(t, moved)

	fm := readFrontmatter(t, moved)
	if fm["workstream"] != "cli" {
		t.Fatalf("workstream = %q, want cli", fm["workstream"])
	}
	if _, ok := fm["hint"]; ok {
		t.Fatalf("hint key must be stripped on triage move, still present: %q", fm["hint"])
	}
}

// TestSetTitleEditKeepsId confirms a title edit is allowed and does NOT change
// the id/filename — id immutability wins, drift accepted (spec §4, §7 Q7).
func TestSetTitleEditKeepsId(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "original-id", approvedFields("Original title", "cli"), "body")

	r := runIn(t, root, "backlog", "set", "original-id", "title=Completely different now")
	wantExit(t, r, 0)

	// File stays at the original id path.
	path := itemPath(root, "cli", "original-id")
	wantFilePresent(t, path)
	fm := readFrontmatter(t, path)
	if fm["title"] != "Completely different now" {
		t.Fatalf("title = %q, want the edited value", fm["title"])
	}
	// No re-slugged file appeared.
	wantFileGone(t, itemPath(root, "cli", "completely-different-now"))
}

// TestSetValueEditable confirms value is editable via set (spec §7 Q7).
func TestSetValueEditable(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "val-item", approvedFields("Val item", "cli"), "body")
	r := runIn(t, root, "backlog", "set", "val-item", "value=a brand new value")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", "val-item"))
	if fm["value"] != "a brand new value" {
		t.Fatalf("value = %q, want edited", fm["value"])
	}
}

// TestSetApprovedRefused is the core §7 Q1 answer: `set status=approved` must be
// REFUSED (approval only via `backlog approve --yes`). The write is rejected
// (non-zero exit) and the file is left untouched at status: idea.
func TestSetApprovedRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "gate-item", func() map[string]string {
		f := approvedFields("Gate item", "cli")
		f["status"] = "idea"
		return f
	}(), "body")

	r := runIn(t, root, "backlog", "set", "gate-item", "status=approved")
	wantNonZero(t, r)

	fm := readFrontmatter(t, itemPath(root, "cli", "gate-item"))
	if fm["status"] == "approved" {
		t.Fatalf("set status=approved must be refused, but status became approved")
	}
}

// TestSetIllegalEnum covers exit 3 for an illegal enum value (validation class).
func TestSetIllegalEnum(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "enum-item", approvedFields("Enum item", "cli"), "body")
	r := runIn(t, root, "backlog", "set", "enum-item", "risk=bogus-value")
	wantExit(t, r, 3)
	// File untouched.
	fm := readFrontmatter(t, itemPath(root, "cli", "enum-item"))
	if fm["risk"] != "additive" {
		t.Fatalf("rejected write must leave file untouched, risk = %q", fm["risk"])
	}
}

// TestSetImmutableId covers exit 3 when trying to set id (immutable key).
func TestSetImmutableId(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "immutable-item", approvedFields("Immutable item", "cli"), "body")
	r := runIn(t, root, "backlog", "set", "immutable-item", "id=new-id")
	wantExit(t, r, 3)
	wantFilePresent(t, itemPath(root, "cli", "immutable-item"))
	wantFileGone(t, itemPath(root, "cli", "new-id"))
}

// TestSetUnknownWorkstream covers exit 3 when workstream= names a non-existent
// workstream directory (validation class).
func TestSetUnknownWorkstream(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", "ws-item", ideaFields("Ws item"), "body")
	r := runIn(t, root, "backlog", "set", "ws-item", "workstream=ghoststream")
	wantExit(t, r, 3)
	wantFilePresent(t, itemPath(root, "intake", "ws-item"))
}

// TestSetNotFound covers exit 4 when the id does not exist.
func TestSetNotFound(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "set", "no-such-item", "priority=high")
	wantExit(t, r, 4)
}

// TestSetNoPair covers exit 2 when no key=value pair is supplied.
func TestSetNoPair(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "nopair-item", approvedFields("Nopair item", "cli"), "body")
	r := runIn(t, root, "backlog", "set", "nopair-item")
	wantExit(t, r, 2)
}

// TestSetMalformedPair covers exit 2 for a malformed (no '=') mutation token.
func TestSetMalformedPair(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "badpair-item", approvedFields("Badpair item", "cli"), "body")
	r := runIn(t, root, "backlog", "set", "badpair-item", "priorityhigh")
	wantExit(t, r, 2)
}

// TestSetJSON confirms set returns the updated item object under --json.
func TestSetJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "json-set-item", approvedFields("Json set item", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "set", "json-set-item", "priority=high")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["priority"] != "high" {
		t.Fatalf("json priority = %v, want high", obj["priority"])
	}
	if obj["id"] != "json-set-item" {
		t.Fatalf("json id = %v, want json-set-item", obj["id"])
	}
}
