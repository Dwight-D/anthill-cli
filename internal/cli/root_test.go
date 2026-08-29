package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	cmd := NewRootCommand()

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(--help) returned error: %v", err)
	}

	if got := out.String(); !strings.Contains(got, "anthill") {
		t.Fatalf("help output = %q, want it to contain %q", got, "anthill")
	}
}

// stubInteractive forces the interactivity seam for one test.
func stubInteractive(t *testing.T, v bool) {
	t.Helper()
	prev := isInteractive
	isInteractive = func() bool { return v }
	t.Cleanup(func() { isInteractive = prev })
}

// TestBareInvocationNonInteractiveShowsHelp: with no TTY, bare `anthill`
// prints help and exits 0 — it must never hang a headless caller.
func TestBareInvocationNonInteractiveShowsHelp(t *testing.T) {
	stubInteractive(t, false)

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "Usage:") {
		t.Fatalf("bare non-interactive output = %q, want help text", got)
	}
}

// TestBareInvocationInteractiveLaunchesTUI: at a terminal with a discoverable
// root, bare `anthill` launches the TUI (observed via the program seam).
func TestBareInvocationInteractiveLaunchesTUI(t *testing.T) {
	stubInteractive(t, true)
	var launchedRoot string
	prevRun := runTUIProgram
	runTUIProgram = func(root string) error {
		launchedRoot = root
		return nil
	}
	t.Cleanup(func() { runTUIProgram = prevRun })

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".anthill"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--root", root})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if launchedRoot != root {
		t.Fatalf("TUI launched with root %q, want %q", launchedRoot, root)
	}
}

// TestBareInvocationInteractiveNoRootShowsHelp: at a terminal but outside any
// installation, bare `anthill` falls back to help rather than erroring.
func TestBareInvocationInteractiveNoRootShowsHelp(t *testing.T) {
	stubInteractive(t, true)
	prevRun := runTUIProgram
	runTUIProgram = func(root string) error {
		t.Fatalf("TUI must not launch without a discoverable root")
		return nil
	}
	t.Cleanup(func() { runTUIProgram = prevRun })

	// No --root flag and no ANTHILL_ROOT: root discovery is the upward walk
	// from CWD, which finds nothing under a fresh temp dir.
	empty := t.TempDir()
	t.Chdir(empty)
	t.Setenv("ANTHILL_ROOT", "")

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "Usage:") {
		t.Fatalf("bare no-root output = %q, want help text", got)
	}
}

// TestHelpSurfacesUnchanged: `--help` and the help command still print help
// even when the terminal is interactive (they never reach the bare default).
func TestHelpSurfacesUnchanged(t *testing.T) {
	stubInteractive(t, true)
	prevRun := runTUIProgram
	runTUIProgram = func(root string) error {
		t.Fatalf("TUI must not launch on a help invocation")
		return nil
	}
	t.Cleanup(func() { runTUIProgram = prevRun })

	for _, args := range [][]string{{"--help"}, {"help"}} {
		cmd := NewRootCommand()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute(%v) returned error: %v", args, err)
		}
		if got := out.String(); !strings.Contains(got, "Usage:") {
			t.Fatalf("%v output = %q, want help text", args, got)
		}
	}
}
