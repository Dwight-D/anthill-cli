package e2e_test

import "testing"

// seedMixed seeds a tree with one untriaged idea, one approved (ready) cli
// item, and one non-ready (idea) cli item.
func seedMixed(t *testing.T) string {
	root := mkTree(t)
	writeItem(t, root, "intake", "an-idea", ideaFields("An idea"), "body")
	writeItem(t, root, "cli", "ready-one", approvedFields("Ready one", "cli"), "body")
	notReady := approvedFields("Not ready", "cli")
	notReady["status"] = "idea"
	writeItem(t, root, "cli", "not-ready", notReady, "body")
	return root
}

// TestListJSONArray covers spec §7 Q5: --json list is a JSON ARRAY of item
// objects, each with a derived ready field. Parsed, not string-matched.
func TestListJSONArray(t *testing.T) {
	root := seedMixed(t)
	r := runIn(t, root, "--json", "backlog", "list")
	wantExit(t, r, 0)

	arr := jsonArr(t, r.stdout)
	if len(arr) != 3 {
		t.Fatalf("list --json length = %d, want 3\npayload: %s", len(arr), r.stdout)
	}
	byID := map[string]map[string]any{}
	for _, it := range arr {
		id, _ := it["id"].(string)
		byID[id] = it
	}
	for _, id := range []string{"an-idea", "ready-one", "not-ready"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("list missing item %q: %s", id, r.stdout)
		}
	}
	if byID["ready-one"]["ready"] != true {
		t.Fatalf("ready-one should have ready=true, got %v", byID["ready-one"]["ready"])
	}
	if byID["an-idea"]["ready"] != false {
		t.Fatalf("an-idea should have ready=false, got %v", byID["an-idea"]["ready"])
	}
}

// TestListReadyFilter covers --ready (only approved + non-empty verify).
func TestListReadyFilter(t *testing.T) {
	root := seedMixed(t)
	r := runIn(t, root, "--json", "backlog", "list", "--ready")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 1 || arr[0]["id"] != "ready-one" {
		t.Fatalf("--ready should return only ready-one, got: %s", r.stdout)
	}
}

// TestListUntriagedFilter covers --untriaged (only intake items, no workstream).
func TestListUntriagedFilter(t *testing.T) {
	root := seedMixed(t)
	r := runIn(t, root, "--json", "backlog", "list", "--untriaged")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 1 || arr[0]["id"] != "an-idea" {
		t.Fatalf("--untriaged should return only an-idea, got: %s", r.stdout)
	}
}

// TestListStatusFilter covers --status filtering by lifecycle status.
func TestListStatusFilter(t *testing.T) {
	root := seedMixed(t)
	r := runIn(t, root, "--json", "backlog", "list", "--status", "approved")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 1 || arr[0]["id"] != "ready-one" {
		t.Fatalf("--status approved should return only ready-one, got: %s", r.stdout)
	}
}

// TestListWorkstreamFilter covers --workstream scoping.
func TestListWorkstreamFilter(t *testing.T) {
	root := seedMixed(t)
	r := runIn(t, root, "--json", "backlog", "list", "--workstream", "cli")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 2 {
		t.Fatalf("--workstream cli should return 2 items, got %d: %s", len(arr), r.stdout)
	}
}

// TestListUnknownWorkstream covers exit 4 when --workstream names a
// non-existent workstream (spec §3.2).
func TestListUnknownWorkstream(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "list", "--workstream", "ghoststream")
	wantExit(t, r, 4)
}

// TestListEmptyExitZero confirms an empty result set is exit 0, not an error.
func TestListEmptyExitZero(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "--json", "backlog", "list")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 0 {
		t.Fatalf("empty tree should list 0 items, got %d", len(arr))
	}
}
