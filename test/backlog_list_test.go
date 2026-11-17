package e2e_test

import "testing"

// seedMixed seeds a tree with one untriaged idea (ANT-1), one approved (ready)
// cli item (ANT-2), and one non-ready (proposed) cli item (ANT-3).
func seedMixed(t *testing.T) string {
	root := mkTree(t)
	writeItem(t, root, "intake", 1, "an-idea", newFields("An idea"), "body")
	writeItem(t, root, "cli", 2, "ready-one", readyFields("Ready one", "cli"), "body")
	notReady := readyFields("Not ready", "cli")
	notReady["dispatch"] = "proposed"
	writeItem(t, root, "cli", 3, "not-ready", notReady, "body")
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
	for _, id := range []string{key(1), key(2), key(3)} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("list missing item %q: %s", id, r.stdout)
		}
	}
	if byID[key(2)]["ready"] != true {
		t.Fatalf("%s (ready-one) should have ready=true, got %v", key(2), byID[key(2)]["ready"])
	}
	if byID[key(1)]["ready"] != false {
		t.Fatalf("%s (an-idea) should have ready=false, got %v", key(1), byID[key(1)]["ready"])
	}
}

// TestListReadyFilter covers --ready (only approved + non-empty verify).
func TestListReadyFilter(t *testing.T) {
	root := seedMixed(t)
	r := runIn(t, root, "--json", "backlog", "list", "--ready")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 1 || arr[0]["id"] != key(2) {
		t.Fatalf("--ready should return only %s (ready-one), got: %s", key(2), r.stdout)
	}
}

// TestListUntriagedFilter covers --untriaged (only intake items, no workstream).
func TestListUntriagedFilter(t *testing.T) {
	root := seedMixed(t)
	r := runIn(t, root, "--json", "backlog", "list", "--untriaged")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 1 || arr[0]["id"] != key(1) {
		t.Fatalf("--untriaged should return only %s (an-idea), got: %s", key(1), r.stdout)
	}
}

// TestListStatusFilter covers --status filtering by lifecycle status.
func TestListStatusFilter(t *testing.T) {
	root := seedMixed(t)
	r := runIn(t, root, "--json", "backlog", "list", "--status", "new")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 1 || arr[0]["id"] != key(1) {
		t.Fatalf("--status new should return only %s (an-idea), got: %s", key(1), r.stdout)
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

// TestListSortIDNumeric covers --sort id comparing keys by number, not as
// strings: ANT-2 lists before ANT-10.
func TestListSortIDNumeric(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 10, "ten", readyFields("Ten", "cli"), "body")
	writeItem(t, root, "cli", 2, "two", readyFields("Two", "cli"), "body")
	r := runIn(t, root, "--json", "backlog", "list", "--sort", "id")
	wantExit(t, r, 0)
	arr := jsonArr(t, r.stdout)
	if len(arr) != 2 || arr[0]["id"] != key(2) || arr[1]["id"] != key(10) {
		t.Fatalf("--sort id should order %s before %s, got: %s", key(2), key(10), r.stdout)
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
