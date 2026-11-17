package e2e_test

import "testing"

// TestExitCodeTable is a consolidated, self-documenting check that each non-zero
// exit class in the spec §2 table is reachable. Each subtest triggers exactly
// one code (2 usage, 3 validation, 4 not_found, 5 conflict, 6 precondition).
// Command-specific behavior is covered in the per-command test files; this test
// exists so the exit-code contract is verified centrally.
func TestExitCodeTable(t *testing.T) {
	t.Run("code2_usage_unknown_flag", func(t *testing.T) {
		root := mkTree(t)
		r := runIn(t, root, "backlog", "list", "--no-such-flag")
		wantExit(t, r, 2)
	})

	t.Run("code2_usage_missing_required", func(t *testing.T) {
		root := mkTree(t)
		r := runIn(t, root, "backlog", "new", "--value", "v") // missing --title
		wantExit(t, r, 2)
	})

	t.Run("code2_usage_malformed_id", func(t *testing.T) {
		root := mkTree(t)
		r := runIn(t, root, "backlog", "show", "../x") // not a key, a number, or a former id
		wantExit(t, r, 2)
		wantContains(t, r.stderr, `invalid item id "../x"`, "malformed id stderr")
	})

	t.Run("code3_validation_illegal_enum", func(t *testing.T) {
		root := mkTree(t)
		writeItem(t, root, "cli", 1, "ec3-item", readyFields("Ec3 item", "cli"), "body")
		r := runIn(t, root, "backlog", "set", key(1), "status=not-a-status")
		wantExit(t, r, 3)
	})

	t.Run("code4_not_found_show", func(t *testing.T) {
		root := mkTree(t)
		r := runIn(t, root, "backlog", "show", "does-not-exist") // slug-shaped: a former id matching nothing
		wantExit(t, r, 4)
	})

	t.Run("code5_conflict_double_claim", func(t *testing.T) {
		root := mkTree(t)
		f := readyFields("Ec5 item", "cli")
		f["status"] = "in-progress"
		f["claimed-at"] = "2026-07-20T10:00:00Z"
		writeItem(t, root, "cli", 1, "ec5-item", f, "body")
		r := runIn(t, root, "backlog", "claim", key(1))
		wantExit(t, r, 5)
	})

	t.Run("code6_precondition_close_unclaimed", func(t *testing.T) {
		root := mkTree(t)
		writeItem(t, root, "cli", 1, "ec6-item", readyFields("Ec6 item", "cli"), "body")
		r := runIn(t, root, "backlog", "close", key(1), "--done") // not claimed
		wantExit(t, r, 6)
	})
}
