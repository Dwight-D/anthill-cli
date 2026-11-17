package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// RenumberResult reports a renumber: the old and new keys, the item's new
// path, and the ledger/escalation records rewritten to the new key.
type RenumberResult struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	Path       string   `json:"path"`
	References []string `json:"references"`
}

var slugTailRe = regexp.MustCompile(`^[0-9]{4,}-(.*)\.md$`)

// Renumber moves an item to number `to` (0: the next free number per
// ITEM_KEYS_SPEC.md §2.2): renames the file in place, raises `sequence`,
// rewrites every sessions-ledger and escalation record naming the old key,
// and appends a Log line. A taken number is ErrConflict. It is the repair
// for a branch-merge collision (two branches allocating the same number).
func (s *Store) Renumber(it *Item, to int, by string) (*RenumberResult, error) {
	if it.Number == 0 {
		return nil, precondition("item %s has no key number; run 'anthill backlog migrate' first", it.ID)
	}
	if to == 0 {
		high, err := s.maxNumber()
		if err != nil {
			return nil, err
		}
		to = high + 1
	} else if to < 1 {
		return nil, invalid("--to must be a positive number")
	} else {
		taken, err := s.numberTaken(to)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, fmt.Errorf("%w: number %d is already taken", ErrConflict, to)
		}
	}
	oldKey := it.ID
	newKey := Key{Prefix: s.prefix(), Number: to}
	slug := ""
	if m := slugTailRe.FindStringSubmatch(filepath.Base(it.Path)); m != nil {
		slug = m[1]
	}
	it.appendLog("renumber "+oldKey+" → "+newKey.String(), ResolveActor(by), "", "")
	oldPath := it.Path
	it.Path = filepath.Join(filepath.Dir(oldPath), FormatFilename(to, slug))
	if err := s.persist(it); err != nil {
		it.Path = oldPath
		return nil, err
	}
	if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("remove old file %s after renumber: %w", oldPath, err)
	}
	it.ID = newKey.String()
	it.Number = to
	if err := s.bumpSequence(to); err != nil {
		s.warnf("warning: sequence file not updated: %v", err)
	}
	refs, err := s.rewriteReferences(map[string]string{oldKey: newKey.String()}, false)
	if err != nil {
		return nil, err
	}
	return &RenumberResult{From: oldKey, To: newKey.String(), Path: it.Path, References: refs}, nil
}
