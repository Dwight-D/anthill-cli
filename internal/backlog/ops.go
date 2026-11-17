package backlog

// IsSettableKey reports whether key may be mutated via `backlog set`.
func IsSettableKey(key string) bool { return settableKeys[key] }

// SetField assigns a free frontmatter field by its schema key name. The
// workstream key is not handled here (it triggers a file move via Store.Move);
// status, dispatch, the stamps, and id are never assignable this way.
func (it *Item) SetField(key, value string) {
	switch key {
	case "title":
		it.Title = value
	case "summary":
		it.Summary = value
	case "value":
		it.Value = value
	case "source":
		it.Source = value
	case "hint":
		it.Hint = value
	case "note":
		it.Note = value
	case "priority":
		it.Priority = value
	case "disposition":
		it.Disposition = value
	case "change-type":
		it.ChangeType = value
	case "risk":
		it.Risk = value
	case "verify":
		it.Verify = value
	}
}

// Field returns a frontmatter field (or derived id/path/ready/active) by its
// schema key name, as a display string. Unknown keys return "".
func (it *Item) Field(key string) string {
	switch key {
	case "id":
		return it.ID
	case "path":
		return it.Path
	case "title":
		return it.Title
	case "summary":
		return it.Summary
	case "value":
		return it.Value
	case "workstream":
		return it.Workstream
	case "status":
		return it.Status
	case "dispatch":
		return it.Dispatch
	case "disposition":
		return it.Disposition
	case "priority":
		return it.Priority
	case "reporter":
		return it.Reporter
	case "change-type":
		return it.ChangeType
	case "risk":
		return it.Risk
	case "verify":
		return it.Verify
	case "source":
		return it.Source
	case "hint":
		return it.Hint
	case "note":
		return it.Note
	case "created-at":
		return it.CreatedAt
	case "triaged-at":
		return it.TriagedAt
	case "claimed-at":
		return it.ClaimedAt
	case "claimed-by":
		return it.ClaimedBy
	case "session":
		return it.Session
	case "delivered-at":
		return it.DeliveredAt
	case "closed-at":
		return it.ClosedAt
	case "ready":
		if it.Ready() {
			return "true"
		}
		return "false"
	case "active":
		if it.Active() {
			return "true"
		}
		return "false"
	}
	return ""
}

// FieldNames is every key Field answers, in schema order (the --fields menu).
var FieldNames = []string{
	"id", "title", "summary", "value", "workstream", "status", "dispatch",
	"disposition", "priority", "reporter", "change-type", "risk", "verify",
	"source", "hint", "note", "created-at", "triaged-at", "claimed-at",
	"claimed-by", "session", "delivered-at", "closed-at", "path", "ready", "active",
}

// IsFieldName reports whether key is a valid --fields name.
func IsFieldName(key string) bool {
	for _, f := range FieldNames {
		if f == key {
			return true
		}
	}
	return false
}
