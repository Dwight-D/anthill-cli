package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/escalation"
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
}

// New builds the root model for the installation rooted at root (the
// directory containing .anthill). The mode registry is assembled here; adding
// a future mode (Escalations, Agents, ...) is one append.
func New(root string) Model {
	return NewWithModes([]Mode{
		NewBacklogMode(backlog.NewStore(root)),
		NewEscalationsMode(escalation.NewStore(root)),
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

// Init implements tea.Model: trigger the initial load.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return reloadMsg{} }
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

// refreshDetail re-renders the selected row's detail into the viewport.
func (m *Model) refreshDetail() {
	if m.mode().Len() == 0 {
		m.viewport.SetContent("no items")
		return
	}
	m.viewport.SetContent(m.mode().Detail(m.selected))
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
	msg, err := act.Do(i)
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

	case tea.WindowSizeMsg:
		widthChanged := msg.Width != m.width
		m.width, m.height = msg.Width, msg.Height
		m.sizeReady = true
		m.viewport.Width = m.leftInnerW()
		m.viewport.Height = m.innerH()
		if widthChanged {
			saved := m.viewport.YOffset
			m.refreshDetail()
			m.viewport.SetYOffset(saved)
		}
		return m, nil

	case tea.KeyMsg:
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

		switch msg.String() {
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
			m.switchMode((m.modeIdx + 1) % len(m.modes))
			return m, nil
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
			if n := int(msg.String()[0] - '1'); len(msg.String()) == 1 && n >= 0 && n < len(m.modes) {
				m.switchMode(n)
				return m, nil
			}
			// Mode action keys.
			for _, act := range m.mode().Actions() {
				if msg.String() != act.Key {
					continue
				}
				if m.mode().Len() == 0 {
					m.setStatus("no items", true)
					return m, nil
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
		}
	}

	// Everything unhandled scrolls the content viewport when it has focus.
	if m.focus == focusDetail {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// switchMode activates mode idx, resetting selection and reloading.
func (m *Model) switchMode(idx int) {
	if idx == m.modeIdx || idx < 0 || idx >= len(m.modes) {
		return
	}
	m.modeIdx = idx
	m.selected = 0
	m.focus = focusList
	m.setStatus("", false)
	m.reload()
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
	main := lipgloss.JoinHorizontal(lipgloss.Top, m.renderDetailPane(), m.renderListPane())
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

	hints := "[tab] switch pane  [↑↓] move  [enter] open  [r] refresh"
	if len(m.modes) > 1 {
		hints += "  [m] mode"
	}
	for _, act := range m.mode().Actions() {
		hints += fmt.Sprintf("  [%s] %s", act.Key, act.Label)
	}
	hints += "  [q] quit"
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
		Width(m.leftInnerW()).
		Height(m.innerH()).
		Render(m.viewport.View())
}

// renderListPane draws the right nav pane: the mode's rows, hand-rolled, with
// a selection marker and a scrolled window keeping the selection visible.
func (m Model) renderListPane() string {
	borderCol := m.styles.inactive
	if m.focus == focusList {
		borderCol = m.styles.active
	}
	w, h := m.rightInnerW(), m.innerH()

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
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol).
		Width(w).
		Height(h).
		Render(strings.Join(lines, "\n"))
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
