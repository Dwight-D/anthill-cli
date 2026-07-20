// Package cli builds the anthill command-line interface.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/version"
)

// NewRootCommand constructs the root "anthill" cobra command.
//
// It defines only the root: subcommands are designed separately. The root
// carries descriptive help text and a --version flag that prints the build
// version string.
func NewRootCommand() *cobra.Command {
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
	cmd.SetVersionTemplate(version.String() + "\n")
	return cmd
}
