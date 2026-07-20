package escalation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// ErrNotFound indicates a referenced record does not exist.
var ErrNotFound = errors.New("not found")

// ValidationError names a schema invariant a rejected write would violate.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) *ValidationError {
	return &ValidationError{Msg: fmt.Sprintf(format, a...)}
}

// PreconditionError indicates the operation is illegal in the record's state.
type PreconditionError struct{ Msg string }

func (e *PreconditionError) Error() string { return e.Msg }

func precond(format string, a ...any) *PreconditionError {
	return &PreconditionError{Msg: fmt.Sprintf(format, a...)}
}

// Store is the file-backed escalation directory.
type Store struct{ root string }

// NewStore returns a store rooted at the directory containing .anthill.
func NewStore(root string) *Store { return &Store{root: root} }

func (s *Store) dir() string     { return filepath.Join(s.root, ".anthill", "escalations") }
func (s *Store) logPath() string { return filepath.Join(s.dir(), "LOG.md") }

// LogPath is the escalations log file.
func (s *Store) LogPath() string { return s.logPath() }

func slug(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 50 {
		out = out[:50]
		if i := strings.LastIndex(out, "-"); i > 0 {
			out = out[:i]
		}
		out = strings.Trim(out, "-")
	}
	return out
}

// RaiseParams carries the fields for creating a record.
type RaiseParams struct {
	To       string
	From     string
	Item     string
	Question string
	Context  string
	Options  string
	BodyFile string
}

// Raise creates and persists a new escalation record.
func (s *Store) Raise(p RaiseParams) (*Record, error) {
	if !toTiers[p.To] {
		return nil, invalid("illegal --to %q (must be dispatcher|supervisor|user)", p.To)
	}
	if strings.TrimSpace(p.From) == "" {
		return nil, invalid("--from is required")
	}
	if strings.TrimSpace(p.Question) == "" {
		return nil, invalid("--question is required")
	}
	date := time.Now().Format("2006-01-02")
	base := date + "-" + slug(p.Question)
	taken, err := s.takenIDs()
	if err != nil {
		return nil, err
	}
	id := uniqueID(base, taken)
	r := &Record{
		To:       p.To,
		From:     p.From,
		Item:     p.Item,
		Status:   "open",
		Opened:   date,
		ID:       id,
		Path:     filepath.Join(s.dir(), id+".md"),
		Sections: map[string]string{},
	}
	if strings.TrimSpace(p.BodyFile) != "" {
		data, err := os.ReadFile(p.BodyFile)
		if err != nil {
			return nil, fmt.Errorf("read --body-file: %w", err)
		}
		r.Sections = parseSections(string(data))
		if r.Section("Question") == "" {
			r.Sections["Question"] = strings.TrimSpace(p.Question)
		}
	} else {
		r.Sections["Question"] = strings.TrimSpace(p.Question)
		r.Sections["Context & attempted remedies"] = strings.TrimSpace(p.Context)
		r.Sections["Options & recommendation"] = strings.TrimSpace(p.Options)
	}
	if err := s.persist(r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) persist(r *Record) error {
	data, err := r.marshalFile()
	if err != nil {
		return err
	}
	return mdfile.WriteAtomic(r.Path, data)
}

func (s *Store) takenIDs() (map[string]bool, error) {
	recs, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	taken := map[string]bool{}
	for _, r := range recs {
		taken[r.ID] = true
	}
	return taken, nil
}

func uniqueID(base string, taken map[string]bool) string {
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		cand := fmt.Sprintf("%s-%d", base, n)
		if !taken[cand] {
			return cand
		}
	}
}

// LoadAll returns every parseable record, sorted by id.
func (s *Store) LoadAll() ([]*Record, error) {
	entries, err := os.ReadDir(s.dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var recs []*Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if e.Name() == "README.md" || e.Name() == "LOG.md" {
			continue
		}
		path := filepath.Join(s.dir(), e.Name())
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, rerr
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		r, perr := parseRecord(data, id, path)
		if perr != nil {
			continue
		}
		recs = append(recs, r)
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].ID < recs[j].ID })
	return recs, nil
}

// Find loads one record by id (its <date>-<slug> filename stem).
func (s *Store) Find(id string) (*Record, error) {
	path := filepath.Join(s.dir(), id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: no escalation with id %q", ErrNotFound, id)
		}
		return nil, err
	}
	return parseRecord(data, id, path)
}

// Answer records a decision on an open record.
func (s *Store) Answer(id, decision string) (*Record, error) {
	r, err := s.Find(id)
	if err != nil {
		return nil, err
	}
	if r.Status != "open" {
		return nil, precond("record %q is %s, not open", id, r.Status)
	}
	r.Sections["Decision"] = strings.TrimSpace(decision)
	r.Status = "answered"
	if err := s.persist(r); err != nil {
		return nil, err
	}
	return r, nil
}

// Apply closes out an answered record: appends Applied, logs it, deletes it.
func (s *Store) Apply(id, note string) (*Record, error) {
	r, err := s.Find(id)
	if err != nil {
		return nil, err
	}
	if r.Status != "answered" {
		return nil, precond("record %q is %s, not answered", id, r.Status)
	}
	if strings.TrimSpace(note) != "" {
		r.Sections["Applied"] = strings.TrimSpace(note)
	}
	r.Status = "applied"
	outcome := r.Section("Applied")
	if outcome == "" {
		outcome = firstLine(r.Section("Decision"))
	}
	if err := s.appendLog(r, outcome); err != nil {
		return nil, err
	}
	if err := os.Remove(r.Path); err != nil {
		return nil, fmt.Errorf("delete record after apply: %w", err)
	}
	return r, nil
}

func (s *Store) appendLog(r *Record, outcome string) error {
	line := fmt.Sprintf("- %s %s — %s/applied: %s\n",
		time.Now().Format("2006-01-02"), r.ID, r.To, firstLine(outcome))
	path := s.logPath()
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(existing)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += line
	return mdfile.WriteAtomic(path, []byte(content))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// ValidateWellFormed reports records whose frontmatter is malformed or whose
// status is not a legal enum value. Returns a slice of "<id>: <problem>".
func (s *Store) ValidateWellFormed() ([]string, error) {
	entries, err := os.ReadDir(s.dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var problems []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || name == "README.md" || name == "LOG.md" {
			continue
		}
		id := strings.TrimSuffix(name, ".md")
		data, rerr := os.ReadFile(filepath.Join(s.dir(), name))
		if rerr != nil {
			return nil, rerr
		}
		r, perr := parseRecord(data, id, "")
		if perr != nil {
			problems = append(problems, id+": "+perr.Error())
			continue
		}
		if !toTiers[r.To] {
			problems = append(problems, id+": illegal to value "+r.To)
		}
		if !statuses[r.Status] {
			problems = append(problems, id+": illegal status "+r.Status)
		}
		if r.Section("Question") == "" {
			problems = append(problems, id+": empty Question section")
		}
	}
	sort.Strings(problems)
	return problems, nil
}

// UnappliedAnswered returns ids of records that are answered but not yet applied.
func (s *Store) UnappliedAnswered() ([]string, error) {
	recs, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range recs {
		if r.Status == "answered" {
			out = append(out, r.ID)
		}
	}
	return out, nil
}
