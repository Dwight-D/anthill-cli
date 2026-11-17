package backlog

// Enum value sets for the item frontmatter schema (BACKLOG_V2_SPEC.md §1.1),
// owned by this CLI as the schema owner.
//
// change-type is deliberately absent here: its vocabulary is the project's
// domain, declared in workstreams.md frontmatter (`change-types`) and read via
// Store.changeTypeConfig. An undeclared change-type is a soft warning, never a
// hard rejection. The never-auto subset lives in the same config
// (`never-auto-change-types`).

// Lifecycle statuses (§2).
const (
	StatusNew        = "new"
	StatusTriaged    = "triaged"
	StatusInProgress = "in-progress"
	StatusValidation = "validation"
	StatusDone       = "done"
	StatusBlocked    = "blocked"
	StatusParked     = "parked"
	StatusDiscarded  = "discarded"
)

// Dispatch values (§2.1).
const (
	DispatchProposed    = "proposed"
	DispatchApproved    = "approved"
	DispatchFastTracked = "fast-tracked"
)

// Archive directory names under backlog/ (§1.3). They are never workstreams.
const (
	DoneDir      = "done"
	DiscardedDir = "discarded"
	IntakeDir    = "intake"
)

var statuses = map[string]bool{
	StatusNew: true, StatusTriaged: true, StatusInProgress: true, StatusValidation: true,
	StatusDone: true, StatusBlocked: true, StatusParked: true, StatusDiscarded: true,
}

var dispatches = map[string]bool{
	DispatchProposed: true, DispatchApproved: true, DispatchFastTracked: true,
}

var dispositions = map[string]bool{
	"auto": true, "design": true, "interactive": true, "open": true,
}

var priorities = map[string]bool{
	"high": true, "normal": true, "low": true,
}

var reporters = map[string]bool{
	"user": true, "supervisor": true, "worker": true,
}

var risks = map[string]bool{
	"additive": true, "reversible": true, "behavior-change": true, "subjective": true,
}

// EnumValues returns the legal values of an enum-typed frontmatter key, or nil
// when the key is not a closed enum.
func EnumValues(key string) map[string]bool {
	switch key {
	case "status":
		return statuses
	case "dispatch":
		return dispatches
	case "disposition":
		return dispositions
	case "priority":
		return priorities
	case "reporter":
		return reporters
	case "risk":
		return risks
	}
	return nil
}

// settableKeys are the free frontmatter keys mutable via `backlog set` (§6.5).
// status and dispatch move only through their verbs; the timestamps, claim
// fields and reporter are CLI-owned.
var settableKeys = map[string]bool{
	"title": true, "summary": true, "value": true, "source": true, "hint": true,
	"note": true, "priority": true, "disposition": true, "change-type": true,
	"risk": true, "verify": true, "workstream": true,
}

// SectionOrder is the canonical order of the item body's "## " sections (§1.2).
var SectionOrder = []string{"Description", "Acceptance", "Triage", "Evidence", "Log"}
