package backlog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// MigrateOptions tunes `anthill backlog migrate`.
type MigrateOptions struct {
	DryRun          bool
	DefaultReporter string // reporter for items that lack one (default "user")
}

// MigrateChange is one migrated item and what changed.
type MigrateChange struct {
	ID      string   `json:"id"`
	Path    string   `json:"path"`
	Changes []string `json:"changes"`
}

// MigrateSkip is a file migration could not process; it is left untouched.
type MigrateSkip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// MigrateResult reports a migration run.
type MigrateResult struct {
	DryRun        bool            `json:"dry_run"`
	Changed       []MigrateChange `json:"changed"`
	Unchanged     int             `json:"unchanged"`
	Skipped       []MigrateSkip   `json:"skipped"`
	SchemaStamped bool            `json:"schema_stamped"`
}

// migrateKnownKeys are the frontmatter keys migration understands: the
// schema-2 keys plus the schema-1 value-verdict it folds into ## Triage.
var migrateKnownKeys = func() map[string]bool {
	m := map[string]bool{"value-verdict": true}
	for _, k := range FieldNames {
		m[k] = true
	}
	delete(m, "id")
	delete(m, "path")
	delete(m, "ready")
	delete(m, "active")
	return m
}()

// Migrate rewrites every item from schema 1 to schema 2 in place (BACKLOG_V2
// §9) and stamps `schema: 2` into workstreams.md. It is idempotent: an item
// already on schema 2 is left byte-identical. Files without parseable
// frontmatter or with keys outside both schemas are skipped and reported,
// never modified.
func (s *Store) Migrate(opt MigrateOptions) (*MigrateResult, error) {
	res := &MigrateResult{DryRun: opt.DryRun, Changed: []MigrateChange{}, Skipped: []MigrateSkip{}}
	reporter := opt.DefaultReporter
	if reporter == "" {
		reporter = "user"
	}
	if !reporters[reporter] {
		return nil, invalid("illegal default reporter %q", reporter)
	}
	_, neverAuto := s.changeTypeConfig()
	migratedAt := Stamp()

	type loc struct{ dir, ws string }
	locs := []loc{{s.intakeDir(), ""}}
	wss, err := s.Workstreams()
	if err != nil {
		return nil, err
	}
	for _, ws := range wss {
		locs = append(locs, loc{s.wsDir(ws), ws})
	}
	locs = append(locs, loc{s.doneDir(), DoneDir}, loc{s.discardedDir(), DiscardedDir})

	for _, l := range locs {
		entries, err := os.ReadDir(l.dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(l.dir, e.Name())
			id := strings.TrimSuffix(e.Name(), ".md")
			ch, skip, err := s.migrateFile(path, id, l.ws, reporter, migratedAt, neverAuto, opt.DryRun)
			if err != nil {
				return nil, err
			}
			switch {
			case skip != nil:
				res.Skipped = append(res.Skipped, *skip)
			case ch != nil:
				res.Changed = append(res.Changed, *ch)
			default:
				res.Unchanged++
			}
		}
	}
	sort.Slice(res.Changed, func(i, j int) bool { return res.Changed[i].ID < res.Changed[j].ID })

	if v, err := s.SchemaVersion(); err != nil {
		return nil, err
	} else if v != CurrentSchema {
		res.SchemaStamped = true
		if !opt.DryRun {
			if err := s.StampSchema(); err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

// migrateFile maps one item file. ws is the containing workstream dir name
// ("" for intake; "done"/"discarded" for the archives).
func (s *Store) migrateFile(path, id, ws, reporter, migratedAt string, neverAuto map[string]bool, dryRun bool) (*MigrateChange, *MigrateSkip, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	fm, _, err := mdfile.Split(data)
	if err != nil {
		return nil, &MigrateSkip{Path: path, Reason: err.Error()}, nil
	}
	var raw map[string]any
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		return nil, &MigrateSkip{Path: path, Reason: "parse frontmatter: " + err.Error()}, nil
	}
	var unknown []string
	for k := range raw {
		if !migrateKnownKeys[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, &MigrateSkip{Path: path, Reason: "unknown keys: " + strings.Join(unknown, ", ")}, nil
	}
	it, err := parseItem(data, id, path)
	if err != nil {
		return nil, &MigrateSkip{Path: path, Reason: err.Error()}, nil
	}
	verdict, _ := raw["value-verdict"].(string)

	var changes []string
	note := func(format string, a ...any) { changes = append(changes, fmt.Sprintf(format, a...)) }
	inWorkstream := ws != "" && ws != DoneDir && ws != DiscardedDir

	// Note qualifiers: expedited → fast-tracked, needs-spec → design.
	noteParts, expedited := dropQualifier(it.Note, "expedited")
	noteParts, needsSpec := dropQualifier(noteParts, "needs-spec")
	if noteParts != it.Note {
		it.Note = noteParts
	}

	// status (and the dispatch it implies).
	switch it.Status {
	case "idea":
		if inWorkstream {
			it.Status = StatusTriaged
		} else {
			it.Status = StatusNew
		}
		note("status idea → %s", it.Status)
	case "approved":
		it.Status = StatusTriaged
		it.Dispatch = DispatchApproved
		note("status approved → triaged, dispatch approved")
	case StatusInProgress, StatusBlocked, StatusParked:
		if it.Dispatch == "" {
			it.Dispatch = DispatchApproved
			note("dispatch approved (was %s)", it.Status)
		}
	}
	if expedited {
		it.Dispatch = DispatchFastTracked
		note("note expedited → dispatch fast-tracked")
	}

	// disposition.
	switch it.Disposition {
	case "AUTO":
		it.Disposition = "auto"
		note("disposition AUTO → auto")
	case "REVIEW":
		if RealVerify(it.Verify) && !neverAuto[it.ChangeType] {
			it.Disposition = "auto"
		} else {
			it.Disposition = "design"
		}
		note("disposition REVIEW → %s", it.Disposition)
	case "DISCARD":
		it.Disposition = "open"
		it.Dispatch = DispatchProposed
		note("disposition DISCARD → open, dispatch proposed")
	}
	if needsSpec {
		it.Disposition = "design"
		note("note needs-spec → disposition design")
	}
	if it.Dispatch == "" {
		it.Dispatch = DispatchProposed
		note("dispatch proposed")
	}
	if it.Priority == "" {
		it.Priority = "normal"
		note("priority normal")
	}
	if it.Reporter == "" {
		it.Reporter = reporter
		note("reporter %s", reporter)
	}
	if it.CreatedAt == "" {
		it.CreatedAt = gitFirstCommit(s.root, path)
		if it.CreatedAt == "" {
			it.CreatedAt = migratedAt
		}
		note("created-at %s", it.CreatedAt)
	}

	// Body: skeleton when empty, value-verdict rationale into ## Triage.
	if strings.TrimSpace(it.Body) == "" {
		it.Body = Skeleton(it.Value)
		note("body skeleton")
	}
	if verdict != "" {
		why := verdict
		for _, sep := range []string{" — ", " - ", " "} {
			if i := strings.Index(verdict, sep); i >= 0 {
				why = strings.TrimSpace(verdict[i+len(sep):])
				break
			}
		}
		if why != "" {
			it.appendSection("Triage", why)
		}
		note("value-verdict folded into ## Triage")
	}

	// done is an archive status: move the file into done/.
	target := ""
	if it.Status == StatusDone && ws != DoneDir {
		target = s.doneDir()
		if it.ClosedAt == "" {
			it.ClosedAt = migratedAt
		}
		note("moved to done/")
	}

	if len(changes) == 0 {
		return nil, nil, nil
	}
	it.appendLog("migrate schema 1 → 2", "anthill", "", "")
	ch := &MigrateChange{ID: id, Path: path, Changes: changes}
	if dryRun {
		return ch, nil, nil
	}
	if target != "" {
		if err := s.relocateUnchecked(it, target); err != nil {
			return nil, nil, err
		}
		ch.Path = it.Path
		return ch, nil, nil
	}
	if err := s.persist(it); err != nil {
		return nil, nil, err
	}
	return ch, nil, nil
}

// relocateUnchecked moves an item file into dir without the persist gate:
// migration writes whatever the source held, and validate reports the gaps.
func (s *Store) relocateUnchecked(it *Item, dir string) error {
	oldPath := it.Path
	it.Path = filepath.Join(dir, it.ID+".md")
	if err := s.persist(it); err != nil {
		it.Path = oldPath
		return err
	}
	if oldPath != it.Path {
		return os.Remove(oldPath)
	}
	return nil
}

// dropQualifier removes a "; "-separated qualifier token from a note,
// reporting whether it was present. A part equal to the token, or starting
// with the token followed by a space or colon, counts.
func dropQualifier(note, token string) (string, bool) {
	if note == "" {
		return note, false
	}
	parts := strings.Split(note, ";")
	var kept []string
	found := false
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == token || strings.HasPrefix(t, token+" ") || strings.HasPrefix(t, token+":") {
			found = true
			continue
		}
		if t != "" {
			kept = append(kept, t)
		}
	}
	if !found {
		return note, false
	}
	return strings.Join(kept, "; "), true
}

// gitFirstCommit returns the author date (RFC3339 UTC) of the commit that
// added path, or "" when git or the history is unavailable.
func gitFirstCommit(root, path string) string {
	cmd := exec.Command("git", "log", "--diff-filter=A", "--follow", "--format=%aI", "--", path)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Fields(strings.TrimSpace(string(out)))
	if len(lines) == 0 {
		return ""
	}
	t, err := time.Parse(time.RFC3339, lines[len(lines)-1])
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
