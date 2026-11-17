package backlog

import (
	"os"
	"time"
)

// PruneResult reports a prune run.
type PruneResult struct {
	DryRun bool     `json:"dry_run"`
	Pruned []string `json:"pruned"`
	Kept   int      `json:"kept"`
}

// Prune deletes items archived in discarded/ — all of them, or only those
// closed longer than olderThan ago (olderThan > 0). An item's age is its
// closed-at stamp, falling back to the file's modification time. Each deletion
// files one CHANGELOG line ("purged"). done/ is never pruned.
func (s *Store) Prune(olderThan time.Duration, dryRun bool) (*PruneResult, error) {
	res := &PruneResult{DryRun: dryRun, Pruned: []string{}}
	items, err := s.scanDir(s.discardedDir())
	if err != nil {
		return nil, err
	}
	cutoff := now().Add(-olderThan)
	for _, it := range items {
		if olderThan > 0 {
			closed, perr := time.Parse(time.RFC3339, it.ClosedAt)
			if perr != nil {
				info, serr := os.Stat(it.Path)
				if serr != nil {
					return nil, serr
				}
				closed = info.ModTime()
			}
			if closed.After(cutoff) {
				res.Kept++
				continue
			}
		}
		res.Pruned = append(res.Pruned, it.ID)
		if dryRun {
			continue
		}
		if err := s.Delete(it); err != nil {
			return nil, err
		}
		if err := s.AppendChangelog(it.ID, "purged", "discarded item pruned"); err != nil {
			return nil, err
		}
	}
	return res, nil
}
