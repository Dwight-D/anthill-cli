package cli

import (
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
)

func itoa(n int) string { return strconv.Itoa(n) }

func sortStrings(s []string) { sort.Strings(s) }

func (a *App) newBacklogList() *cobra.Command {
	var workstream string
	var untriaged, ready bool
	var status []string
	var sortKey string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List items across intake and workstreams",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !backlog.IsSortKey(sortKey) {
				return usageErr("--sort must be one of rank|priority|age|updated|id")
			}
			store, err := a.backlogStore()
			if err != nil {
				return err
			}
			if workstream != "" {
				ok, werr := store.IsWorkstream(workstream)
				if werr != nil {
					return internalErr(werr.Error())
				}
				if !ok {
					return notFoundErr("no workstream "+workstream, workstream)
				}
			}
			items, err := store.LoadAll()
			if err != nil {
				return internalErr(err.Error())
			}
			statusSet := toSet(status)
			var filtered []*backlog.Item
			for _, it := range items {
				if workstream != "" && it.Workstream != workstream {
					continue
				}
				if untriaged && it.Status != backlog.StatusNew {
					continue
				}
				if ready && !it.Ready() {
					continue
				}
				if len(statusSet) > 0 && !statusSet[it.Status] {
					continue
				}
				filtered = append(filtered, it)
			}
			order, _, err := store.SweepOrder()
			if err != nil {
				return internalErr(err.Error())
			}
			backlog.SortItems(filtered, sortKey, order, false)
			if a.json {
				return a.emitJSON(viewItems(filtered))
			}
			if len(filtered) == 0 {
				a.note("no items match")
				return nil
			}
			a.printItemTable(filtered)
			return nil
		},
	}
	cmd.Flags().StringVar(&workstream, "workstream", "", "restrict to one workstream")
	cmd.Flags().BoolVar(&untriaged, "untriaged", false, "only new (untriaged) items")
	cmd.Flags().BoolVar(&ready, "ready", false, "only ready items")
	cmd.Flags().StringSliceVar(&status, "status", nil, "filter by lifecycle status (repeatable)")
	cmd.Flags().StringVar(&sortKey, "sort", "rank", "rank|priority|age|updated|id")
	return cmd
}
