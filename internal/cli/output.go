package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/escalation"
)

// itemView is the stable JSON shape of a backlog item (frontmatter fields plus
// derived id/path/ready, and optionally the body).
type itemView struct {
	ID           string `json:"id"`
	Workstream   string `json:"workstream,omitempty"`
	Title        string `json:"title"`
	Value        string `json:"value"`
	Source       string `json:"source,omitempty"`
	Hint         string `json:"hint,omitempty"`
	ChangeType   string `json:"change-type,omitempty"`
	Risk         string `json:"risk,omitempty"`
	Verify       string `json:"verify,omitempty"`
	ValueVerdict string `json:"value-verdict,omitempty"`
	Disposition  string `json:"disposition,omitempty"`
	Status       string `json:"status"`
	Priority     string `json:"priority,omitempty"`
	Note         string `json:"note,omitempty"`
	ClaimedAt    string `json:"claimed-at,omitempty"`
	Path         string `json:"path"`
	Ready        bool   `json:"ready"`
	Body         string `json:"body,omitempty"`
}

func viewItem(it *backlog.Item, withBody bool) itemView {
	v := itemView{
		ID:           it.ID,
		Workstream:   it.Workstream,
		Title:        it.Title,
		Value:        it.Value,
		Source:       it.Source,
		Hint:         it.Hint,
		ChangeType:   it.ChangeType,
		Risk:         it.Risk,
		Verify:       it.Verify,
		ValueVerdict: it.ValueVerdict,
		Disposition:  it.Disposition,
		Status:       it.Status,
		Priority:     it.Priority,
		Note:         it.Note,
		ClaimedAt:    it.ClaimedAt,
		Path:         it.Path,
		Ready:        it.Ready(),
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

func settableKeyOK(k string) bool { return backlog.IsSettableKey(k) }

func applyItemKey(it *backlog.Item, k, v string) { it.SetField(k, v) }

// printItemDetail renders one item's fields (and optionally body) to stdout.
func (a *App) printItemDetail(it *backlog.Item, withBody bool) {
	fmt.Fprintf(a.out, "id:            %s\n", it.ID)
	if it.Workstream != "" {
		fmt.Fprintf(a.out, "workstream:    %s\n", it.Workstream)
	}
	fmt.Fprintf(a.out, "title:         %s\n", it.Title)
	fmt.Fprintf(a.out, "value:         %s\n", it.Value)
	writeOpt(a.out, "source", it.Source)
	writeOpt(a.out, "hint", it.Hint)
	writeOpt(a.out, "change-type", it.ChangeType)
	writeOpt(a.out, "risk", it.Risk)
	writeOpt(a.out, "verify", it.Verify)
	writeOpt(a.out, "value-verdict", it.ValueVerdict)
	writeOpt(a.out, "disposition", it.Disposition)
	fmt.Fprintf(a.out, "status:        %s\n", it.Status)
	writeOpt(a.out, "priority", it.Priority)
	writeOpt(a.out, "note", it.Note)
	writeOpt(a.out, "claimed-at", it.ClaimedAt)
	fmt.Fprintf(a.out, "ready:         %v\n", it.Ready())
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

// printItemTable renders items as an aligned human table on stdout.
func (a *App) printItemTable(items []*backlog.Item) {
	w := tabwriter.NewWriter(a.out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tWORKSTREAM\tSTATUS\tPRIORITY\tTITLE")
	for _, it := range items {
		ws := it.Workstream
		if ws == "" {
			ws = "intake"
		}
		pr := it.Priority
		if pr == "" {
			pr = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", it.ID, ws, it.Status, pr, truncate(it.Title, 50))
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
