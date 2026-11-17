package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureStatuses are the stored statuses a transition can be attempted from.
var fixtureStatuses = []string{"new", "triaged", "in-progress", "validation", "blocked", "parked", "done", "discarded"}

// writeAtStatus writes item number n (file NNNN-<slug>.md, title slug) at
// status st in its canonical directory and returns that directory name.
func writeAtStatus(t *testing.T, root string, n int, slug, st string) string {
	t.Helper()
	var dir string
	var f map[string]string
	switch st {
	case "new":
		dir, f = "intake", newFields(slug)
	case "triaged":
		dir, f = "cli", readyFields(slug, "cli")
	case "in-progress":
		dir, f = "cli", claimedFields(slug, "cli")
	case "validation":
		dir, f = "cli", validationFields(slug, "cli")
	case "blocked":
		dir, f = "cli", withField(claimedFields(slug, "cli"), "status", "blocked")
	case "parked":
		dir, f = "cli", withField(readyFields(slug, "cli"), "status", "parked")
	case "done":
		dir, f = "done", withField(validationFields(slug, "cli"), "status", "done")
		f["closed-at"] = "2026-11-05T00:00:00Z"
	case "discarded":
		dir, f = "discarded", withField(readyFields(slug, "cli"), "status", "discarded")
		f["closed-at"] = "2026-11-05T00:00:00Z"
	default:
		t.Fatalf("unknown fixture status %s", st)
	}
	if err := os.MkdirAll(filepath.Join(backlogDir(root), dir), 0o755); err != nil {
		t.Fatal(err)
	}
	writeItem(t, root, dir, n, slug, f, "## Description\nd\n\n## Log\n")
	return dir
}

// transition is one verb attempted against every fixture status.
type transition struct {
	name  string
	args  func(id string) []string
	legal map[string]string // from status -> resulting status
	dir   string            // dir the item lives in afterwards ("" = unchanged)
	log   string            // substring the Log gains on success
}

var transitions = []transition{
	{
		name: "triage",
		args: func(id string) []string {
			return []string{"backlog", "triage", id, "workstream=cli", "summary=s", "change-type=doc",
				"risk=additive", "verify=go test ./...", "disposition=auto"}
		},
		legal: map[string]string{"new": "triaged"}, dir: "cli", log: "triage → cli",
	},
	{
		name:  "claim",
		args:  func(id string) []string { return []string{"backlog", "claim", id, "--by", "worker-a"} },
		legal: map[string]string{"triaged": "in-progress"}, log: "claim by worker-a",
	},
	{
		name: "deliver",
		args: func(id string) []string {
			return []string{"backlog", "deliver", id, "--evidence", "go test exit 0", "--by", "worker-a"}
		},
		legal: map[string]string{"in-progress": "validation"}, log: "deliver by worker-a",
	},
	{
		name:  "close --done",
		args:  func(id string) []string { return []string{"backlog", "close", id, "--done"} },
		legal: map[string]string{"validation": "done"}, dir: "done", log: "close --done",
	},
	{
		name:  "close --reject",
		args:  func(id string) []string { return []string{"backlog", "close", id, "--reject", "no evidence"} },
		legal: map[string]string{"validation": "in-progress"}, log: "close --reject",
	},
	{
		name:  "block",
		args:  func(id string) []string { return []string{"backlog", "block", id, "waiting on dep"} },
		legal: map[string]string{"in-progress": "blocked"}, log: "block by",
	},
	{
		name:  "unblock",
		args:  func(id string) []string { return []string{"backlog", "unblock", id} },
		legal: map[string]string{"blocked": "in-progress"}, log: "unblock by",
	},
	{
		name:  "unblock --release",
		args:  func(id string) []string { return []string{"backlog", "unblock", id, "--release"} },
		legal: map[string]string{"blocked": "triaged"}, log: "unblock --release",
	},
	{
		name:  "release",
		args:  func(id string) []string { return []string{"backlog", "release", id} },
		legal: map[string]string{"in-progress": "triaged"}, log: "release by",
	},
	{
		name:  "park",
		args:  func(id string) []string { return []string{"backlog", "park", id, "later"} },
		legal: map[string]string{"new": "parked", "triaged": "parked"}, log: "park by",
	},
	{
		name:  "unpark",
		args:  func(id string) []string { return []string{"backlog", "unpark", id} },
		legal: map[string]string{"parked": "triaged"}, log: "unpark → triaged",
	},
}

// TestTransitionTable drives every verb from every stored status: legal moves
// land on the §2 target status (and directory) with a Log line; every other
// source status is refused with exit 6 (exit 5 for claiming a claimed item)
// and leaves the file byte-identical. Moves keep the filename.
func TestTransitionTable(t *testing.T) {
	for _, tr := range transitions {
		for _, from := range fixtureStatuses {
			t.Run(tr.name+"/"+from, func(t *testing.T) {
				root := mkTree(t)
				n, slug := 1, "item"
				dir := writeAtStatus(t, root, n, slug, from)
				before := readAll(t, itemPath(root, dir, n, slug))
				r := runIn(t, root, tr.args(key(n))...)
				to, ok := tr.legal[from]
				if !ok {
					want := 6
					if tr.name == "claim" && from == "in-progress" {
						want = 5
					}
					wantExit(t, r, want)
					if readAll(t, itemPath(root, dir, n, slug)) != before {
						t.Fatal("refused transition modified the file")
					}
					return
				}
				wantExit(t, r, 0)
				newDir := dir
				if tr.dir != "" {
					newDir = tr.dir
				}
				if newDir != dir {
					wantFileGone(t, itemPath(root, dir, n, slug))
				}
				path := itemPath(root, newDir, n, slug)
				fm := readFrontmatter(t, path)
				if fm["status"] != to {
					t.Fatalf("status = %q, want %q", fm["status"], to)
				}
				wantContains(t, readAll(t, path), tr.log, "Log after "+tr.name)
			})
		}
	}
}

// TestUnparkIntakeReturnsToNew covers unpark's other prior status: an item
// parked in intake returns to new.
func TestUnparkIntakeReturnsToNew(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", 1, "p", withField(newFields("P"), "status", "parked"), "")
	wantExit(t, runIn(t, root, "backlog", "unpark", key(1)), 0)
	if fm := readFrontmatter(t, itemPath(root, "intake", 1, "p")); fm["status"] != "new" {
		t.Fatalf("status = %q, want new", fm["status"])
	}
}

// TestReleaseDropsClaim covers release / unblock --release clearing the claim.
func TestReleaseDropsClaim(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "r", claimedFields("R", "cli"), "")
	wantExit(t, runIn(t, root, "backlog", "release", key(1)), 0)
	fm := readFrontmatter(t, itemPath(root, "cli", 1, "r"))
	for _, k := range []string{"claimed-at", "claimed-by", "session"} {
		if _, ok := fm[k]; ok {
			t.Fatalf("release must drop %s: %v", k, fm)
		}
	}
	writeItem(t, root, "cli", 2, "b", withField(claimedFields("B", "cli"), "status", "blocked"), "")
	wantExit(t, runIn(t, root, "backlog", "unblock", key(2), "--release"), 0)
	if fm := readFrontmatter(t, itemPath(root, "cli", 2, "b")); fm["claimed-by"] != "" {
		t.Fatalf("unblock --release must drop the claim: %v", fm)
	}
}

// TestBlockKeepsNote covers §2: the block reason goes to the Log, note is
// untouched; unblock resumes the same claim.
func TestBlockKeepsNote(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "b", withField(claimedFields("B", "cli"), "note", "keep me"), "")
	wantExit(t, runIn(t, root, "backlog", "block", key(1), "escalated: q.md"), 0)
	path := itemPath(root, "cli", 1, "b")
	fm := readFrontmatter(t, path)
	if fm["note"] != "keep me" {
		t.Fatalf("block clobbered note: %q", fm["note"])
	}
	wantContains(t, readAll(t, path), "block by", "Log")
	wantContains(t, readAll(t, path), "escalated: q.md", "Log reason")
	wantExit(t, runIn(t, root, "backlog", "unblock", key(1)), 0)
	if fm := readFrontmatter(t, path); fm["status"] != "in-progress" || fm["claimed-by"] != "worker-a" {
		t.Fatalf("unblock: %v", fm)
	}
	wantExit(t, runIn(t, root, "backlog", "block", key(1)), 2)
}

// TestTriageAppliesFieldSet covers §6.3: hint stripped, triaged-at stamped,
// optional keys applied, --triage-note under ## Triage. The move into the
// workstream dir keeps the filename.
func TestTriageAppliesFieldSet(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", 1, "t", withField(newFields("T"), "hint", "dev"), "")
	r := runIn(t, root, "backlog", "triage", key(1), "workstream=dev", "summary=does a thing", "change-type=doc",
		"risk=additive", "verify=none", "disposition=design", "priority=high", "--triage-note", "needs a spec first")
	wantExit(t, r, 0)
	path := itemPath(root, "dev", 1, "t")
	fm := readFrontmatter(t, path)
	if fm["status"] != "triaged" || fm["workstream"] != "dev" || fm["priority"] != "high" ||
		fm["triaged-at"] == "" || fm["hint"] != "" || fm["disposition"] != "design" {
		t.Fatalf("triaged frontmatter = %v", fm)
	}
	body := readAll(t, path)
	if i, j := strings.Index(body, "## Triage"), strings.Index(body, "needs a spec first"); i < 0 || j < i {
		t.Fatalf("triage note not under ## Triage:\n%s", body)
	}
}

// TestTriageRefusals covers exit 3 for a missing field, an incoherent auto
// disposition, an unknown workstream, and status=/dispatch= keys.
func TestTriageRefusals(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 3\nkey-prefix: "+testPrefix+"\nsweep-order: bugs, cli, dev, process\nnever-auto-change-types: design\n---\n")
	writeItem(t, root, "intake", 1, "t", newFields("T"), "")
	full := []string{"workstream=cli", "summary=s", "change-type=doc", "risk=additive", "verify=go test", "disposition=auto"}
	cases := map[string][]string{
		"missing summary":  {"workstream=cli", "change-type=doc", "risk=additive", "verify=go test", "disposition=auto"},
		"auto none verify": append(append([]string{}, full[:4]...), "verify=none", "disposition=auto"),
		"auto never-auto":  {"workstream=cli", "summary=s", "change-type=design", "risk=additive", "verify=go test", "disposition=auto"},
		"bad workstream":   append(append([]string{}, full[1:]...), "workstream=ghost"),
		"status key":       append(append([]string{}, full...), "status=triaged"),
		"dispatch key":     append(append([]string{}, full...), "dispatch=approved"),
		"bad enum":         append(append([]string{}, full[:5]...), "disposition=AUTO"),
	}
	for name, pairs := range cases {
		r := runIn(t, root, append([]string{"backlog", "triage", key(1)}, pairs...)...)
		if r.exit != 3 {
			t.Fatalf("%s: exit = %d, want 3\n%s", name, r.exit, r.stderr)
		}
		wantFilePresent(t, itemPath(root, "intake", 1, "t"))
	}
}

// TestDeliverClaimantOnly covers §2/§7: only the claimant delivers; --force
// lets a human deliver on its behalf (logged as forced). Identity comes from
// --by or ANTHILL_ACTOR.
func TestDeliverClaimantOnly(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "d", claimedFields("D", "cli"), "")
	path := itemPath(root, "cli", 1, "d")
	wantExit(t, runIn(t, root, "backlog", "deliver", key(1), "--evidence", "x", "--by", "worker-b"), 6)
	wantExit(t, runEnv(t, root, []string{"ANTHILL_ACTOR=worker-b"}, "backlog", "deliver", key(1), "--evidence", "x"), 6)
	if fm := readFrontmatter(t, path); fm["status"] != "in-progress" {
		t.Fatalf("refused deliver changed status: %v", fm)
	}
	wantExit(t, runIn(t, root, "backlog", "deliver", key(1), "--evidence", "x", "--by", "worker-b", "--force"), 0)
	wantContains(t, readAll(t, path), "deliver (forced) by worker-b", "forced deliver Log")

	writeItem(t, root, "cli", 2, "e", claimedFields("E", "cli"), "")
	r := runEnv(t, root, []string{"ANTHILL_ACTOR=worker-a"}, "backlog", "deliver", key(2), "--evidence", "go test ./... exit 0")
	wantExit(t, r, 0)
	body := readAll(t, itemPath(root, "cli", 2, "e"))
	fm := readFrontmatter(t, itemPath(root, "cli", 2, "e"))
	if fm["status"] != "validation" || fm["delivered-at"] == "" {
		t.Fatalf("deliver frontmatter: %v", fm)
	}
	if i, j := strings.Index(body, "## Evidence"), strings.Index(body, "go test ./... exit 0"); i < 0 || j < i {
		t.Fatalf("evidence not under ## Evidence:\n%s", body)
	}
}

// TestDeliverEvidenceFileAndUsage covers --evidence-file and exit 2 without
// evidence.
func TestDeliverEvidenceFileAndUsage(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "d", claimedFields("D", "cli"), "")
	wantExit(t, runIn(t, root, "backlog", "deliver", key(1), "--by", "worker-a"), 2)
	ev := filepath.Join(t.TempDir(), "ev.txt")
	writeRaw(t, ev, "artifact: build.log\n")
	wantExit(t, runIn(t, root, "backlog", "deliver", key(1), "--by", "worker-a", "--evidence-file", ev), 0)
	wantContains(t, readAll(t, itemPath(root, "cli", 1, "d")), "artifact: build.log", "evidence file content")
}

// TestDispatchVerbs covers approve / fast-track / unapprove (§6.4): multiple
// ids, orthogonal to status, logged, refused on archived items, all-or-nothing
// on unknown ids.
func TestDispatchVerbs(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", 1, "a", newFields("A"), "")
	writeItem(t, root, "cli", 2, "b", withField(readyFields("B", "cli"), "dispatch", "proposed"), "")
	wantExit(t, runIn(t, root, "backlog", "approve", key(1), key(2)), 0)
	for _, p := range []string{itemPath(root, "intake", 1, "a"), itemPath(root, "cli", 2, "b")} {
		if fm := readFrontmatter(t, p); fm["dispatch"] != "approved" {
			t.Fatalf("%s dispatch = %q", p, fm["dispatch"])
		}
		wantContains(t, readAll(t, p), "approve by", "Log")
	}
	if fm := readFrontmatter(t, itemPath(root, "intake", 1, "a")); fm["status"] != "new" {
		t.Fatalf("approve must not change status: %v", fm)
	}
	wantExit(t, runIn(t, root, "backlog", "fast-track", key(1)), 0)
	if fm := readFrontmatter(t, itemPath(root, "intake", 1, "a")); fm["dispatch"] != "fast-tracked" {
		t.Fatalf("fast-track: %v", fm)
	}
	wantExit(t, runIn(t, root, "backlog", "unapprove", key(1)), 0)
	if fm := readFrontmatter(t, itemPath(root, "intake", 1, "a")); fm["dispatch"] != "proposed" {
		t.Fatalf("unapprove: %v", fm)
	}
	// Unknown (slug-shaped) id: nothing written.
	wantExit(t, runIn(t, root, "backlog", "approve", key(1), "ghost"), 4)
	if fm := readFrontmatter(t, itemPath(root, "intake", 1, "a")); fm["dispatch"] != "proposed" {
		t.Fatalf("approve with an unknown id must write nothing: %v", fm)
	}
	// Archived item: refused.
	writeAtStatus(t, root, 3, "old", "done")
	wantExit(t, runIn(t, root, "backlog", "approve", key(3)), 6)
	// Legacy --yes is accepted.
	wantExit(t, runIn(t, root, "backlog", "approve", key(1), "--yes"), 0)
	// --json is an array.
	r := runIn(t, root, "--json", "backlog", "fast-track", key(1), key(2))
	wantExit(t, r, 0)
	if arr := jsonArr(t, r.stdout); len(arr) != 2 || arr[0]["dispatch"] != "fast-tracked" {
		t.Fatalf("json = %s", r.stdout)
	}
}

// TestDispatchMakesReady covers derived ready: triaged + approved + auto + real
// verify, and its absence for each missing condition.
func TestDispatchMakesReady(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "r", withField(readyFields("R", "cli"), "dispatch", "proposed"), "")
	ready := func() any {
		r := runIn(t, root, "--json", "backlog", "show", key(1))
		wantExit(t, r, 0)
		return jsonObj(t, r.stdout)["ready"]
	}
	if ready() != false {
		t.Fatal("proposed item must not be ready")
	}
	wantExit(t, runIn(t, root, "backlog", "approve", key(1)), 0)
	if ready() != true {
		t.Fatal("approved auto item with verify must be ready")
	}
	wantExit(t, runIn(t, root, "backlog", "set", key(1), "disposition=interactive"), 0)
	if ready() != false {
		t.Fatal("interactive item must not be ready")
	}
}

// TestNewFlags covers --summary, --description(-file), --priority, --reporter
// and the reporter precedence (§4). A fresh tree allocates ANT-1, ANT-2, ….
func TestNewFlags(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "new", "--title", "Rich", "--value", "v", "--summary", "short",
		"--description", "the long brief", "--priority", "low", "--reporter", "supervisor")
	wantExit(t, r, 0)
	if got := strings.TrimSpace(r.stdout); got != key(1) {
		t.Fatalf("new stdout = %q, want %s", got, key(1))
	}
	path := itemPath(root, "intake", 1, "rich")
	fm := readFrontmatter(t, path)
	if fm["summary"] != "short" || fm["priority"] != "low" || fm["reporter"] != "supervisor" {
		t.Fatalf("new flags: %v", fm)
	}
	body := readAll(t, path)
	if i, j := strings.Index(body, "## Description"), strings.Index(body, "the long brief"); i < 0 || j < i {
		t.Fatalf("description not under ## Description:\n%s", body)
	}

	desc := filepath.Join(t.TempDir(), "d.md")
	writeRaw(t, desc, "from a file\n")
	wantExit(t, runIn(t, root, "backlog", "new", "--title", "Filed", "--value", "v", "--description-file", desc), 0)
	wantContains(t, readAll(t, itemPath(root, "intake", 2, "filed")), "from a file", "description file")
	wantExit(t, runIn(t, root, "backlog", "new", "--title", "X", "--value", "v", "--description", "a", "--description-file", desc), 2)
	wantExit(t, runIn(t, root, "backlog", "new", "--title", "X", "--value", "v", "--priority", "urgent"), 2)
	wantExit(t, runIn(t, root, "backlog", "new", "--title", "X", "--value", "v", "--reporter", "robot"), 2)

	for _, tc := range []struct {
		title string
		env   []string
		flag  string
		want  string
	}{
		{"Env reporter", []string{"ANTHILL_REPORTER=supervisor", "CLAUDECODE=1"}, "", "supervisor"},
		{"Inferred worker", []string{"CLAUDECODE=1"}, "", "worker"},
		{"Inferred user", nil, "", "user"},
		{"Flag wins", []string{"ANTHILL_REPORTER=supervisor"}, "user", "user"},
	} {
		args := []string{"--json", "backlog", "new", "--title", tc.title, "--value", "v"}
		if tc.flag != "" {
			args = append(args, "--reporter", tc.flag)
		}
		r := runEnv(t, root, tc.env, args...)
		wantExit(t, r, 0)
		if got := jsonObj(t, r.stdout)["reporter"]; got != tc.want {
			t.Fatalf("%s: reporter = %v, want %s", tc.title, got, tc.want)
		}
	}
}

// TestShowFindsArchivedItems covers show resolving ids in done/ and discarded/.
func TestShowFindsArchivedItems(t *testing.T) {
	root := mkTree(t)
	writeAtStatus(t, root, 1, "gone", "done")
	r := runIn(t, root, "--json", "backlog", "show", key(1))
	wantExit(t, r, 0)
	if obj := jsonObj(t, r.stdout); obj["status"] != "done" || obj["active"] != false {
		t.Fatalf("show archived = %v", obj)
	}
}
