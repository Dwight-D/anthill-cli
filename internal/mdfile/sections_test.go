package mdfile

import (
	"strings"
	"testing"
)

func TestParseSectionsOrderAndUnknown(t *testing.T) {
	body := "intro line\n\n## Description\nfirst\n\n## Custom\nkeep me\n## Log\n- a\n- b\n"
	secs := ParseSections(body)
	want := []Section{{"", "intro line"}, {"Description", "first"}, {"Custom", "keep me"}, {"Log", "- a\n- b"}}
	if len(secs) != len(want) {
		t.Fatalf("got %d sections: %+v", len(secs), secs)
	}
	for i := range want {
		if secs[i] != want[i] {
			t.Fatalf("section %d = %+v, want %+v", i, secs[i], want[i])
		}
	}
}

func TestParseSectionsIgnoresFencedHeadings(t *testing.T) {
	body := "## Description\n```\n## not a heading\n```\n## Log\n"
	m := SectionMap(body)
	if !strings.Contains(m["Description"], "## not a heading") {
		t.Fatalf("fenced heading split the section: %+v", m)
	}
	if _, ok := m["not a heading"]; ok {
		t.Fatal("fenced line parsed as heading")
	}
}

func TestParseSectionsEmptyAndPreambleOnly(t *testing.T) {
	if s := ParseSections(""); len(s) != 0 {
		t.Fatalf("empty body: %+v", s)
	}
	s := ParseSections("just text")
	if len(s) != 1 || s[0].Heading != "" || s[0].Body != "just text" {
		t.Fatalf("preamble only: %+v", s)
	}
}

func TestRenderRoundTrip(t *testing.T) {
	body := "## Description\nd\n\n## Acceptance\n\n## Log\n- x"
	if got := RenderSections(ParseSections(body)); got != body {
		t.Fatalf("round trip:\n%q\nwant\n%q", got, body)
	}
}

var order = []string{"Description", "Acceptance", "Triage", "Evidence", "Log"}

func TestAppendToExistingSection(t *testing.T) {
	body := "## Description\nprose  with  spacing\n\n## Evidence\n\n## Log\n- one\n"
	got := AppendToSection(body, "Log", "- two", order)
	want := "## Description\nprose  with  spacing\n\n## Evidence\n\n## Log\n- one\n- two"
	if got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	got = AppendToSection(got, "Evidence", "ran tests", order)
	want = "## Description\nprose  with  spacing\n\n## Evidence\nran tests\n\n## Log\n- one\n- two"
	if got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

func TestAppendCreatesMissingSectionInOrder(t *testing.T) {
	body := "## Description\nd\n\n## Log\n- one"
	got := AppendToSection(body, "Evidence", "e", order)
	want := "## Description\nd\n\n## Evidence\ne\n\n## Log\n- one"
	if got != want {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
	if got := AppendToSection("", "Log", "- first", order); got != "## Log\n- first" {
		t.Fatalf("empty body append: %q", got)
	}
	if got := AppendToSection("free text", "Log", "- first", order); got != "free text\n\n## Log\n- first" {
		t.Fatalf("preamble append: %q", got)
	}
	// Unknown heading with no canonical anchor goes to the end.
	if got := AppendToSection("## Custom\nx", "Log", "- l", order); got != "## Custom\nx\n\n## Log\n- l" {
		t.Fatalf("custom-only append: %q", got)
	}
}
