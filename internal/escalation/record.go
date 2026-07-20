// Package escalation is the file-backed store for escalation records under
// <root>/.anthill/escalations. The CLI owns the escalation frontmatter schema
// and section skeleton, invariant-checking every write.
package escalation

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// SectionOrder is the canonical order of the record's markdown sections.
var SectionOrder = []string{
	"Question",
	"Context & attempted remedies",
	"Options & recommendation",
	"Decision",
	"Applied",
}

// Tiers that may receive an escalation.
var toTiers = map[string]bool{"dispatcher": true, "supervisor": true, "user": true}

// Statuses of a record.
var statuses = map[string]bool{"open": true, "answered": true, "applied": true}

// Record is one escalation: frontmatter fields plus the ordered section bodies.
type Record struct {
	To     string `yaml:"to"`
	From   string `yaml:"from"`
	Item   string `yaml:"item,omitempty"`
	Status string `yaml:"status"`
	Opened string `yaml:"opened"`

	// Derived, never serialised into frontmatter.
	ID       string            `yaml:"-"`
	Path     string            `yaml:"-"`
	Sections map[string]string `yaml:"-"`
}

// Section returns the trimmed body of a named section (empty if absent).
func (r *Record) Section(name string) string { return strings.TrimSpace(r.Sections[name]) }

// marshalFile renders the record to a full markdown file.
func (r *Record) marshalFile() ([]byte, error) {
	fmStruct := struct {
		To     string `yaml:"to"`
		From   string `yaml:"from"`
		Item   string `yaml:"item,omitempty"`
		Status string `yaml:"status"`
		Opened string `yaml:"opened"`
	}{r.To, r.From, r.Item, r.Status, r.Opened}
	y, err := yaml.Marshal(fmStruct)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}
	var b strings.Builder
	for _, name := range SectionOrder {
		b.WriteString("## ")
		b.WriteString(name)
		b.WriteString("\n")
		if body := strings.TrimSpace(r.Sections[name]); body != "" {
			b.WriteString(body)
			b.WriteString("\n")
		}
	}
	return mdfile.Compose(y, b.String()), nil
}

// parseRecord decodes a record file's bytes.
func parseRecord(data []byte, id, path string) (*Record, error) {
	fm, body, err := mdfile.Split(data)
	if err != nil {
		return nil, err
	}
	var r Record
	if err := yaml.Unmarshal(fm, &r); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	r.ID = id
	r.Path = path
	r.Sections = parseSections(body)
	return &r, nil
}

// parseSections splits a markdown body into a map of "## Header" -> body text.
func parseSections(body string) map[string]string {
	sections := map[string]string{}
	var cur string
	var buf []string
	flush := func() {
		if cur != "" {
			sections[cur] = strings.TrimSpace(strings.Join(buf, "\n"))
		}
		buf = nil
	}
	for _, line := range strings.Split(body, "\n") {
		if h := strings.TrimPrefix(line, "## "); h != line {
			flush()
			cur = strings.TrimSpace(h)
			continue
		}
		buf = append(buf, line)
	}
	flush()
	return sections
}
