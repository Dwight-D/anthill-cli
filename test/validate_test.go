package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateClean covers `backlog validate` on a well-formed tree, including
// archived items: exit 0, json ok=true.
func TestValidateClean(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "good-item", readyFields("Good item", "cli"), "body")
	writeItem(t, root, "intake", "good-idea", newFields("Good idea"), "body")
	writeAtStatus(t, root, "good-done", "done")
	writeAtStatus(t, root, "good-gone", "discarded")
	for _, st := range []string{"in-progress", "validation", "blocked", "parked"} {
		writeAtStatus(t, root, "good-"+st, st)
	}

	r := runIn(t, root, "--json", "backlog", "validate")
	wantExit(t, r, 0)
	if obj := jsonObj(t, r.stdout); obj["ok"] != true || obj["checked"] != float64(8) {
		t.Fatalf("clean tree: %s", r.stdout)
	}
}

// violationCase seeds one rule violation into a fresh tree.
type violationCase struct {
	rule string
	id   string
	seed func(t *testing.T, root string)
}

var violationCases = []violationCase{
	{"frontmatter", "no-fm", func(t *testing.T, root string) {
		writeRaw(t, itemPath(root, "intake", "no-fm"), "## Description\nno frontmatter\n")
	}},
	{"unknown-key", "legacy-key", func(t *testing.T, root string) {
		writeItem(t, root, "cli", "legacy-key", withField(readyFields("L", "cli"), "value-verdict", "ADVANCE"), "")
	}},
	{"location", "misplaced", func(t *testing.T, root string) {
		writeItem(t, root, "cli", "misplaced", readyFields("M", "dev"), "")
	}},
	{"location", "wrong-dir-status", func(t *testing.T, root string) {
		writeItem(t, root, "intake", "wrong-dir-status", withField(newFields("W"), "status", "in-progress"), "")
	}},
	{"location", "undone", func(t *testing.T, root string) {
		writeAtStatus(t, root, "undone", "done")
		p := itemPath(root, "done", "undone")
		writeRaw(t, p, strings.Replace(readAll(t, p), "status: done", "status: validation", 1))
	}},
	{"enum", "bad-enum", func(t *testing.T, root string) {
		writeItem(t, root, "cli", "bad-enum", withField(readyFields("B", "cli"), "risk", "totally-bogus"), "")
	}},
	{"enum", "old-status", func(t *testing.T, root string) {
		writeItem(t, root, "intake", "old-status", withField(newFields("O"), "status", "idea"), "")
	}},
	{"required", "no-summary", func(t *testing.T, root string) {
		writeItem(t, root, "cli", "no-summary", withField(readyFields("N", "cli"), "summary", ""), "")
	}},
	{"required", "no-created", func(t *testing.T, root string) {
		writeItem(t, root, "intake", "no-created", withField(newFields("N"), "created-at", ""), "")
	}},
	{"required", "no-delivered", func(t *testing.T, root string) {
		writeItem(t, root, "cli", "no-delivered", withField(validationFields("N", "cli"), "delivered-at", ""), "")
	}},
	{"disposition-coherence", "auto-none", func(t *testing.T, root string) {
		writeItem(t, root, "cli", "auto-none", withField(readyFields("A", "cli"), "verify", "none"), "")
	}},
	{"hint-stripped", "hinted", func(t *testing.T, root string) {
		writeItem(t, root, "cli", "hinted", withField(readyFields("H", "cli"), "hint", "dev"), "")
	}},
	{"id-unique", "dup-id", func(t *testing.T, root string) {
		writeItem(t, root, "cli", "dup-id", readyFields("D1", "cli"), "")
		writeItem(t, root, "dev", "dup-id", readyFields("D2", "dev"), "")
	}},
	{"stray-file", "notes.txt", func(t *testing.T, root string) {
		writeRaw(t, filepath.Join(backlogDir(root), "cli", "notes.txt"), "x")
	}},
}

// TestValidateListsEveryViolation seeds one violation per rule into one tree
// and asserts that `backlog validate`, `anthill validate`, and `doctor` each
// list every one as "<id>: <rule>: <message>" (never a bare count), and that
// the --json forms carry the same violations array.
func TestValidateListsEveryViolation(t *testing.T) {
	root := scaffoldFresh(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 2\nsweep-order: bugs, cli, dev, process\n---\n")
	for _, d := range []string{"cli", "dev", "process", "bugs"} {
		if err := os.MkdirAll(filepath.Join(backlogDir(root), d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Keep the scaffold's product dir out of the sweep-order check's way.
	for _, c := range violationCases {
		c.seed(t, root)
	}
	outputs := map[string]result{
		"backlog validate": runIn(t, root, "backlog", "validate"),
		"validate":         runIn(t, root, "validate"),
		"doctor":           runIn(t, root, "doctor"),
	}
	for name, r := range outputs {
		wantExit(t, r, 3)
		for _, c := range violationCases {
			if !strings.Contains(r.stdout, c.id+": "+c.rule+": ") {
				t.Fatalf("%s does not list %q: %q\n%s", name, c.id, c.rule, r.stdout)
			}
		}
	}

	var jsonCounts []int
	for _, args := range [][]string{{"backlog", "validate"}, {"validate"}, {"doctor"}} {
		r := runIn(t, root, append([]string{"--json"}, args...)...)
		wantExit(t, r, 3)
		obj := jsonObj(t, r.stdout)
		viols, ok := obj["violations"].([]any)
		if !ok {
			t.Fatalf("%v --json has no violations array: %s", args, r.stdout)
		}
		for _, c := range violationCases {
			found := false
			for _, v := range viols {
				m := v.(map[string]any)
				if m["id"] == c.id && m["rule"] == c.rule && m["message"] != "" {
					found = true
				}
			}
			if !found {
				t.Fatalf("%v --json lacks %s/%s", args, c.id, c.rule)
			}
		}
		jsonCounts = append(jsonCounts, len(viols))
	}
	if jsonCounts[0] != jsonCounts[1] || jsonCounts[0] != jsonCounts[2] {
		t.Fatalf("validate and doctor disagree on the rule set: counts %v", jsonCounts)
	}
}

// TestValidateStrictPromotesWarnings covers --strict: an out-of-vocabulary
// change-type warns by default and fails under --strict.
func TestValidateStrictPromotesWarnings(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 2\nsweep-order: bugs, cli, dev, process\nchange-types: doc\n---\n")
	writeItem(t, root, "cli", "odd", readyFields("Odd", "cli"), "")
	r := runIn(t, root, "backlog", "validate")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "warn: odd: change-type-vocab: ", "validate warning")
	wantExit(t, runIn(t, root, "backlog", "validate", "--strict"), 3)
}

// TestValidateRequiresSchemaStamp covers the migration gate: a tree with items
// but no `schema: 2` stamp is refused (exit 6) with a pointer to migrate, and
// doctor fails backlog-valid with the same pointer.
func TestValidateRequiresSchemaStamp(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nsweep-order: bugs, cli, dev, process\n---\n")
	writeItem(t, root, "intake", "old", withField(newFields("Old"), "status", "idea"), "")
	r := runIn(t, root, "backlog", "validate")
	wantExit(t, r, 6)
	wantContains(t, r.stderr, "anthill backlog migrate", "validate stderr")
	wantExit(t, runIn(t, root, "validate"), 6)
	r = runIn(t, root, "doctor")
	wantExit(t, r, 3)
	wantContains(t, r.stdout, "anthill backlog migrate", "doctor stdout")
}

// TestValidateTopLevelAlias covers `anthill validate` on a clean tree.
func TestValidateTopLevelAlias(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", "alias-good", readyFields("Alias good", "cli"), "body")
	wantExit(t, runIn(t, root, "validate"), 0)
}

// TestNewOnFreshTreeStampsSchema covers the empty-tree rule: a scaffold has no
// items and no stamp; the first `new` stamps `schema: 2` so validate applies.
func TestNewOnFreshTreeStampsSchema(t *testing.T) {
	root := scaffoldFresh(t)
	wantExit(t, runIn(t, root, "backlog", "validate"), 0)
	wantExit(t, runIn(t, root, "backlog", "new", "--title", "First", "--value", "v"), 0)
	wantContains(t, readAll(t, filepath.Join(backlogDir(root), "workstreams.md")), "schema: 2", "workstreams.md")
	wantExit(t, runIn(t, root, "backlog", "validate"), 0)
}
