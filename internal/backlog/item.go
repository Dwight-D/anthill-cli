package backlog

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// Item is a single backlog item: the YAML frontmatter fields plus derived,
// never-stored fields (ID, Path, Body). Serialisation order matches the schema
// table so files stay git-friendly and human-readable.
type Item struct {
	Workstream   string `yaml:"workstream,omitempty"`
	Title        string `yaml:"title"`
	Value        string `yaml:"value"`
	Source       string `yaml:"source,omitempty"`
	Hint         string `yaml:"hint,omitempty"`
	ChangeType   string `yaml:"change-type,omitempty"`
	Risk         string `yaml:"risk,omitempty"`
	Verify       string `yaml:"verify,omitempty"`
	ValueVerdict string `yaml:"value-verdict,omitempty"`
	Disposition  string `yaml:"disposition,omitempty"`
	Status       string `yaml:"status"`
	Priority     string `yaml:"priority,omitempty"`
	Note         string `yaml:"note,omitempty"`
	ClaimedAt    string `yaml:"claimed-at,omitempty"`

	// Derived, never serialised into frontmatter.
	ID   string `yaml:"-"`
	Path string `yaml:"-"`
	Body string `yaml:"-"`
}

// Ready reports whether the item is dispatchable: approved with a concrete
// verify (non-empty and not the literal "none").
func (it *Item) Ready() bool {
	return it.Status == "approved" && it.Verify != "" && it.Verify != "none"
}

// marshalFile renders the item to a full markdown file (frontmatter + body).
func (it *Item) marshalFile() ([]byte, error) {
	y, err := yaml.Marshal(it)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}
	return mdfile.Compose(y, it.Body), nil
}

// parseItem decodes a markdown file's bytes into an Item. id and path are the
// derived fields supplied by the caller. Unknown keys are ignored here; the
// strict unknown-key check lives in validation.
func parseItem(data []byte, id, path string) (*Item, error) {
	fm, body, err := mdfile.Split(data)
	if err != nil {
		return nil, err
	}
	var it Item
	if err := yaml.Unmarshal(fm, &it); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	it.ID = id
	it.Path = path
	it.Body = body
	return &it, nil
}
