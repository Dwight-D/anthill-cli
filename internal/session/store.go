// Package session owns the runtime-only sessions ledger under
// <root>/.anthill/sessions/ (BACKLOG_V2_SPEC.md §5): one JSON record per
// harness session (and per subagent inside it) naming the backlog item it is
// working, its role, transcript, and when it was last seen.
//
// The ledger is untracked: the first write drops a .gitignore ignoring *.json
// into the directory.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ErrNotFound indicates no ledger record exists for a session/agent key.
var ErrNotFound = errors.New("not found")

// ValidationError names a rejected input (an unsafe id, a bad enum value).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) *ValidationError {
	return &ValidationError{Msg: fmt.Sprintf(format, a...)}
}

// now is the clock used for every stamp; tests may replace it.
var now = func() time.Time { return time.Now().UTC() }

// Roles are the accepted values of a record's role (the reporter vocabulary).
var Roles = map[string]bool{"user": true, "supervisor": true, "worker": true}

// Verbs are the backlog verbs a link may name as the reason for the update.
var Verbs = map[string]bool{"claim": true, "deliver": true, "close": true, "block": true, "release": true}

// Record is one ledger entry. Its JSON keys are the §5 record shape; verb is
// the backlog verb that last refreshed the record, when one is known.
type Record struct {
	Session    string `json:"session"`
	Agent      string `json:"agent"`
	Handle     string `json:"handle"`
	Item       string `json:"item"`
	Role       string `json:"role"`
	Transcript string `json:"transcript"`
	Cwd        string `json:"cwd"`
	Started    string `json:"started"`
	LastSeen   string `json:"last-seen"`
	Verb       string `json:"verb,omitempty"`
}

// Key is the record's ledger key: <session>[.<agent>].
func (r *Record) Key() string { return Key(r.Session, r.Agent) }

// Key builds a ledger key from a session id and an optional agent id.
func Key(session, agent string) string {
	if agent == "" {
		return session
	}
	return session + "." + agent
}

// Seen returns the record's liveness time: last-seen, falling back to the
// transcript file's modification time when last-seen is absent or
// unparseable. ok is false when neither is available.
func (r *Record) Seen() (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, r.LastSeen); err == nil {
		return t, true
	}
	if r.Transcript != "" {
		if info, err := os.Stat(r.Transcript); err == nil {
			return info.ModTime().UTC(), true
		}
	}
	return time.Time{}, false
}

// Age is the time since the record was last seen, formatted by FormatAge, or
// "-" when no liveness time is available.
func (r *Record) Age() string {
	t, ok := r.Seen()
	if !ok {
		return "-"
	}
	return FormatAge(now().Sub(t))
}

// FormatAge renders a duration as a compact age: 42s, 7m, 3h, 2d.
func FormatAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// Store is the file-backed ledger under <root>/.anthill/sessions.
type Store struct {
	root string
}

// NewStore returns a store rooted at the directory containing .anthill.
func NewStore(root string) *Store { return &Store{root: root} }

// Dir is the ledger directory.
func (s *Store) Dir() string { return filepath.Join(s.root, ".anthill", "sessions") }

func (s *Store) path(key string) string { return filepath.Join(s.Dir(), key+".json") }

// idPattern bounds session and agent ids to filename-safe characters; the dot
// is excluded because it separates the two in a ledger key.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// CheckID rejects an id that cannot form a ledger filename.
func CheckID(label, id string) error {
	if !idPattern.MatchString(id) {
		return invalid("%s %q must be non-empty and use only letters, digits, '-' and '_'", label, id)
	}
	return nil
}

func checkKey(session, agent string) error {
	if err := CheckID("session", session); err != nil {
		return err
	}
	if agent != "" {
		return CheckID("agent", agent)
	}
	return nil
}

// Get loads one record by session and agent id.
func (s *Store) Get(session, agent string) (*Record, error) {
	if err := checkKey(session, agent); err != nil {
		return nil, err
	}
	path := s.path(Key(session, agent))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: no session record %q", ErrNotFound, Key(session, agent))
		}
		return nil, err
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &r, nil
}

// List loads every record, sorted by key. Unparseable files are skipped.
func (s *Store) List() ([]*Record, error) {
	entries, err := os.ReadDir(s.Dir())
	if err != nil {
		if os.IsNotExist(err) {
			return []*Record{}, nil
		}
		return nil, err
	}
	out := []*Record{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.Dir(), e.Name()))
		if err != nil {
			return nil, err
		}
		var r Record
		if json.Unmarshal(data, &r) != nil || r.Session == "" {
			continue
		}
		out = append(out, &r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out, nil
}

// LinkParams carries the fields of a link call. Empty optional fields leave
// an existing record's value unchanged.
type LinkParams struct {
	Session    string
	Agent      string
	Handle     string
	Item       string
	Role       string
	Transcript string
	Cwd        string
	Verb       string
}

// Link creates or updates the record keyed by (session, agent) and stamps
// last-seen (and started on create). created reports whether it was new. The
// caller checks that the item exists.
func (s *Store) Link(p LinkParams) (rec *Record, created bool, err error) {
	if err := checkKey(p.Session, p.Agent); err != nil {
		return nil, false, err
	}
	if p.Role != "" && !Roles[p.Role] {
		return nil, false, invalid("role %q must be one of supervisor|user|worker", p.Role)
	}
	if p.Verb != "" && !Verbs[p.Verb] {
		return nil, false, invalid("verb %q must be one of block|claim|close|deliver|release", p.Verb)
	}
	ts := stamp()
	rec, err = s.Get(p.Session, p.Agent)
	switch {
	case errors.Is(err, ErrNotFound):
		rec = &Record{Session: p.Session, Agent: p.Agent, Started: ts}
		created = true
	case err != nil:
		return nil, false, err
	}
	set := func(dst *string, v string) {
		if v != "" {
			*dst = v
		}
	}
	set(&rec.Handle, p.Handle)
	set(&rec.Item, p.Item)
	set(&rec.Role, p.Role)
	set(&rec.Transcript, p.Transcript)
	set(&rec.Cwd, p.Cwd)
	set(&rec.Verb, p.Verb)
	rec.LastSeen = ts
	if err := s.save(rec); err != nil {
		return nil, false, err
	}
	return rec, created, nil
}

// Touch bumps last-seen on an existing record.
func (s *Store) Touch(session, agent string) (*Record, error) {
	rec, err := s.Get(session, agent)
	if err != nil {
		return nil, err
	}
	rec.LastSeen = stamp()
	return rec, s.save(rec)
}

// TouchItem bumps last-seen (and records verb) on every record of session
// that names item — the session's own record and any of its subagents'. It
// returns the records touched; none is not an error.
func (s *Store) TouchItem(session, item, verb string) ([]*Record, error) {
	if item == "" || CheckID("session", session) != nil {
		return nil, nil
	}
	recs, err := s.List()
	if err != nil {
		return nil, err
	}
	var touched []*Record
	ts := stamp()
	for _, r := range recs {
		if r.Session != session || r.Item != item {
			continue
		}
		r.LastSeen = ts
		if verb != "" {
			r.Verb = verb
		}
		if err := s.save(r); err != nil {
			return touched, err
		}
		touched = append(touched, r)
	}
	return touched, nil
}

// ForItem returns the records naming item, most recently seen first.
func (s *Store) ForItem(item string) ([]*Record, error) {
	recs, err := s.List()
	if err != nil {
		return nil, err
	}
	var out []*Record
	for _, r := range recs {
		if r.Item == item {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ti, _ := out[i].Seen()
		tj, _ := out[j].Seen()
		return ti.After(tj)
	})
	return out, nil
}

// PruneResult reports a prune run.
type PruneResult struct {
	DryRun bool     `json:"dry_run"`
	Pruned []string `json:"pruned"`
	Kept   int      `json:"kept"`
}

// Prune deletes records not seen within olderThan (liveness per Seen; a record
// with no liveness time at all is stale).
func (s *Store) Prune(olderThan time.Duration, dryRun bool) (*PruneResult, error) {
	res := &PruneResult{DryRun: dryRun, Pruned: []string{}}
	recs, err := s.List()
	if err != nil {
		return nil, err
	}
	cutoff := now().Add(-olderThan)
	for _, r := range recs {
		if t, ok := r.Seen(); ok && t.After(cutoff) {
			res.Kept++
			continue
		}
		res.Pruned = append(res.Pruned, r.Key())
		if dryRun {
			continue
		}
		if err := os.Remove(s.path(r.Key())); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return res, nil
}

// RewriteItems points every record whose item is a key of mapping at the
// mapped value (a backlog migration or renumber). dryRun reports without
// writing. It returns one line per record: "<key>.json: item <old> → <new>".
func (s *Store) RewriteItems(mapping map[string]string, dryRun bool) ([]string, error) {
	if len(mapping) == 0 {
		return nil, nil
	}
	recs, err := s.List()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range recs {
		to, ok := mapping[r.Item]
		if !ok || to == r.Item {
			continue
		}
		out = append(out, fmt.Sprintf("sessions/%s.json: item %s → %s", r.Key(), r.Item, to))
		if dryRun {
			continue
		}
		r.Item = to
		if err := s.save(r); err != nil {
			return out, err
		}
	}
	return out, nil
}

func stamp() string { return now().Format(time.RFC3339) }

// save writes rec atomically, creating the directory and its .gitignore on
// first write.
func (s *Store) save(rec *Record) error {
	if err := os.MkdirAll(s.Dir(), 0o755); err != nil {
		return err
	}
	ignore := filepath.Join(s.Dir(), ".gitignore")
	if _, err := os.Stat(ignore); os.IsNotExist(err) {
		if err := os.WriteFile(ignore, []byte("*.json\n"), 0o644); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir(), ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), s.path(rec.Key())); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}
