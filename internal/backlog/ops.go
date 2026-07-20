package backlog

// IsSettableKey reports whether key may be mutated via `backlog set`.
func IsSettableKey(key string) bool { return settableKeys[key] }

// SetField assigns a plain frontmatter field by its schema key name. The
// workstream key is not handled here (it triggers a file move via Store.Move);
// id is immutable and never assignable.
func (it *Item) SetField(key, value string) {
	switch key {
	case "title":
		it.Title = value
	case "value":
		it.Value = value
	case "source":
		it.Source = value
	case "hint":
		it.Hint = value
	case "change-type":
		it.ChangeType = value
	case "risk":
		it.Risk = value
	case "verify":
		it.Verify = value
	case "value-verdict":
		it.ValueVerdict = value
	case "disposition":
		it.Disposition = value
	case "status":
		it.Status = value
	case "priority":
		it.Priority = value
	case "note":
		it.Note = value
	}
}
