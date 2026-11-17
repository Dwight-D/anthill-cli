package backlog

import "sort"

// Sort keys accepted by SortItems.
var SortKeys = []string{"rank", "priority", "age", "updated", "id"}

// IsSortKey reports whether k is a legal sort key.
func IsSortKey(k string) bool {
	for _, s := range SortKeys {
		if s == k {
			return true
		}
	}
	return false
}

// SortItems orders items in place (stable). key is one of:
//
//	"rank"     — fast-tracked first → priority (high, normal, low) →
//	             workstream sweep-order → oldest created-at first → id (default)
//	"priority" — priority, then rank
//	"age"      — oldest created-at first, then id
//	"updated"  — most recent stamp first, then id
//	"id"       — lexicographic id
//
// order is the workstream sweep order (from Store.SweepOrder); workstreams
// absent from it rank after listed ones, and intake items rank last. reverse
// inverts the final order.
func SortItems(items []*Item, key string, order []string, reverse bool) {
	wsIndex := map[string]int{}
	for i, w := range order {
		wsIndex[w] = i
	}
	wsRank := func(it *Item) int {
		if it.Workstream == "" {
			return len(order) + 1
		}
		if i, ok := wsIndex[it.Workstream]; ok {
			return i
		}
		return len(order)
	}
	ft := func(it *Item) int {
		if it.Dispatch == DispatchFastTracked {
			return 0
		}
		return 1
	}
	// age compares created-at ascending; a missing stamp sorts after any stamp.
	ageLess := func(a, b *Item) (bool, bool) {
		if a.CreatedAt == b.CreatedAt {
			return false, false
		}
		if a.CreatedAt == "" {
			return false, true
		}
		if b.CreatedAt == "" {
			return true, true
		}
		return a.CreatedAt < b.CreatedAt, true
	}
	rankLess := func(a, b *Item) bool {
		if ft(a) != ft(b) {
			return ft(a) < ft(b)
		}
		if PriorityRank(a.Priority) != PriorityRank(b.Priority) {
			return PriorityRank(a.Priority) < PriorityRank(b.Priority)
		}
		if wsRank(a) != wsRank(b) {
			return wsRank(a) < wsRank(b)
		}
		if l, decided := ageLess(a, b); decided {
			return l
		}
		return a.ID < b.ID
	}
	less := func(a, b *Item) bool {
		switch key {
		case "id":
			return a.ID < b.ID
		case "priority":
			if PriorityRank(a.Priority) != PriorityRank(b.Priority) {
				return PriorityRank(a.Priority) < PriorityRank(b.Priority)
			}
			return rankLess(a, b)
		case "age":
			if l, decided := ageLess(a, b); decided {
				return l
			}
			return a.ID < b.ID
		case "updated":
			ua, ub := a.Updated(), b.Updated()
			if ua != ub {
				return ua > ub
			}
			return a.ID < b.ID
		default:
			return rankLess(a, b)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if reverse {
			return less(items[j], items[i])
		}
		return less(items[i], items[j])
	})
}

// PriorityRank orders priorities: high 0, normal (or absent) 1, low 2.
func PriorityRank(p string) int {
	switch p {
	case "high":
		return 0
	case "low":
		return 2
	}
	return 1
}

// Updated is the latest of the item's RFC3339 stamps (lexicographic order
// equals chronological order for RFC3339 UTC).
func (it *Item) Updated() string {
	latest := ""
	for _, s := range []string{it.CreatedAt, it.TriagedAt, it.ClaimedAt, it.DeliveredAt, it.ClosedAt} {
		if s > latest {
			latest = s
		}
	}
	return latest
}
