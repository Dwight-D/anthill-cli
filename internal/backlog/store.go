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
func (s *Store) intakeDir() string      { return filepath.Join(s.dir(), IntakeDir) }
func (s *Store) wsDir(ws string) string { return filepath.Join(s.dir(), ws) }
func (s *Store) doneDir() string        { return filepath.Join(s.dir(), DoneDir) }
func (s *Store) discardedDir() string   { return filepath.Join(s.dir(), DiscardedDir) }

// Root returns the directory that contains .anthill.
func (s *Store) Root() string { return s.root }

// reservedDir reports whether name is a backlog subdirectory that is never a
// workstream (intake and the two archives).
func reservedDir(name string) bool {
	return name == IntakeDir || name == DoneDir || name == DiscardedDir
}

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
		if e.IsDir() && !reservedDir(e.Name()) && !strings.HasPrefix(e.Name(), ".") {
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
	return info.IsDir() && !reservedDir(ws), nil
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

// changeTypeConfig reads the project's change-type vocabulary from
// workstreams.md frontmatter. `declared` is the accepted menu (`change-types`);
// an empty map means the project has declared none, so any change-type is
// accepted free-form (no vocabulary warning). `neverAuto` is the subset
// (`never-auto-change-types`) that can never carry an AUTO disposition. The read
// is tolerant: a missing/unparseable file yields empty maps, never an error, so
// validation degrades to free-form rather than failing.
func (s *Store) changeTypeConfig() (declared, neverAuto map[string]bool) {
	declared, neverAuto = map[string]bool{}, map[string]bool{}
	data, err := os.ReadFile(filepath.Join(s.dir(), "workstreams.md"))
	if err != nil {
		return declared, neverAuto
	}
	fm, _, ferr := mdfile.Split(data)
	if ferr != nil {
		return declared, neverAuto
	}
	var meta struct {
		ChangeTypes string `yaml:"change-types"`
		NeverAuto   string `yaml:"never-auto-change-types"`
	}
	if yaml.Unmarshal(fm, &meta) != nil {
		return declared, neverAuto
	}
	for _, c := range splitList(meta.ChangeTypes) {
		declared[c] = true
	}
	for _, c := range splitList(meta.NeverAuto) {
		neverAuto[c] = true
	}
	return declared, neverAuto
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

// LoadAll returns every parseable live item across intake/ and all workstream
// dirs (the archives done/ and discarded/ are skipped).
func (s *Store) LoadAll() ([]*Item, error) { return s.Load(false) }

// Load returns every parseable item; includeArchived also walks done/ and
// discarded/.
func (s *Store) Load(includeArchived bool) ([]*Item, error) {
	dirs, err := s.itemDirs(includeArchived)
	if err != nil {
		return nil, err
	}
	var all []*Item
	for _, d := range dirs {
		items, err := s.scanDir(d)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all, nil
}

// itemDirs lists the directories that hold items: intake, every workstream,
// and (optionally) the archives.
func (s *Store) itemDirs(includeArchived bool) ([]string, error) {
	dirs := []string{s.intakeDir()}
	wss, err := s.Workstreams()
	if err != nil {
		return nil, err
	}
	for _, ws := range wss {
		dirs = append(dirs, s.wsDir(ws))
	}
	if includeArchived {
		dirs = append(dirs, s.doneDir(), s.discardedDir())
	}
	return dirs, nil
}

// takenIDs returns the set of ids currently used across the tree.
func (s *Store) takenIDs() (map[string]bool, error) {
	items, err := s.Load(true)
	if err != nil {
		return nil, err
	}
	taken := map[string]bool{}
	for _, it := range items {
		taken[it.ID] = true
	}
	return taken, nil
}

// Find loads a single item by id, searching intake/, the workstream dirs, then
// the archives.
func (s *Store) Find(id string) (*Item, error) {
	dirs, err := s.itemDirs(true)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		path := filepath.Join(d, id+".md")
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
	Title       string
	Value       string
	Summary     string
	Description string
	Source      string
	Hint        string
	Priority    string
	Reporter    string
}

// Skeleton returns the section skeleton of a new item body, with description
// under "## Description".
func Skeleton(description string) string {
	var b strings.Builder
	for i, h := range SectionOrder {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("## " + h + "\n")
		if h == "Description" && strings.TrimSpace(description) != "" {
			b.WriteString(strings.TrimSpace(description) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// New creates and persists an intake item (status new, dispatch proposed),
// generating a unique id from the title and stamping created-at. It validates
// the result before persisting. On a tree with no schema stamp and no items
// yet, it stamps `schema: 2` first (there is nothing to migrate).
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
	priority := p.Priority
	if priority == "" {
		priority = "normal"
	}
	it := &Item{
		Title:     p.Title,
		Summary:   p.Summary,
		Value:     p.Value,
		Status:    StatusNew,
		Dispatch:  DispatchProposed,
		Priority:  priority,
		Reporter:  ResolveReporter(p.Reporter),
		Source:    p.Source,
		Hint:      p.Hint,
		CreatedAt: Stamp(),
		ID:        id,
		Path:      filepath.Join(s.intakeDir(), id+".md"),
		Body:      Skeleton(p.Description),
	}
	it.appendLog("new", ResolveActor(""), "", "")
	if err := s.validateForPersist(it); err != nil {
		return nil, err
	}
	if len(taken) == 0 {
		if v, _ := s.SchemaVersion(); v == 0 {
			if err := s.StampSchema(); err != nil {
				return nil, err
			}
		}
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
	if err := s.validateForPersist(it); err != nil {
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
	oldWs := it.Workstream
	it.Workstream = ws
	if err := s.relocate(it, s.wsDir(ws)); err != nil {
		it.Workstream = oldWs
		return err
	}
	return nil
}

// relocate writes the item into dir (validated, atomic) and removes the old
// file. On failure the in-memory Path is rolled back.
func (s *Store) relocate(it *Item, dir string) error {
	oldPath := it.Path
	newPath := filepath.Join(dir, it.ID+".md")
	it.Path = newPath
	if err := s.validateForPersist(it); err != nil {
		it.Path = oldPath
		return err
	}
	if err := s.persist(it); err != nil {
		it.Path = oldPath
		return err
	}
	if oldPath != "" && oldPath != newPath {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
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
	if !strings.HasPrefix(disposition, "done") {
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
