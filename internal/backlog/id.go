package backlog

import (
	"fmt"
	"strings"
)

const maxIDLen = 50

// Slug converts a title into a kebab id slug: lowercase; every run of
// non-alphanumeric characters collapses to a single '-'; leading/trailing '-'
// stripped; truncated to <=50 chars on a hyphen boundary where possible, never
// leaving a trailing hyphen.
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
	if len(s) > maxIDLen {
		s = s[:maxIDLen]
		if i := strings.LastIndex(s, "-"); i > 0 {
			s = s[:i]
		}
		s = strings.Trim(s, "-")
	}
	return s
}

// UniqueID returns an id derived from base that is not present in taken. On
// collision it appends the lowest free numeric suffix ("-2", "-3", …); the
// suffix counts toward the 50-char budget, truncating base further if needed.
func UniqueID(base string, taken map[string]bool) string {
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		suffix := fmt.Sprintf("-%d", n)
		b := base
		if len(b)+len(suffix) > maxIDLen {
			cut := maxIDLen - len(suffix)
			if cut < 0 {
				cut = 0
			}
			b = strings.Trim(b[:cut], "-")
		}
		cand := b + suffix
		if !taken[cand] {
			return cand
		}
	}
}
