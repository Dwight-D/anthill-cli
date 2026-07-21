package backlog

import "strings"

// Enum value sets for the item frontmatter schema (from bindings.md, owned by
// this CLI as the schema owner).
//
// change-type is deliberately absent here: its vocabulary is the project's
// domain, declared in workstreams.md frontmatter (`change-types`) and read via
// Store.changeTypeConfig. An undeclared change-type is a soft warning, never a
// hard rejection. The never-auto subset lives in the same config
// (`never-auto-change-types`).

var risks = map[string]bool{
	"additive": true, "reversible": true, "behavior-change": true, "subjective": true,
}

var dispositions = map[string]bool{
	"AUTO": true, "REVIEW": true, "DISCARD": true,
}

var statuses = map[string]bool{
	"idea": true, "approved": true, "in-progress": true,
	"blocked": true, "parked": true, "done": true,
}

var priorities = map[string]bool{
	"high": true, "normal": true,
}

var valueVerdictPrefixes = []string{"ADVANCE", "HOLD", "DISCARD"}

// validValueVerdict reports whether v is a legal value-verdict: one of the
// prefixes optionally followed by " — <why>".
func validValueVerdict(v string) bool {
	for _, p := range valueVerdictPrefixes {
		if v == p || strings.HasPrefix(v, p+" ") || strings.HasPrefix(v, p+" —") {
			return true
		}
	}
	return false
}

// settableKeys are the frontmatter keys mutable via `backlog set`. id is
// immutable and absent here; claimed-at is managed by claim/close, not set.
var settableKeys = map[string]bool{
	"workstream": true, "title": true, "value": true, "source": true,
	"hint": true, "change-type": true, "risk": true, "verify": true,
	"value-verdict": true, "disposition": true, "status": true,
	"priority": true, "note": true,
}

// knownKeys is every frontmatter key the schema recognises (for the unknown-key
// typo guard in validation).
var knownKeys = map[string]bool{
	"workstream": true, "title": true, "value": true, "source": true,
	"hint": true, "change-type": true, "risk": true, "verify": true,
	"value-verdict": true, "disposition": true, "status": true,
	"priority": true, "note": true, "claimed-at": true,
}
