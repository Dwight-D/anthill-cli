package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sessions ledger (BACKLOG_V2_SPEC.md §5, §6.6): .anthill/sessions/
// <session>[.<agent>].json, untracked, written by `anthill session link`.

func sessionsDir(root string) string { return filepath.Join(root, ".anthill", "sessions") }

func sessionPath(root, key string) string { return filepath.Join(sessionsDir(root), key+".json") }

// readRecord parses one ledger record file.
func readRecord(t *testing.T, root, key string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(sessionPath(root, key))
	if err != nil {
		t.Fatalf("read ledger record %s: %v", key, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("ledger record %s is not JSON: %v\n%s", key, err, data)
	}
	return m
}

// writeRecord writes a ledger record directly (to seed stale entries).
func writeRecord(t *testing.T, root, key string, rec map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sessionsDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	writeRaw(t, sessionPath(root, key), string(data))
}

func TestSessionLinkCreate(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "work-item", claimedFields("Work item", "cli"), "body")

	r := runIn(t, root, "--json", "session", "link", "--session", "sess-1", "--item", "work-item",
		"--handle", "cli-worker", "--role", "worker", "--transcript", "/tmp/t.jsonl", "--cwd", "/repo", "--verb", "claim")
	wantExit(t, r, 0)
	out := jsonObj(t, r.stdout)
	if out["session"] != "sess-1" || out["item"] != "work-item" || out["handle"] != "cli-worker" {
		t.Fatalf("link --json = %v", out)
	}

	rec := readRecord(t, root, "sess-1")
	for k, want := range map[string]string{
		"session": "sess-1", "agent": "", "handle": "cli-worker", "item": "work-item",
		"role": "worker", "transcript": "/tmp/t.jsonl", "cwd": "/repo",
	} {
		if rec[k] != want {
			t.Errorf("record %s = %v, want %q", k, rec[k], want)
		}
	}
	if rec["started"] == "" || rec["last-seen"] == "" || rec["started"] == nil || rec["last-seen"] == nil {
		t.Errorf("record must stamp started and last-seen: %v", rec)
	}
	// The ledger is untracked: the first write drops a .gitignore.
	ign := readAll(t, filepath.Join(sessionsDir(root), ".gitignore"))
	wantContains(t, ign, "*.json", ".anthill/sessions/.gitignore")
}

func TestSessionLinkAgentKey(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "work-item", claimedFields("Work item", "cli"), "body")
	r := runIn(t, root, "session", "link", "--session", "sess-1", "--agent", "a7f3", "--item", "work-item")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "sess-1.a7f3", "link stdout")
	if rec := readRecord(t, root, "sess-1.a7f3"); rec["agent"] != "a7f3" {
		t.Fatalf("agent = %v", rec["agent"])
	}
}

func TestSessionLinkUpdate(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "first", claimedFields("First", "cli"), "body")
	writeItem(t, root, "cli", "second", claimedFields("Second", "cli"), "body")
	writeRecord(t, root, "sess-1", map[string]any{
		"session": "sess-1", "agent": "", "handle": "h", "item": "first", "role": "worker",
		"transcript": "/t.jsonl", "cwd": "/repo", "started": "2026-01-01T00:00:00Z", "last-seen": "2026-01-01T00:00:00Z",
	})

	r := runIn(t, root, "session", "link", "--session", "sess-1", "--item", "second")
	wantExit(t, r, 0)
	rec := readRecord(t, root, "sess-1")
	if rec["item"] != "second" {
		t.Errorf("item = %v, want second", rec["item"])
	}
	// Unspecified fields are kept; started is kept; last-seen is bumped.
	if rec["handle"] != "h" || rec["transcript"] != "/t.jsonl" || rec["started"] != "2026-01-01T00:00:00Z" {
		t.Errorf("update clobbered fields: %v", rec)
	}
	if rec["last-seen"] == "2026-01-01T00:00:00Z" {
		t.Error("update must bump last-seen")
	}
}

func TestSessionLinkMissingItem(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "session", "link", "--session", "sess-1", "--item", "no-such-item")
	wantExit(t, r, 4)
	if fileExists(t, sessionPath(root, "sess-1")) {
		t.Error("a failed link must not write a record")
	}
}

func TestSessionLinkUsage(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "work-item", claimedFields("Work item", "cli"), "body")
	wantExit(t, runIn(t, root, "session", "link", "--item", "work-item"), 2)
	wantExit(t, runIn(t, root, "session", "link", "--session", "../evil", "--item", "work-item"), 2)
	wantExit(t, runIn(t, root, "session", "link", "--session", "s", "--item", "work-item", "--role", "boss"), 2)
}

func TestSessionTouch(t *testing.T) {
	root := mkTree(t)
	writeRecord(t, root, "sess-1.ag", map[string]any{
		"session": "sess-1", "agent": "ag", "item": "x", "started": "2026-01-01T00:00:00Z", "last-seen": "2026-01-01T00:00:00Z",
	})
	r := runIn(t, root, "--json", "session", "touch", "--session", "sess-1", "--agent", "ag")
	wantExit(t, r, 0)
	if out := jsonObj(t, r.stdout); out["last-seen"] == "2026-01-01T00:00:00Z" {
		t.Fatalf("touch --json shows stale last-seen: %v", out)
	}
	if rec := readRecord(t, root, "sess-1.ag"); rec["last-seen"] == "2026-01-01T00:00:00Z" || rec["item"] != "x" {
		t.Fatalf("touch must bump last-seen only: %v", rec)
	}
	// No record → not found.
	wantExit(t, runIn(t, root, "session", "touch", "--session", "nope"), 4)
}

func TestSessionList(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "live", claimedFields("Live", "cli"), "body")
	writeItem(t, root, "cli", "idle", readyFields("Idle", "cli"), "body")
	writeRecord(t, root, "s-live", map[string]any{
		"session": "s-live", "agent": "", "handle": "w1", "item": "live", "role": "worker",
		"transcript": "/t1.jsonl", "last-seen": "2026-01-01T00:00:00Z",
	})
	writeRecord(t, root, "s-idle", map[string]any{
		"session": "s-idle", "agent": "", "handle": "w2", "item": "idle", "role": "worker",
		"last-seen": "2026-01-01T00:00:00Z",
	})

	r := runIn(t, root, "session", "list")
	wantExit(t, r, 0)
	for _, want := range []string{"SESSION", "LAST-SEEN", "s-live", "s-idle", "w1", "/t1.jsonl"} {
		wantContains(t, r.stdout, want, "session list")
	}

	r = runIn(t, root, "--json", "session", "list", "--item", "idle")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 1 || arr[0]["session"] != "s-idle" {
		t.Fatalf("list --item idle = %v", arr)
	}

	// --active keeps only records whose item is in-progress or validation.
	r = runIn(t, root, "--json", "session", "list", "--active")
	wantExit(t, r, 0)
	arr = jsonArr(t, r.stdout)
	if len(arr) != 1 || arr[0]["session"] != "s-live" {
		t.Fatalf("list --active = %v", arr)
	}

	// Empty ledger → empty array, not null.
	empty := mkTree(t)
	r = runIn(t, empty, "--json", "session", "list")
	wantExit(t, r, 0)
	if strings.TrimSpace(r.stdout) != "[]" {
		t.Fatalf("empty list --json = %q", r.stdout)
	}
}

func TestSessionPrune(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "work-item", claimedFields("Work item", "cli"), "body")
	writeRecord(t, root, "old", map[string]any{"session": "old", "item": "work-item", "last-seen": "2020-01-01T00:00:00Z"})
	wantExit(t, runIn(t, root, "session", "link", "--session", "fresh", "--item", "work-item"), 0)

	// --dry-run reports without deleting.
	r := runIn(t, root, "--json", "session", "prune", "--older-than", "1d", "--dry-run")
	wantExit(t, r, 0)
	out := jsonObj(t, r.stdout)
	pruned, _ := out["pruned"].([]any)
	if out["dry_run"] != true || len(pruned) != 1 || pruned[0] != "old" || out["kept"] != float64(1) {
		t.Fatalf("prune --dry-run = %v", out)
	}
	wantFilePresent(t, sessionPath(root, "old"))

	// A real run deletes only the stale record.
	r = runIn(t, root, "session", "prune", "--older-than", "24h")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "old", "prune stdout")
	wantFileGone(t, sessionPath(root, "old"))
	wantFilePresent(t, sessionPath(root, "fresh"))

	wantExit(t, runIn(t, root, "session", "prune", "--older-than", "soon"), 2)
}

// TestClaimWritesLedgerFromEnv covers deliverable: claim records the ledger
// itself when CLAUDE_CODE_SESSION_ID is set, with no hook installed.
func TestClaimWritesLedgerFromEnv(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "env-claim", readyFields("Env claim", "cli"), "body")
	r := runEnv(t, root, []string{"ANTHILL_ACTOR=cli-worker", "CLAUDE_CODE_SESSION_ID=sess-env-1"},
		"backlog", "claim", "env-claim")
	wantExit(t, r, 0)
	rec := readRecord(t, root, "sess-env-1")
	if rec["session"] != "sess-env-1" || rec["item"] != "env-claim" || rec["handle"] != "cli-worker" || rec["agent"] != "" {
		t.Fatalf("ledger record = %v", rec)
	}
	if rec["last-seen"] == nil || rec["last-seen"] == "" {
		t.Fatalf("ledger record lacks last-seen: %v", rec)
	}
	// Backlog --json views are unchanged: no ledger fields leak into them.
	r = runIn(t, root, "--json", "backlog", "show", "env-claim")
	wantExit(t, r, 0)
	if _, ok := jsonObj(t, r.stdout)["last-seen"]; ok {
		t.Error("backlog show --json must not carry ledger fields")
	}
}

func TestClaimWritesLedgerFromFlag(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "flag-claim", readyFields("Flag claim", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "claim", "flag-claim", "--session", "sess-flag"), 0)
	if rec := readRecord(t, root, "sess-flag"); rec["item"] != "flag-claim" {
		t.Fatalf("ledger record = %v", rec)
	}
}

func TestClaimNoSessionNoLedger(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "plain-claim", readyFields("Plain claim", "cli"), "body")
	wantExit(t, runIn(t, root, "backlog", "claim", "plain-claim"), 0)
	if fileExists(t, sessionsDir(root)) {
		t.Fatal("claim with no known session must not create the ledger")
	}
}

// TestDeliverTouchesLedger covers deliver bumping the claimant session's record.
func TestDeliverTouchesLedger(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "deliver-me", readyFields("Deliver me", "cli"), "body")
	env := []string{"ANTHILL_ACTOR=cli-worker", "CLAUDE_CODE_SESSION_ID=sess-d"}
	wantExit(t, runEnv(t, root, env, "backlog", "claim", "deliver-me"), 0)

	rec := readRecord(t, root, "sess-d")
	rec["last-seen"] = "2020-01-01T00:00:00Z"
	writeRecord(t, root, "sess-d", rec)

	wantExit(t, runEnv(t, root, env, "backlog", "deliver", "deliver-me", "--evidence", "go test ok"), 0)
	rec = readRecord(t, root, "sess-d")
	if rec["last-seen"] == "2020-01-01T00:00:00Z" {
		t.Fatal("deliver must bump the ledger record's last-seen")
	}
	if rec["verb"] != "deliver" {
		t.Errorf("verb = %v, want deliver", rec["verb"])
	}

	// No record for the session: deliver still succeeds and writes nothing.
	root2 := mkTree(t)
	writeItem(t, root2, "cli", "no-ledger", claimedFields("No ledger", "cli"), "body")
	wantExit(t, runEnv(t, root2, []string{"ANTHILL_ACTOR=worker-a"}, "backlog", "deliver", "no-ledger", "--evidence", "ok"), 0)
	if fileExists(t, sessionsDir(root2)) {
		t.Fatal("deliver without a record must not create the ledger")
	}
}
