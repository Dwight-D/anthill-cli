package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Item numbers of the seedQuery fixture. The numbers are chosen so that
// numeric id order differs from rank order and spans a two-digit key
// (ANT-10), which a string sort would misplace before ANT-2.
const (
	qRaw      = 1  // intake/raw         new, proposed, reporter worker, high, created 01
	qCliNew   = 2  // cli/cli-new        triaged, approved, normal, created 03
	qCliOld   = 3  // cli/cli-old        triaged, approved, normal, created 01
	qBugOld   = 4  // bugs/bug-old       triaged, approved, normal, created 02
	qFtLow    = 6  // cli/ft-low         triaged, fast-tracked, low, created 05
	qDesign   = 7  // dev/design-item    triaged, approved, design (not ready), normal
	qWorking  = 8  // cli/working        in-progress, claimed-by worker-a, session sess-1
	qChecking = 9  // dev/checking       validation, claimed-by worker-b, session sess-2
	qHi       = 10 // cli/hi             triaged, approved, high, created 04
	qFinished = 11 // done/finished      done (archive)
	qDropped  = 12 // discarded/dropped  discarded (archive)
)

// seedQuery seeds a tree whose items differ along every list axis (see the
// number table above).
func seedQuery(t *testing.T) string {
	t.Helper()
	root := mkTree(t)
	at := func(f map[string]string, created string) map[string]string {
		f["created-at"] = "2026-11-" + created + "T00:00:00Z"
		return f
	}
	raw := at(newFields("Raw idea about widgets"), "01")
	raw["reporter"] = "worker"
	raw["priority"] = "high"
	writeItem(t, root, "intake", qRaw, "raw", raw, "")
	writeItem(t, root, "cli", qFtLow, "ft-low", at(withField(withField(readyFields("Ft low", "cli"), "dispatch", "fast-tracked"), "priority", "low"), "05"), "")
	writeItem(t, root, "cli", qHi, "hi", at(withField(readyFields("Hi", "cli"), "priority", "high"), "04"), "")
	writeItem(t, root, "bugs", qBugOld, "bug-old", at(readyFields("Bug old", "bugs"), "02"), "")
	writeItem(t, root, "cli", qCliOld, "cli-old", at(readyFields("Cli old", "cli"), "01"), "")
	writeItem(t, root, "cli", qCliNew, "cli-new", at(readyFields("Cli new", "cli"), "03"), "")
	writeItem(t, root, "dev", qDesign, "design-item", at(withField(withField(readyFields("Design item", "dev"), "disposition", "design"), "risk", "subjective"), "02"), "")
	writeItem(t, root, "cli", qWorking, "working", at(claimedFields("Working", "cli"), "02"), "")
	checking := at(validationFields("Checking", "dev"), "02")
	checking["claimed-by"] = "worker-b"
	checking["session"] = "sess-2"
	writeItem(t, root, "dev", qChecking, "checking", checking, "")
	for _, d := range []string{"done", "discarded"} {
		if err := os.MkdirAll(filepath.Join(backlogDir(root), d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fin := withField(validationFields("Finished", "cli"), "status", "done")
	fin["closed-at"] = "2026-11-06T00:00:00Z"
	writeItem(t, root, "done", qFinished, "finished", fin, "")
	drop := withField(newFields("Dropped"), "status", "discarded")
	drop["closed-at"] = "2026-11-06T00:00:00Z"
	writeItem(t, root, "discarded", qDropped, "dropped", drop, "")
	return root
}

// listIDs runs `list --json` with args and returns the ids in order.
func listIDs(t *testing.T, root string, args ...string) []string {
	t.Helper()
	r := runIn(t, root, append([]string{"--json", "backlog", "list"}, args...)...)
	wantExit(t, r, 0)
	var ids []string
	for _, o := range jsonArr(t, r.stdout) {
		ids = append(ids, o["id"].(string))
	}
	return ids
}

func wantIDs(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ids = %v, want %v", got, want)
	}
}

func wantSet(t *testing.T, got []string, want ...string) {
	t.Helper()
	g := map[string]bool{}
	for _, id := range got {
		g[id] = true
	}
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want set %v", got, want)
	}
	for _, id := range want {
		if !g[id] {
			t.Fatalf("ids = %v, want set %v", got, want)
		}
	}
}

// TestListRankOrder covers the default sort: fast-tracked > priority >
// workstream sweep-order (bugs, cli, dev, process; intake last) > oldest.
func TestListRankOrder(t *testing.T) {
	root := seedQuery(t)
	wantIDs(t, listIDs(t, root, "--ready"), key(qFtLow), key(qHi), key(qBugOld), key(qCliOld), key(qCliNew))
	wantIDs(t, listIDs(t, root, "--ready", "--sort", "rank"), key(qFtLow), key(qHi), key(qBugOld), key(qCliOld), key(qCliNew))
	wantIDs(t, listIDs(t, root, "--ready", "--reverse"), key(qCliNew), key(qCliOld), key(qBugOld), key(qHi), key(qFtLow))
	// Intake ranks after every workstream at equal tier: raw is high priority
	// and ranks just after hi.
	got := listIDs(t, root)
	if got[0] != key(qFtLow) || got[1] != key(qHi) || got[2] != key(qRaw) {
		t.Fatalf("full rank order = %v", got)
	}
}

// TestListOtherSorts covers --sort priority|age|updated|id and --limit.
// --sort id compares numerically: ANT-10 (hi) lists last, after ANT-6.
func TestListOtherSorts(t *testing.T) {
	root := seedQuery(t)
	wantIDs(t, listIDs(t, root, "--ready", "--sort", "priority"), key(qHi), key(qBugOld), key(qCliOld), key(qCliNew), key(qFtLow))
	wantIDs(t, listIDs(t, root, "--ready", "--sort", "age"), key(qCliOld), key(qBugOld), key(qCliNew), key(qHi), key(qFtLow))
	wantIDs(t, listIDs(t, root, "--ready", "--sort", "id"), key(qCliNew), key(qCliOld), key(qBugOld), key(qFtLow), key(qHi))
	wantIDs(t, listIDs(t, root, "--active", "--sort", "updated"), key(qChecking), key(qWorking))
	wantIDs(t, listIDs(t, root, "--ready", "--limit", "2"), key(qFtLow), key(qHi))
	wantExit(t, runIn(t, root, "backlog", "list", "--sort", "sweep"), 2)
	wantExit(t, runIn(t, root, "backlog", "list", "--limit", "-1"), 2)
}

// TestListFilters covers each repeatable filter and their AND/OR semantics.
func TestListFilters(t *testing.T) {
	root := seedQuery(t)
	wantSet(t, listIDs(t, root, "--status", "in-progress", "--status", "validation"), key(qWorking), key(qChecking))
	wantSet(t, listIDs(t, root, "--status", "in-progress,validation"), key(qWorking), key(qChecking))
	wantSet(t, listIDs(t, root, "--dispatch", "fast-tracked"), key(qFtLow))
	wantSet(t, listIDs(t, root, "--disposition", "design"), key(qDesign))
	wantSet(t, listIDs(t, root, "--reporter", "worker"), key(qRaw))
	wantSet(t, listIDs(t, root, "--priority", "high"), key(qHi), key(qRaw))
	wantSet(t, listIDs(t, root, "--priority", "high", "--reporter", "user"), key(qHi))
	wantSet(t, listIDs(t, root, "--workstream", "dev"), key(qDesign), key(qChecking))
	wantSet(t, listIDs(t, root, "--risk", "subjective"), key(qDesign))
	wantSet(t, listIDs(t, root, "--change-type", "tooling", "--workstream", "bugs"), key(qBugOld))
	wantSet(t, listIDs(t, root, "--claimed-by", "worker-b"), key(qChecking))
	wantSet(t, listIDs(t, root, "--session", "sess-1"), key(qWorking))
	wantSet(t, listIDs(t, root, "--search", "WIDGET"), key(qRaw))
	wantSet(t, listIDs(t, root, "--search", "summary of hi"), key(qHi))
	wantExit(t, runIn(t, root, "backlog", "list", "--status", "idea"), 2)
	wantExit(t, runIn(t, root, "backlog", "list", "--workstream", "ghost"), 4)
}

// TestListPresets covers --ready, --active, --untriaged, --all and the
// archive-skipping default.
func TestListPresets(t *testing.T) {
	root := seedQuery(t)
	wantSet(t, listIDs(t, root, "--active"), key(qWorking), key(qChecking))
	wantSet(t, listIDs(t, root, "--untriaged"), key(qRaw))
	all := listIDs(t, root)
	for _, id := range all {
		if id == key(qFinished) || id == key(qDropped) {
			t.Fatalf("default list must skip archives: %v", all)
		}
	}
	if got := listIDs(t, root, "--all"); len(got) != len(all)+2 {
		t.Fatalf("--all = %v", got)
	}
	wantSet(t, listIDs(t, root, "--status", "done"), key(qFinished))
	wantSet(t, listIDs(t, root, "--all", "--status", "discarded"), key(qDropped))
}

// TestListFields covers --fields for the human table and the json projection.
func TestListFields(t *testing.T) {
	root := seedQuery(t)
	r := runIn(t, root, "--json", "backlog", "list", "--active", "--fields", "id,summary,claimed-by,session,claimed-at,ready")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 2 {
		t.Fatalf("json = %s", r.stdout)
	}
	for _, o := range arr {
		if len(o) != 6 || o["claimed-by"] == "" || o["ready"] != false {
			t.Fatalf("projected object = %v", o)
		}
		if _, ok := o["title"]; ok {
			t.Fatalf("unrequested key present: %v", o)
		}
	}
	r = runIn(t, root, "backlog", "list", "--active", "--fields", "id,claimed-by,session")
	wantExit(t, r, 0)
	lines := strings.Split(strings.TrimSpace(r.stdout), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "ID") || !strings.Contains(lines[0], "CLAIMED-BY") ||
		!strings.Contains(r.stdout, "worker-b") || strings.Contains(lines[0], "STATUS") {
		t.Fatalf("table = %q", r.stdout)
	}
	wantExit(t, runIn(t, root, "backlog", "list", "--fields", "id,bogus"), 2)
}

// TestListSpecExamples runs the §6.2 example queries.
func TestListSpecExamples(t *testing.T) {
	root := seedQuery(t)
	wantIDs(t, listIDs(t, root, "--ready"), key(qFtLow), key(qHi), key(qBugOld), key(qCliOld), key(qCliNew))
	wantSet(t, listIDs(t, root, "--reporter", "user", "--dispatch", "fast-tracked"), key(qFtLow))
	wantSet(t, listIDs(t, root, "--disposition", "interactive", "--dispatch", "approved"))
	wantSet(t, listIDs(t, root, "--disposition", "design", "--sort", "priority"), key(qDesign))
}

// TestNextUsesRank covers `next` choosing the highest-ranked ready item and
// claim --next taking the same one.
func TestNextUsesRank(t *testing.T) {
	root := seedQuery(t)
	r := runIn(t, root, "--json", "backlog", "next")
	wantExit(t, r, 0)
	if id := jsonObj(t, r.stdout)["id"]; id != key(qFtLow) {
		t.Fatalf("next = %v, want %s (ft-low: fast-tracked beats priority)", id, key(qFtLow))
	}
	r = runIn(t, root, "--json", "backlog", "next", "--workstream", "bugs")
	wantExit(t, r, 0)
	if id := jsonObj(t, r.stdout)["id"]; id != key(qBugOld) {
		t.Fatalf("next --workstream bugs = %v", id)
	}
	wantExit(t, runIn(t, root, "backlog", "claim", "--next", "--by", "w"), 0)
	if fm := readFrontmatter(t, itemPath(root, "cli", qFtLow, "ft-low")); fm["status"] != "in-progress" {
		t.Fatalf("claim --next took the wrong item: %v", fm)
	}
}
