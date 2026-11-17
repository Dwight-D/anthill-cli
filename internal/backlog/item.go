package backlog

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// Item is a single backlog item: the YAML frontmatter fields plus derived,
// never-stored fields (ID, Path, Body). Serialisation order matches the schema
// table (BACKLOG_V2_SPEC.md §1.1) so files stay git-friendly and readable.
type Item struct {
	Title       string `yaml:"title"`
	Summary     string `yaml:"summary,omitempty"`
	Value       string `yaml:"value"`
	Workstream  string `yaml:"workstream,omitempty"`
	Status      string `yaml:"status"`
	Dispatch    string `yaml:"dispatch,omitempty"`
	Disposition string `yaml:"disposition,omitempty"`
	Priority    string `yaml:"priority,omitempty"`
	Reporter    string `yaml:"reporter,omitempty"`
	ChangeType  string `yaml:"change-type,omitempty"`
	Risk        string `yaml:"risk,omitempty"`
	Verify      string `yaml:"verify,omitempty"`
	Source      string `yaml:"source,omitempty"`
	Hint        string `yaml:"hint,omitempty"`
	Note        string `yaml:"note,omitempty"`
	CreatedAt   string `yaml:"created-at,omitempty"`
	TriagedAt   string `yaml:"triaged-at,omitempty"`
	ClaimedAt   string `yaml:"claimed-at,omitempty"`
	ClaimedBy   string `yaml:"claimed-by,omitempty"`
	Session     string `yaml:"session,omitempty"`
	DeliveredAt string `yaml:"delivered-at,omitempty"`
	ClosedAt    string `yaml:"closed-at,omitempty"`
	FormerID    string `yaml:"former-id,omitempty"` // pre-key slug id, written only by migrate

	// Derived, never serialised into frontmatter. ID is the canonical key
	// (`ANT-74`); Number is its numeric part (0 for an item whose filename is
	// not in key form).
	ID     string `yaml:"-"`
	Number int    `yaml:"-"`
	Path   string `yaml:"-"`
	Body   string `yaml:"-"`
}

// RealVerify reports whether v is a concrete acceptance test (non-empty and not
// the literal "none").
func RealVerify(v string) bool { return v != "" && v != "none" }

// Ready reports whether an agent may pick the item up unattended: triaged,
// dispatch said go (approved or fast-tracked), disposition auto, and a real
// verify.
func (it *Item) Ready() bool {
	return it.Status == StatusTriaged &&
		(it.Dispatch == DispatchApproved || it.Dispatch == DispatchFastTracked) &&
		it.Disposition == "auto" && RealVerify(it.Verify)
}

// Active reports whether the item is being worked (in-progress or validation).
func (it *Item) Active() bool {
	return it.Status == StatusInProgress || it.Status == StatusValidation
}

// Terminal reports whether the item is archived (done or discarded).
func (it *Item) Terminal() bool {
	return it.Status == StatusDone || it.Status == StatusDiscarded
}

// Sections returns the item body's "## " sections as heading -> trimmed text.
func (it *Item) Sections() map[string]string { return mdfile.SectionMap(it.Body) }

// marshalFile renders the item to a full markdown file (frontmatter + body).
func (it *Item) marshalFile() ([]byte, error) {
	y, err := yaml.Marshal(it)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}
	return mdfile.Compose(y, it.Body), nil
}

// parseItem decodes a markdown file's bytes into an Item. id and path are the
// derived fields supplied by the caller; the number is parsed from the
// filename when it is in key form. Unknown keys are ignored here; the strict
// unknown-key check lives in validation.
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
	if n, ok := ParseFilename(filepath.Base(path)); ok {
		it.Number = n
	}
	return &it, nil
}
