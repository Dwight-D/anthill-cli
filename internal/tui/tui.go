package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/escalation"
	"github.com/Dwight-D/anthill-cli/internal/session"
)

// Reserved band heights: header (mode line + hints line) and status line are
// always present so the layout never shifts.
const (
	headerHeight = 2
	statusHeight = 1
)

const selectionMarker = "▶ "

// focusedPane names which pane receives navigation keys.
type focusedPane int

const (
	focusList   focusedPane = iota // right-hand nav list
	focusDetail                    // left-hand content viewport
)

// reloadMsg asks the root to reload the active mode's data.
type reloadMsg struct{}

// tickMsg is a polling mode's timer firing. gen ties it to the mode
// activation that scheduled it; a tick from an earlier activation is dropped.
type tickMsg struct{ gen int }

// pendingConfirm is an armed destructive action awaiting a y keypress.
type pendingConfirm struct {
	action Action
	index  int
	rowID  string
}

// Model is the single root Bubble Tea model. Modes plug in behind the Mode
// interface; the root owns layout, focus, selection, confirm prompts, and
// mode switching (see the package doc for the mode concept).
type Model struct {
	modes   []Mode
	modeIdx int

	focus    focusedPane
	selected int
	viewport viewport.Model

	width, height int
	sizeReady     bool

	styles styleSet

	status    string
	statusErr bool
	confirm   *pendingConfirm
	overlay   *textOverlay // nil = no text-entry overlay open

	tickGen int // bumped on every mode switch; stale tickMsgs are dropped
}

// New builds the root model for the installation rooted at root (the
// directory containing .anthill). The mode registry is assembled here; adding
// a future mode (Escalations, Agents, ...) is one append.
func New(root string) Model {
	return NewWithModes([]Mode{
		NewBacklogMode(backlog.NewStore(root)),
		NewEscalationsMode(escalation.NewStore(root)),
		NewNowMode(backlog.NewStore(root), session.NewStore(root)),
	})
}

// NewWithModes builds the root model over an explicit mode registry.
func NewWithModes(modes []Mode) Model {
	return Model{
		modes:  modes,
		styles: newStyles(),
	}
}

func (m Model) mode() Mode { return m.modes[m.modeIdx] }

// Init implements tea.Model: trigger the initial load, and start polling if
// the first mode polls.
func (m Model) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg { return reloadMsg{} }, m.pollCmd())
}

// pollCmd schedules the active mode's next poll tick, or nil when the mode
// does not poll.
func (m Model) pollCmd() tea.Cmd {
	p, ok := m.mode().(Poller)
	if !ok {
		return nil
	}
	gen := m.tickGen
	return tea.Tick(p.PollInterval(), func(time.Time) tea.Msg { return tickMsg{gen: gen} })
}

// singlePane reports whether the active mode requested the Board layout.
func (m Model) singlePane() bool {
	_, ok := m.mode().(Board)
	return ok
}

// reload refreshes the active mode from disk, preserving the selection by row
// id where the row survives and clamping the index otherwise. Loads are
// synchronous: the store is a handful of small local files, so there is
// nothing to gain from a tea.Cmd round trip.
func (m *Model) reload() {
	var prevID string
	if m.mode().Len() > 0 && m.selected < m.mode().Len() {
		prevID = m.mode().RowID(m.selected)
	}
	if err := m.mode().Reload(); err != nil {
		m.setStatus("load failed: "+err.Error(), true)
		return
	}
	n := m.mode().Len()
	if n == 0 {
		m.selected = 0
	} else {
		found := false
		if prevID != "" {
			for i := 0; i < n; i++ {
				if m.mode().RowID(i) == prevID {
					m.selected = i
					found = true
					break
				}
			}
		}
		if !found && m.selected >= n {
			m.selected = n - 1
		}
	}
	m.refreshDetail()
}

// refreshDetail re-renders the selected row's detail into the viewport,
// wrapped to the detail pane's inner width so nothing is clipped off-screen.
// The root re-invokes this on width change (preserving the scroll offset), so
// the wrap always matches the live pane width.
func (m *Model) refreshDetail() {
	if m.mode().Len() == 0 {
		m.viewport.Width = m.detailInnerW()
		m.viewport.SetContent("no items")
		return
	}
	m.viewport.Width = m.detailInnerW()
	m.viewport.SetContent(wrapCells(m.mode().Detail(m.selected), m.detailInnerW()))
}

func (m *Model) setStatus(s string, isErr bool) {
	m.status = s
	m.statusErr = isErr
}

// runAction executes an action on row i and reloads in place.
func (m *Model) runAction(act Action, i int) {
	if i >= m.mode().Len() {
		return
	}
	m.finishAction(act.Do(i))
}

// runInputAction executes a free-text action on row i and reloads in place.
func (m *Model) runInputAction(act Action, i int, text string) {
	if i >= m.mode().Len() {
		return
	}
	m.finishAction(act.DoInput(i, text))
}

// finishAction surfaces an action's outcome in the status line and reloads.
func (m *Model) finishAction(msg string, err error) {
	if err != nil {
		m.setStatus("error: "+err.Error(), true)
	} else {
		m.setStatus(msg, false)
	}
	m.reload()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reloadMsg:
		m.reload()
		return m, nil

	case statusMsg:
		m.setStatus(msg.text, msg.isErr)
		return m, nil

	case tickMsg:
		if msg.gen != m.tickGen {
			return m, nil // scheduled by an earlier mode activation
		}
		// Never reload under an open prompt: its row index must stay valid.
		if m.overlay == nil && m.confirm == nil {
			m.reload()
		}
		return m, m.pollCmd()

	case tea.WindowSizeMsg:
		widthChanged := msg.Width != m.width
		m.width, m.height = msg.Width, msg.Height
		m.sizeReady = true
		m.viewport.Width = m.detailInnerW()
		m.viewport.Height = m.innerH()
		if widthChanged {
			saved := m.viewport.YOffset
			m.refreshDetail()
			m.viewport.SetYOffset(saved)
		}
		if m.overlay != nil {
			m.overlay.resize(m.width, m.innerH())
		}
		return m, nil

	case tea.KeyMsg:
		// An open overlay owns every key event.
		if m.overlay != nil {
			cmd, keepOpen := m.overlay.update(msg)
			if !keepOpen {
				o := m.overlay
				m.overlay = nil
				if o.text != "" {
					m.runInputAction(o.act, o.index, o.text)
				} else {
					m.setStatus("cancelled", false)
				}
			}
			return m, cmd
		}

		// An armed confirm prompt gets first refusal on every key.
		if m.confirm != nil {
			c := *m.confirm
			m.confirm = nil
			if msg.String() == "y" {
				m.runAction(c.action, c.index)
			} else {
				m.setStatus("cancelled", false)
			}
			return m, nil
		}

		k := msg.String()
		if !rootReservedKey(k) {
			for _, act := range m.mode().Actions() {
				if k == act.Key {
					return m.triggerAction(act)
				}
			}
		}

		switch k {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.focus == focusList {
				m.focus = focusDetail
			} else {
				m.focus = focusList
			}
			return m, nil
		case "enter":
			m.focus = focusDetail
			return m, nil
		case "esc":
			m.focus = focusList
			return m, nil
		case "r":
			m.reload()
			m.setStatus("refreshed", false)
			return m, nil
		case "m":
			return m, m.switchMode((m.modeIdx + 1) % len(m.modes))
		case "up", "k":
			if m.focus == focusList {
				if m.selected > 0 {
					m.selected--
					m.refreshDetail()
				}
				return m, nil
			}
		case "down", "j":
			if m.focus == focusList {
				if m.selected < m.mode().Len()-1 {
					m.selected++
					m.refreshDetail()
				}
				return m, nil
			}
		default:
			// Number keys select a mode directly.
			if n := int(k[0] - '1'); len(k) == 1 && n >= 0 && n < len(m.modes) {
				return m, m.switchMode(n)
			}
		}
	}

	// Non-key messages an open overlay still needs (cursor blinks).
	if m.overlay != nil {
		cmd, _ := m.overlay.update(msg)
		return m, cmd
	}

	// Everything unhandled scrolls the content viewport when it has focus.
	if m.focus == focusDetail {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// rootReservedKey reports whether k is a root key no mode action may shadow:
// quit, pane focus, mode switching. Every other root key (refresh, j/k
// movement) yields to a mode action bound to the same key.
func rootReservedKey(k string) bool {
	switch k {
	case "ctrl+c", "q", "tab", "enter", "esc", "m":
		return true
	}
	return len(k) == 1 && k[0] >= '1' && k[0] <= '9'
}

// triggerAction runs a mode action on the selected row: free-text actions
// open the overlay, confirm-gated ones arm the prompt, plain ones run now.
func (m Model) triggerAction(act Action) (tea.Model, tea.Cmd) {
	if m.mode().Len() == 0 {
		m.setStatus("no items", true)
		return m, nil
	}
	if act.Input {
		m.overlay = newTextOverlay(act, m.selected, m.mode().RowID(m.selected), m.width, m.innerH())
		return m, textarea.Blink
	}
	if act.Confirm {
		id := m.mode().RowID(m.selected)
		m.confirm = &pendingConfirm{action: act, index: m.selected, rowID: id}
		m.setStatus(fmt.Sprintf("%s %s? [y] confirm  [any other key] cancel", act.Label, id), false)
		return m, nil
	}
	m.runAction(act, m.selected)
	return m, nil
}

// switchMode activates mode idx, resetting selection and reloading. It
// returns the new mode's first poll tick (nil for a non-polling mode); the
// generation bump orphans any tick the previous mode had scheduled.
func (m *Model) switchMode(idx int) tea.Cmd {
	if idx == m.modeIdx || idx < 0 || idx >= len(m.modes) {
		return nil
	}
	m.modeIdx = idx
	m.tickGen++
	m.selected = 0
	m.focus = focusList
	m.setStatus("", false)
	m.reload()
	return m.pollCmd()
}

// ---- derived dimensions (pure, recomputed each frame) ----

// leftInnerW is the content pane's inner width: 3/5 of the terminal minus the
// border cells, floored so it never goes negative.
func (m Model) leftInnerW() int {
	w := m.width*3/5 - 2
	if w < 4 {
		return 4
	}
	return w
}

// rightInnerW is the nav pane's inner width: the remainder minus both panes'
// border cells.
func (m Model) rightInnerW() int {
	w := m.width - 4 - m.leftInnerW()
	if w < 4 {
		return 4
	}
	return w
}

// detailInnerW is the detail viewport's inner width: the left pane in the
// two-pane layout, the full width (minus borders) in the single-pane layout.
func (m Model) detailInnerW() int {
	if !m.singlePane() {
		return m.leftInnerW()
	}
	if w := m.width - 2; w >= 4 {
		return w
	}
	return 4
}

// innerH is the main-area inner height: total minus the reserved bands and
// the pane borders.
func (m Model) innerH() int {
	h := m.height - headerHeight - statusHeight - 2
	if h < 4 {
		return 4
	}
	return h
}

// ---- rendering ----

// View implements tea.Model: header band, two bordered panes, status line.
func (m Model) View() string {
	if !m.sizeReady {
		return "loading..."
	}
	var main string
	switch {
	case m.singlePane() && m.focus == focusDetail:
		main = m.renderDetailPane()
	case m.singlePane():
		main = m.renderBoard()
	default:
		main = lipgloss.JoinHorizontal(lipgloss.Top, m.renderDetailPane(), m.renderListPane())
	}
	if m.overlay != nil {
		// The overlay replaces the main area, centered in the same band so
		// the header and status line never shift.
		main = lipgloss.Place(m.width, m.innerH()+2, lipgloss.Center, lipgloss.Center, m.overlay.view(m.styles))
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), main, m.renderStatus())
}

// renderHeader draws the mode-tab line and the key-hint line.
func (m Model) renderHeader() string {
	var tabs []string
	for i, md := range m.modes {
		label := fmt.Sprintf("%d:%s", i+1, md.Name())
		if i == m.modeIdx {
			tabs = append(tabs, m.styles.activeTab.Render(label))
		} else {
			tabs = append(tabs, m.styles.dim.Render(label))
		}
	}
	title := " anthill  " + strings.Join(tabs, "  ")

	var hints string
	if m.overlay != nil {
		// An open overlay swaps the whole hint string for its own keys.
		hints = m.overlay.title + "  ·  [ctrl+s] submit  [esc] cancel"
	} else {
		hints = "[tab] switch pane  [↑↓] move  [enter] open"
		if m.singlePane() {
			hints = "[↑↓] move  [enter] show  [esc] back"
		}
		if !m.modeBinds("r") {
			hints += "  [r] refresh"
		}
		if len(m.modes) > 1 {
			hints += "  [m] mode"
		}
		for _, act := range m.mode().Actions() {
			hints += fmt.Sprintf("  [%s] %s", act.Key, act.Label)
		}
		hints += "  [q] quit"
	}
	hintLine := m.styles.dim.Render(truncateCells(" "+hints, m.width))

	return lipgloss.NewStyle().Width(m.width).Render(title + "\n" + hintLine)
}

// renderDetailPane draws the left content pane (selected item detail).
func (m Model) renderDetailPane() string {
	borderCol := m.styles.inactive
	if m.focus == focusDetail {
		borderCol = m.styles.active
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol).
		Width(m.detailInnerW()).
		Height(m.innerH()).
		Render(m.viewport.View())
}

// modeBinds reports whether the active mode binds key to an action.
func (m Model) modeBinds(key string) bool {
	for _, act := range m.mode().Actions() {
		if act.Key == key {
			return true
		}
	}
	return false
}

// renderBoard draws the single-pane layout: the mode's top lines, its rows
// (selection-marked and scrolled), then its bottom lines, in one full-width
// bordered pane.
func (m Model) renderBoard() string {
	w := m.width - 2
	if w < 4 {
		w = 4
	}
	top, bottom := m.mode().(Board).BoardLines(w)
	h := m.innerH() - len(top) - len(bottom)
	if h < 1 {
		h = 1
	}
	var lines []string
	for _, l := range top {
		lines = append(lines, truncateCells(l, w))
	}
	lines = append(lines, m.renderRows(w, h)...)
	for _, l := range bottom {
		lines = append(lines, truncateCells(l, w))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.active).
		Width(w).
		Height(m.innerH()).
		Render(strings.Join(lines, "\n"))
}

// renderListPane draws the right nav pane: the mode's rows, hand-rolled, with
// a selection marker and a scrolled window keeping the selection visible.
func (m Model) renderListPane() string {
	borderCol := m.styles.inactive
	if m.focus == focusList {
		borderCol = m.styles.active
	}
	w, h := m.rightInnerW(), m.innerH()
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol).
		Width(w).
		Height(h).
		Render(strings.Join(m.renderRows(w, h), "\n"))
}

// renderRows renders at most h of the mode's rows at width w, with the
// selection marker and a window scrolled to keep the selection visible.
func (m Model) renderRows(w, h int) []string {
	n := m.mode().Len()
	top := 0
	if m.selected >= h {
		top = m.selected - h + 1
	}
	var lines []string
	for i := top; i < n && i-top < h; i++ {
		prefix := "  "
		if i == m.selected {
			prefix = selectionMarker
		}
		line := truncateCells(prefix+m.mode().Row(i, w-lipgloss.Width(prefix)), w)
		if i == m.selected {
			if m.focus == focusList {
				line = m.styles.selection.Render(line)
			} else {
				line = m.styles.dim.Render(line)
			}
		}
		lines = append(lines, line)
	}
	if n == 0 {
		lines = []string{m.styles.dim.Render("no items")}
	}
	return lines
}

// renderStatus draws the one-line status bar.
func (m Model) renderStatus() string {
	s := m.status
	if s == "" {
		n := m.mode().Len()
		s = fmt.Sprintf("%d items", n)
		if n > 0 && m.selected < n {
			s += "  ·  " + m.mode().RowID(m.selected)
		}
	}
	style := m.styles.dim
	if m.statusErr {
		style = m.styles.errText
	}
	return style.Render(truncateCells(" "+s, m.width))
}
