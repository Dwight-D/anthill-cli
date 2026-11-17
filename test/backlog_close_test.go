package e2e_test

import (
	"strings"
	"testing"
)

// TestCloseDoneArchives covers `close --done` (§2): validation → done; the
// file moves to done/, closed-at is stamped, and one CHANGELOG line is filed.
func TestCloseDoneArchives(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "done-item", validationFields("Done item", "cli"), "body")

	r := runIn(t, root, "backlog", "close", "done-item", "--done")
	wantExit(t, r, 0)

	wantFileGone(t, itemPath(root, "cli", "done-item"))
	fm := readFrontmatter(t, itemPath(root, "done", "done-item"))
	if fm["status"] != "done" || fm["closed-at"] == "" || fm["workstream"] != "cli" {
		t.Fatalf("done item frontmatter = %v", fm)
	}
	changelog := readAll(t, changelogPath(root))
	wantContains(t, changelog, "done-item — done", "CHANGELOG after --done")
	wantContains(t, readAll(t, itemPath(root, "done", "done-item")), "close --done", "Log after --done")
}

// TestCloseDoneFromInProgressRefused covers exit 6: done needs validation.
func TestCloseDoneFromInProgressRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "wip", claimedFields("Wip", "cli"), "body")
	r := runIn(t, root, "backlog", "close", "wip", "--done")
	wantExit(t, r, 6)
	wantFilePresent(t, itemPath(root, "cli", "wip"))
}

// TestCloseDoneForce covers --force: a human closes an in-progress item.
func TestCloseDoneForce(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "forced", claimedFields("Forced", "cli"), "body")
	r := runIn(t, root, "backlog", "close", "forced", "--done", "--force")
	wantExit(t, r, 0)
	wantContains(t, readAll(t, itemPath(root, "done", "forced")), "close --done (forced)", "forced Log line")
}

// TestCloseRejectReturnsToInProgress covers validation → in-progress keeping
// the claim, with the reason in the Log and no CHANGELOG line.
func TestCloseRejectReturnsToInProgress(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "rej", validationFields("Rej", "cli"), "body")
	before := readAll(t, changelogPath(root))
	r := runIn(t, root, "backlog", "close", "rej", "--reject", "tests not run")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", "rej"))
	if fm["status"] != "in-progress" || fm["claimed-by"] != "worker-a" {
		t.Fatalf("after reject: %v", fm)
	}
	wantContains(t, readAll(t, itemPath(root, "cli", "rej")), "close --reject by", "Log after reject")
	wantContains(t, readAll(t, itemPath(root, "cli", "rej")), "tests not run", "Log reason")
	if readAll(t, changelogPath(root)) != before {
		t.Fatal("reject must not write CHANGELOG")
	}
}

// TestCloseRejectNeedsReason covers exit 2 for a reasonless reject.
func TestCloseRejectNeedsReason(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "rej", validationFields("Rej", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "close", "rej", "--reject"), 2)
}

// TestCloseRejectFromInProgressRefused covers exit 6: reject needs validation.
func TestCloseRejectFromInProgressRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "wip", claimedFields("Wip", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "close", "wip", "--reject", "no"), 6)
}

// TestCloseDiscardArchives covers any → discarded: moved to discarded/.
func TestCloseDiscardArchives(t *testing.T) {
	for _, tc := range []struct {
		dir    string
		fields map[string]string
	}{
		{"intake", newFields("Disc")},
		{"cli", readyFields("Disc", "cli")},
		{"cli", claimedFields("Disc", "cli")},
		{"cli", withField(readyFields("Disc", "cli"), "status", "parked")},
	} {
		root := mkTree(t)
		writeItem(t, root, tc.dir, "disc", tc.fields, "body")
		r := runIn(t, root, "backlog", "close", "disc", "--discard", "not worth doing")
		wantExit(t, r, 0)
		wantFileGone(t, itemPath(root, tc.dir, "disc"))
		fm := readFrontmatter(t, itemPath(root, "discarded", "disc"))
		if fm["status"] != "discarded" || fm["closed-at"] == "" {
			t.Fatalf("discarded item from %s: %v", tc.fields["status"], fm)
		}
		wantContains(t, readAll(t, changelogPath(root)), "disc — discarded: not worth doing", "CHANGELOG")
	}
}

// TestCloseDiscardTwiceRefused covers exit 6: an archived item cannot be
// discarded again.
func TestCloseDiscardTwiceRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", "d", newFields("D"), "body")
	wantExit(t, runIn(t, root, "backlog", "close", "d", "--discard"), 0)
	wantExit(t, runIn(t, root, "backlog", "close", "d", "--discard"), 6)
}

// TestCloseRemoveDeletes covers any → removed: the file is deleted outright.
func TestCloseRemoveDeletes(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "remove-item", claimedFields("Remove item", "cli"), "body")
	r := runIn(t, root, "backlog", "close", "remove-item", "--remove", "filed in error")
	wantExit(t, r, 0)
	wantFileGone(t, itemPath(root, "cli", "remove-item"))
	wantFileGone(t, itemPath(root, "discarded", "remove-item"))
	wantContains(t, readAll(t, changelogPath(root)), "remove-item — removed: filed in error", "CHANGELOG")
}

// sectionedChangelog is the CHANGELOG shape a real (scaffolded) install ships:
// an empty "## Done" above "## Discarded".
const sectionedChangelog = "# Improvement Changelog\n\nOne line per closed item, newest first.\n\n## Done\n\n## Discarded (triaged out, not done)\n"

// TestCloseDoneFilesUnderDoneHeading: the done line falls between "## Done"
// and "## Discarded".
func TestCloseDoneFilesUnderDoneHeading(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, changelogPath(root), sectionedChangelog)
	writeItem(t, root, "cli", "done-item", validationFields("Done item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "close", "done-item", "--done"), 0)
	cl := readAll(t, changelogPath(root))
	doneIdx := strings.Index(cl, "## Done")
	discIdx := strings.Index(cl, "## Discarded")
	lineIdx := strings.Index(cl, "done-item")
	if lineIdx < 0 || !(lineIdx > doneIdx && lineIdx < discIdx) {
		t.Fatalf("close --done filed under the wrong heading:\n%s", cl)
	}
}

// TestCloseDiscardFilesUnderDiscardedHeading: a discarded item lands under
// "## Discarded".
func TestCloseDiscardFilesUnderDiscardedHeading(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, changelogPath(root), sectionedChangelog)
	writeItem(t, root, "cli", "discard-item", claimedFields("Discard item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "close", "discard-item", "--discard", "not worth doing"), 0)
	cl := readAll(t, changelogPath(root))
	if i := strings.Index(cl, "discard-item"); i < 0 || i < strings.Index(cl, "## Discarded") {
		t.Fatalf("close --discard did not file under ## Discarded:\n%s", cl)
	}
}

// TestCloseJSON covers the json result: id, mode, status, changelog.
func TestCloseJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "done-json", validationFields("Done json", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "close", "done-json", "--done")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["mode"] != "done" || obj["status"] != "done" || obj["changelog"] != true {
		t.Fatalf("json = %v", obj)
	}
	writeItem(t, root, "cli", "rej-json", validationFields("Rej json", "cli"), "body")
	r = runIn(t, root, "--json", "backlog", "close", "rej-json", "--reject", "x")
	wantExit(t, r, 0)
	if obj := jsonObj(t, r.stdout); obj["changelog"] != false || obj["status"] != "in-progress" {
		t.Fatalf("reject json = %v", obj)
	}
}

// TestCloseBareModeAfterID: a bare boolean mode after the id works, and the
// CHANGELOG line falls back to the title.
func TestCloseBareModeAfterID(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "bare-item", claimedFields("Bare item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "close", "bare-item", "--remove"), 0)
	wantContains(t, readAll(t, changelogPath(root)), "Bare item", "CHANGELOG reason falls back to title")
}

// TestCloseModeBeforeID: the id parses as the positional in either order.
func TestCloseModeBeforeID(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "order-item", claimedFields("Order item", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "close", "--discard", "order-item", "duplicate of other-item"), 0)
	wantContains(t, readAll(t, changelogPath(root)), "duplicate of other-item", "CHANGELOG reason")
}

// TestCloseUsageErrors covers exit 2: no mode, several modes, --force off done,
// and the removed --block.
func TestCloseUsageErrors(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "u", validationFields("U", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "close", "u"), 2)
	wantExit(t, runIn(t, root, "backlog", "close", "u", "--done", "--discard", "x"), 2)
	wantExit(t, runIn(t, root, "backlog", "close", "u", "--discard", "--force"), 2)
	wantExit(t, runIn(t, root, "backlog", "close", "u", "--block", "x"), 2)
}

// TestCloseNotFound covers exit 4 for a missing id.
func TestCloseNotFound(t *testing.T) {
	root := mkTree(t)
	wantExit(t, runIn(t, root, "backlog", "close", "ghost-item", "--done"), 4)
}
