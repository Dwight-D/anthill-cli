// Package tui implements `anthill tui`, the interactive full-screen browser
// over the harness's durable state.
//
// # Modes
//
// The app is organised around top-level modes — switchable screens such as
// Backlog (implemented), and later Escalations or Agents. A Mode supplies the
// data behind one shared three-band layout: rows for the right-hand nav list,
// detail text for the left-hand content pane, and the keypress actions that
// operate on the selected row. The root Model owns everything visual and
// interactive — layout, pane focus, selection, scrolling, confirm prompts,
// refresh, and mode switching — so adding a mode is one struct implementing
// Mode plus one entry in the registry built by New; the root model needs no
// changes.
//
// Every mode action maps 1:1 to an existing non-interactive command's domain
// operation (the TUI is a viewport onto the verbs, never a privileged path).
package tui

import "time"

// Mode is one top-level screen of the TUI. Implementations hold their own
// domain handles (stores) and backing data; the root model drives them through
// this interface only.
type Mode interface {
	// Name labels the mode in the header band (e.g. "Backlog").
	Name() string
	// Reload refreshes the mode's backing data from disk. The root calls it
	// on mode entry, on explicit refresh, and after every action.
	Reload() error
	// Len is the number of rows currently loaded.
	Len() int
	// RowID is a stable identity for row i, used to preserve the selection
	// across reloads.
	RowID(i int) string
	// Row renders row i as one plain-text line at most width cells wide.
	// The root adds the selection marker and styling.
	Row(i, width int) string
	// Detail renders the left-pane text for row i (scrolled by the root).
	Detail(i int) string
	// Actions lists the mode's keypress actions in hint-display order.
	Actions() []Action
}

// Action is one keypress operation on the selected row. Exactly one of Do
// and DoInput is set: plain actions run Do (optionally confirm-gated),
// free-text actions set Input and run DoInput with the submitted text.
type Action struct {
	// Key is the Bubble Tea key string that triggers the action (e.g. "a").
	Key string
	// Label is the human hint label (e.g. "approve").
	Label string
	// Confirm marks a destructive action: the root interposes a y/n prompt
	// in the status line before running Do.
	Confirm bool
	// Input marks a free-text action: the root opens a text-entry overlay
	// and passes the submitted text to DoInput instead of calling Do.
	Input bool
	// Do performs the action on row i via the domain layer and returns a
	// short status-line message. The root reloads the mode afterwards,
	// preserving the selection position where sensible.
	Do func(i int) (string, error)
	// DoInput performs an Input action on row i with the overlay's submitted
	// text (never empty). Same contract as Do otherwise.
	DoInput func(i int, text string) (string, error)
}

// Poller is an optional Mode extension. While a mode implementing it is
// active, the root reloads it every PollInterval via tea.Tick; ticks are
// dropped once another mode is selected, so non-polling modes never reload
// on a timer.
type Poller interface {
	PollInterval() time.Duration
}

// Board is an optional Mode extension requesting the single-pane layout: the
// rows span the full width, framed by the lines BoardLines returns above and
// below them (column headings, summary blocks, counts). enter swaps the board
// for the selected row's full-width detail; esc returns to the board.
type Board interface {
	BoardLines(width int) (top, bottom []string)
}
