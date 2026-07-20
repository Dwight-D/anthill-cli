package backlog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// ErrNotFound indicates a referenced item or workstream does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict indicates a compare-and-set lost: the item's on-disk status is no
// longer the value a claim was computed against, or it is already claimed.
var ErrConflict = errors.New("conflict")

// ValidationError names the schema invariant a rejected write would violate.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) *ValidationError {
	return &ValidationError{Msg: fmt.Sprintf(format, a...)}
}

// Store is the file-backed backlog under <root>/.anthill/backlog.
type Store struct {
	root string
}

// NewStore returns a store rooted at the directory containing .anthill.
func NewStore(root string) *Store { return &Store{root: root} }

func (s *Store) dir() string            { return filepath.Join(s.root, ".anthill", "backlog") }
func (s *Store) intakeDir() string      { return filepath.Join(s.dir(), "intake") }
func (s *Store) wsDir(ws string) string { return filepath.Join(s.dir(), ws) }

// ChangelogPath is the backlog changelog file.
func (s *Store) ChangelogPath() string { return filepath.Join(s.dir(), "CHANGELOG.md") }

// Workstreams returns the workstream directory names (every subdirectory of
// backlog/ except intake), sorted.
func (s *Store) Workstreams() ([]string, error) {
	entries, err := os.ReadDir(s.dir())
	if err != nil {
		return nil, fmt.Errorf("read backlog dir: %w", err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "intake" {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// IsWorkstream reports whether ws names an existing workstream directory.
func (s *Store) IsWorkstream(ws string) (bool, error) {
	info, err := os.Stat(s.wsDir(ws))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir() && ws != "intake", nil
}

// SweepOrder reads workstreams.md frontmatter and returns the sweep order plus
// the never-implicit set. Workstreams present on disk but missing from the
// sweep list are appended in sorted order.
func (s *Store) SweepOrder() (order []string, neverImplicit map[string]bool, err error) {
	neverImplicit = map[string]bool{}
	path := filepath.Join(s.dir(), "workstreams.md")
	data, err := os.ReadFile(path)
	var listed []string
	if err == nil {
		fm, _, ferr := mdfile.Split(data)
		if ferr == nil {
			var meta struct {
				SweepOrder    string `yaml:"sweep-order"`
				NeverImplicit string `yaml:"never-implicit"`
			}
			if yaml.Unmarshal(fm, &meta) == nil {
				listed = splitList(meta.SweepOrder)
				for _, n := range splitList(meta.NeverImplicit) {
					neverImplicit[n] = true
				}
			}
		}
	}
	present, err := s.Workstreams()
	if err != nil {
		return nil, nil, err
	}
	presentSet := map[string]bool{}
	for _, w := range present {
		presentSet[w] = true
	}
	seen := map[string]bool{}
	for _, w := range listed {
		if presentSet[w] && !seen[w] {
			order = append(order, w)
			seen[w] = true
		}
	}
	for _, w := range present {
		if !seen[w] {
			order = append(order, w)
			seen[w] = true
		}
	}
	return order, neverImplicit, nil
}

// ListedSweepOrder returns the sweep-order names exactly as written in
// workstreams.md frontmatter, without reconciling against on-disk directories.
// It is the raw list an integrity check compares against existing dirs.
func (s *Store) ListedSweepOrder() ([]string, error) {
	path := filepath.Join(s.dir(), "workstreams.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	fm, _, ferr := mdfile.Split(data)
	if ferr != nil {
		return nil, nil
	}
	var meta struct {
		SweepOrder string `yaml:"sweep-order"`
	}
	if yaml.Unmarshal(fm, &meta) != nil {
		return nil, nil
	}
	return splitList(meta.SweepOrder), nil
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// scanDir loads all *.md items in one directory, assigning each item its
// workstream context is left to the caller (dir name). Files that fail to parse
// are returned as errors keyed by path only via the collect callback.
func (s *Store) scanDir(dir string) ([]*Item, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var items []*Item
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, rerr
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		it, perr := parseItem(data, id, path)
		if perr != nil {
			continue // malformed files are surfaced by Validate, not here
		}
		items = append(items, it)
	}
	return items, nil
}

// LoadAll returns every parseable item across intake/ and all workstream dirs.
func (s *Store) LoadAll() ([]*Item, error) {
	var all []*Item
	intake, err := s.scanDir(s.intakeDir())
	if err != nil {
		return nil, err
	}
	all = append(all, intake...)
	wss, err := s.Workstreams()
	if err != nil {
		return nil, err
	}
	for _, ws := range wss {
		items, err := s.scanDir(s.wsDir(ws))
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all, nil
}

// takenIDs returns the set of ids currently used across the tree.
func (s *Store) takenIDs() (map[string]bool, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	taken := map[string]bool{}
	for _, it := range items {
		taken[it.ID] = true
	}
	return taken, nil
}

// Find loads a single item by id, searching intake/ then workstream dirs.
func (s *Store) Find(id string) (*Item, error) {
	candidates := []string{filepath.Join(s.intakeDir(), id+".md")}
	wss, err := s.Workstreams()
	if err != nil {
		return nil, err
	}
	for _, ws := range wss {
		candidates = append(candidates, filepath.Join(s.wsDir(ws), id+".md"))
	}
	for _, path := range candidates {
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			if os.IsNotExist(rerr) {
				continue
			}
			return nil, rerr
		}
		return parseItem(data, id, path)
	}
	return nil, fmt.Errorf("%w: no item with id %q", ErrNotFound, id)
}

// NewParams carries the submitter-provided fields for backlog new.
type NewParams struct {
	Title    string
	Value    string
	Source   string
	Hint     string
	Priority string
}

// New creates and persists an intake item, generating a unique id from the
// title. It validates the result before persisting.
func (s *Store) New(p NewParams) (*Item, error) {
	base := Slug(p.Title)
	if base == "" {
		return nil, invalid("title slugifies to empty; provide an alphanumeric title")
	}
	taken, err := s.takenIDs()
	if err != nil {
		return nil, err
	}
	id := UniqueID(base, taken)
	it := &Item{
		Title:    p.Title,
		Value:    p.Value,
		Source:   p.Source,
		Hint:     p.Hint,
		Priority: p.Priority,
		Status:   "idea",
		ID:       id,
		Path:     filepath.Join(s.intakeDir(), id+".md"),
	}
	if err := s.validateForPersist(it, true); err != nil {
		return nil, err
	}
	if err := s.persist(it); err != nil {
		return nil, err
	}
	return it, nil
}

// persist atomically writes the item to its Path.
func (s *Store) persist(it *Item) error {
	data, err := it.marshalFile()
	if err != nil {
		return err
	}
	return mdfile.WriteAtomic(it.Path, data)
}

// Save re-validates and atomically writes an existing item in place.
func (s *Store) Save(it *Item) error {
	if err := s.validateForPersist(it, false); err != nil {
		return err
	}
	return s.persist(it)
}

// Move relocates an item into workstream ws (a git-friendly rename preserving
// id/filename), re-validating first. The old file is removed only after the new
// one is written.
func (s *Store) Move(it *Item, ws string) error {
	ok, err := s.IsWorkstream(ws)
	if err != nil {
		return err
	}
	if !ok {
		return invalid("workstream target %q is not an existing workstream directory", ws)
	}
	oldPath := it.Path
	newPath := filepath.Join(s.wsDir(ws), it.ID+".md")
	it.Workstream = ws
	it.Path = newPath
	if err := s.validateForPersist(it, false); err != nil {
		it.Path = oldPath // roll back the in-memory change
		return err
	}
	if err := s.persist(it); err != nil {
		it.Path = oldPath
		return err
	}
	if oldPath != newPath {
		if err := os.Remove(oldPath); err != nil {
			return fmt.Errorf("remove old file %s after move: %w", oldPath, err)
		}
	}
	return nil
}

// Delete removes an item file (terminal close).
func (s *Store) Delete(it *Item) error {
	return os.Remove(it.Path)
}

// AppendChangelog files one line in backlog/CHANGELOG.md under the section that
// matches the disposition: "done" items go under "## Done"; everything else
// (discarded / removed) under "## Discarded". The line is inserted newest-first
// (immediately below the heading). If the target section is absent it is
// created, so the filing is correct on both scaffolded and minimally-seeded
// changelogs.
func (s *Store) AppendChangelog(id, disposition, reason string) error {
	line := fmt.Sprintf("- %s %s — %s: %s",
		time.Now().Format("2006-01-02"), id, disposition, reason)
	heading := "## Done"
	if disposition != "done" {
		heading = "## Discarded"
	}
	path := s.ChangelogPath()
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return mdfile.WriteAtomic(path, []byte(insertUnderHeading(string(existing), heading, line)))
}

// insertUnderHeading inserts line immediately below the first line whose trimmed
// text starts with heading (so "## Discarded" matches "## Discarded (triaged out,
// not done)"). If no such heading exists the section is appended to the end.
func insertUnderHeading(content, heading, line string) string {
	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), heading) {
			out := make([]string, 0, len(lines)+1)
			out = append(out, lines[:i+1]...)
			out = append(out, line)
			out = append(out, lines[i+1:]...)
			return ensureTrailingNewline(strings.Join(out, "\n"))
		}
	}
	body := strings.TrimRight(content, "\n")
	if body == "" {
		return heading + "\n" + line + "\n"
	}
	return body + "\n\n" + heading + "\n" + line + "\n"
}

// ensureTrailingNewline guarantees exactly a trailing newline.
func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
