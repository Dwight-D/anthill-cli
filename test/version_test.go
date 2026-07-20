package e2e_test

import "testing"

// TestVersionFlag covers the root --version flag (spec §2 global flags).
func TestVersionFlag(t *testing.T) {
	r := run(t, "--version")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "anthill", "version --version stdout")
	wantContains(t, r.stdout, "0.0.0-dev", "version --version stdout")
}

// TestVersionVerb covers the `version` subcommand (spec §3 command tree).
func TestVersionVerb(t *testing.T) {
	r := run(t, "version")
	wantExit(t, r, 0)
	wantContains(t, r.stdout, "anthill", "version verb stdout")
}

// TestRootHelp confirms the root command emits help and exits 0.
func TestRootHelp(t *testing.T) {
	r := run(t, "--help")
	wantExit(t, r, 0)
	wantContains(t, r.stdout+r.stderr, "anthill", "root help")
}

// TestUnknownCommandUsageExit covers exit code 2 for an unknown command
// (spec §2: Cobra usage errors map to exit 2).
func TestUnknownCommandUsageExit(t *testing.T) {
	r := run(t, "not-a-real-command")
	wantExit(t, r, 2)
}

// TestUnknownFlagUsageExit covers exit code 2 for an unknown flag.
func TestUnknownFlagUsageExit(t *testing.T) {
	root := mkTree(t)
	r := runIn(t, root, "backlog", "list", "--definitely-not-a-flag")
	wantExit(t, r, 2)
}

// TestRootDiscoveryMissing covers --root discovery failure: a directory with no
// .anthill/ ancestor should error (spec §2 global flags: "error if none found").
func TestRootDiscoveryMissing(t *testing.T) {
	empty := t.TempDir()
	r := runIn(t, empty, "backlog", "list")
	wantNonZero(t, r)
}
