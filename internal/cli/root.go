// Package cli builds the anthill command-line interface.
package cli

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/bootstrap"
	"github.com/Dwight-D/anthill-cli/internal/version"
)

// isInteractive reports whether stdin and stdout are both terminals. A seam
// var so tests can force either branch of the bare-invocation default.
var isInteractive = func() bool {
	return isTerm(os.Stdin.Fd()) && isTerm(os.Stdout.Fd())
}

func isTerm(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// NewRootCommand constructs the root "anthill" cobra command with all
// subcommands wired to an App bound to the process stdio.
func NewRootCommand() *cobra.Command {
	return newRootCommand(newApp(os.Stdout, os.Stderr))
}

// newRootCommand builds the root command against a specific App (streams +
// global flags). Every subcommand closes over this App.
func newRootCommand(a *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anthill",
		Short: "Anthill backlog and escalation CLI",
		Long: "anthill is the command-line interface for the Anthill backlog and " +
			"escalation harness. It drives the durable state that coordinates " +
			"autonomous agent workstreams.",
		Version:       version.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bare `anthill` opens the TUI when a human is at a terminal and
			// an installation is discoverable; otherwise fall back to help
			// (exit 0) rather than hanging a headless caller. `--help`,
			// `anthill help`, and unknown commands never reach this RunE and
			// behave as before.
			if isInteractive() {
				if _, err := a.resolveRoot(); err == nil {
					return a.launchTUI()
				}
			}
			return cmd.Help()
		},
	}
	cmd.SetVersionTemplate(version.String() + "\ntemplate ref: " + bootstrap.TemplateRef + "\n")

	pf := cmd.PersistentFlags()
	pf.StringVar(&a.rootFlag, "root", "", "directory containing .anthill/ (default: search upward from CWD)")
	pf.BoolVar(&a.json, "json", false, "emit machine-readable JSON on stdout")
	pf.BoolVarP(&a.quiet, "quiet", "q", false, "suppress non-essential human output")
	pf.BoolVar(&a.noColor, "no-color", false, "disable ANSI styling")

	cmd.AddCommand(
		a.newBacklogCommand(),
		a.newEscalationCommand(),
		a.newInitCommand(),
		a.newDoctorCommand(),
		a.newValidateCommand(),
		a.newVersionCommand(),
		a.newBootstrapCommand(),
		a.newScaffoldCommand(),
		a.newSyncCommand(),
		a.newTUICommand(),
	)
	return cmd
}

// Run builds the root command bound to the process stdio, executes it against
// args, and returns the process exit code (mapping errors per the exit table).
func Run(args []string, stdout, stderr io.Writer) int {
	a := newApp(stdout, stderr)
	root := newRootCommand(a)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	return a.exitCode(root.Execute())
}
