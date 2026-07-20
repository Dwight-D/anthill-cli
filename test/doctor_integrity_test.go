package e2e_test

import (
	"strings"
	"testing"
)

// TestDoctorCleanScaffoldHealthy covers `anthill doctor` on a clean scaffold
// (spec §4.4): healthy, exit 0, --json { ok:true, checks:[...] } with BOTH
// Section A (install integrity) and Section B (runtime data integrity) present.
func TestDoctorCleanScaffoldHealthy(t *testing.T) {
	dir := scaffoldFresh(t)

	r := runIn(t, dir, "--json", "doctor")
	wantExit(t, r, 0)
	obj := jsonObj(t, r.stdout)
	if obj["ok"] != true {
		t.Fatalf("clean scaffold doctor ok = %v, want true\n%s", obj["ok"], r.stdout)
	}
	checks := doctorChecks(t, r.stdout)
	if !hasSection(checks, "A") {
		t.Fatalf("no check labeled Section A\n%s", r.stdout)
	}
	if !hasSection(checks, "B") {
		t.Fatalf("no check labeled Section B\n%s", r.stdout)
	}
}

// TestDoctorTamperedSkill covers skill integrity (spec §4.4 Section A): an
// installed general-tier skill edited to differ from the embedded template is
// flagged as an illegal local edit, exit 3.
func TestDoctorTamperedSkill(t *testing.T) {
	dir := scaffoldFresh(t)
	appendJunk(t, skillPath(dir, "triage"), "LOCAL EDIT — should be illegal")

	r := runIn(t, dir, "--json", "doctor")
	wantExit(t, r, 3)
	obj := jsonObj(t, r.stdout)
	if obj["ok"] != false {
		t.Fatalf("tampered-skill doctor ok = %v, want false", obj["ok"])
	}
	checks := doctorChecks(t, r.stdout)
	if _, found := findFailingMentioning(checks, "skill"); !found {
		t.Fatalf("no failing skill-integrity check reported\n%s", r.stdout)
	}
}

// TestDoctorFlagsAutonomousLocalEdit covers the retired exemption (spec §doctor
// Section A): the autonomous skill has no exempted regions, so editing its
// "## Proceed freely" region is a flat integrity failure — doctor flags
// autonomous and exits 3.
func TestDoctorFlagsAutonomousLocalEdit(t *testing.T) {
	dir := scaffoldFresh(t)
	setProceedList(t, skillPath(dir, "autonomous"), []string{
		"- Create/edit/delete anything under the widget project's src/ and tests/.",
		"- Run make build and make test.",
		"- git: path-scoped add, commit, push to the work branch.",
	})

	r := runIn(t, dir, "--json", "doctor")
	wantExit(t, r, 3)
	obj := jsonObj(t, r.stdout)
	if obj["ok"] != false {
		t.Fatalf("edited-autonomous doctor ok = %v, want false", obj["ok"])
	}
	checks := doctorChecks(t, r.stdout)
	if _, found := findFailingMentioning(checks, "skill"); !found {
		t.Fatalf("no failing skill-integrity check reported for the autonomous edit\n%s", r.stdout)
	}
}

// TestDoctorDerivationPlaceholders covers derivation-status (spec §4.4 Section
// A): a fresh scaffold still holds template placeholders. Plain doctor may pass
// (informational), but doctor --strict exits 3 on remaining placeholders.
func TestDoctorDerivationPlaceholders(t *testing.T) {
	dir := scaffoldFresh(t)

	// Plain: placeholders are informational, not a hard failure.
	wantExit(t, runIn(t, dir, "doctor"), 0)

	// Strict: remaining placeholders are a failure.
	r := runIn(t, dir, "--json", "doctor", "--strict")
	wantExit(t, r, 3)
	checks := doctorChecks(t, r.stdout)
	if !anyFailingMentioning(checks, "deriv", "placeholder", "template") {
		t.Fatalf("no failing derivation-status check under --strict\n%s", r.stdout)
	}
}

// TestDoctorSyncStatusUpToDate covers sync-status (spec §4.4 Section A): a fresh
// scaffold is stamped at the embedded ref and reports up-to-date.
func TestDoctorSyncStatusUpToDate(t *testing.T) {
	dir := scaffoldFresh(t)

	r := runIn(t, dir, "--json", "doctor")
	obj := jsonObj(t, r.stdout)
	checks := doctorChecks(t, r.stdout)
	c, found := findMentioning(checks, "sync")
	if !found {
		t.Fatalf("no sync-status check reported\n%s", r.stdout)
	}
	if ok, _ := c["ok"].(bool); !ok {
		t.Fatalf("sync-status check not ok on a fresh scaffold: %v", c)
	}
	_ = obj
}

// TestDoctorSyncStatusBehind covers sync-status behind (spec §4.4 Section A):
// rewriting synced-through to a bogus old value makes doctor report behind.
func TestDoctorSyncStatusBehind(t *testing.T) {
	dir := scaffoldFresh(t)
	ref := templateRef(t)

	// Rewrite the stamped ref to a bogus older value.
	bogus := "0000000000000000000000000000000000000000"
	fw := readAll(t, frameworkPath(dir))
	if !strings.Contains(fw, ref) {
		t.Fatalf("framework.md unexpectedly lacks the stamped ref %q", ref)
	}
	writeRaw(t, frameworkPath(dir), strings.ReplaceAll(fw, ref, bogus))

	r := runIn(t, dir, "--json", "doctor")
	checks := doctorChecks(t, r.stdout)
	c, found := findMentioning(checks, "sync")
	if !found {
		t.Fatalf("no sync-status check reported\n%s", r.stdout)
	}
	if detail, _ := c["detail"].(string); !strings.Contains(strings.ToLower(detail), "behind") {
		t.Fatalf("sync-status detail does not report behind: %v", c)
	}
}

// anyFailingMentioning reports whether some ok==false check mentions any of subs.
func anyFailingMentioning(checks []map[string]any, subs ...string) bool {
	for _, sub := range subs {
		if _, ok := findFailingMentioning(checks, sub); ok {
			return true
		}
	}
	return false
}
