package backlog

import "fmt"

// PreconditionError indicates an item is not in a state that permits the
// requested lifecycle transition (e.g. approving a non-idea item).
type PreconditionError struct {
	Msg string
}

func (e *PreconditionError) Error() string { return e.Msg }

func precondition(format string, a ...any) *PreconditionError {
	return &PreconditionError{Msg: fmt.Sprintf(format, a...)}
}

// Approve performs the gated idea → approved transition. The human
// confirmation gate (--yes on the CLI, an explicit keypress in the TUI) is the
// caller's responsibility; this method enforces only the state precondition.
func (s *Store) Approve(it *Item) error {
	if it.Status != "idea" {
		return precondition("item %s is %s; only an item in status idea can be approved", it.ID, it.Status)
	}
	it.Status = "approved"
	return s.Save(it)
}

// Close terminates an item: the file is deleted and one line is filed in the
// backlog changelog. disposition is one of done | discard | remove; done
// additionally requires the item to be claimed (in-progress). reason is the
// changelog text and defaults to the item's title.
func (s *Store) Close(it *Item, disposition, reason string) error {
	var logWord string
	switch disposition {
	case "done":
		if it.Status != "in-progress" {
			return precondition("item %s is %s, not in-progress; only a claimed item can be closed --done", it.ID, it.Status)
		}
		logWord = "done"
	case "discard":
		logWord = "discarded"
	case "remove":
		logWord = "removed"
	default:
		return invalid("unknown close disposition %q (want done|discard|remove)", disposition)
	}
	logText := reason
	if logText == "" {
		logText = it.Title
	}
	if err := s.Delete(it); err != nil {
		return err
	}
	return s.AppendChangelog(it.ID, logWord, logText)
}
