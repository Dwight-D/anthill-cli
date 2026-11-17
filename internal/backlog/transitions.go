package backlog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// PreconditionError indicates an item is not in a state that permits the
// requested lifecycle transition (e.g. delivering an item that is not claimed).
type PreconditionError struct {
	Msg string
}

func (e *PreconditionError) Error() string { return e.Msg }

func precondition(format string, a ...any) *PreconditionError {
	return &PreconditionError{Msg: fmt.Sprintf(format, a...)}
}

// oneLine collapses whitespace runs (including newlines) to single spaces so a
// reason fits on one Log line.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// appendLog appends one "## Log" line for a transition:
//
//   - <RFC3339> <verb> by <actor> (session <id>): <detail>
func (it *Item) appendLog(verb, actor, session, detail string) {
	line := "- " + Stamp() + " " + verb
	if actor != "" {
		line += " by " + actor
	}
	if session != "" {
		sid := shortID(session)
		if sid != session {
			sid += "…"
		}
		line += " (session " + sid + ")"
	}
	if d := oneLine(detail); d != "" {
		line += ": " + d
	}
	it.Body = mdfile.AppendToSection(it.Body, "Log", line, SectionOrder)
}

// appendSection appends text to a named body section.
func (it *Item) appendSection(heading, text string) {
	it.Body = mdfile.AppendToSection(it.Body, heading, strings.TrimSpace(text), SectionOrder)
}

func forcedVerb(verb string, forced bool) string {
	if forced {
		return verb + " (forced)"
	}
	return verb
}

// TriagedRequired is the field set a triaged item must carry (§6.3).
var TriagedRequired = []string{"workstream", "summary", "change-type", "risk", "verify", "disposition"}

// Triage performs new → triaged: applies fields (free keys plus workstream),
// requires the full triaged field set, checks disposition coherence, strips
// hint, stamps triaged-at, appends note to "## Triage", logs, and moves the
// file into the workstream dir.
func (s *Store) Triage(it *Item, fields map[string]string, note, by string) error {
	if it.Status != StatusNew {
		return precondition("item %s is %s; only a new item can be triaged", it.ID, it.Status)
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ws := ""
	for _, k := range keys {
		v := fields[k]
		switch {
		case k == "workstream":
			ws = strings.TrimSpace(v)
		case k == "status" || k == "dispatch":
			return invalid("%s cannot be set at triage; it moves only through its verbs", k)
		case !settableKeys[k]:
			return invalid("unknown or CLI-owned key %q", k)
		default:
			it.SetField(k, v)
		}
	}
	it.Workstream = ws
	var missing []string
	for _, k := range TriagedRequired {
		if strings.TrimSpace(it.Field(k)) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		it.Workstream = ""
		return invalid("triage requires %s", strings.Join(missing, ", "))
	}
	if err := s.checkDispositionCoherence(it); err != nil {
		it.Workstream = ""
		return err
	}
	it.Hint = ""
	it.Status = StatusTriaged
	it.TriagedAt = Stamp()
	if strings.TrimSpace(note) != "" {
		it.appendSection("Triage", note)
	}
	it.appendLog("triage → "+ws, ResolveActor(by), "", "")
	return s.Move(it, ws)
}

// checkDispositionCoherence enforces §3: auto needs a real verify and a
// change-type outside never-auto-change-types.
func (s *Store) checkDispositionCoherence(it *Item) error {
	if it.Disposition != "auto" {
		return nil
	}
	if !RealVerify(it.Verify) {
		return invalid("disposition auto requires a real verify (not empty or none)")
	}
	_, neverAuto := s.changeTypeConfig()
	if neverAuto[it.ChangeType] {
		return invalid("disposition auto is illegal for never-auto change-type %q", it.ChangeType)
	}
	return nil
}

// Claim performs triaged → in-progress with a compare-and-set on status. A
// non-ready triaged item, or reclaiming an in-progress item, needs force. It
// stamps claimed-at, claimed-by (ResolveActor(by)) and session
// (ResolveSession(session)).
func (s *Store) Claim(it *Item, by, session string, force bool) (*Item, error) {
	forced := false
	switch it.Status {
	case StatusInProgress:
		if !force {
			return nil, fmt.Errorf("%w: item %s is already claimed (in-progress) by %s; use --force to reclaim an orphan",
				ErrConflict, it.ID, orDash(it.ClaimedBy))
		}
		forced = true
	case StatusTriaged:
		if !it.Ready() {
			if !force {
				return nil, precondition("item %s is not ready (needs dispatch approved/fast-tracked, disposition auto, and a real verify); use --force to override", it.ID)
			}
			forced = true
		}
	default:
		return nil, precondition("item %s is %s; only a triaged item can be claimed", it.ID, it.Status)
	}
	expected := it.Status
	fresh, err := s.Find(it.ID)
	if err != nil {
		return nil, err
	}
	if fresh.Status != expected {
		return nil, fmt.Errorf("%w: item %s changed under us (expected %s, found %s)", ErrConflict, it.ID, expected, fresh.Status)
	}
	fresh.Status = StatusInProgress
	fresh.ClaimedAt = Stamp()
	fresh.ClaimedBy = ResolveActor(by)
	fresh.Session = ResolveSession(session)
	fresh.appendLog(forcedVerb("claim", forced), fresh.ClaimedBy, fresh.Session, "")
	if err := s.Save(fresh); err != nil {
		return nil, err
	}
	return fresh, nil
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// Deliver performs in-progress → validation. Only the claimant may deliver
// (caller identity is ResolveActor(by)) unless force. evidence is appended to
// "## Evidence".
func (s *Store) Deliver(it *Item, by, evidence string, force bool) error {
	if it.Status != StatusInProgress {
		return precondition("item %s is %s; only an in-progress item can be delivered", it.ID, it.Status)
	}
	caller := ResolveActor(by)
	forced := false
	if it.ClaimedBy != "" && caller != it.ClaimedBy {
		if !force {
			return precondition("item %s is claimed by %s, not %s; only the claimant may deliver (use --force to override)", it.ID, it.ClaimedBy, caller)
		}
		forced = true
	}
	ts := Stamp()
	it.Status = StatusValidation
	it.DeliveredAt = ts
	it.appendSection("Evidence", "**"+ts+"** — "+caller+"\n"+strings.TrimSpace(evidence))
	it.appendLog(forcedVerb("deliver", forced), caller, "", "")
	return s.Save(it)
}

// Block performs in-progress → blocked; reason is logged, note is untouched.
func (s *Store) Block(it *Item, reason, by string) error {
	if it.Status != StatusInProgress {
		return precondition("item %s is %s; only an in-progress item can be blocked", it.ID, it.Status)
	}
	it.Status = StatusBlocked
	it.appendLog("block", ResolveActor(by), "", reason)
	return s.Save(it)
}

// Unblock performs blocked → in-progress (same claim), or blocked → triaged
// dropping the claim when release is set.
func (s *Store) Unblock(it *Item, release bool, by string) error {
	if it.Status != StatusBlocked {
		return precondition("item %s is %s; only a blocked item can be unblocked", it.ID, it.Status)
	}
	if release {
		it.Status = StatusTriaged
		clearClaim(it)
		it.appendLog("unblock --release", ResolveActor(by), "", "")
	} else {
		it.Status = StatusInProgress
		it.appendLog("unblock", ResolveActor(by), "", "")
	}
	return s.Save(it)
}

func clearClaim(it *Item) {
	it.ClaimedAt, it.ClaimedBy, it.Session = "", "", ""
}

// Release performs in-progress → triaged, dropping the claim fields.
func (s *Store) Release(it *Item, by string) error {
	if it.Status != StatusInProgress {
		return precondition("item %s is %s; only an in-progress item can be released", it.ID, it.Status)
	}
	prev := it.ClaimedBy
	it.Status = StatusTriaged
	clearClaim(it)
	it.appendLog("release", ResolveActor(by), "", "was claimed by "+orDash(prev))
	return s.Save(it)
}

// Park performs new|triaged → parked.
func (s *Store) Park(it *Item, reason, by string) error {
	if it.Status != StatusNew && it.Status != StatusTriaged {
		return precondition("item %s is %s; only a new or triaged item can be parked", it.ID, it.Status)
	}
	it.Status = StatusParked
	it.appendLog("park", ResolveActor(by), "", reason)
	return s.Save(it)
}

// Unpark returns a parked item to its prior status: new when it sits in
// intake (no workstream), triaged otherwise.
func (s *Store) Unpark(it *Item, by string) error {
	if it.Status != StatusParked {
		return precondition("item %s is %s; only a parked item can be unparked", it.ID, it.Status)
	}
	if it.Workstream == "" {
		it.Status = StatusNew
	} else {
		it.Status = StatusTriaged
	}
	it.appendLog("unpark → "+it.Status, ResolveActor(by), "", "")
	return s.Save(it)
}

// dispatchVerb names the verb that sets each dispatch value.
var dispatchVerb = map[string]string{
	DispatchApproved:    "approve",
	DispatchFastTracked: "fast-track",
	DispatchProposed:    "unapprove",
}

// SetDispatch moves the user's go/no-go (approve, fast-track, unapprove). It
// reports whether the value changed; an unchanged value is a no-op.
func (s *Store) SetDispatch(it *Item, value, by string) (bool, error) {
	if !dispatches[value] {
		return false, invalid("illegal dispatch %q", value)
	}
	if it.Terminal() {
		return false, precondition("item %s is %s (archived); dispatch cannot change", it.ID, it.Status)
	}
	if it.Dispatch == value {
		return false, nil
	}
	it.Dispatch = value
	it.appendLog(dispatchVerb[value], ResolveActor(by), "", "")
	return true, s.Save(it)
}

// Approve sets dispatch to approved (the TUI's approve action).
func (s *Store) Approve(it *Item) error {
	_, err := s.SetDispatch(it, DispatchApproved, "")
	return err
}

// Reassign moves a triaged (or later, non-terminal) item to another
// workstream. Intake items move only through Triage.
func (s *Store) Reassign(it *Item, ws, by string) error {
	if it.Status == StatusNew || it.Workstream == "" {
		return precondition("item %s is untriaged; use 'anthill backlog triage' to give it a workstream", it.ID)
	}
	if it.Terminal() {
		return precondition("item %s is %s (archived); its workstream is history", it.ID, it.Status)
	}
	if ws == it.Workstream {
		return nil
	}
	from := it.Workstream
	it.appendLog("move "+from+" → "+ws, ResolveActor(by), "", "")
	return s.Move(it, ws)
}

// Close modes.
const (
	CloseDone    = "done"
	CloseReject  = "reject"
	CloseDiscard = "discard"
	CloseRemove  = "remove"
)

// Close applies one close mode:
//
//	done    validation → done; archived in done/ (force: from any live status)
//	reject  validation → in-progress, keeping the claim (reason required)
//	discard any live status → discarded; archived in discarded/
//	remove  any status → file deleted
//
// done, discard and remove file one CHANGELOG line (reason, or the title).
// It reports whether a CHANGELOG line was written.
func (s *Store) Close(it *Item, mode, reason, by string, force bool) (bool, error) {
	actor := ResolveActor(by)
	logText := oneLine(reason)
	if logText == "" {
		logText = it.Title
	}
	switch mode {
	case CloseDone:
		forced := false
		if it.Status != StatusValidation {
			if !force || it.Terminal() {
				return false, precondition("item %s is %s; only an item in validation can be closed --done", it.ID, it.Status)
			}
			forced = true
		}
		it.Status = StatusDone
		it.ClosedAt = Stamp()
		it.appendLog(forcedVerb("close --done", forced), actor, "", reason)
		if err := s.relocate(it, s.doneDir()); err != nil {
			return false, err
		}
		return true, s.AppendChangelog(it.ID, "done", logText)
	case CloseReject:
		if it.Status != StatusValidation {
			return false, precondition("item %s is %s; only an item in validation can be rejected", it.ID, it.Status)
		}
		if strings.TrimSpace(reason) == "" {
			return false, invalid("close --reject requires a reason")
		}
		it.Status = StatusInProgress
		it.appendLog("close --reject", actor, "", reason)
		return false, s.Save(it)
	case CloseDiscard:
		if it.Terminal() {
			return false, precondition("item %s is already %s", it.ID, it.Status)
		}
		it.Status = StatusDiscarded
		it.ClosedAt = Stamp()
		it.appendLog("close --discard", actor, "", reason)
		if err := s.relocate(it, s.discardedDir()); err != nil {
			return false, err
		}
		return true, s.AppendChangelog(it.ID, "discarded", logText)
	case CloseRemove:
		if err := s.Delete(it); err != nil {
			return false, err
		}
		return true, s.AppendChangelog(it.ID, "removed", logText)
	}
	return false, invalid("unknown close mode %q (want done|reject|discard|remove)", mode)
}
