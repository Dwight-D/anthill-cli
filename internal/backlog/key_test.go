package backlog

import (
	"errors"
	"sort"
	"testing"
)

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
	long := "this is a very long title that will certainly exceed the forty character budget for slugs"
	got := Slug(long)
	if len(got) > maxSlugLen {
		t.Errorf("Slug len = %d, want <= %d (%q)", len(got), maxSlugLen, got)
	}
	if got[len(got)-1] == '-' {
		t.Errorf("Slug %q ends with a hyphen", got)
	}
}

func TestParseRefTable(t *testing.T) {
	cases := []struct {
		in     string
		prefix string
		key    string // expected canonical key, "" when a former id or invalid
		former string
		err    bool
	}{
		{"ANT-74", "ANT", "ANT-74", "", false},
		{"ant-74", "ANT", "ANT-74", "", false},
		{" Ant-74 ", "ANT", "ANT-74", "", false},
		{"74", "ANT", "ANT-74", "", false},
		{"BH-3", "", "BH-3", "", false}, // no installation prefix: any well-formed prefix
		{"add-a-json-flag", "ANT", "", "add-a-json-flag", false},
		{"2026-item", "ANT", "", "2026-item", false},
		{"../x", "ANT", "", "", true},
		{"ANT-0", "ANT", "", "", true},
		{"ANT-074", "ANT", "", "", true},
		{"A-1", "ANT", "", "", true},
		{"ABCDEF-1", "ANT", "", "", true},
		{"BH-3", "ANT", "", "", true}, // wrong prefix for this installation
		{"", "ANT", "", "", true},
		{"Foo", "ANT", "", "", true},
		{"a/b", "ANT", "", "", true},
		{"0", "ANT", "", "", true},
	}
	for _, c := range cases {
		ref, err := ParseRef(c.in, c.prefix)
		if c.err {
			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("ParseRef(%q) err = %v, want ErrInvalidID", c.in, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRef(%q): %v", c.in, err)
			continue
		}
		if c.former != "" {
			if ref.IsKey() || ref.Former != c.former {
				t.Errorf("ParseRef(%q) = %+v, want former %q", c.in, ref, c.former)
			}
			continue
		}
		if !ref.IsKey() || ref.Key.String() != c.key {
			t.Errorf("ParseRef(%q) = %+v, want key %q", c.in, ref, c.key)
		}
	}
	if _, err := ParseKey("add-a-json-flag", "ANT"); !errors.Is(err, ErrInvalidID) {
		t.Errorf("ParseKey must reject a former id, got %v", err)
	}
}

func TestKeyRendering(t *testing.T) {
	k := Key{Prefix: "ANT", Number: 74}
	if k.String() != "ANT-74" || k.FileStem() != "0074" {
		t.Errorf("String/FileStem = %q/%q", k.String(), k.FileStem())
	}
	if (Key{Prefix: "ANT", Number: 12345}).FileStem() != "12345" {
		t.Error("FileStem must not truncate five digits")
	}
	if (Key{Number: 7}).String() != "7" {
		t.Error("a prefix-less key renders as the bare number")
	}
	if !(Key{Number: 74}).Less(Key{Number: 100}) {
		t.Error("Less must compare numerically")
	}
}

func TestFilenameTable(t *testing.T) {
	cases := []struct {
		name string
		n    int
		ok   bool
	}{
		{"0074-anthill-claude-now-pane.md", 74, true},
		{"0001-item.md", 1, true},
		{"12345-x.md", 12345, true},
		{"074-x.md", 0, false},
		{"0074.md", 0, false},
		{"0074-.md", 0, false},
		{"0074-Upper.md", 0, false},
		{"anthill-claude-now-pane.md", 0, false},
		{"0000-x.md", 0, false},
		{"0074-x.txt", 0, false},
	}
	for _, c := range cases {
		n, ok := ParseFilename(c.name)
		if ok != c.ok || n != c.n {
			t.Errorf("ParseFilename(%q) = %d,%v want %d,%v", c.name, n, ok, c.n, c.ok)
		}
	}
	if got := FormatFilename(74, "a-slug"); got != "0074-a-slug.md" {
		t.Errorf("FormatFilename = %q", got)
	}
	if got := FormatFilename(100, ""); got != "0100-item.md" {
		t.Errorf("FormatFilename empty slug = %q", got)
	}
}

func TestNumericSort(t *testing.T) {
	items := []*Item{
		{ID: "ANT-100", Number: 100},
		{ID: "ANT-74", Number: 74},
		{ID: "ANT-9", Number: 9},
	}
	sort.Slice(items, func(i, j int) bool { return idLess(items[i], items[j]) })
	if items[0].ID != "ANT-9" || items[1].ID != "ANT-74" || items[2].ID != "ANT-100" {
		t.Errorf("numeric order broken: %s %s %s", items[0].ID, items[1].ID, items[2].ID)
	}
	SortItems(items, "id", nil, true)
	if items[0].ID != "ANT-100" {
		t.Errorf("--sort id reversed: first = %s", items[0].ID)
	}
}

func TestStampFrontmatterKey(t *testing.T) {
	in := "---\nschema: 2\nsweep-order: cli\n---\n\n# Workstreams\n"
	got := stampFrontmatterKey(in, "key-prefix", "ANT")
	want := "---\nschema: 2\nsweep-order: cli\nkey-prefix: ANT\n---\n\n# Workstreams\n"
	if got != want {
		t.Errorf("insert:\n%q\nwant\n%q", got, want)
	}
	got = stampFrontmatterKey(got, "key-prefix", "BH")
	if got != "---\nschema: 2\nsweep-order: cli\nkey-prefix: BH\n---\n\n# Workstreams\n" {
		t.Errorf("replace:\n%q", got)
	}
	got = stampFrontmatterKey("---\nkey-prefix: <PREFIX>   # filled by init\nsweep-order: cli\n---\n", "key-prefix", "TST")
	if got != "---\nkey-prefix: TST\nsweep-order: cli\n---\n" {
		t.Errorf("placeholder replace:\n%q", got)
	}
	if got := stampFrontmatterKey("", "schema", "3"); got != "---\nschema: 3\n---\n" {
		t.Errorf("empty file:\n%q", got)
	}
}

func TestDefaultKeyPrefix(t *testing.T) {
	cases := []struct{ root, want string }{
		{"E:/Dev/frameworks/anthill", "ANT"},
		{"/home/me/battlehounds", "BAT"},
		{"/x/b-h", "BH"},
		{"/x/b", "ANT"},
		{"/x/1234", "ANT"},
		{"/x/ab", "AB"},
	}
	for _, c := range cases {
		if got := DefaultKeyPrefix(c.root); got != c.want {
			t.Errorf("DefaultKeyPrefix(%q) = %q, want %q", c.root, got, c.want)
		}
	}
}
