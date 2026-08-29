package tui

import (
	"fmt"
	"strings"

	"github.com/Dwight-D/anthill-cli/internal/escalation"
)

// EscalationsMode is the escalations screen: every record in the escalation
// directory, with the lifecycle actions of the escalation verbs. Answering an
// open record takes free text through the root's overlay; raising stays on
// the scriptable `escalation raise` verb.
type EscalationsMode struct {
	store *escalation.Store
	recs  []*escalation.Record
}

// NewEscalationsMode builds the escalations mode over a store.
func NewEscalationsMode(store *escalation.Store) *EscalationsMode {
	return &EscalationsMode{store: store}
}

// Name implements Mode.
func (e *EscalationsMode) Name() string { return "Escalations" }

// Reload implements Mode: re-scan the escalation directory (sorted by id).
func (e *EscalationsMode) Reload() error {
	recs, err := e.store.LoadAll()
	if err != nil {
		return err
	}
	e.recs = recs
	return nil
}

// Len implements Mode.
func (e *EscalationsMode) Len() int { return len(e.recs) }

// RowID implements Mode.
func (e *EscalationsMode) RowID(i int) string { return e.recs[i].ID }

// List column widths (cells) for the fixed columns; the id column flexes.
const (
	escToColW     = 10
	escStatusColW = 8
	minEscIDColW  = 12
)

// Row implements Mode: "id  to  status" columns fitted to width.
func (e *EscalationsMode) Row(i, width int) string {
	r := e.recs[i]
	idW := width - escToColW - escStatusColW - 2
	if idW < minEscIDColW {
		idW = minEscIDColW
	}
	line := padCells(r.ID, idW) + " " + padCells(r.To, escToColW) + " " + truncateCells(r.Status, escStatusColW)
	return truncateCells(line, width)
}

// Detail implements Mode: the record's frontmatter fields followed by its
// sections, mirroring `escalation show`.
func (e *EscalationsMode) Detail(i int) string {
	r := e.recs[i]
	var sb strings.Builder
	field := func(label, val string) {
		if val != "" {
			fmt.Fprintf(&sb, "%-8s %s\n", label+":", val)
		}
	}
	field("id", r.ID)
	field("to", r.To)
	field("from", r.From)
	field("item", r.Item)
	field("status", r.Status)
	field("opened", r.Opened)
	for _, name := range escalation.SectionOrder {
		if body := r.Section(name); body != "" {
			fmt.Fprintf(&sb, "\n## %s\n%s\n", name, body)
		}
	}
	return sb.String()
}

// Actions implements Mode. Each Do/DoInput maps 1:1 to an escalation verb's
// domain transition; close and remove are confirm-gated as destructive.
// Answer takes the [a] key as the primary act on an open record; apply sits
// on its mnemonic [p].
func (e *EscalationsMode) Actions() []Action {
	return []Action{
		{Key: "a", Label: "answer", Input: true, DoInput: e.answer},
		{Key: "p", Label: "apply", Do: e.apply},
		{Key: "x", Label: "close", Confirm: true, Do: e.close},
		{Key: "X", Label: "remove", Confirm: true, Do: e.remove},
	}
}

func (e *EscalationsMode) answer(i int, text string) (string, error) {
	r, err := e.store.Answer(e.recs[i].ID, text)
	if err != nil {
		return "", err
	}
	return "answered " + r.ID, nil
}

func (e *EscalationsMode) apply(i int) (string, error) {
	r, err := e.store.Apply(e.recs[i].ID, "")
	if err != nil {
		return "", err
	}
	return "applied " + r.ID, nil
}

func (e *EscalationsMode) close(i int) (string, error) {
	r, err := e.store.Close(e.recs[i].ID, "")
	if err != nil {
		return "", err
	}
	return "closed " + r.ID, nil
}

func (e *EscalationsMode) remove(i int) (string, error) {
	r, err := e.store.Remove(e.recs[i].ID, "")
	if err != nil {
		return "", err
	}
	return "removed " + r.ID, nil
}
