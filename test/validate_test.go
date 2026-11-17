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
	writeItem(t, root, "cli", 1, "good-item", readyFields("Good item", "cli"), "body")
	writeItem(t, root, "intake", 2, "good-idea", newFields("Good idea"), "body")
	writeAtStatus(t, root, 3, "good-done", "done")
	writeAtStatus(t, root, 4, "good-gone", "discarded")
	for i, st := range []string{"in-progress", "validation", "blocked", "parked"} {
		writeAtStatus(t, root, 5+i, "good-"+st, st)
	}

	r := runIn(t, root, "--json", "backlog", "validate")
	wantExit(t, r, 0)
	if obj := jsonObj(t, r.stdout); obj["ok"] != true || obj["checked"] != float64(8) {
		t.Fatalf("clean tree: %s", r.stdout)
	}
}

// violationCase seeds one rule violation into a fresh tree. id is the
// violation ID validate reports: the item key for item rules, the file name
// for file rules.
type violationCase struct {
	rule string
	id   string
	seed func(t *testing.T, root string)
}

var violationCases = []violationCase{
	{"frontmatter", key(1), func(t *testing.T, root string) {
		writeRaw(t, itemPath(root, "intake", 1, "no-fm"), "## Description\nno frontmatter\n")
	}},
	{"unknown-key", key(2), func(t *testing.T, root string) {
		writeItem(t, root, "cli", 2, "legacy-key", withField(readyFields("L", "cli"), "value-verdict", "ADVANCE"), "")
	}},
	{"location", key(3), func(t *testing.T, root string) {
		writeItem(t, root, "cli", 3, "misplaced", readyFields("M", "dev"), "")
	}},
	{"location", key(4), func(t *testing.T, root string) {
		writeItem(t, root, "intake", 4, "wrong-dir-status", withField(newFields("W"), "status", "in-progress"), "")
	}},
	{"location", key(5), func(t *testing.T, root string) {
		writeAtStatus(t, root, 5, "undone", "done")
		p := itemPath(root, "done", 5, "undone")
		writeRaw(t, p, strings.Replace(readAll(t, p), "status: done", "status: validation", 1))
	}},
	{"enum", key(6), func(t *testing.T, root string) {
		writeItem(t, root, "cli", 6, "bad-enum", withField(readyFields("B", "cli"), "risk", "totally-bogus"), "")
	}},
	{"enum", key(7), func(t *testing.T, root string) {
		writeItem(t, root, "intake", 7, "old-status", withField(newFields("O"), "status", "idea"), "")
	}},
	{"required", key(8), func(t *testing.T, root string) {
		writeItem(t, root, "cli", 8, "no-summary", withField(readyFields("N", "cli"), "summary", ""), "")
	}},
	{"required", key(9), func(t *testing.T, root string) {
		writeItem(t, root, "intake", 9, "no-created", withField(newFields("N"), "created-at", ""), "")
	}},
	{"required", key(10), func(t *testing.T, root string) {
		writeItem(t, root, "cli", 10, "no-delivered", withField(validationFields("N", "cli"), "delivered-at", ""), "")
	}},
	{"disposition-coherence", key(11), func(t *testing.T, root string) {
		writeItem(t, root, "cli", 11, "auto-none", withField(readyFields("A", "cli"), "verify", "none"), "")
	}},
	{"hint-stripped", key(12), func(t *testing.T, root string) {
		writeItem(t, root, "cli", 12, "hinted", withField(readyFields("H", "cli"), "hint", "dev"), "")
	}},
	{"id-unique", key(13), func(t *testing.T, root string) {
		writeItem(t, root, "cli", 13, "dup-id", readyFields("D1", "cli"), "")
		writeItem(t, root, "dev", 13, "dup-id", readyFields("D2", "dev"), "")
	}},
	{"stray-file", "notes.txt", func(t *testing.T, root string) {
		writeRaw(t, filepath.Join(backlogDir(root), "cli", "notes.txt"), "x")
	}},
	{"filename-form", "loose-slug.md", func(t *testing.T, root string) {
		writeItem(t, root, "cli", 14, "loose-slug", readyFields("Loose", "cli"), "")
		mustRename(t, itemPath(root, "cli", 14, "loose-slug"), filepath.Join(backlogDir(root), "cli", "loose-slug.md"))
	}},
}

func mustRename(t *testing.T, from, to string) {
	t.Helper()
	if err := os.Rename(from, to); err != nil {
		t.Fatalf("rename %s -> %s: %v", from, to, err)
	}
}

// TestValidateListsEveryViolation seeds one violation per rule into one tree
// and asserts that `backlog validate`, `anthill validate`, and `doctor` each
// list every one as "<id>: <rule>: <message>" (never a bare count), and that
// the --json forms carry the same violations array.
func TestValidateListsEveryViolation(t *testing.T) {
	root := scaffoldFresh(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 3\nkey-prefix: "+testPrefix+"\nsweep-order: bugs, cli, dev, process\n---\n")
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

// TestValidateDuplicateNumberAcrossDirs covers id-unique keyed on the number:
// the same number in a workstream dir and in done/ is one violation whose ID
// is the key (exit 3).
func TestValidateDuplicateNumberAcrossDirs(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 3, "dup-live", readyFields("Dup live", "cli"), "")
	if err := os.MkdirAll(filepath.Join(backlogDir(root), "done"), 0o755); err != nil {
		t.Fatal(err)
	}
	done := withField(validationFields("Dup done", "cli"), "status", "done")
	done["closed-at"] = "2026-11-05T00:00:00Z"
	writeItem(t, root, "done", 3, "dup-done", done, "")

	r := runIn(t, root, "backlog", "validate")
	wantExit(t, r, 3)
	wantContains(t, r.stdout, key(3)+": id-unique", "validate stdout")
}

// TestValidateFilenameForm covers filename-form: a well-formed item whose file
// name is not NNNN-<slug>.md is reported under its file name.
func TestValidateFilenameForm(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "loose-slug", readyFields("Loose slug", "cli"), "body")
	mustRename(t, itemPath(root, "cli", 1, "loose-slug"), filepath.Join(backlogDir(root), "cli", "loose-slug.md"))

	r := runIn(t, root, "backlog", "validate")
	wantExit(t, r, 3)
	wantContains(t, r.stdout, "loose-slug.md: filename-form", "validate stdout")
}

// TestValidateKeyPrefixMissing covers key-prefix-missing: a schema-3 tree
// holding items whose workstreams.md has no key-prefix line is reported under
// workstreams.md.
func TestValidateKeyPrefixMissing(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 3\nsweep-order: bugs, cli, dev, process\n---\n")
	writeItem(t, root, "cli", 1, "no-prefix", readyFields("No prefix", "cli"), "body")

	r := runIn(t, root, "backlog", "validate")
	wantExit(t, r, 3)
	wantContains(t, r.stdout, "workstreams.md: key-prefix-missing", "validate stdout")
}

// TestValidateStrictPromotesWarnings covers --strict: an out-of-vocabulary
// change-type warns by default and fails under --strict.
func TestValidateStrictPromotesWarnings(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 3\nkey-prefix: "+testPrefix+"\nsweep-order: bugs, cli, dev, process\nchange-types: doc\n---\n")
	writeItem(t, root, "cli", 1, "odd", readyFields("Odd", "cli"), "")
	r := runIn(t, root, "backlog", "validate")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "warn: "+key(1)+": change-type-vocab: ", "validate warning")
	wantExit(t, runIn(t, root, "backlog", "validate", "--strict"), 3)
}

// TestValidateRequiresSchemaStamp covers the migration gate: a tree with items
// but no `schema: 3` stamp (unstamped, or stamped 2) is refused (exit 6) with
// a pointer to migrate, and doctor fails backlog-valid with the same pointer.
func TestValidateRequiresSchemaStamp(t *testing.T) {
	for name, fm := range map[string]string{
		"unstamped": "---\nsweep-order: bugs, cli, dev, process\n---\n",
		"schema-2":  "---\nschema: 2\nsweep-order: bugs, cli, dev, process\n---\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := mkTree(t)
			writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"), fm)
			writeItem(t, root, "intake", 1, "old", withField(newFields("Old"), "status", "idea"), "")
			r := runIn(t, root, "backlog", "validate")
			wantExit(t, r, 6)
			wantContains(t, r.stderr, "not on schema 3", "validate stderr")
			wantContains(t, r.stderr, "anthill backlog migrate", "validate stderr")
			wantExit(t, runIn(t, root, "validate"), 6)
			r = runIn(t, root, "doctor")
			wantExit(t, r, 3)
			wantContains(t, r.stdout, "anthill backlog migrate", "doctor stdout")
		})
	}
}

// TestValidateTopLevelAlias covers `anthill validate` on a clean tree.
func TestValidateTopLevelAlias(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "cli", 1, "alias-good", readyFields("Alias good", "cli"), "body")
	wantExit(t, runIn(t, root, "validate"), 0)
}

// TestNewOnFreshTreeStampsSchema covers the scaffold rule: the template ships
// `schema: 3` with a `<PREFIX>` placeholder for key-prefix, so an empty tree
// validates, `new` refuses until a prefix is set, and `init --force` fills
// the placeholder in place; the first `new` then allocates under it.
func TestNewOnFreshTreeStampsSchema(t *testing.T) {
	root := scaffoldFresh(t)
	wantExit(t, runIn(t, root, "backlog", "validate"), 0)
	r := runIn(t, root, "backlog", "new", "--title", "First", "--value", "v")
	wantExit(t, r, 6)
	wantContains(t, r.stderr, "key-prefix", "new stderr")
	wantExit(t, runIn(t, root, "init", "--force", "--key-prefix", "TST"), 0)
	r = runIn(t, root, "backlog", "new", "--title", "First", "--value", "v")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "TST-1", "new stdout")
	fm := readFrontmatter(t, filepath.Join(backlogDir(root), "workstreams.md"))
	if fm["schema"] != "3" || fm["key-prefix"] != "TST" {
		t.Fatalf("init did not fill schema 3 + key-prefix: %v", fm)
	}
	wantExit(t, runIn(t, root, "backlog", "validate"), 0)
}
