package e2e_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
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

// snapshot returns path -> content for every file under root's .anthill
// (backlog, sessions ledger, escalations).
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(filepath.Join(root, ".anthill"), func(p string, d fs.DirEntry, err error) error {
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

// showPath resolves ref through `backlog show --json` and returns the item's
// id and path.
func showPath(t *testing.T, root, ref string) (id, path string) {
	t.Helper()
	r := runIn(t, root, "--json", "backlog", "show", ref, "--no-body")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	return obj["id"].(string), obj["path"].(string)
}

// TestMigrateRealItems runs migrate over this installation's real schema-1
// items (testdata/migrate-v1, copied from the parent .anthill/backlog): the
// 1 → 2 field mapping followed by the 2 → 3 keying, ending at schema 3.
func TestMigrateRealItems(t *testing.T) {
	root := copyFixture(t, "migrate-v1")
	before := snapshot(t, root)

	// The unstamped tree is refused by validate until migrated.
	wantExit(t, runIn(t, root, "backlog", "validate"), 6)

	// Without a key prefix anywhere the migration cannot key the items.
	r := runIn(t, root, "backlog", "migrate", "--dry-run")
	wantExit(t, r, 6)
	wantContains(t, r.stderr, "key-prefix", "missing prefix message")

	// --dry-run reports but writes nothing: 5 field mappings + 5 keyings.
	r = runIn(t, root, "--json", "backlog", "migrate", "--dry-run", "--key-prefix", "ANT")
	wantExit(t, r, 0)
	dry := jsonObj(t, r.stdout)
	if dry["dry_run"] != true || len(dry["changed"].([]any)) != 10 || dry["schema_stamped"] != true || dry["key_prefix_stamped"] != true {
		t.Fatalf("dry run = %s", r.stdout)
	}
	after := snapshot(t, root)
	for p, c := range before {
		if after[p] != c {
			t.Fatalf("--dry-run modified %s", p)
		}
	}

	r = runIn(t, root, "backlog", "migrate", "--key-prefix", "ANT")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "skipped ", "migrate reports the frontmatter-less file")
	ws := readAll(t, filepath.Join(backlogDir(root), "workstreams.md"))
	wantContains(t, ws, "schema: 3", "schema stamp")
	wantContains(t, ws, "key-prefix: ANT", "prefix stamp")
	// The skipped file is untouched (and keeps its slug name).
	skipped := filepath.Join(backlogDir(root), "intake", "anthill-claude-language-sanitization-mod-to-defus.md")
	if readAll(t, skipped) != before[skipped] {
		t.Fatal("skipped file was modified")
	}

	// Intake idea → new, proposed, defaults stamped, skeleton seeded with
	// value; the item is now keyed and resolves by its former id.
	id, p := showPath(t, root, "doctor-don-t-warn-on-pristine-runtime-seed-files")
	if !strings.HasPrefix(id, "ANT-") || !strings.HasSuffix(p, "-doctor-don-t-warn-on-pristine-runtime.md") {
		t.Fatalf("keyed item = %s at %s (slug tail capped at 40 chars on a dash boundary)", id, p)
	}
	fm := readFrontmatter(t, p)
	if fm["status"] != "new" || fm["dispatch"] != "proposed" || fm["priority"] != "normal" ||
		fm["reporter"] != "user" || fm["created-at"] == "" || fm["hint"] != "cli" ||
		fm["former-id"] != "doctor-don-t-warn-on-pristine-runtime-seed-files" {
		t.Fatalf("migrated intake item = %v", fm)
	}
	body := readAll(t, p)
	wantContains(t, body, "## Description\nFresh scaffolds permanently show", "description seeded with value")
	wantContains(t, body, "migrate schema 1 → 2", "Log")
	wantContains(t, body, "migrate schema 2 → 3 by anthill: doctor-don-t-warn-on-pristine-runtime-seed-files → "+id, "Log")

	// In-progress workstream item keeps status and claim, gains dispatch approved.
	_, p = showPath(t, root, "tui-text-wrap-in-panels")
	fm = readFrontmatter(t, p)
	if fm["status"] != "in-progress" || fm["dispatch"] != "approved" || fm["claimed-at"] != "2026-08-29T14:38:51Z" {
		t.Fatalf("migrated in-progress item = %v", fm)
	}
	wantContains(t, readAll(t, filepath.Join(backlogDir(root), "sequence")), "5", "sequence high-water mark")

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

	// validate now applies the schema rules and lists what triage still owes,
	// per item: no enum or unknown-key violations remain.
	r = runIn(t, root, "backlog", "validate")
	wantExit(t, r, 3)
	wantContains(t, r.stdout, "anthill-claude-language-sanitization-mod-to-defus.md: filename-form: ", "skipped file listed")
	wantContains(t, r.stdout, ": required: missing summary", "triage gap listed")
	for _, rule := range []string{": enum: ", ": unknown-key: ", ": location: "} {
		if strings.Contains(r.stdout, rule) {
			t.Fatalf("migration left a %q violation:\n%s", rule, r.stdout)
		}
	}
}

// TestMigrateSchema2ToKeys runs the 2 → 3 pass over this installation's
// schema-2 items (testdata/migrate-v2, copied from the parent
// .anthill/backlog plus a ledger record and an escalation naming old ids):
// numbering in created-at order, former-id, sequence, reference rewrites,
// idempotency (ITEM_KEYS_SPEC.md §3.5).
func TestMigrateSchema2ToKeys(t *testing.T) {
	root := copyFixture(t, "migrate-v2")
	before := snapshot(t, root)
	wantExit(t, runIn(t, root, "backlog", "validate"), 6)
	wantExit(t, runIn(t, root, "backlog", "new", "--title", "x", "--value", "v"), 6)

	r := runIn(t, root, "backlog", "migrate", "--dry-run")
	wantExit(t, r, 6)
	wantContains(t, r.stderr, "migrate --key-prefix", "missing prefix fix")

	// created-at ascending, then old id ascending.
	wantOrder := []string{
		"doctor-don-t-warn-on-pristine-runtime-seed-files",
		"doctor-backlog-valid-reports-violations-without",
		"tui-free-text-escalation-answering",
		"tui-text-wrap-in-panels",
		"anthill-claude-in-session-now-pane-showing-active",
		"release-ps1-aborts-after-git-push-under",
		"backlog-item-keys-prefix-n-ids-ant-74-with-file",
		"validation-instructions-when-submitting-to",
		"framework-item-keys-ant-74-in-skills-templates",
	}
	r = runIn(t, root, "backlog", "migrate", "--dry-run", "--key-prefix", "TST")
	wantExit(t, r, 0)
	lines := strings.Split(strings.TrimSpace(r.stdout), "\n")
	for i, old := range wantOrder {
		want := old + " → TST-" + itoa(i+1)
		if i >= len(lines) || lines[i] != want {
			t.Fatalf("dry-run line %d = %q, want %q\nfull:\n%s", i, lineAt(lines, i), want, r.stdout)
		}
	}
	wantContains(t, r.stdout, "rewrite sessions/abcd1234-sess.json: item framework-item-keys-ant-74-in-skills-templates → TST-9", "ledger rewrite listed")
	wantContains(t, r.stdout, "rewrite escalations/2026-11-17-should-panels-wrap.md: item tui-text-wrap-in-panels → TST-4", "escalation rewrite listed")
	wantContains(t, r.stderr, "would stamp schema: 3, key-prefix: TST", "stamps announced")
	for p, c := range before {
		if readAll(t, p) != c {
			t.Fatalf("--dry-run modified %s", p)
		}
	}

	// A mismatching prefix is a usage error once one is stamped; first the
	// real run.
	r = runIn(t, root, "backlog", "migrate", "--key-prefix", "TST")
	wantExit(t, r, 0)
	wantExit(t, runIn(t, root, "backlog", "migrate", "--key-prefix", "ZZZ"), 2)

	ws := readAll(t, filepath.Join(backlogDir(root), "workstreams.md"))
	wantContains(t, ws, "schema: 3", "schema stamp")
	wantContains(t, ws, "key-prefix: TST", "prefix stamp")
	if got := strings.TrimSpace(readAll(t, filepath.Join(backlogDir(root), "sequence"))); got != "9" {
		t.Fatalf("sequence = %q, want 9", got)
	}
	for i, old := range wantOrder {
		n := i + 1
		id, p := showPath(t, root, old)
		if id != "TST-"+itoa(n) {
			t.Fatalf("%s resolved to %s, want TST-%d", old, id, n)
		}
		if _, ok := strings.CutPrefix(filepath.Base(p), itoaPad(n)+"-"); !ok {
			t.Fatalf("%s filed as %s", id, p)
		}
		fm := readFrontmatter(t, p)
		if fm["former-id"] != old {
			t.Fatalf("%s former-id = %q", id, fm["former-id"])
		}
		wantContains(t, readAll(t, p), "migrate schema 2 → 3 by anthill: "+old+" → "+id, "Log line")
		// Every accepted reference form resolves.
		if got, _ := showPath(t, root, itoa(n)); got != id {
			t.Fatalf("show %d = %s", n, got)
		}
		if got, _ := showPath(t, root, "tst-"+itoa(n)); got != id {
			t.Fatalf("show tst-%d = %s", n, got)
		}
	}
	// Archive placement survives: the done item stays in done/.
	if _, p := showPath(t, root, "9"); filepath.Base(filepath.Dir(p)) != "done" {
		t.Fatalf("done item moved to %s", p)
	}
	// Slug tails are capped at 40 characters.
	if _, p := showPath(t, root, "9"); filepath.Base(p) != "0009-framework-item-keys-ant-74-in-skills.md" {
		t.Fatalf("truncated slug = %s", filepath.Base(p))
	}
	// Old slug files are gone.
	for _, old := range wantOrder {
		matches, _ := filepath.Glob(filepath.Join(backlogDir(root), "*", old+".md"))
		if len(matches) != 0 {
			t.Fatalf("old slug file still present: %v", matches)
		}
	}
	// References rewritten: ledger item and the escalation's item: line only.
	wantContains(t, readAll(t, filepath.Join(root, ".anthill", "sessions", "abcd1234-sess.json")), `"item": "TST-9"`, "ledger item")
	escPath := filepath.Join(escalDir(root), "2026-11-17-should-panels-wrap.md")
	esc := readAll(t, escPath)
	wantContains(t, esc, "item: TST-4\n", "escalation item")
	if strings.Replace(before[escPath], "item: tui-text-wrap-in-panels", "item: TST-4", 1) != esc {
		t.Fatalf("escalation record changed beyond its item line:\n%s", esc)
	}
	wantContains(t, readAll(t, filepath.Join(backlogDir(root), "CHANGELOG.md")), "", "changelog untouched")
	if readAll(t, filepath.Join(backlogDir(root), "CHANGELOG.md")) != before[filepath.Join(backlogDir(root), "CHANGELOG.md")] {
		t.Fatal("CHANGELOG.md must be left as history")
	}

	// Idempotent: a second run reports nothing and changes nothing.
	mid := snapshot(t, root)
	r = runIn(t, root, "backlog", "migrate")
	wantExit(t, r, 0)
	if strings.TrimSpace(r.stdout) != "" {
		t.Fatalf("second migrate stdout = %q, want empty", r.stdout)
	}
	wantContains(t, r.stderr, "migrated 0 item(s), 9 already on schema 3", "second run summary")
	for p, c := range mid {
		if readAll(t, p) != c {
			t.Fatalf("second migrate modified %s", p)
		}
	}

	// The keyed tree validates without key or filename violations, and new
	// allocation continues after the sequence.
	r = runIn(t, root, "backlog", "validate")
	for _, rule := range []string{"filename-form", "key-prefix", "id-unique", "former-id"} {
		if strings.Contains(r.stdout, rule) {
			t.Fatalf("validate reports %s after migrate:\n%s", rule, r.stdout)
		}
	}
	if got := newID(t, root, "After migration"); got != "TST-10" {
		t.Fatalf("new after migrate = %s, want TST-10", got)
	}
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

func itoa(n int) string { return strconv.Itoa(n) }

// itoaPad is the 4-digit zero-padded form of n (an item filename stem).
func itoaPad(n int) string { return fmt.Sprintf("%04d", n) }

// TestMigrateMappingTable covers each §9 row on synthetic schema-1 items,
// which the run also keys (ITEM_KEYS_SPEC.md §3.5); items are located by
// their former id afterwards.
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
	slugItem := func(slug string, fields map[string]string, body string) {
		var b strings.Builder
		b.WriteString("---\n")
		for k, v := range fields {
			b.WriteString(k + ": " + v + "\n")
		}
		b.WriteString("---\n" + body)
		writeRaw(t, filepath.Join(backlogDir(root), "cli", slug+".md"), b.String())
	}
	slugItem("ws-idea", v1("idea", nil), "")
	slugItem("approved", v1("approved", map[string]string{"disposition": "AUTO", "value-verdict": "ADVANCE — removes daily friction"}), "")
	slugItem("expedited", v1("approved", map[string]string{"note": "expedited; remainder of x"}), "")
	slugItem("needs-spec", v1("idea", map[string]string{"disposition": "AUTO", "note": "needs-spec"}), "")
	slugItem("review-real", v1("idea", map[string]string{"disposition": "REVIEW"}), "")
	slugItem("review-none", v1("idea", map[string]string{"disposition": "REVIEW", "verify": "none"}), "")
	slugItem("review-never", v1("idea", map[string]string{"disposition": "REVIEW", "change-type": "design"}), "")
	slugItem("discard", v1("idea", map[string]string{"disposition": "DISCARD"}), "")
	slugItem("blocked", v1("blocked", map[string]string{"claimed-at": "2026-01-01T00:00:00Z"}), "")
	slugItem("parked", v1("parked", nil), "")
	slugItem("was-done", v1("done", nil), "")
	slugItem("has-body", v1("idea", map[string]string{"value-verdict": "HOLD"}), "## Description\nhand written\n")

	r := runIn(t, root, "backlog", "migrate", "--default-reporter", "supervisor", "--key-prefix", "ANT")
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
	paths := map[string]string{}
	for former := range want {
		_, paths[former] = showPath(t, root, former)
	}
	for former, kv := range want {
		fm := readFrontmatter(t, paths[former])
		for k, v := range kv {
			if fm[k] != v {
				t.Fatalf("%s: %s = %q, want %q (%v)", former, k, fm[k], v, fm)
			}
		}
		if _, ok := fm["value-verdict"]; ok {
			t.Fatalf("%s still carries value-verdict", former)
		}
		if fm["former-id"] != former {
			t.Fatalf("%s: former-id = %q", former, fm["former-id"])
		}
	}
	wantContains(t, readAll(t, paths["approved"]), "## Triage\nremoves daily friction", "verdict rationale")
	hb := readAll(t, paths["has-body"])
	wantContains(t, hb, "## Description\nhand written", "existing prose preserved")
	wantContains(t, hb, "## Triage\nHOLD", "bare verdict kept")
	wantFileGone(t, filepath.Join(backlogDir(root), "cli", "was-done.md"))
	_, donePath := showPath(t, root, "was-done")
	if fm := readFrontmatter(t, donePath); fm["status"] != "done" || fm["closed-at"] == "" || filepath.Base(filepath.Dir(donePath)) != "done" {
		t.Fatalf("done item: %v at %s", fm, donePath)
	}
}

// TestMigrateSkipsUnknownKeys covers the no-data-loss rule: a file carrying a
// key outside both schemas is reported and left untouched (and not keyed).
func TestMigrateSkipsUnknownKeys(t *testing.T) {
	root := mkTree(t)
	path := filepath.Join(backlogDir(root), "intake", "odd.md")
	writeRaw(t, path, "---\ntitle: O\nvalue: v\nstatus: idea\nowner: max\n---\n")
	writeRaw(t, filepath.Join(backlogDir(root), "workstreams.md"),
		"---\nschema: 1\nkey-prefix: ANT\nsweep-order: bugs, cli, dev, process\n---\n")
	before := readAll(t, path)
	r := runIn(t, root, "--json", "backlog", "migrate")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	sk := obj["skipped"].([]any)
	if len(sk) != 1 || !strings.Contains(sk[0].(map[string]any)["reason"].(string), "owner") {
		t.Fatalf("skipped = %v", obj["skipped"])
	}
	if readAll(t, path) != before {
		t.Fatal("file with unknown keys was modified")
	}
}

// TestPrune covers `backlog prune`: deletes discarded items (all, or older
// than a duration), files a CHANGELOG line each, never touches done/.
func TestPrune(t *testing.T) {
	root := mkTree(t)
	writeAtStatus(t, root, 1, "old-drop", "discarded") // closed 2026-11-05
	writeAtStatus(t, root, 2, "kept-done", "done")
	writeItem(t, root, "intake", 3, "fresh", newFields("Fresh"), "")
	wantExit(t, runIn(t, root, "backlog", "close", "3", "--discard", "nah"), 0) // closed now

	r := runIn(t, root, "--json", "backlog", "prune", "--older-than", "1d", "--dry-run")
	wantExit(t, r, 0)
	if obj := jsonObj(t, r.stdout); len(obj["pruned"].([]any)) != 1 || obj["kept"] != float64(1) {
		t.Fatalf("dry-run = %s", r.stdout)
	}
	wantFilePresent(t, itemPath(root, "discarded", 1, "old-drop"))

	wantExit(t, runIn(t, root, "backlog", "prune", "--older-than", "1d"), 0)
	wantFileGone(t, itemPath(root, "discarded", 1, "old-drop"))
	wantFilePresent(t, itemPath(root, "discarded", 3, "fresh"))
	wantContains(t, readAll(t, changelogPath(root)), key(1)+" — purged", "CHANGELOG")

	wantExit(t, runIn(t, root, "backlog", "prune"), 0)
	wantFileGone(t, itemPath(root, "discarded", 3, "fresh"))
	wantFilePresent(t, itemPath(root, "done", 2, "kept-done"))
	wantExit(t, runIn(t, root, "backlog", "prune", "--older-than", "soon"), 2)

	// Pruned numbers are never reused: the sequence remembers them.
	if got := newID(t, root, "Next"); got != key(4) {
		t.Fatalf("new after prune = %s, want %s", got, key(4))
	}
}
