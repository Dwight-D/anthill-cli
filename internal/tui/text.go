package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

// wrapCells soft-wraps s so no line exceeds w terminal cells: word boundaries
// are preferred and a token wider than w (a URL, a hash) is hard-broken rather
// than left to overflow. Cell-accurate — wide runes count as their display
// width — so a wrapped block can never bleed past a pane edge. The viewport
// does not soft-wrap, so detail content has to arrive pre-wrapped or long
// lines are clipped at the pane edge and become unreadable.
func wrapCells(s string, w int) string {
	if w <= 0 {
		return s
	}
	// A literal tab has no fixed cell width; expand before measuring.
	s = strings.ReplaceAll(s, "\t", "    ")
	out := xansi.Wrap(s, w, "-")
	// Wrapping can leave break whitespace at line ends; trim it per line so
	// trailing padding never widens a line past w.
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

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
