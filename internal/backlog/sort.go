package backlog

import "sort"

// SortItems orders items in place. mode is one of:
//
//	"sweep"    — workstream sweep-order rank, then priority, then id (default)
//	"priority" — priority (high first), then id
//	"id"       — lexicographic id
//
// order is the workstream sweep order (from Store.SweepOrder); workstreams
// absent from it rank after listed ones, and intake items rank last.
func SortItems(items []*Item, mode string, order []string) {
	wsIndex := map[string]int{}
	for i, w := range order {
		wsIndex[w] = i
	}
	rank := func(it *Item) int {
		if it.Workstream == "" {
			return len(order) + 1
		}
		if i, ok := wsIndex[it.Workstream]; ok {
			return i
		}
		return len(order)
	}
	prio := func(it *Item) int {
		if it.Priority == "high" {
			return 0
		}
		return 1
	}
	sort.SliceStable(items, func(i, j int) bool {
		switch mode {
		case "id":
			return items[i].ID < items[j].ID
		case "priority":
			if prio(items[i]) != prio(items[j]) {
				return prio(items[i]) < prio(items[j])
			}
			return items[i].ID < items[j].ID
		default: // sweep
			if rank(items[i]) != rank(items[j]) {
				return rank(items[i]) < rank(items[j])
			}
			if prio(items[i]) != prio(items[j]) {
				return prio(items[i]) < prio(items[j])
			}
			return items[i].ID < items[j].ID
		}
	})
}
