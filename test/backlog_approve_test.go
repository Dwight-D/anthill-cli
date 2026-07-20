package e2e_test

import "testing"

// approvableFields is a triaged item sitting at status: idea, ready to be
// approved (all triaged fields present, non-empty verify).
func approvableFields(title, ws string) map[string]string {
	f := approvedFields(title, ws)
	f["status"] = "idea"
	return f
}

// TestApproveGatedSuccess covers the only sanctioned approval path (spec §7 Q1
// "Separate approve verb"): `backlog approve <id> --yes` sets status: approved.
func TestApproveGatedSuccess(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "approve-me", approvableFields("Approve me", "cli"), "body")

	r := runIn(t, root, "backlog", "approve", "approve-me", "--yes")
	wantExit(t, r, 0)

	fm := readFrontmatter(t, itemPath(root, "cli", "approve-me"))
	if fm["status"] != "approved" {
		t.Fatalf("status = %q, want approved", fm["status"])
	}
}

// TestApproveWithoutYesRefused confirms the confirmation gate: approve without
// --yes must NOT approve the item (headless, no interactive prompt).
func TestApproveWithoutYesRefused(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "gate-approve", approvableFields("Gate approve", "cli"), "body")

	r := runIn(t, root, "backlog", "approve", "gate-approve")
	wantNonZero(t, r)

	fm := readFrontmatter(t, itemPath(root, "cli", "gate-approve"))
	if fm["status"] == "approved" {
		t.Fatalf("approve without --yes must not approve, but status became approved")
	}
}

// TestApproveNotFound covers exit 4 for approving a missing id.
func TestApproveNotFound(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "approve", "no-such-item", "--yes")
	wantExit(t, r, 4)
}

// TestApproveMakesReady confirms an approved item with a verify is reported
// ready by list --json (the readiness derivation, spec §4).
func TestApproveMakesReady(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "ready-after", approvableFields("Ready after", "cli"), "body")
	runIn(t, root, "backlog", "approve", "ready-after", "--yes")

	r := runIn(t, root, "--json", "backlog", "list", "--workstream", "cli")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	found := false
	for _, it := range arr {
		if it["id"] == "ready-after" {
			found = true
			if it["ready"] != true {
				t.Fatalf("approved item with verify should be ready=true, got %v", it["ready"])
			}
		}
	}
	if !found {
		t.Fatalf("approved item not found in list: %v", arr)
	}
}
