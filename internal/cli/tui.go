package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/tui"
)

// newTUICommand builds `anthill tui`: the interactive full-screen browser.
func (a *App) newTUICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Browse the backlog interactively (full-screen)",
		Long: "tui opens a full-screen interactive browser over the harness's durable\n" +
			"state, organised as switchable modes (currently: Backlog). The list pane\n" +
			"shows every item; the content pane shows the selected item's frontmatter\n" +
			"and body. Keypress actions map 1:1 to the backlog verbs: approve, close\n" +
			"as done, discard, and remove (destructive actions confirm first).\n" +
			"Requires an interactive terminal.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := a.resolveRoot()
			if err != nil {
				return err
			}
			p := tea.NewProgram(tui.New(root), tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return internalErr(err.Error())
			}
			return nil
		},
	}
	return cmd
}
