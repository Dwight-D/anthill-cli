package e2e_test

import (
	"strings"
	"testing"
)

// claimedFields is an in-progress (claimed) item, ready to be closed.
func claimedFields(title, ws string) map[string]string {
	f := approvedFields(title, ws)
	f["status"] = "in-progress"
	f["claimed-at"] = "2026-07-20T10:00:00Z"
	return f
}

// TestCloseDoneTerminal covers `close --done` (spec §3.7): terminal — the item
// file is DELETED and one line is appended to CHANGELOG.md.
func TestCloseDoneTerminal(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "done-item", claimedFields("Done item", "cli"), "body")

	r := runIn(t, root, "backlog", "close", "done-item", "--done")
	wantExit(t, r, 0)

	wantFileGone(t, itemPath(root, "cli", "done-item"))
	changelog := readAll(t, changelogPath(root))
	wantContains(t, changelog, "done-item", "CHANGELOG after --done")
	wantContains(t, changelog, "done", "CHANGELOG after --done")
}

// TestCloseDiscardTerminal covers `close --discard` (terminal + CHANGELOG).
func TestCloseDiscardTerminal(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "discard-item", claimedFields("Discard item", "cli"), "body")

	r := runIn(t, root, "backlog", "close", "discard-item", "--discard", "not worth doing")
	wantExit(t, r, 0)

	wantFileGone(t, itemPath(root, "cli", "discard-item"))
	changelog := readAll(t, changelogPath(root))
	wantContains(t, changelog, "discard-item", "CHANGELOG after --discard")
	wantContains(t, changelog, "discarded", "CHANGELOG after --discard")
}

// TestCloseRemoveTerminal covers `close --remove` (terminal + CHANGELOG).
func TestCloseRemoveTerminal(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "remove-item", claimedFields("Remove item", "cli"), "body")

	r := runIn(t, root, "backlog", "close", "remove-item", "--remove", "superseded")
	wantExit(t, r, 0)

	wantFileGone(t, itemPath(root, "cli", "remove-item"))
	changelog := readAll(t, changelogPath(root))
	wantContains(t, changelog, "remove-item", "CHANGELOG after --remove")
	wantContains(t, changelog, "removed", "CHANGELOG after --remove")
}

// sectionedChangelog is the CHANGELOG shape a real (scaffolded) install ships:
// an empty "## Done" above "## Discarded". It is the shape under which the
// mis-filing bug manifests.
const sectionedChangelog = "# Improvement Changelog\n\nOne line per closed item, newest first.\n\n## Done\n\n## Discarded (triaged out, not done)\n"

// TestCloseDoneFilesUnderDoneHeading is the regression for the bug where
// `close --done` appended the changelog line at the end of the file — landing it
// under the trailing "## Discarded" heading instead of the empty "## Done". The
// done line must fall between "## Done" and "## Discarded".
func TestCloseDoneFilesUnderDoneHeading(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, changelogPath(root), sectionedChangelog)
	writeItem(t, root, "cli", "done-item", claimedFields("Done item", "cli"), "body")

	r := runIn(t, root, "backlog", "close", "done-item", "--done")
	wantExit(t, r, 0)

	cl := readAll(t, changelogPath(root))
	doneIdx := strings.Index(cl, "## Done")
	discIdx := strings.Index(cl, "## Discarded")
	lineIdx := strings.Index(cl, "done-item")
	if lineIdx < 0 {
		t.Fatalf("changelog missing the closed item:\n%s", cl)
	}
	if !(lineIdx > doneIdx && lineIdx < discIdx) {
		t.Fatalf("close --done filed the item under the wrong heading:\n%s", cl)
	}
}

// TestCloseDiscardFilesUnderDiscardedHeading is the companion: a discarded item
// must land under "## Discarded", not "## Done".
func TestCloseDiscardFilesUnderDiscardedHeading(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, changelogPath(root), sectionedChangelog)
	writeItem(t, root, "cli", "discard-item", claimedFields("Discard item", "cli"), "body")

	r := runIn(t, root, "backlog", "close", "discard-item", "--discard", "not worth doing")
	wantExit(t, r, 0)

	cl := readAll(t, changelogPath(root))
	discIdx := strings.Index(cl, "## Discarded")
	lineIdx := strings.Index(cl, "discard-item")
	if lineIdx < 0 || lineIdx < discIdx {
		t.Fatalf("close --discard did not file under ## Discarded:\n%s", cl)
	}
}

// TestCloseBlockNonTerminal covers `close --block` (spec §3.7, §7 Q3): the file
// STAYS, status becomes blocked, the reason lands in note, and NO CHANGELOG
// line is written.
func TestCloseBlockNonTerminal(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "block-item", claimedFields("Block item", "cli"), "body")

	before := readAll(t, changelogPath(root))
	r := runIn(t, root, "backlog", "close", "block-item", "--block", "waiting on escalation")
	wantExit(t, r, 0)

	// File remains, status blocked.
	path := itemPath(root, "cli", "block-item")
	wantFilePresent(t, path)
	fm := readFrontmatter(t, path)
	if fm["status"] != "blocked" {
		t.Fatalf("status = %q, want blocked", fm["status"])
	}
	if !strings.Contains(fm["note"], "waiting on escalation") {
		t.Fatalf("block reason should land in note, got note=%q", fm["note"])
	}

	// No CHANGELOG line appended.
	after := readAll(t, changelogPath(root))
	if after != before {
		t.Fatalf("close --block must NOT append to CHANGELOG.\nbefore: %q\nafter: %q", before, after)
	}
}

// TestCloseBlockJSON covers the json result for block: changelog=false.
func TestCloseBlockJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "block-json", claimedFields("Block json", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "close", "block-json", "--block", "dep")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["disposition"] != "block" {
		t.Fatalf("json disposition = %v, want block", obj["disposition"])
	}
	if obj["changelog"] != false {
		t.Fatalf("json changelog = %v, want false for block", obj["changelog"])
	}
}

// TestCloseDoneJSON covers the json result for done: changelog=true.
func TestCloseDoneJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "done-json", claimedFields("Done json", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "close", "done-json", "--done")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["disposition"] != "done" {
		t.Fatalf("json disposition = %v, want done", obj["disposition"])
	}
	if obj["changelog"] != true {
		t.Fatalf("json changelog = %v, want true for done", obj["changelog"])
	}
}

// TestCloseNoDisposition covers exit 2 when no disposition flag is given.
func TestCloseNoDisposition(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "nodisp-item", claimedFields("Nodisp item", "cli"), "body")
	r := runIn(t, root, "backlog", "close", "nodisp-item")
	wantExit(t, r, 2)
}

// TestCloseMultipleDispositions covers exit 2 when more than one disposition
// flag is given (mutually exclusive).
func TestCloseMultipleDispositions(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "multidisp-item", claimedFields("Multidisp item", "cli"), "body")
	r := runIn(t, root, "backlog", "close", "multidisp-item", "--done", "--discard", "x")
	wantExit(t, r, 2)
}

// TestCloseNotFound covers exit 4 for a missing id.
func TestCloseNotFound(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "close", "ghost-item", "--done")
	wantExit(t, r, 4)
}

// TestCloseUnclaimedPrecondition covers exit 6: closing an item that is not in a
// closeable state (not claimed) — here an approved-but-unclaimed item.
func TestCloseUnclaimedPrecondition(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "unclaimed-item", approvedFields("Unclaimed item", "cli"), "body")
	r := runIn(t, root, "backlog", "close", "unclaimed-item", "--done")
	wantExit(t, r, 6)
	// Still present (not terminated).
	wantFilePresent(t, itemPath(root, "cli", "unclaimed-item"))
}
