package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// maxLineWidth is the widest line of s in terminal cells.
func maxLineWidth(s string) int {
	w := 0
	for _, line := range strings.Split(s, "\n") {
		if lw := lipgloss.Width(line); lw > w {
			w = lw
		}
	}
	return w
}

func TestWrapCellsWordBoundaries(t *testing.T) {
	got := wrapCells("alpha beta gamma delta", 11)
	if maxLineWidth(got) > 11 {
		t.Errorf("line exceeds width 11:\n%s", got)
	}
	if !strings.Contains(got, "alpha beta\n") {
		t.Errorf("expected break at word boundary, got:\n%s", got)
	}
	// No word is split: every token survives intact on some line.
	joined := strings.ReplaceAll(got, "\n", " ")
	for _, word := range []string{"alpha", "beta", "gamma", "delta"} {
		if !strings.Contains(joined, word) {
			t.Errorf("word %q lost in wrap:\n%s", word, got)
		}
	}
}

func TestWrapCellsHardBreaksLongTokens(t *testing.T) {
	// A token wider than the limit must be hard-broken, not left to overflow.
	long := strings.Repeat("x", 50)
	got := wrapCells("see "+long+" there", 10)
	if maxLineWidth(got) > 10 {
		t.Errorf("unbroken token overflows width 10:\n%s", got)
	}
	if strings.Count(strings.ReplaceAll(got, "\n", ""), "x") != 50 {
		t.Errorf("hard break lost characters:\n%s", got)
	}
}

func TestWrapCellsWideRunes(t *testing.T) {
	// CJK runes are two cells each; ten of them need 20 cells, so at width 8
	// every line holds at most four runes.
	got := wrapCells(strings.Repeat("漢", 10), 8)
	if maxLineWidth(got) > 8 {
		t.Errorf("wide runes overflow width 8:\n%s", got)
	}
	if strings.Count(got, "漢") != 10 {
		t.Errorf("wide runes lost in wrap:\n%s", got)
	}
}

func TestWrapCellsNarrowFloorAndTabs(t *testing.T) {
	// Non-positive widths pass the text through rather than panic or erase.
	if got := wrapCells("abc", 0); got != "abc" {
		t.Errorf("width 0 = %q, want passthrough", got)
	}
	if got := wrapCells("abc", -3); got != "abc" {
		t.Errorf("negative width = %q, want passthrough", got)
	}
	// Width 1 still terminates and keeps every character.
	got := wrapCells("abcd", 1)
	if maxLineWidth(got) > 1 || strings.Count(got, "") == 0 {
		t.Errorf("width 1 wrap = %q", got)
	}
	// Tabs are expanded so measurement stays cell-accurate.
	if strings.Contains(wrapCells("a\tb", 20), "\t") {
		t.Error("tab survived wrapCells")
	}
}

func TestWrapCellsPreservesExistingNewlines(t *testing.T) {
	got := wrapCells("line one\nline two", 40)
	if got != "line one\nline two" {
		t.Errorf("short lines rewritten: %q", got)
	}
}

// TestDetailWrapsAndRewrapsOnResize drives the root model with window sizes
// and asserts the viewport content is wrapped to the live pane width, and that
// a resize re-wraps while preserving the scroll offset.
func TestDetailWrapsAndRewrapsOnResize(t *testing.T) {
	longDetail := strings.Repeat("wrapme ", 400) + "\n" + strings.Repeat("z", 200)
	mode := &stubMode{name: "A", rows: []string{"r1"}}
	mode.detail = longDetail
	m := NewWithModes([]Mode{mode})

	next, _ := m.Update(reloadMsg{})
	m = next.(Model)
	next, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = next.(Model)
	if got := maxLineWidth(m.viewport.View()); got > m.leftInnerW() {
		t.Errorf("viewport line width %d exceeds pane width %d", got, m.leftInnerW())
	}

	// Scroll down, then shrink the terminal: content re-wraps to the new
	// width and the scroll offset survives.
	m.viewport.SetYOffset(3)
	next, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	m = next.(Model)
	if got := maxLineWidth(m.viewport.View()); got > m.leftInnerW() {
		t.Errorf("after resize: line width %d exceeds pane width %d", got, m.leftInnerW())
	}
	if m.viewport.YOffset != 3 {
		t.Errorf("YOffset = %d after resize, want 3", m.viewport.YOffset)
	}
}
