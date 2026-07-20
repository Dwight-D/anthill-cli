package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const autonomousPath = ".claude/skills/autonomous/SKILL.md"

func mustReadTemplate(t *testing.T, p string) []byte {
	t.Helper()
	b, err := ReadTemplateFile(p)
	if err != nil {
		t.Fatalf("ReadTemplateFile(%q): %v", p, err)
	}
	return b
}

// replaceProceedBody swaps the body of the "## Proceed freely" section for the
// given bullet lines, leaving the rest of the file byte-identical.
func replaceProceedBody(src string, body []string) string {
	lines := strings.Split(src, "\n")
	var out []string
	in := false
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "## ") {
			if strings.HasPrefix(t, "## Proceed freely") {
				in = true
				out = append(out, ln)
				out = append(out, body...)
				continue
			}
			in = false
		}
		if in {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// TestAutonomousComparedByteExact asserts the autonomous skill has no exempted
// regions: it is byte-compared exactly like every other general-tier skill, so
// a derived proceed-list — once a sanctioned adaptation — now counts as a
// divergence.
func TestAutonomousComparedByteExact(t *testing.T) {
	tmpl := mustReadTemplate(t, autonomousPath)

	// Identical bytes → equal.
	if !filesEqual(tmpl, tmpl) {
		t.Fatal("identical autonomous bytes should be equal")
	}

	// A derived proceed-list is no longer exempt — it diverges from the template.
	derived := replaceProceedBody(string(tmpl), []string{
		"- Build, test, and run the project's own toolchain.",
		"- git: path-scoped add, commit, push to the work branch.",
	})
	if derived == string(tmpl) {
		t.Fatal("test precondition: proceed-list replacement produced no change")
	}
	if filesEqual([]byte(derived), tmpl) {
		t.Fatal("a derived proceed-list must now count as a divergence (no exemption)")
	}

	// A relocated decisions-log path is likewise no longer exempt.
	relocated := strings.Replace(string(tmpl), "`.anthill/decisions.md`", "`.anthill/log/decisions.md`", 1)
	if relocated == string(tmpl) {
		t.Fatal("test precondition: template must contain the decisions-log path")
	}
	if filesEqual([]byte(relocated), tmpl) {
		t.Fatal("a relocated decisions-log path must now count as a divergence")
	}
}

func TestFrameworkStampAndRead(t *testing.T) {
	content := mustReadTemplate(t, ".anthill/framework.md")

	stamped, err := StampFramework(content, "REF123456789abc", "2026-07-20")
	if err != nil {
		t.Fatalf("StampFramework: %v", err)
	}
	s := string(stamped)
	if !strings.Contains(s, "- **synced-through:** REF123456789abc (installed 2026-07-20)") {
		t.Fatalf("stamped line missing; got:\n%s", s)
	}
	// The multi-line angle-bracket placeholder must be gone.
	if strings.Contains(s, "<the upstream release this install is current") {
		t.Fatal("angle-bracket placeholder was not collapsed")
	}
	ref, err := ReadSyncedThroughRef(stamped)
	if err != nil {
		t.Fatalf("ReadSyncedThroughRef: %v", err)
	}
	if ref != "REF123456789abc" {
		t.Fatalf("ref = %q, want REF123456789abc", ref)
	}

	// A pristine (un-stamped) placeholder reads as "" (manual).
	ref0, err := ReadSyncedThroughRef(content)
	if err != nil {
		t.Fatalf("ReadSyncedThroughRef(pristine): %v", err)
	}
	if ref0 != "" {
		t.Fatalf("pristine ref = %q, want empty", ref0)
	}

	// Idempotent: re-stamping replaces the single line.
	stamped2, err := StampFramework(stamped, "OTHER", "2026-07-21")
	if err != nil {
		t.Fatalf("re-stamp: %v", err)
	}
	if got := strings.Count(string(stamped2), "**synced-through:**"); got != 1 {
		t.Fatalf("synced-through appears %d times after re-stamp, want 1", got)
	}
}

func TestScaffoldClassifyWriteSkipRefuse(t *testing.T) {
	dir := t.TempDir()

	// Empty dir → everything is a write.
	entries, err := ClassifyScaffold(dir)
	if err != nil {
		t.Fatalf("ClassifyScaffold: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected payload entries")
	}
	for _, e := range entries {
		want := StatusWrite
		if e.Path == gitignoreRelPath {
			// An absent .gitignore is merged (marker-wrapped block), not written
			// verbatim, so a re-scaffold stays idempotent via the marker.
			want = StatusAppend
		}
		if e.Status != want {
			t.Fatalf("%s: status %v, want %v", e.Path, e.Status, want)
		}
	}

	// Apply → writes all, refuses none, stamps framework.md.
	res, err := Scaffold(dir, false, false)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if len(res.Refused) != 0 || len(res.Written) == 0 {
		t.Fatalf("first scaffold: written=%d refused=%d", len(res.Written), len(res.Refused))
	}
	fwPath := filepath.Join(dir, filepath.FromSlash(frameworkRelPath))
	fw, err := os.ReadFile(fwPath)
	if err != nil {
		t.Fatalf("read framework.md: %v", err)
	}
	if ref, _ := ReadSyncedThroughRef(fw); ref != TemplateRef {
		t.Fatalf("stamped ref = %q, want %q", ref, TemplateRef)
	}

	// Idempotent: re-classify → everything identical, including the stamped
	// framework.md (it differs from the pristine template only by scaffold's own
	// synced-through stamp).
	entries, _ = ClassifyScaffold(dir)
	for _, e := range entries {
		if e.Status != StatusIdentical {
			t.Fatalf("%s: status %v, want StatusIdentical on idempotent re-run", e.Path, e.Status)
		}
	}

	// Second apply is a true no-op: nothing written, nothing refused.
	res2, err := Scaffold(dir, false, false)
	if err != nil {
		t.Fatalf("second Scaffold: %v", err)
	}
	if len(res2.Written) != 0 || len(res2.Refused) != 0 {
		t.Fatalf("idempotent re-run: written=%v refused=%v, want both empty", res2.Written, res2.Refused)
	}
	if fw2, _ := os.ReadFile(fwPath); string(fw2) != string(fw) {
		t.Fatal("idempotent re-run modified the stamped framework.md")
	}

	// A genuine user derivation of framework.md (a change beyond synced-through)
	// is refused without --force.
	derived := strings.Replace(string(fw), "**Role:** consumer", "**Role:** MAINTAINER (derived)", 1)
	if derived == string(fw) {
		t.Fatal("test precondition: expected role line in framework.md")
	}
	if err := os.WriteFile(fwPath, []byte(derived), 0o644); err != nil {
		t.Fatalf("write derived framework.md: %v", err)
	}
	res3, err := Scaffold(dir, false, false)
	if err != nil {
		t.Fatalf("third Scaffold: %v", err)
	}
	if !contains(res3.Refused, frameworkRelPath) {
		t.Fatalf("expected derived framework.md refused; refused=%v", res3.Refused)
	}
	if fw3, _ := os.ReadFile(fwPath); string(fw3) != derived {
		t.Fatal("refused framework.md was modified without --force")
	}

	// --force overwrites the derived file.
	res4, err := Scaffold(dir, true, false)
	if err != nil {
		t.Fatalf("force Scaffold: %v", err)
	}
	if len(res4.Refused) != 0 || !contains(res4.Written, frameworkRelPath) {
		t.Fatalf("force run: written=%v refused=%v", res4.Written, res4.Refused)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// TestSyncFlagsAutonomousLocalEdit covers the retired-exemption contract: a
// derived autonomous proceed-list on a current install is now an unexpected
// local edit → conflict (exit 3) without --force, and --force overwrites it
// verbatim like any other skill.
func TestSyncFlagsAutonomousLocalEdit(t *testing.T) {
	dir := t.TempDir()
	if _, err := Scaffold(dir, false, false); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	// After scaffold, synced-through == TemplateRef (install claims current).

	autPath := filepath.Join(dir, filepath.FromSlash(autonomousPath))
	tmpl, _ := os.ReadFile(autPath)
	marker := "- Build the project's widgets via the derived toolchain."
	derived := replaceProceedBody(string(tmpl), []string{marker})
	if err := os.WriteFile(autPath, []byte(derived), 0o644); err != nil {
		t.Fatalf("write derived autonomous: %v", err)
	}

	// A proceed-list edit is no longer exempt → conflict, file left unchanged.
	res, err := Sync(dir, false, false)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !contains(res.Conflicts, "autonomous") {
		t.Fatalf("expected autonomous conflict; got updated=%v conflicts=%v", res.Updated, res.Conflicts)
	}
	if after, _ := os.ReadFile(autPath); !strings.Contains(string(after), marker) {
		t.Fatal("conflicting sync (no --force) must leave the local edit untouched")
	}

	// --force overwrites verbatim → autonomous becomes an update, no conflict.
	res2, err := Sync(dir, false, true)
	if err != nil {
		t.Fatalf("Sync (force): %v", err)
	}
	if len(res2.Conflicts) != 0 || !contains(res2.Updated, "autonomous") {
		t.Fatalf("force sync: updated=%v conflicts=%v", res2.Updated, res2.Conflicts)
	}
	if forced, _ := os.ReadFile(autPath); string(forced) != string(tmpl) {
		t.Fatal("force sync did not restore the pristine autonomous skill verbatim")
	}
}

func TestSyncUpdatesBehindInstall(t *testing.T) {
	dir := t.TempDir()
	if _, err := Scaffold(dir, false, false); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	// Mark the install as behind by rewriting synced-through to an older ref.
	fwPath := filepath.Join(dir, filepath.FromSlash(frameworkRelPath))
	fw, _ := os.ReadFile(fwPath)
	behind, err := StampFramework(fw, "0000000000000000000000000000000000000000", "2026-01-01")
	if err != nil {
		t.Fatalf("stamp behind: %v", err)
	}
	if err := os.WriteFile(fwPath, behind, 0o644); err != nil {
		t.Fatalf("write behind: %v", err)
	}

	// Diverge a non-autonomous skill (simulating an upstream change to apply).
	triagePath := filepath.Join(dir, filepath.FromSlash(".claude/skills/triage/SKILL.md"))
	if err := os.WriteFile(triagePath, []byte("stale local copy\n"), 0o644); err != nil {
		t.Fatalf("write stale triage: %v", err)
	}

	res, err := Sync(dir, false, false)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !contains(res.Updated, "triage") {
		t.Fatalf("behind install should update triage; updated=%v conflicts=%v", res.Updated, res.Conflicts)
	}
	// Re-copied verbatim + synced-through bumped back to the embedded ref.
	got, _ := os.ReadFile(triagePath)
	want := mustReadTemplate(t, ".claude/skills/triage/SKILL.md")
	if string(got) != string(want) {
		t.Fatal("triage was not re-copied verbatim")
	}
	fw2, _ := os.ReadFile(fwPath)
	if ref, _ := ReadSyncedThroughRef(fw2); ref != TemplateRef {
		t.Fatalf("synced-through not bumped: %q", ref)
	}
}
