package backlog

import "strings"

// maxSlugLen bounds the cosmetic slug tail of an item filename
// (`NNNN-<slug>.md`, ITEM_KEYS_SPEC.md §2.1).
const maxSlugLen = 40

// Slug converts a title into a kebab slug: lowercase; every run of
// non-alphanumeric characters collapses to a single '-'; leading/trailing '-'
// stripped; truncated to <=40 chars on a hyphen boundary where possible, never
// leaving a trailing hyphen. The slug is the cosmetic tail of an item's
// filename; it is never the id.
func Slug(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > maxSlugLen {
		s = s[:maxSlugLen]
		if i := strings.LastIndex(s, "-"); i > 0 {
			s = s[:i]
		}
		s = strings.Trim(s, "-")
	}
	return s
}
