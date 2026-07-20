package backlog

import "testing"

func TestSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Add a --json flag", "add-a-json-flag"},
		{"  Trailing & leading  ", "trailing-leading"},
		{"CamelCase Title", "camelcase-title"},
		{"multiple---separators!!!here", "multiple-separators-here"},
		{"123 numbers ok", "123-numbers-ok"},
		{"", ""},
		{"!!!", ""},
	}
	for _, c := range cases {
		if got := Slug(c.in); got != c.want {
			t.Errorf("Slug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSlugTruncatesOnHyphenBoundary(t *testing.T) {
	long := "this is a very long title that will certainly exceed the fifty character budget for ids"
	got := Slug(long)
	if len(got) > maxIDLen {
		t.Errorf("Slug len = %d, want <= %d (%q)", len(got), maxIDLen, got)
	}
	if got[len(got)-1] == '-' {
		t.Errorf("Slug %q ends with a hyphen", got)
	}
}

func TestUniqueID(t *testing.T) {
	taken := map[string]bool{"foo": true, "foo-2": true}
	if got := UniqueID("foo", taken); got != "foo-3" {
		t.Errorf("UniqueID collision = %q, want foo-3", got)
	}
	if got := UniqueID("bar", taken); got != "bar" {
		t.Errorf("UniqueID free = %q, want bar", got)
	}
}

func TestUniqueIDSuffixWithinBudget(t *testing.T) {
	base := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 50 chars
	taken := map[string]bool{base: true}
	got := UniqueID(base, taken)
	if len(got) > maxIDLen {
		t.Errorf("UniqueID len = %d (%q), want <= %d", len(got), got, maxIDLen)
	}
}
