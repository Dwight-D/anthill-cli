package e2e_test

import "testing"

// TestClaimSuccess covers `backlog claim <id>` transitioning a ready item to
// in-progress and stamping a claimed-at marker (spec §3.6, §7 Q4).
func TestClaimSuccess(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "claim-me", approvedFields("Claim me", "cli"), "the body")

	r := runIn(t, root, "backlog", "claim", "claim-me")
	wantExit(t, r, 0)
	// claim prints the item body on stdout (dispatch packages it into a brief).
	wantContains(t, r.stdout, "the body", "claim stdout")

	fm := readFrontmatter(t, itemPath(root, "cli", "claim-me"))
	if fm["status"] != "in-progress" {
		t.Fatalf("status = %q, want in-progress", fm["status"])
	}
	if fm["claimed-at"] == "" {
		t.Fatalf("claim must stamp a non-empty claimed-at field, got %q", fm["claimed-at"])
	}
}

// TestClaimConflict covers exit 5 (conflict): a second claim of an already
// in-progress item WITHOUT --force fails the compare-and-set (spec §3.6, §7 Q4).
func TestClaimConflict(t *testing.T) {
	root := mkTree(t)
	f := approvedFields("Taken item", "cli")
	f["status"] = "in-progress"
	f["claimed-at"] = "2026-07-20T10:00:00Z"
	writeItem(t, root, "cli", "taken-item", f, "body")

	r := runIn(t, root, "backlog", "claim", "taken-item")
	wantExit(t, r, 5)
	// Untouched.
	fm := readFrontmatter(t, itemPath(root, "cli", "taken-item"))
	if fm["status"] != "in-progress" {
		t.Fatalf("conflicting claim must not change status, got %q", fm["status"])
	}
}

// TestClaimForceReclaims covers `claim --force` reclaiming an in-progress orphan.
func TestClaimForceReclaims(t *testing.T) {
	root := mkTree(t)
	f := approvedFields("Orphan item", "cli")
	f["status"] = "in-progress"
	f["claimed-at"] = "2026-07-20T10:00:00Z"
	writeItem(t, root, "cli", "orphan-item", f, "body")

	r := runIn(t, root, "backlog", "claim", "orphan-item", "--force")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", "orphan-item"))
	if fm["status"] != "in-progress" {
		t.Fatalf("reclaim should keep status in-progress, got %q", fm["status"])
	}
}

// TestClaimNotReadyPrecondition covers exit 6: claiming a non-ready (idea) item
// without --force is a precondition failure (spec §3.6).
func TestClaimNotReadyPrecondition(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", "not-ready-claim", ideaFields("Not ready claim"), "body")
	r := runIn(t, root, "backlog", "claim", "not-ready-claim")
	wantExit(t, r, 6)
	fm := readFrontmatter(t, itemPath(root, "intake", "not-ready-claim"))
	if fm["status"] != "idea" {
		t.Fatalf("failed claim must leave status idea, got %q", fm["status"])
	}
}

// TestClaimForceNonReady covers --force overriding the readiness gate (spec §3.6:
// a human naming an item is an explicit override).
func TestClaimForceNonReady(t *testing.T) {
	root := mkTree(t)
	f := approvedFields("Forced not ready", "cli")
	f["status"] = "idea"
	writeItem(t, root, "cli", "forced-not-ready", f, "body")
	r := runIn(t, root, "backlog", "claim", "forced-not-ready", "--force")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", "forced-not-ready"))
	if fm["status"] != "in-progress" {
		t.Fatalf("forced claim should set in-progress, got %q", fm["status"])
	}
}

// TestClaimNotFound covers exit 4 for a missing id.
func TestClaimNotFound(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "claim", "ghost-item")
	wantExit(t, r, 4)
}

// TestClaimNext covers `claim --next` claiming the item next would select.
func TestClaimNext(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "next-claim", approvedFields("Next claim", "cli"), "body")
	r := runIn(t, root, "backlog", "claim", "--next")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", "next-claim"))
	if fm["status"] != "in-progress" {
		t.Fatalf("claim --next should set in-progress, got %q", fm["status"])
	}
}

// TestClaimNextEmpty covers spec §3.6 note: `claim --next` with an empty ready
// set exits 4 (asked to produce a claim, could not) — distinct from next's
// exit-0 empty.
func TestClaimNextEmpty(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "claim", "--next")
	wantExit(t, r, 4)
}

// TestClaimJSON confirms the claimed item object under --json.
func TestClaimJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "json-claim", approvedFields("Json claim", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "claim", "json-claim")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["status"] != "in-progress" {
		t.Fatalf("json status = %v, want in-progress", obj["status"])
	}
}
