package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedQuery seeds a tree whose items differ along every list axis:
//
//	intake/raw         new, proposed, reporter worker, high, created 01
//	cli/ft-low         triaged, fast-tracked, low, created 05
//	cli/hi             triaged, approved, high, created 04
//	bugs/bug-old       triaged, approved, normal, created 02
//	cli/cli-old        triaged, approved, normal, created 01
//	cli/cli-new        triaged, approved, normal, created 03
//	dev/design-item    triaged, approved, design (not ready), normal
//	cli/working        in-progress, claimed-by worker-a, session sess-1
//	dev/checking       validation, claimed-by worker-b, session sess-2
//	done/finished      done (archive)
//	discarded/dropped  discarded (archive)
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
	writeItem(t, root, "intake", "raw", raw, "")
	writeItem(t, root, "cli", "ft-low", at(withField(withField(readyFields("Ft low", "cli"), "dispatch", "fast-tracked"), "priority", "low"), "05"), "")
	writeItem(t, root, "cli", "hi", at(withField(readyFields("Hi", "cli"), "priority", "high"), "04"), "")
	writeItem(t, root, "bugs", "bug-old", at(readyFields("Bug old", "bugs"), "02"), "")
	writeItem(t, root, "cli", "cli-old", at(readyFields("Cli old", "cli"), "01"), "")
	writeItem(t, root, "cli", "cli-new", at(readyFields("Cli new", "cli"), "03"), "")
	writeItem(t, root, "dev", "design-item", at(withField(withField(readyFields("Design item", "dev"), "disposition", "design"), "risk", "subjective"), "02"), "")
	writeItem(t, root, "cli", "working", at(claimedFields("Working", "cli"), "02"), "")
	checking := at(validationFields("Checking", "dev"), "02")
	checking["claimed-by"] = "worker-b"
	checking["session"] = "sess-2"
	writeItem(t, root, "dev", "checking", checking, "")
	for _, d := range []string{"done", "discarded"} {
		if err := os.MkdirAll(filepath.Join(backlogDir(root), d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fin := withField(validationFields("Finished", "cli"), "status", "done")
	fin["closed-at"] = "2026-11-06T00:00:00Z"
	writeItem(t, root, "done", "finished", fin, "")
	drop := withField(newFields("Dropped"), "status", "discarded")
	drop["closed-at"] = "2026-11-06T00:00:00Z"
	writeItem(t, root, "discarded", "dropped", drop, "")
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
	wantIDs(t, listIDs(t, root, "--ready"), "ft-low", "hi", "bug-old", "cli-old", "cli-new")
	wantIDs(t, listIDs(t, root, "--ready", "--sort", "rank"), "ft-low", "hi", "bug-old", "cli-old", "cli-new")
	wantIDs(t, listIDs(t, root, "--ready", "--reverse"), "cli-new", "cli-old", "bug-old", "hi", "ft-low")
	// Intake ranks after every workstream at equal tier: raw is high priority
	// and ranks just after hi.
	got := listIDs(t, root)
	if got[0] != "ft-low" || got[1] != "hi" || got[2] != "raw" {
		t.Fatalf("full rank order = %v", got)
	}
}

// TestListOtherSorts covers --sort priority|age|updated|id and --limit.
func TestListOtherSorts(t *testing.T) {
	root := seedQuery(t)
	wantIDs(t, listIDs(t, root, "--ready", "--sort", "priority"), "hi", "bug-old", "cli-old", "cli-new", "ft-low")
	wantIDs(t, listIDs(t, root, "--ready", "--sort", "age"), "cli-old", "bug-old", "cli-new", "hi", "ft-low")
	wantIDs(t, listIDs(t, root, "--ready", "--sort", "id"), "bug-old", "cli-new", "cli-old", "ft-low", "hi")
	wantIDs(t, listIDs(t, root, "--active", "--sort", "updated"), "checking", "working")
	wantIDs(t, listIDs(t, root, "--ready", "--limit", "2"), "ft-low", "hi")
	wantExit(t, runIn(t, root, "backlog", "list", "--sort", "sweep"), 2)
	wantExit(t, runIn(t, root, "backlog", "list", "--limit", "-1"), 2)
}

// TestListFilters covers each repeatable filter and their AND/OR semantics.
func TestListFilters(t *testing.T) {
	root := seedQuery(t)
	wantSet(t, listIDs(t, root, "--status", "in-progress", "--status", "validation"), "working", "checking")
	wantSet(t, listIDs(t, root, "--status", "in-progress,validation"), "working", "checking")
	wantSet(t, listIDs(t, root, "--dispatch", "fast-tracked"), "ft-low")
	wantSet(t, listIDs(t, root, "--disposition", "design"), "design-item")
	wantSet(t, listIDs(t, root, "--reporter", "worker"), "raw")
	wantSet(t, listIDs(t, root, "--priority", "high"), "hi", "raw")
	wantSet(t, listIDs(t, root, "--priority", "high", "--reporter", "user"), "hi")
	wantSet(t, listIDs(t, root, "--workstream", "dev"), "design-item", "checking")
	wantSet(t, listIDs(t, root, "--risk", "subjective"), "design-item")
	wantSet(t, listIDs(t, root, "--change-type", "tooling", "--workstream", "bugs"), "bug-old")
	wantSet(t, listIDs(t, root, "--claimed-by", "worker-b"), "checking")
	wantSet(t, listIDs(t, root, "--session", "sess-1"), "working")
	wantSet(t, listIDs(t, root, "--search", "WIDGET"), "raw")
	wantSet(t, listIDs(t, root, "--search", "summary of hi"), "hi")
	wantExit(t, runIn(t, root, "backlog", "list", "--status", "idea"), 2)
	wantExit(t, runIn(t, root, "backlog", "list", "--workstream", "ghost"), 4)
}

// TestListPresets covers --ready, --active, --untriaged, --all and the
// archive-skipping default.
func TestListPresets(t *testing.T) {
	root := seedQuery(t)
	wantSet(t, listIDs(t, root, "--active"), "working", "checking")
	wantSet(t, listIDs(t, root, "--untriaged"), "raw")
	all := listIDs(t, root)
	for _, id := range all {
		if id == "finished" || id == "dropped" {
			t.Fatalf("default list must skip archives: %v", all)
		}
	}
	if got := listIDs(t, root, "--all"); len(got) != len(all)+2 {
		t.Fatalf("--all = %v", got)
	}
	wantSet(t, listIDs(t, root, "--status", "done"), "finished")
	wantSet(t, listIDs(t, root, "--all", "--status", "discarded"), "dropped")
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
	wantIDs(t, listIDs(t, root, "--ready"), "ft-low", "hi", "bug-old", "cli-old", "cli-new")
	wantSet(t, listIDs(t, root, "--reporter", "user", "--dispatch", "fast-tracked"), "ft-low")
	wantSet(t, listIDs(t, root, "--disposition", "interactive", "--dispatch", "approved"))
	wantSet(t, listIDs(t, root, "--disposition", "design", "--sort", "priority"), "design-item")
}

// TestNextUsesRank covers `next` choosing the highest-ranked ready item and
// claim --next taking the same one.
func TestNextUsesRank(t *testing.T) {
	root := seedQuery(t)
	r := runIn(t, root, "--json", "backlog", "next")
	wantExit(t, r, 0)
	if id := jsonObj(t, r.stdout)["id"]; id != "ft-low" {
		t.Fatalf("next = %v, want ft-low (fast-tracked beats priority)", id)
	}
	r = runIn(t, root, "--json", "backlog", "next", "--workstream", "bugs")
	wantExit(t, r, 0)
	if id := jsonObj(t, r.stdout)["id"]; id != "bug-old" {
		t.Fatalf("next --workstream bugs = %v", id)
	}
	wantExit(t, runIn(t, root, "backlog", "claim", "--next", "--by", "w"), 0)
	if fm := readFrontmatter(t, itemPath(root, "cli", "ft-low")); fm["status"] != "in-progress" {
		t.Fatalf("claim --next took the wrong item: %v", fm)
	}
}
