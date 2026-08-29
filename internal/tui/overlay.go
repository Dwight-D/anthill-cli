package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// statusMsg carries a status-line update through the message loop, so overlay
// validation errors flow the same path as any other status text.
type statusMsg struct {
	text  string
	isErr bool
}

func statusCmd(text string, isErr bool) tea.Cmd {
	return func() tea.Msg { return statusMsg{text: text, isErr: isErr} }
}

// Overlay box geometry.
const (
	overlayMaxW      = 64
	overlayMinW      = 24
	overlayTextMaxH  = 8
	overlayFixedRows = 2 // title line + hint line
)

// textOverlay is the free-text entry overlay behind every Input action. It is
// held as a nil-able pointer on the root model and owns every key event while
// open: esc cancels, ctrl+s submits (the textarea itself eats enter for
// newlines and ctrl+d for delete-forward, so neither can be the submit key).
// On submit with text the overlay closes with text set; the root then runs
// the action. Empty submissions are rejected in the status line and the
// overlay stays open.
type textOverlay struct {
	title string // box title, e.g. "answer 2026-08-29-open-q"
	act   Action // the Input action to run on submit
	index int    // row the action targets
	text  string // trimmed submission; empty means cancelled

	ta            textarea.Model
	width, height int // terminal width, main-area inner height
}

// newTextOverlay opens the overlay for an Input action on row index.
func newTextOverlay(act Action, index int, rowID string, w, h int) *textOverlay {
	ta := textarea.New()
	ta.Placeholder = "type " + act.Label + " text"
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.Focus()
	o := &textOverlay{title: act.Label + " " + rowID, act: act, index: index, ta: ta}
	o.resize(w, h)
	return o
}

// update handles one message for the open overlay. It returns the command to
// run and whether the overlay should stay open.
func (o *textOverlay) update(msg tea.Msg) (tea.Cmd, bool) {
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return o.delegate(msg), true
	}
	switch key.String() {
	case "esc":
		return nil, false
	case "ctrl+s":
		text := strings.TrimSpace(o.ta.Value())
		if text == "" {
			return statusCmd(o.act.Label+" text required", true), true
		}
		o.text = text
		return nil, false
	}
	return o.delegate(msg), true
}

// delegate routes a message to the textarea (typing, cursor blinks).
func (o *textOverlay) delegate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	o.ta, cmd = o.ta.Update(msg)
	return cmd
}

// boxW is the overlay box width: clamped to the terminal, capped for
// readability.
func (o *textOverlay) boxW() int {
	w := o.width - 2
	if w > overlayMaxW {
		w = overlayMaxW
	}
	if w < overlayMinW {
		w = overlayMinW
	}
	return w
}

// innerW is the text width inside the box: lipgloss Width covers the padding
// (2) but not the border.
func (o *textOverlay) innerW() int { return o.boxW() - 2 }

// resize refits the box and textarea to the terminal width and the main-area
// inner height.
func (o *textOverlay) resize(w, h int) {
	o.width, o.height = w, h
	o.ta.SetWidth(o.innerW())
	taH := overlayTextMaxH
	if lim := h - overlayFixedRows; taH > lim {
		taH = lim
	}
	if taH < 1 {
		taH = 1
	}
	o.ta.SetHeight(taH)
}

// view renders the bordered overlay box: title, textarea, hint line.
func (o *textOverlay) view(st styleSet) string {
	titleS := lipgloss.NewStyle().Bold(true).Foreground(st.active)
	hint := st.dim.Render(truncateCells("[ctrl+s] submit  [esc] cancel", o.innerW()))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(st.active).
		Padding(0, 1).
		Width(o.boxW()).
		Render(titleS.Render(truncateCells(o.title, o.innerW())) + "\n" + o.ta.View() + "\n" + hint)
}
