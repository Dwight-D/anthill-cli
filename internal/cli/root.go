// Package cli builds the anthill command-line interface.
package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/bootstrap"
	"github.com/Dwight-D/anthill-cli/internal/version"
)

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
