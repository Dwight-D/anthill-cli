package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/escalation"
)

// itemView is the stable JSON shape of a backlog item: every frontmatter
// field, the derived id/path/ready/active, and optionally the body.
type itemView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary,omitempty"`
	Value       string `json:"value"`
	Workstream  string `json:"workstream,omitempty"`
	Status      string `json:"status"`
	Dispatch    string `json:"dispatch,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Reporter    string `json:"reporter,omitempty"`
	ChangeType  string `json:"change-type,omitempty"`
	Risk        string `json:"risk,omitempty"`
	Verify      string `json:"verify,omitempty"`
	Source      string `json:"source,omitempty"`
	Hint        string `json:"hint,omitempty"`
	Note        string `json:"note,omitempty"`
	CreatedAt   string `json:"created-at,omitempty"`
	TriagedAt   string `json:"triaged-at,omitempty"`
	ClaimedAt   string `json:"claimed-at,omitempty"`
	ClaimedBy   string `json:"claimed-by,omitempty"`
	Session     string `json:"session,omitempty"`
	DeliveredAt string `json:"delivered-at,omitempty"`
	ClosedAt    string `json:"closed-at,omitempty"`
	FormerID    string `json:"former-id,omitempty"`
	Path        string `json:"path"`
	Ready       bool   `json:"ready"`
	Active      bool   `json:"active"`
	Body        string `json:"body,omitempty"`
}

func viewItem(it *backlog.Item, withBody bool) itemView {
	v := itemView{
		ID:          it.ID,
		Title:       it.Title,
		Summary:     it.Summary,
		Value:       it.Value,
		Workstream:  it.Workstream,
		Status:      it.Status,
		Dispatch:    it.Dispatch,
		Disposition: it.Disposition,
		Priority:    it.Priority,
		Reporter:    it.Reporter,
		ChangeType:  it.ChangeType,
		Risk:        it.Risk,
		Verify:      it.Verify,
		Source:      it.Source,
		Hint:        it.Hint,
		Note:        it.Note,
		CreatedAt:   it.CreatedAt,
		TriagedAt:   it.TriagedAt,
		ClaimedAt:   it.ClaimedAt,
		ClaimedBy:   it.ClaimedBy,
		Session:     it.Session,
		DeliveredAt: it.DeliveredAt,
		ClosedAt:    it.ClosedAt,
		FormerID:    it.FormerID,
		Path:        it.Path,
		Ready:       it.Ready(),
		Active:      it.Active(),
	}
	if withBody {
		v.Body = it.Body
	}
	return v
}

func viewItems(items []*backlog.Item) []itemView {
	out := make([]itemView, 0, len(items))
	for _, it := range items {
		out = append(out, viewItem(it, false))
	}
	return out
}

// recordView is the stable JSON shape of an escalation record.
type recordView struct {
	ID       string `json:"id"`
	To       string `json:"to"`
	From     string `json:"from"`
	Item     string `json:"item,omitempty"`
	Status   string `json:"status"`
	Opened   string `json:"opened"`
	Path     string `json:"path,omitempty"`
	Question string `json:"question,omitempty"`
	Context  string `json:"context,omitempty"`
	Options  string `json:"options,omitempty"`
	Decision string `json:"decision,omitempty"`
	Applied  string `json:"applied,omitempty"`
}

func viewRecord(r *escalation.Record) recordView {
	return recordView{
		ID:       r.ID,
		To:       r.To,
		From:     r.From,
		Item:     r.Item,
		Status:   r.Status,
		Opened:   r.Opened,
		Path:     r.Path,
		Question: r.Section("Question"),
		Context:  r.Section("Context & attempted remedies"),
		Options:  r.Section("Options & recommendation"),
		Decision: r.Section("Decision"),
		Applied:  r.Section("Applied"),
	}
}

func viewRecords(recs []*escalation.Record) []recordView {
	out := make([]recordView, 0, len(recs))
	for _, r := range recs {
		out = append(out, viewRecord(r))
	}
	return out
}

// printItemDetail renders one item's fields (and optionally body) to stdout.
func (a *App) printItemDetail(it *backlog.Item, withBody bool) {
	for _, k := range backlog.FieldNames {
		if k == "path" {
			continue
		}
		writeOpt(a.out, k, it.Field(k))
	}
	if withBody && it.Body != "" {
		fmt.Fprintf(a.out, "\n%s\n", it.Body)
	}
}

func writeOpt(w io.Writer, label, val string) {
	if val != "" {
		fmt.Fprintf(w, "%-14s %s\n", label+":", val)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// defaultListFields are the human table columns when --fields is not given.
var defaultListFields = []string{"id", "workstream", "status", "dispatch", "priority", "summary"}

// printItemTable renders items as an aligned human table on stdout with the
// given columns. The summary column falls back to the title when empty.
func (a *App) printItemTable(items []*backlog.Item, fields ...string) {
	if len(fields) == 0 {
		fields = defaultListFields
	}
	w := tabwriter.NewWriter(a.out, 0, 2, 2, ' ', 0)
	head := make([]string, len(fields))
	for i, f := range fields {
		head[i] = strings.ToUpper(f)
	}
	fmt.Fprintln(w, strings.Join(head, "\t"))
	for _, it := range items {
		row := make([]string, len(fields))
		for i, f := range fields {
			v := it.Field(f)
			switch {
			case f == "workstream" && v == "":
				v = "intake"
			case f == "summary" && v == "":
				v = it.Title
			}
			if v == "" {
				v = "-"
			}
			if f == "summary" || f == "title" || f == "value" {
				v = truncate(v, 60)
			}
			row[i] = v
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// printRecordTable renders records as an aligned human table on stdout.
func (a *App) printRecordTable(recs []*escalation.Record) {
	w := tabwriter.NewWriter(a.out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "FILE\tTO\tFROM\tSTATUS\tITEM\tQUESTION")
	for _, r := range recs {
		item := r.Item
		if item == "" {
			item = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.To, r.From, r.Status, item, truncate(r.Section("Question"), 50))
	}
	w.Flush()
}
