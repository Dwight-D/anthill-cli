package cli

import (
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
)

func itoa(n int) string { return strconv.Itoa(n) }

func sortStrings(s []string) { sort.Strings(s) }

// listFilterKeys are the repeatable, AND-ed (OR within one key) field filters
// of `backlog list`, in flag order.
var listFilterKeys = []string{
	"status", "dispatch", "disposition", "reporter", "priority", "workstream",
	"change-type", "risk", "claimed-by", "session",
}

// listQuery is a parsed `backlog list` invocation.
type listQuery struct {
	filters   map[string][]string
	ready     bool
	active    bool
	untriaged bool
	all       bool
	search    string
	sortKey   string
	reverse   bool
	limit     int
	fields    []string
}

func (q *listQuery) match(it *backlog.Item) bool {
	for key, vals := range q.filters {
		if len(vals) == 0 {
			continue
		}
		got := it.Field(key)
		ok := false
		for _, v := range vals {
			if got == v {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if q.ready && !it.Ready() {
		return false
	}
	if q.active && !it.Active() {
		return false
	}
	if q.untriaged && it.Status != backlog.StatusNew {
		return false
	}
	if q.search != "" {
		needle := strings.ToLower(q.search)
		hay := strings.ToLower(it.Title + "\n" + it.Summary + "\n" + it.Value)
		if !strings.Contains(hay, needle) {
			return false
		}
	}
	return true
}

// walkArchives reports whether done/ and discarded/ must be loaded: --all, or
// a status filter naming an archived status.
func (q *listQuery) walkArchives() bool {
	if q.all {
		return true
	}
	for _, s := range q.filters["status"] {
		if s == backlog.StatusDone || s == backlog.StatusDiscarded {
			return true
		}
	}
	return false
}

// splitValues flattens repeatable, comma-separable flag values.
func splitValues(vals []string) []string {
	var out []string
	for _, v := range vals {
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func (a *App) newBacklogList() *cobra.Command {
	q := &listQuery{filters: map[string][]string{}}
	raw := map[string]*[]string{}
	var fields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List items: filter, sort, and choose columns",
		Long: "Filters are repeatable (OR within a key, AND across keys): --status --dispatch --disposition\n" +
			"--reporter --priority --workstream --change-type --risk --claimed-by --session, plus --search.\n" +
			"Presets: --ready, --active (in-progress + validation), --untriaged (status new), --all (also\n" +
			"walk done/ and discarded/). Sort: rank (fast-tracked, priority, sweep-order, oldest) |\n" +
			"priority | age | updated | id; --reverse; --limit n; --fields a,b,c.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !backlog.IsSortKey(q.sortKey) {
				return usageErr("--sort must be one of " + strings.Join(backlog.SortKeys, "|"))
			}
			if q.limit < 0 {
				return usageErr("--limit must be >= 0")
			}
			for _, k := range listFilterKeys {
				vals := splitValues(*raw[k])
				if enum := backlog.EnumValues(k); enum != nil {
					for _, v := range vals {
						if !enum[v] {
							return usageErr("--" + k + " must be one of " + enumList(k))
						}
					}
				}
				q.filters[k] = vals
			}
			q.fields = nil
			for _, f := range splitValues([]string{fields}) {
				if !backlog.IsFieldName(f) {
					return usageErr("unknown field " + f + " (one of " + strings.Join(backlog.FieldNames, ",") + ")")
				}
				q.fields = append(q.fields, f)
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			for _, ws := range q.filters["workstream"] {
				ok, werr := store.IsWorkstream(ws)
				if werr != nil {
					return internalErr(werr.Error())
				}
				if !ok {
					return notFoundErr("no workstream "+ws, ws)
				}
			}
			items, err := store.Load(q.walkArchives())
			if err != nil {
				return internalErr(err.Error())
			}
			var filtered []*backlog.Item
			for _, it := range items {
				if q.match(it) {
					filtered = append(filtered, it)
				}
			}
			order, _, err := store.SweepOrder()
			if err != nil {
				return internalErr(err.Error())
			}
			backlog.SortItems(filtered, q.sortKey, order, q.reverse)
			if q.limit > 0 && len(filtered) > q.limit {
				filtered = filtered[:q.limit]
			}
			if a.json {
				if len(q.fields) > 0 {
					out := make([]map[string]any, 0, len(filtered))
					for _, it := range filtered {
						out = append(out, projectFields(it, q.fields))
					}
					return a.emitJSON(out)
				}
				return a.emitJSON(viewItems(filtered))
			}
			if len(filtered) == 0 {
				a.note("no items match")
				return nil
			}
			a.printItemTable(filtered, q.fields...)
			return nil
		},
	}
	for _, k := range listFilterKeys {
		raw[k] = cmd.Flags().StringSlice(k, nil, "filter by "+k+" (repeatable)")
	}
	cmd.Flags().BoolVar(&q.ready, "ready", false, "only ready items (agent-implementable unattended)")
	cmd.Flags().BoolVar(&q.active, "active", false, "only in-progress and validation items")
	cmd.Flags().BoolVar(&q.untriaged, "untriaged", false, "only new (untriaged) items")
	cmd.Flags().BoolVar(&q.all, "all", false, "also list archived items (done/, discarded/)")
	cmd.Flags().StringVar(&q.search, "search", "", "case-insensitive substring over title, summary, value")
	cmd.Flags().StringVar(&q.sortKey, "sort", "rank", strings.Join(backlog.SortKeys, "|"))
	cmd.Flags().BoolVar(&q.reverse, "reverse", false, "reverse the sort order")
	cmd.Flags().IntVar(&q.limit, "limit", 0, "show at most n items (0 = all)")
	cmd.Flags().StringVar(&fields, "fields", "", "columns / json keys to show, comma-separated")
	return cmd
}

// projectFields renders the chosen fields of an item as a JSON object; ready
// and active are booleans, everything else a string.
func projectFields(it *backlog.Item, fields []string) map[string]any {
	m := map[string]any{}
	for _, f := range fields {
		switch f {
		case "ready":
			m[f] = it.Ready()
		case "active":
			m[f] = it.Active()
		default:
			m[f] = it.Field(f)
		}
	}
	return m
}
