package tui

import "github.com/charmbracelet/lipgloss"

// styleSet is the prebuilt palette, computed once on model construction so a
// future theme swap is a single rebuild.
type styleSet struct {
	active   lipgloss.Color // focused pane border
	inactive lipgloss.Color // unfocused pane border

	dim       lipgloss.Style // hints, unfocused selection, idle status
	errText   lipgloss.Style // error status line
	activeTab lipgloss.Style // active mode indicator in the header
	selection lipgloss.Style // focused selected list row
}

func newStyles() styleSet {
	return styleSet{
		active:   lipgloss.Color("6"), // cyan
		inactive: lipgloss.Color("8"), // bright black
		dim:      lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		errText:  lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		activeTab: lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("6")),
		// Selection pairs color with an achromatic channel (bold + reverse)
		// so it survives monochrome terminals.
		selection: lipgloss.NewStyle().Bold(true).Reverse(true),
	}
}
