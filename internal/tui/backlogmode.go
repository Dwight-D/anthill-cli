package tui

import (
	"fmt"
	"strings"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
)

// BacklogMode is the backlog screen: every item across intake and workstreams
// in rank order, with the lifecycle actions of the backlog verbs.
type BacklogMode struct {
	store *backlog.Store
	items []*backlog.Item
}

// NewBacklogMode builds the backlog mode over a store.
func NewBacklogMode(store *backlog.Store) *BacklogMode {
	return &BacklogMode{store: store}
}

// Name implements Mode.
func (b *BacklogMode) Name() string { return "Backlog" }

// Reload implements Mode: re-scan the live tree and sort by rank.
func (b *BacklogMode) Reload() error {
	items, err := b.store.LoadAll()
	if err != nil {
		return err
	}
	order, _, err := b.store.SweepOrder()
	if err != nil {
		return err
	}
	backlog.SortItems(items, "rank", order, false)
	b.items = items
	return nil
}

// Len implements Mode.
func (b *BacklogMode) Len() int { return len(b.items) }

// RowID implements Mode.
func (b *BacklogMode) RowID(i int) string { return b.items[i].ID }

// List column widths (cells) for the fixed columns; the id column flexes.
const (
	wsColW     = 10
	statusColW = 11
	minIDColW  = 8
)

// Row implements Mode: "id  workstream  status" columns fitted to width.
func (b *BacklogMode) Row(i, width int) string {
	it := b.items[i]
	ws := it.Workstream
	if ws == "" {
		ws = "intake"
	}
	idW := width - wsColW - statusColW - 2
	if idW < minIDColW {
		idW = minIDColW
	}
	line := padCells(it.ID, idW) + " " + padCells(ws, wsColW) + " " + truncateCells(it.Status, statusColW)
	return truncateCells(line, width)
}

// Detail implements Mode: the item's frontmatter fields followed by its body,
// mirroring `backlog show`.
func (b *BacklogMode) Detail(i int) string { return itemDetail(b.items[i]) }

// itemDetail renders an item's frontmatter fields followed by its body, the
// detail text shared by every mode that shows backlog items.
func itemDetail(it *backlog.Item) string {
	var sb strings.Builder
	field := func(label, val string) {
		if val != "" {
			fmt.Fprintf(&sb, "%-14s %s\n", label+":", val)
		}
	}
	for _, k := range backlog.FieldNames {
		if k == "path" {
			continue
		}
		field(k, it.Field(k))
	}
	if it.Body != "" {
		sb.WriteString("\n" + strings.TrimRight(it.Body, "\n") + "\n")
	}
	return sb.String()
}

// Actions implements Mode. Each Do maps 1:1 to a backlog verb: approve →
// `approve` (dispatch approved), done → `close --done`, discard →
// `close --discard`, remove → `close --remove`. discard and remove are
// confirm-gated as destructive.
func (b *BacklogMode) Actions() []Action {
	return []Action{
		{Key: "a", Label: "approve", Do: b.approve},
		{Key: "d", Label: "done", Do: b.closer(backlog.CloseDone)},
		{Key: "x", Label: "discard", Confirm: true, Do: b.closer(backlog.CloseDiscard)},
		{Key: "X", Label: "remove", Confirm: true, Do: b.closer(backlog.CloseRemove)},
	}
}

func (b *BacklogMode) approve(i int) (string, error) {
	it := b.items[i]
	if err := b.store.Approve(it); err != nil {
		return "", err
	}
	return "approved " + it.ID, nil
}

func (b *BacklogMode) closer(mode string) func(int) (string, error) {
	return func(i int) (string, error) {
		it := b.items[i]
		if _, err := b.store.Close(it, mode, "", "", false); err != nil {
			return "", err
		}
		return fmt.Sprintf("closed %s (%s)", it.ID, mode), nil
	}
}
