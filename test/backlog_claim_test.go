package e2e_test

import "testing"

// TestClaimSuccess covers `backlog claim <id>` transitioning a ready item to
// in-progress and stamping claimed-at / claimed-by / session (§2, §5).
func TestClaimSuccess(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "claim-me", readyFields("Claim me", "cli"), "the body")

	r := runIn(t, root, "backlog", "claim", key(1))
	wantExit(t, r, 0)
	// claim prints the item body on stdout (dispatch packages it into a brief).
	wantContains(t, r.stdout, "the body", "claim stdout")

	fm := readFrontmatter(t, itemPath(root, "cli", 1, "claim-me"))
	if fm["status"] != "in-progress" {
		t.Fatalf("status = %q, want in-progress", fm["status"])
	}
	if fm["claimed-at"] == "" {
		t.Fatalf("claim must stamp a non-empty claimed-at field, got %q", fm["claimed-at"])
	}
	if fm["claimed-by"] == "" {
		t.Fatal("claim must stamp claimed-by (OS user fallback)")
	}
	if _, ok := fm["session"]; ok {
		t.Fatalf("no session known: session must stay empty, got %q", fm["session"])
	}
	wantContains(t, readAll(t, itemPath(root, "cli", 1, "claim-me")), " claim by ", "Log after claim")
}

// TestClaimStampsFromEnv covers §5: ANTHILL_ACTOR → claimed-by and
// CLAUDE_CODE_SESSION_ID → session.
func TestClaimStampsFromEnv(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "env-claim", readyFields("Env claim", "cli"), "body")
	r := runEnv(t, root, []string{"ANTHILL_ACTOR=supervisor:cli-worker", "CLAUDE_CODE_SESSION_ID=2f1c9a00-1111-2222"},
		"backlog", "claim", key(1))
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "env-claim"))
	if fm["claimed-by"] != "supervisor:cli-worker" || fm["session"] != "2f1c9a00-1111-2222" {
		t.Fatalf("stamps = by %q session %q", fm["claimed-by"], fm["session"])
	}
	wantContains(t, readAll(t, itemPath(root, "cli", 1, "env-claim")), "claim by supervisor:cli-worker (session 2f1c9a00…)", "Log line")
}

// TestClaimFlagsBeatEnv covers --by / --session taking precedence over env.
func TestClaimFlagsBeatEnv(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "flag-claim", readyFields("Flag claim", "cli"), "body")
	r := runEnv(t, root, []string{"ANTHILL_ACTOR=env-actor", "CLAUDE_CODE_SESSION_ID=env-session"},
		"backlog", "claim", key(1), "--by", "worker-b", "--session", "flag-session")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "flag-claim"))
	if fm["claimed-by"] != "worker-b" || fm["session"] != "flag-session" {
		t.Fatalf("stamps = by %q session %q", fm["claimed-by"], fm["session"])
	}
}

// TestClaimRoleSessionFallback covers §5's third tier: with no handle and no
// ANTHILL_ACTOR but a harness session, claimed-by is <role>:<session prefix>.
func TestClaimRoleSessionFallback(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "role-claim", readyFields("Role claim", "cli"), "body")
	r := runEnv(t, root, []string{"CLAUDECODE=1", "CLAUDE_CODE_SESSION_ID=abcdef0123456789"},
		"backlog", "claim", key(1))
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "role-claim"))
	if fm["claimed-by"] != "worker:abcdef01" {
		t.Fatalf("claimed-by = %q, want worker:abcdef01", fm["claimed-by"])
	}
}

// TestClaimConflict covers exit 5 (conflict): a second claim of an already
// in-progress item WITHOUT --force fails the compare-and-set (spec §3.6, §7 Q4).
func TestClaimConflict(t *testing.T) {
	root := mkTree(t)
	f := readyFields("Taken item", "cli")
	f["status"] = "in-progress"
	f["claimed-at"] = "2026-07-20T10:00:00Z"
	writeItem(t, root, "cli", 1, "taken-item", f, "body")

	r := runIn(t, root, "backlog", "claim", key(1))
	wantExit(t, r, 5)
	// Untouched.
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "taken-item"))
	if fm["status"] != "in-progress" {
		t.Fatalf("conflicting claim must not change status, got %q", fm["status"])
	}
}

// TestClaimForceReclaims covers `claim --force` reclaiming an in-progress orphan.
func TestClaimForceReclaims(t *testing.T) {
	root := mkTree(t)
	f := readyFields("Orphan item", "cli")
	f["status"] = "in-progress"
	f["claimed-at"] = "2026-07-20T10:00:00Z"
	writeItem(t, root, "cli", 1, "orphan-item", f, "body")

	r := runIn(t, root, "backlog", "claim", key(1), "--force")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "orphan-item"))
	if fm["status"] != "in-progress" {
		t.Fatalf("reclaim should keep status in-progress, got %q", fm["status"])
	}
}

// TestClaimNotReadyPrecondition covers exit 6: claiming a new item is refused
// even with --force (only triaged items are claimable).
func TestClaimNotReadyPrecondition(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", 1, "not-ready-claim", newFields("Not ready claim"), "body")
	r := runIn(t, root, "backlog", "claim", key(1))
	wantExit(t, r, 6)
	wantExit(t, runIn(t, root, "backlog", "claim", key(1), "--force"), 6)
	fm := readFrontmatter(t, itemPath(root, "intake", 1, "not-ready-claim"))
	if fm["status"] != "new" {
		t.Fatalf("failed claim must leave status new, got %q", fm["status"])
	}
}

// TestClaimForceNonReady covers --force overriding the readiness gate (spec §3.6:
// a human naming an item is an explicit override).
func TestClaimForceNonReady(t *testing.T) {
	root := mkTree(t)
	f := readyFields("Forced not ready", "cli")
	f["dispatch"] = "proposed"
	writeItem(t, root, "cli", 1, "forced-not-ready", f, "body")
	wantExit(t, runIn(t, root, "backlog", "claim", key(1)), 6)
	r := runIn(t, root, "backlog", "claim", key(1), "--force")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "forced-not-ready"))
	if fm["status"] != "in-progress" {
		t.Fatalf("forced claim should set in-progress, got %q", fm["status"])
	}
}

// TestClaimNotFound covers exit 4 for a missing id (key, bare number, and a
// slug-shaped former id that no item carries).
func TestClaimNotFound(t *testing.T) {
	root := mkTree(t)
	wantExit(t, runIn(t, root, "backlog", "claim", key(99)), 4)
	wantExit(t, runIn(t, root, "backlog", "claim", "99"), 4)
	wantExit(t, runIn(t, root, "backlog", "claim", "ghost-item"), 4)
}

// TestClaimNext covers `claim --next` claiming the item next would select.
func TestClaimNext(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "next-claim", readyFields("Next claim", "cli"), "body")
	r := runIn(t, root, "backlog", "claim", "--next")
	wantExit(t, r, 0)
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "next-claim"))
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

// TestClaimJSON confirms the claimed item object under --json, with the
// canonical key as id whatever reference form the argument used.
func TestClaimJSON(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "json-claim", readyFields("Json claim", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "claim", "1")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["status"] != "in-progress" {
		t.Fatalf("json status = %v, want in-progress", obj["status"])
	}
	if obj["id"] != key(1) {
		t.Fatalf("json id = %v, want %q", obj["id"], key(1))
	}
}
