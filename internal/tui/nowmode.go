package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/session"
)

// NowPollInterval is how often the Now board reloads while it is shown.
const NowPollInterval = 2 * time.Second

// NowMode is the live board of work in flight: one row per active item
// (in-progress or validation) with its claimant, session, and liveness from
// the sessions ledger; a block of validation items awaiting a controller; and
// counts of ready / proposed / blocked items. It polls (Poller) and draws
// single-pane (Board).
type NowMode struct {
	store    *backlog.Store
	sessions *session.Store

	items      []*backlog.Item            // active items, rank order
	validation []*backlog.Item            // the validation subset, rank order
	records    map[string]*session.Record // item id → its best ledger record

	ready, proposed, blocked int
}

// NewNowMode builds the Now mode over the backlog and the sessions ledger.
func NewNowMode(store *backlog.Store, sessions *session.Store) *NowMode {
	return &NowMode{store: store, sessions: sessions}
}

// Name implements Mode.
func (n *NowMode) Name() string { return "Now" }

// PollInterval implements Poller.
func (n *NowMode) PollInterval() time.Duration { return NowPollInterval }

// Reload implements Mode: re-scan the live tree and the ledger.
func (n *NowMode) Reload() error {
	items, err := n.store.LoadAll()
	if err != nil {
		return err
	}
	order, _, err := n.store.SweepOrder()
	if err != nil {
		return err
	}
	backlog.SortItems(items, "rank", order, false)
	recs, err := n.sessions.List()
	if err != nil {
		return err
	}

	n.items, n.validation = nil, nil
	n.ready, n.proposed, n.blocked = 0, 0, 0
	for _, it := range items {
		if it.Active() {
			n.items = append(n.items, it)
		}
		if it.Status == backlog.StatusValidation {
			n.validation = append(n.validation, it)
		}
		if it.Ready() {
			n.ready++
		}
		if it.Status == backlog.StatusBlocked {
			n.blocked++
		}
		if it.Dispatch == backlog.DispatchProposed && !it.Terminal() {
			n.proposed++
		}
	}
	n.records = map[string]*session.Record{}
	for _, it := range n.items {
		if r := bestRecord(it, recs); r != nil {
			n.records[it.ID] = r
		}
	}
	return nil
}

// bestRecord picks the ledger record for an item: among the records naming
// it, those of the item's stamped session win, then the most recently seen.
func bestRecord(it *backlog.Item, recs []*session.Record) *session.Record {
	var best *session.Record
	score := func(r *session.Record) (bool, time.Time) {
		t, _ := r.Seen()
		return it.Session != "" && r.Session == it.Session, t
	}
	for _, r := range recs {
		if r.Item != it.ID {
			continue
		}
		if best == nil {
			best = r
			continue
		}
		bm, bt := score(best)
		rm, rt := score(r)
		if (rm && !bm) || (rm == bm && rt.After(bt)) {
			best = r
		}
	}
	return best
}

// Len implements Mode.
func (n *NowMode) Len() int { return len(n.items) }

// RowID implements Mode.
func (n *NowMode) RowID(i int) string { return n.items[i].ID }

// Fixed column widths (cells) of the board; id and summary share the rest.
const (
	nowStatusW  = 11
	nowByW      = 14
	nowSessionW = 8
	nowAgeW     = 5
	nowWsW      = 10
	nowFixedW   = nowStatusW + nowByW + nowSessionW + nowAgeW + nowWsW + 6 // + separators
)

// nowCols lays out the seven board columns at width.
func nowCols(width int, id, summary, status, by, sess, age, ws string) string {
	flex := width - nowFixedW
	idW := flex * 2 / 5
	if idW < minIDColW {
		idW = minIDColW
	}
	sumW := flex - idW
	if sumW < 1 {
		sumW = 1
	}
	line := padCells(id, idW) + " " + padCells(summary, sumW) + " " +
		padCells(status, nowStatusW) + " " + padCells(by, nowByW) + " " +
		padCells(sess, nowSessionW) + " " + padCells(age, nowAgeW) + " " + truncateCells(ws, nowWsW)
	return truncateCells(line, width)
}

// Row implements Mode: id · summary · status · claimed-by · session (first 8)
// · last-seen age · workstream.
func (n *NowMode) Row(i, width int) string {
	it := n.items[i]
	return nowCols(width, it.ID, orTitle(it), it.Status, dashIfEmpty(it.ClaimedBy),
		dashIfEmpty(short(it.Session)), n.Age(it.ID), dashIfEmpty(it.Workstream))
}

// Age is the last-seen age of an item's session: the ledger's last-seen,
// else the recorded transcript's modification time, else "-".
func (n *NowMode) Age(id string) string {
	if r := n.records[id]; r != nil {
		return r.Age()
	}
	return "-"
}

// BoardLines implements Board: column headings above the rows; the
// validation block and the queue counts below.
func (n *NowMode) BoardLines(width int) (top, bottom []string) {
	pad := strings.Repeat(" ", lipgloss.Width(selectionMarker))
	top = []string{pad + nowCols(width-len(pad), "ID", "SUMMARY", "STATUS", "CLAIMED-BY", "SESSION", "SEEN", "WORKSTREAM")}

	bottom = []string{"", fmt.Sprintf("Awaiting a controller (validation): %d", len(n.validation))}
	for _, it := range n.validation {
		line := "  " + it.ID + " · " + orTitle(it) + " · " + dashIfEmpty(it.ClaimedBy)
		if age := stampAge(it.DeliveredAt); age != "" {
			line += " · delivered " + age + " ago"
		}
		bottom = append(bottom, line)
	}
	bottom = append(bottom, "", fmt.Sprintf("ready %d · proposed %d · blocked %d", n.ready, n.proposed, n.blocked))
	return top, bottom
}

// Detail implements Mode: the item's full detail, then its ledger record.
func (n *NowMode) Detail(i int) string {
	it := n.items[i]
	d := itemDetail(it)
	if r := n.records[it.ID]; r != nil {
		var sb strings.Builder
		sb.WriteString("\n## Session\n")
		for _, kv := range [][2]string{
			{"session", r.Session}, {"agent", r.Agent}, {"handle", r.Handle}, {"role", r.Role},
			{"last-seen", r.LastSeen + " (" + r.Age() + " ago)"}, {"transcript", r.Transcript}, {"cwd", r.Cwd},
		} {
			if kv[1] != "" {
				fmt.Fprintf(&sb, "%-14s %s\n", kv[0]+":", kv[1])
			}
		}
		d += sb.String()
	}
	return d
}

// Actions implements Mode. Each maps 1:1 to a backlog verb: done →
// `close --done` (confirm), reject → `close --reject "<reason>"` (input),
// release → `release` (confirm). transcript prints the ledger's transcript
// path in the status line.
func (n *NowMode) Actions() []Action {
	return []Action{
		{Key: "d", Label: "done", Confirm: true, Do: n.done},
		{Key: "j", Label: "reject", Input: true, DoInput: n.reject},
		{Key: "r", Label: "release", Confirm: true, Do: n.release},
		{Key: "o", Label: "transcript", Do: n.transcript},
	}
}

func (n *NowMode) done(i int) (string, error) {
	it := n.items[i]
	if _, err := n.store.Close(it, backlog.CloseDone, "", "", false); err != nil {
		return "", err
	}
	return "closed " + it.ID + " (done)", nil
}

func (n *NowMode) reject(i int, reason string) (string, error) {
	it := n.items[i]
	if _, err := n.store.Close(it, backlog.CloseReject, reason, "", false); err != nil {
		return "", err
	}
	return "rejected " + it.ID + " (back to in-progress)", nil
}

func (n *NowMode) release(i int) (string, error) {
	it := n.items[i]
	if err := n.store.Release(it, ""); err != nil {
		return "", err
	}
	return "released " + it.ID, nil
}

func (n *NowMode) transcript(i int) (string, error) {
	it := n.items[i]
	if r := n.records[it.ID]; r != nil && r.Transcript != "" {
		return "transcript: " + r.Transcript, nil
	}
	return "", fmt.Errorf("no transcript recorded for %s", it.ID)
}

func orTitle(it *backlog.Item) string {
	if it.Summary != "" {
		return it.Summary
	}
	return it.Title
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// stampAge renders the age of an RFC3339 stamp, or "" when unparseable.
func stampAge(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return session.FormatAge(time.Since(t))
}
