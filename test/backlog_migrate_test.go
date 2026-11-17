package e2e_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyFixture copies testdata/<name> (a tree holding .anthill/) into a fresh
// temp root and returns it.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("testdata", name)
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture %s: %v", name, err)
	}
	if err := os.MkdirAll(filepath.Join(dst, ".anthill", "escalations"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dst
}

// snapshot returns path -> content for every file under root's backlog.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(backlogDir(root), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		m[p] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestMigrateRealItems runs migrate over this installation's real schema-1
// items (testdata/migrate-v1, copied from the parent .anthill/backlog).
func TestMigrateRealItems(t *testing.T) {
	root := copyFixture(t, "migrate-v1")
	before := snapshot(t, root)

	// The unstamped tree is refused by validate until migrated.
	wantExit(t, runIn(t, root, "backlog", "validate"), 6)

	// --dry-run reports but writes nothing.
	r := runIn(t, root, "--json", "backlog", "migrate", "--dry-run")
	wantExit(t, r, 0)
	dry := jsonObj(t, r.stdout)
	if dry["dry_run"] != true || len(dry["changed"].([]any)) != 5 || dry["schema_stamped"] != true {
		t.Fatalf("dry run = %s", r.stdout)
	}
	after := snapshot(t, root)
	for p, c := range before {
		if after[p] != c {
			t.Fatalf("--dry-run modified %s", p)
		}
	}

	r = runIn(t, root, "backlog", "migrate")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "skipped ", "migrate reports the frontmatter-less file")
	wantContains(t, readAll(t, filepath.Join(backlogDir(root), "workstreams.md")), "schema: 2", "schema stamp")
	// The skipped file is untouched.
	skipped := itemPath(root, "intake", "anthill-claude-language-sanitization-mod-to-defus")
	if readAll(t, skipped) != before[skipped] {
		t.Fatal("skipped file was modified")
	}

	// Intake idea → new, proposed, defaults stamped, skeleton seeded with value.
	p := itemPath(root, "intake", "doctor-don-t-warn-on-pristine-runtime-seed-files")
	fm := readFrontmatter(t, p)
	if fm["status"] != "new" || fm["dispatch"] != "proposed" || fm["priority"] != "normal" ||
		fm["reporter"] != "user" || fm["created-at"] == "" || fm["hint"] != "cli" {
		t.Fatalf("migrated intake item = %v", fm)
	}
	body := readAll(t, p)
	wantContains(t, body, "## Description\nFresh scaffolds permanently show", "description seeded with value")
	wantContains(t, body, "migrate schema 1 → 2", "Log")

	// In-progress workstream item keeps status and claim, gains dispatch approved.
	p = itemPath(root, "cli", "tui-text-wrap-in-panels")
	fm = readFrontmatter(t, p)
	if fm["status"] != "in-progress" || fm["dispatch"] != "approved" || fm["claimed-at"] != "2026-08-29T14:38:51Z" {
		t.Fatalf("migrated in-progress item = %v", fm)
	}

	// Idempotent: a second run changes nothing.
	mid := snapshot(t, root)
	r = runIn(t, root, "--json", "backlog", "migrate")
	wantExit(t, r, 0)
	again := jsonObj(t, r.stdout)
	if len(again["changed"].([]any)) != 0 || again["schema_stamped"] != false || again["unchanged"] != float64(5) {
		t.Fatalf("second migrate = %s", r.stdout)
	}
	end := snapshot(t, root)
	for p, c := range mid {
		if end[p] != c {
			t.Fatalf("second migrate modified %s", p)
		}
	}

	// validate now applies schema-2 rules and lists what triage still owes,
	// per item: no enum or unknown-key violations remain.
	r = runIn(t, root, "backlog", "validate")
	wantExit(t, r, 3)
	wantContains(t, r.stdout, "anthill-claude-language-sanitization-mod-to-defus: frontmatter: ", "skipped file listed")
	wantContains(t, r.stdout, "tui-text-wrap-in-panels: required: missing summary", "triage gap listed")
	for _, rule := range []string{": enum: ", ": unknown-key: ", ": location: "} {
		if strings.Contains(r.stdout, rule) {
			t.Fatalf("migration left a %q violation:\n%s", rule, r.stdout)
		}
	}
}

// TestMigrateMappingTable covers each §9 row on synthetic schema-1 items.
func TestMigrateMappingTable(t *testing.T) {
	root := mkTree(t)
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nsweep-order: bugs, cli, dev, process\nnever-auto-change-types: design\n---\n\n# Workstreams\n")
	v1 := func(status string, extra map[string]string) map[string]string {
		f := map[string]string{"workstream": "cli", "title": "T", "value": "the value", "status": status,
			"change-type": "tooling", "risk": "additive", "verify": "go test ./..."}
		for k, v := range extra {
			f[k] = v
		}
		return f
	}
	writeItem(t, root, "cli", "ws-idea", v1("idea", nil), "")
	writeItem(t, root, "cli", "approved", v1("approved", map[string]string{"disposition": "AUTO", "value-verdict": "ADVANCE — removes daily friction"}), "")
	writeItem(t, root, "cli", "expedited", v1("approved", map[string]string{"note": "expedited; remainder of x"}), "")
	writeItem(t, root, "cli", "needs-spec", v1("idea", map[string]string{"disposition": "AUTO", "note": "needs-spec"}), "")
	writeItem(t, root, "cli", "review-real", v1("idea", map[string]string{"disposition": "REVIEW"}), "")
	writeItem(t, root, "cli", "review-none", v1("idea", map[string]string{"disposition": "REVIEW", "verify": "none"}), "")
	writeItem(t, root, "cli", "review-never", v1("idea", map[string]string{"disposition": "REVIEW", "change-type": "design"}), "")
	writeItem(t, root, "cli", "discard", v1("idea", map[string]string{"disposition": "DISCARD"}), "")
	writeItem(t, root, "cli", "blocked", v1("blocked", map[string]string{"claimed-at": "2026-01-01T00:00:00Z"}), "")
	writeItem(t, root, "cli", "parked", v1("parked", nil), "")
	writeItem(t, root, "cli", "was-done", v1("done", nil), "")
	writeItem(t, root, "cli", "has-body", v1("idea", map[string]string{"value-verdict": "HOLD"}), "## Description\nhand written\n")

	r := runIn(t, root, "backlog", "migrate", "--default-reporter", "supervisor")
	wantExit(t, r, 0)

	want := map[string]map[string]string{
		"ws-idea":      {"status": "triaged", "dispatch": "proposed"},
		"approved":     {"status": "triaged", "dispatch": "approved", "disposition": "auto"},
		"expedited":    {"status": "triaged", "dispatch": "fast-tracked", "note": "remainder of x"},
		"needs-spec":   {"status": "triaged", "disposition": "design", "note": ""},
		"review-real":  {"disposition": "auto"},
		"review-none":  {"disposition": "design"},
		"review-never": {"disposition": "design"},
		"discard":      {"disposition": "open", "dispatch": "proposed", "status": "triaged"},
		"blocked":      {"status": "blocked", "dispatch": "approved"},
		"parked":       {"status": "parked", "dispatch": "approved"},
		"has-body":     {"reporter": "supervisor", "priority": "normal"},
	}
	for id, kv := range want {
		fm := readFrontmatter(t, itemPath(root, "cli", id))
		for k, v := range kv {
			if fm[k] != v {
				t.Fatalf("%s: %s = %q, want %q (%v)", id, k, fm[k], v, fm)
			}
		}
		if _, ok := fm["value-verdict"]; ok {
			t.Fatalf("%s still carries value-verdict", id)
		}
	}
	wantContains(t, readAll(t, itemPath(root, "cli", "approved")), "## Triage\nremoves daily friction", "verdict rationale")
	hb := readAll(t, itemPath(root, "cli", "has-body"))
	wantContains(t, hb, "## Description\nhand written", "existing prose preserved")
	wantContains(t, hb, "## Triage\nHOLD", "bare verdict kept")
	wantFileGone(t, itemPath(root, "cli", "was-done"))
	if fm := readFrontmatter(t, itemPath(root, "done", "was-done")); fm["status"] != "done" || fm["closed-at"] == "" {
		t.Fatalf("done item: %v", fm)
	}
}

// TestMigrateSkipsUnknownKeys covers the no-data-loss rule: a file carrying a
// key outside both schemas is reported and left untouched.
func TestMigrateSkipsUnknownKeys(t *testing.T) {
	root := mkTree(t)
	writeItem(t, root, "intake", "odd", map[string]string{"title": "O", "value": "v", "status": "idea", "owner": "max"}, "")
	before := readAll(t, itemPath(root, "intake", "odd"))
	r := runIn(t, root, "--json", "backlog", "migrate")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	sk := obj["skipped"].([]any)
	if len(sk) != 1 || !strings.Contains(sk[0].(map[string]any)["reason"].(string), "owner") {
		t.Fatalf("skipped = %v", obj["skipped"])
	}
	if readAll(t, itemPath(root, "intake", "odd")) != before {
		t.Fatal("file with unknown keys was modified")
	}
}

// TestPrune covers `backlog prune`: deletes discarded items (all, or older
// than a duration), files a CHANGELOG line each, never touches done/.
func TestPrune(t *testing.T) {
	root := mkTree(t)
	writeAtStatus(t, root, "old-drop", "discarded") // closed 2026-11-05
	writeAtStatus(t, root, "kept-done", "done")
	writeItem(t, root, "intake", "fresh", newFields("Fresh"), "")
	wantExit(t, runIn(t, root, "backlog", "close", "fresh", "--discard", "nah"), 0) // closed now

	r := runIn(t, root, "--json", "backlog", "prune", "--older-than", "1d", "--dry-run")
	wantExit(t, r, 0)
	if obj := jsonObj(t, r.stdout); len(obj["pruned"].([]any)) != 1 || obj["kept"] != float64(1) {
		t.Fatalf("dry-run = %s", r.stdout)
	}
	wantFilePresent(t, itemPath(root, "discarded", "old-drop"))

	wantExit(t, runIn(t, root, "backlog", "prune", "--older-than", "1d"), 0)
	wantFileGone(t, itemPath(root, "discarded", "old-drop"))
	wantFilePresent(t, itemPath(root, "discarded", "fresh"))
	wantContains(t, readAll(t, changelogPath(root)), "old-drop — purged", "CHANGELOG")

	wantExit(t, runIn(t, root, "backlog", "prune"), 0)
	wantFileGone(t, itemPath(root, "discarded", "fresh"))
	wantFilePresent(t, itemPath(root, "done", "kept-done"))
	wantExit(t, runIn(t, root, "backlog", "prune", "--older-than", "soon"), 2)
}
