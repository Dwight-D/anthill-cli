package e2e_test

// Helpers shared by the bootstrapping / integrity suite (bootstrap, scaffold,
// version-ref, doctor Section A, sync). Kept in a separate file from
// harness_test.go so the two never collide on declarations.
//
// These drive the same TestMain-built binary. Commands that need a working
// directory (scaffold's git-repo precondition, bootstrap's no-side-effects
// guarantee) run via runInDir, which sets the child process CWD — never
// inheriting the test package's own directory (which is itself inside a git
// repo and would mask the precondition checks).

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The autonomous skill's sanctioned adaptation region. Edits confined between
// these two headings are the derived proceed-list: doctor must exempt them and
// sync must preserve them.
const (
	proceedHeading    = "## Proceed freely (do not ask permission)"
	afterProceedStart = "## Working rules"
)

// runInDir invokes the binary with CWD set to dir and no --root injected. Use
// for scaffold (whose git-repo precondition and default --into both key off
// CWD) and for bootstrap (to assert it creates nothing in an arbitrary dir).
func runInDir(t *testing.T, dir string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	var so, se strings.Builder
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := asExit(err); ok {
			code = ee
		} else {
			t.Fatalf("runInDir %v in %s: process failed to start: %v", args, dir, err)
		}
	}
	return result{stdout: so.String(), stderr: se.String(), exit: code}
}

// asExit extracts an exit code from an *exec.ExitError.
func asExit(err error) (int, bool) {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), true
	}
	return 0, false
}

// gitInitRepo runs `git init` in dir, failing the test if git is unavailable or
// errors. git is available in the environment per the suite's contract.
func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init in %s: %v\n%s", dir, err, out)
	}
}

// newGitRepo returns a fresh temp dir that has been `git init`ed (satisfies
// scaffold's precondition) but not scaffolded.
func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitInitRepo(t, dir)
	return dir
}

// scaffoldFresh returns a temp git repo into which the embedded template has
// been scaffolded cleanly (exit 0). The returned path is the install root (it
// CONTAINS .anthill/ and .claude/), usable as --root for doctor/sync.
func scaffoldFresh(t *testing.T) string {
	t.Helper()
	dir := newGitRepo(t)
	r := runInDir(t, dir, "scaffold")
	wantExit(t, r, 0)
	return dir
}

// templateRef reads the embedded template ref the binary reports, via
// `version --json` template_ref. This is the single source of the expected ref
// — tests never hardcode it.
func templateRef(t *testing.T) string {
	t.Helper()
	r := run(t, "--json", "version")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	ref, _ := obj["template_ref"].(string)
	if strings.TrimSpace(ref) == "" {
		t.Fatalf("version --json template_ref empty: %s", r.stdout)
	}
	return ref
}

// ---- path helpers ------------------------------------------------------------

func skillPath(root, skill string) string {
	return filepath.Join(root, ".claude", "skills", skill, "SKILL.md")
}

func frameworkPath(root string) string {
	return filepath.Join(root, ".anthill", "framework.md")
}

func claudeTemplatePath(root string) string {
	return filepath.Join(root, "CLAUDE.template.md")
}

// ---- JSON list helpers -------------------------------------------------------

// jsonStrings flattens a decoded JSON array field into strings, whether its
// elements are bare strings (a path/skill manifest) or objects (in which case
// every string-valued field is collected). Robust to the manifest carrying
// either shape, so substring assertions work against both.
func jsonStrings(v any) []string {
	var out []string
	arr, ok := v.([]any)
	if !ok {
		return out
	}
	for _, e := range arr {
		switch x := e.(type) {
		case string:
			out = append(out, x)
		case map[string]any:
			for _, mv := range x {
				if s, ok := mv.(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// listContains reports whether any string in list contains sub.
func listContains(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// wantListContains asserts field `key` of obj is an array with an element
// containing sub.
func wantListContains(t *testing.T, obj map[string]any, key, sub string) {
	t.Helper()
	got := jsonStrings(obj[key])
	if !listContains(got, sub) {
		t.Fatalf("%s array does not contain %q\ngot: %v", key, sub, got)
	}
}

// wantHasKey asserts obj contains key.
func wantHasKey(t *testing.T, obj map[string]any, key string) {
	t.Helper()
	if _, ok := obj[key]; !ok {
		t.Fatalf("json object missing key %q\ngot keys: %v", key, keysOf(obj))
	}
}

func keysOf(m map[string]any) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// ---- doctor check helpers ----------------------------------------------------

// doctorChecks parses a doctor --json payload into its check objects.
func doctorChecks(t *testing.T, stdout string) []map[string]any {
	t.Helper()
	obj := jsonObj(t, stdout)
	raw, ok := obj["checks"].([]any)
	if !ok {
		t.Fatalf("doctor --json missing checks array: %s", stdout)
	}
	var out []map[string]any
	for _, e := range raw {
		if m, ok := e.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// sectionIs normalizes a check's `section` label and matches it to the bare
// letter "A"/"B", accepting either the bare letter or a "Section A"-style
// label. (The spec labels the two doctor halves "Section A" / "Section B"; the
// JSON section field is one of those forms.)
func sectionIs(section, letter string) bool {
	s := strings.ToUpper(strings.TrimSpace(section))
	l := strings.ToUpper(strings.TrimSpace(letter))
	return s == l || strings.Contains(s, "SECTION "+l)
}

// hasSection reports whether any check is labeled with the given section letter.
func hasSection(checks []map[string]any, letter string) bool {
	for _, c := range checks {
		if s, ok := c["section"].(string); ok && sectionIs(s, letter) {
			return true
		}
	}
	return false
}

// checkText returns a check's name + detail joined, lowercased, for loose
// content matching.
func checkText(c map[string]any) string {
	name, _ := c["name"].(string)
	detail, _ := c["detail"].(string)
	return strings.ToLower(name + " " + detail)
}

// findFailingMentioning returns the first check with ok==false whose name or
// detail mentions sub (case-insensitive).
func findFailingMentioning(checks []map[string]any, sub string) (map[string]any, bool) {
	sub = strings.ToLower(sub)
	for _, c := range checks {
		ok, _ := c["ok"].(bool)
		if !ok && strings.Contains(checkText(c), sub) {
			return c, true
		}
	}
	return nil, false
}

// findMentioning returns the first check whose name or detail mentions sub.
func findMentioning(checks []map[string]any, sub string) (map[string]any, bool) {
	sub = strings.ToLower(sub)
	for _, c := range checks {
		if strings.Contains(checkText(c), sub) {
			return c, true
		}
	}
	return nil, false
}

// ---- autonomous proceed-list editing -----------------------------------------

// setProceedList rewrites the autonomous skill at path, replacing the body of
// the "## Proceed freely" region (up to the next "## Working rules" heading)
// with the given bullet lines. The surrounding skill text is left untouched.
// Since the exemption for this region has been retired, such an edit now
// diverges the installed skill from the pinned template.
func setProceedList(t *testing.T, path string, bullets []string) {
	t.Helper()
	content := readAll(t, path)
	i := strings.Index(content, proceedHeading)
	if i < 0 {
		t.Fatalf("proceed heading %q not found in %s", proceedHeading, path)
	}
	bodyStart := i + len(proceedHeading)
	rest := content[bodyStart:]
	j := strings.Index(rest, afterProceedStart)
	if j < 0 {
		t.Fatalf("region terminator %q not found in %s", afterProceedStart, path)
	}
	var b strings.Builder
	b.WriteString(content[:bodyStart])
	b.WriteString("\n\n")
	for _, line := range bullets {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(rest[j:])
	writeRaw(t, path, b.String())
}

// appendJunk appends a distinguishing marker line to the file at path, so the
// installed copy diverges from the embedded template (an illegal general-tier
// edit for the doctor/sync integrity checks to catch).
func appendJunk(t *testing.T, path, marker string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open %s for append: %v", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString("\n" + marker + "\n"); err != nil {
		t.Fatalf("append to %s: %v", path, err)
	}
}
