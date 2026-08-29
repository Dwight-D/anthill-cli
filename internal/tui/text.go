package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// truncateCells hard-truncates s to at most w terminal cells, appending an
// ellipsis when it had to cut. Cell-accurate (wide runes count as their
// display width), so fixed columns and no-wrap guards survive CJK titles.
func truncateCells(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// padCells pads s with spaces to exactly w cells (truncating if over).
func padCells(s string, w int) string {
	s = truncateCells(s, w)
	if d := w - lipgloss.Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}
