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
			"state, organised as switchable modes (Backlog, Escalations, Now). The list\n" +
			"pane shows every row; the content pane shows the selected row's full detail.\n" +
			"Now is a single-pane board of active items and their sessions, refreshed\n" +
			"every 2s. Keypress actions map 1:1 to the scriptable verbs — Backlog:\n" +
			"approve, close as done, discard, remove; Escalations: apply, close, remove;\n" +
			"Now: close as done, reject, release (destructive actions confirm first).\n" +
			"Requires an interactive terminal.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.launchTUI()
		},
	}
	return cmd
}

// runTUIProgram builds and runs the Bubble Tea program over the installation
// at root. A seam var so the root-command test can observe a launch without
// driving a real terminal.
var runTUIProgram = func(root string) error {
	p := tea.NewProgram(tui.New(root), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// launchTUI resolves the root and runs the full-screen TUI.
func (a *App) launchTUI() error {
	root, err := a.resolveRoot()
	if err != nil {
		return err
	}
	if err := runTUIProgram(root); err != nil {
		return internalErr(err.Error())
	}
	return nil
}
