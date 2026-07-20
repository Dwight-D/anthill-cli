package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// scaffoldPayloadPaths are representative install-time paths the mechanical
// scaffold must write (spec §3 payload, §4.2).
func scaffoldPayloadPaths(root string) []string {
	return []string{
		skillPath(root, "autonomous"),
		filepath.Join(root, ".anthill", "backlog", "workstreams.md"),
		claudeTemplatePath(root),
		filepath.Join(root, "tools"),
		filepath.Join(root, ".gitignore"),
	}
}

// TestScaffoldWritesPayload covers `anthill scaffold` into a git repo (spec
// §4.2): the embedded template payload lands on disk, exit 0.
func TestScaffoldWritesPayload(t *testing.T) {
	dir := newGitRepo(t)
	r := runInDir(t, dir, "scaffold")
	wantExit(t, r, 0)

	for _, p := range scaffoldPayloadPaths(dir) {
		wantFilePresent(t, p)
	}
	// tools/ must carry the launcher scripts.
	wantFilePresent(t, filepath.Join(dir, "tools", "supervise.sh"))
	wantFilePresent(t, filepath.Join(dir, "tools", "supervise.ps1"))
}

// TestScaffoldJSONShape covers the manifest shape (spec §4.2): { written,
// skipped, refused, ref }. A first scaffold writes the payload and stamps the
// embedded ref.
func TestScaffoldJSONShape(t *testing.T) {
	dir := newGitRepo(t)
	r := runInDir(t, dir, "--json", "scaffold")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)

	for _, k := range []string{"written", "skipped", "refused", "ref"} {
		wantHasKey(t, obj, k)
	}
	if ref, _ := obj["ref"].(string); ref != templateRef(t) {
		t.Fatalf("scaffold ref = %q, want embedded template_ref %q", obj["ref"], templateRef(t))
	}
	// A fresh scaffold writes the skills; assert one representative entry.
	wantListContains(t, obj, "written", filepath.Join(".claude", "skills", "autonomous"))
}

// TestScaffoldIdempotent covers the non-destructive convergence rule (spec
// §4.2): a second scaffold over an identical install writes nothing, skips
// everything, exit 0.
func TestScaffoldIdempotent(t *testing.T) {
	dir := scaffoldFresh(t)

	before := snapshotTree(t, dir)
	r := runInDir(t, dir, "--json", "scaffold")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)

	if written := jsonStrings(obj["written"]); len(written) != 0 {
		t.Fatalf("second scaffold wrote files, want none: %v", written)
	}
	if skipped := jsonStrings(obj["skipped"]); len(skipped) == 0 {
		t.Fatalf("second scaffold skipped nothing, want all payload skipped")
	}
	after := snapshotTree(t, dir)
	if before != after {
		t.Fatalf("idempotent scaffold changed the tree on disk")
	}
}

// TestScaffoldRefusesDiffering covers the refuse rule (spec §4.2): a
// pre-existing file that differs from the template is refused (exit 3, listed
// in refused, original preserved), and --force overwrites it.
func TestScaffoldRefusesDiffering(t *testing.T) {
	dir := newGitRepo(t)
	// Plant a conflicting CLAUDE.template.md before scaffolding.
	conflict := "MY OWN ALWAYS-ON FILE — do not clobber\n"
	writeRaw(t, claudeTemplatePath(dir), conflict)

	r := runInDir(t, dir, "--json", "scaffold")
	wantExit(t, r, 3)
	obj := jsonObj(t, r.stdout)
	wantListContains(t, obj, "refused", "CLAUDE.template.md")

	// Original content preserved (not overwritten).
	if got := readAll(t, claudeTemplatePath(dir)); got != conflict {
		t.Fatalf("refused file was modified\n got: %q\nwant: %q", got, conflict)
	}

	// --force overwrites the differing file.
	rf := runInDir(t, dir, "scaffold", "--force")
	wantExit(t, rf, 0)
	pristine := scaffoldFresh(t) // an untouched reference install
	wantSameFile(t, claudeTemplatePath(dir), claudeTemplatePath(pristine))
}

// TestScaffoldDryRun covers --dry-run (spec §4.2): compute the manifest, write
// nothing, exit 0.
func TestScaffoldDryRun(t *testing.T) {
	dir := newGitRepo(t)
	before := snapshotTree(t, dir)

	r := runInDir(t, dir, "scaffold", "--dry-run")
	wantExit(t, r, 0)

	after := snapshotTree(t, dir)
	if before != after {
		t.Fatalf("--dry-run modified the tree")
	}
	// The install payload must NOT have been written.
	if fileExists(t, skillPath(dir, "autonomous")) {
		t.Fatalf("--dry-run wrote payload files")
	}
}

// TestScaffoldOutsideGitRepo covers the precondition (spec §4.2): scaffold
// outside a git repository fails with exit 6.
func TestScaffoldOutsideGitRepo(t *testing.T) {
	dir := t.TempDir() // NOT git-initialized
	r := runInDir(t, dir, "scaffold")
	wantExit(t, r, 6)
}

// TestScaffoldStampsFrameworkRef covers the synced-through stamp (spec §4.2):
// scaffold stamps .anthill/framework.md with the embedded ref reported by
// `version --json` template_ref.
func TestScaffoldStampsFrameworkRef(t *testing.T) {
	dir := scaffoldFresh(t)
	ref := templateRef(t)
	fw := readAll(t, frameworkPath(dir))
	if !strings.Contains(fw, ref) {
		t.Fatalf("framework.md does not carry the embedded ref %q\n%s", ref, fw)
	}
}

// ---- local helpers -----------------------------------------------------------

// snapshotTree returns a stable string fingerprint of every regular file under
// root (excluding the .git dir), path + content, so a test can assert a command
// left the tree byte-for-byte unchanged.
func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			b.WriteString("D:" + filepath.ToSlash(rel) + "\n")
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		b.WriteString("F:" + filepath.ToSlash(rel) + ":" + string(data) + "\n")
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return b.String()
}

// wantSameFile asserts two files are byte-identical.
func wantSameFile(t *testing.T, a, b string) {
	t.Helper()
	da := readAll(t, a)
	db := readAll(t, b)
	if da != db {
		t.Fatalf("files differ:\n %s\n %s\n---a---\n%q\n---b---\n%q", a, b, da, db)
	}
}
