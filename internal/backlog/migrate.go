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

	"github.com/Dwight-D/anthill-cli/internal/escalation"
	"github.com/Dwight-D/anthill-cli/internal/mdfile"
	"github.com/Dwight-D/anthill-cli/internal/session"
)

// MigrateOptions tunes `anthill backlog migrate`.
type MigrateOptions struct {
	DryRun          bool
	DefaultReporter string // reporter for items that lack one (default "user")
	KeyPrefix       string // key prefix to stamp when workstreams.md has none
}

// MigrateChange is one migrated item and what changed. FormerID is set by the
// schema 2 → 3 pass: the pre-key slug the item was filed under.
type MigrateChange struct {
	ID       string   `json:"id"`
	Path     string   `json:"path"`
	FormerID string   `json:"former-id,omitempty"`
	Changes  []string `json:"changes"`
}

// MigrateSkip is a file migration could not process; it is left untouched.
type MigrateSkip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// MigrateResult reports a migration run. References lists the sessions-ledger
// and escalation records whose item was rewritten to a key.
type MigrateResult struct {
	DryRun        bool            `json:"dry_run"`
	Changed       []MigrateChange `json:"changed"`
	Unchanged     int             `json:"unchanged"`
	Skipped       []MigrateSkip   `json:"skipped"`
	SchemaStamped bool            `json:"schema_stamped"`
	KeyPrefix     string          `json:"key_prefix,omitempty"`
	PrefixStamped bool            `json:"key_prefix_stamped"`
	Sequence      int             `json:"sequence,omitempty"`
	References    []string        `json:"references"`
}

// migrateKnownKeys are the frontmatter keys migration understands: the
// current schema keys plus the schema-1 value-verdict it folds into ## Triage.
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

// Migrate upgrades the tree to the current schema and is idempotent at it.
// A tree at schema 1 (or unstamped) first gets the 1 → 2 field mapping
// (BACKLOG_V2 §9); then every item still filed under its slug is keyed
// (ITEM_KEYS_SPEC.md §3.5): numbered in created-at order, renamed to
// `NNNN-<slug>.md`, stamped with `former-id`, and the sessions ledger and
// escalation records that name the old id are rewritten. `sequence`,
// `schema: 3` and `key-prefix` are stamped into place. Files without
// parseable frontmatter or with keys outside the schemas are skipped and
// reported, never modified.
func (s *Store) Migrate(opt MigrateOptions) (*MigrateResult, error) {
	res := &MigrateResult{DryRun: opt.DryRun, Changed: []MigrateChange{}, Skipped: []MigrateSkip{}, References: []string{}}
	reporter := opt.DefaultReporter
	if reporter == "" {
		reporter = "user"
	}
	if !reporters[reporter] {
		return nil, invalid("illegal default reporter %q", reporter)
	}
	v, err := s.SchemaVersion()
	if err != nil {
		return nil, err
	}
	fmPrefix, err := s.KeyPrefix()
	if err != nil {
		return nil, err
	}
	prefix := fmPrefix
	if prefix == "" && opt.KeyPrefix != "" {
		if !ValidPrefix(opt.KeyPrefix) {
			return nil, invalid("key prefix %q must be 2 to 5 uppercase letters", opt.KeyPrefix)
		}
		prefix = opt.KeyPrefix
	}
	nItems, err := s.countItemFiles()
	if err != nil {
		return nil, err
	}
	if prefix == "" && nItems > 0 {
		return nil, ErrKeyPrefixMissing
	}
	res.KeyPrefix = prefix

	if v <= 1 && nItems > 0 {
		if err := s.migrateFields(res, reporter, opt.DryRun); err != nil {
			return nil, err
		}
	}
	if nItems > 0 {
		if err := s.migrateKeys(res, prefix, opt.DryRun); err != nil {
			return nil, err
		}
	}
	if err := s.countUnchanged(res); err != nil {
		return nil, err
	}

	if v != CurrentSchema {
		res.SchemaStamped = true
		if !opt.DryRun {
			if err := s.StampSchema(); err != nil {
				return nil, err
			}
		}
	}
	if fmPrefix == "" && prefix != "" {
		res.PrefixStamped = true
		if !opt.DryRun {
			if err := s.StampKeyPrefix(prefix); err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

// itemLoc is one item directory and its workstream label ("" for intake,
// "done"/"discarded" for the archives).
type itemLoc struct{ dir, ws string }

func (s *Store) itemLocs() ([]itemLoc, error) {
	locs := []itemLoc{{s.intakeDir(), ""}}
	wss, err := s.Workstreams()
	if err != nil {
		return nil, err
	}
	for _, ws := range wss {
		locs = append(locs, itemLoc{s.wsDir(ws), ws})
	}
	return append(locs, itemLoc{s.doneDir(), DoneDir}, itemLoc{s.discardedDir(), DiscardedDir}), nil
}

// migrateFields is the schema 1 → 2 pass: it maps every item's fields in
// place, keeping its filename.
func (s *Store) migrateFields(res *MigrateResult, reporter string, dryRun bool) error {
	_, neverAuto := s.changeTypeConfig()
	migratedAt := Stamp()
	locs, err := s.itemLocs()
	if err != nil {
		return err
	}
	var changed []MigrateChange
	for _, l := range locs {
		entries, err := os.ReadDir(l.dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(l.dir, e.Name())
			id := strings.TrimSuffix(e.Name(), ".md")
			ch, skip, err := s.migrateFile(path, id, l.ws, reporter, migratedAt, neverAuto, dryRun)
			if err != nil {
				return err
			}
			switch {
			case skip != nil:
				res.Skipped = append(res.Skipped, *skip)
			case ch != nil:
				changed = append(changed, *ch)
			}
		}
	}
	sort.Slice(changed, func(i, j int) bool { return changed[i].ID < changed[j].ID })
	res.Changed = append(res.Changed, changed...)
	return nil
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
		if err := s.relocateUnchecked(it, filepath.Join(target, filepath.Base(path))); err != nil {
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

// migrateKeys is the schema 2 → 3 pass (ITEM_KEYS_SPEC.md §3.5): every item
// file not yet in `NNNN-<slug>.md` form is numbered in created-at order
// (then old id), renamed in place, stamped with former-id and a Log line,
// and the ledger/escalation references to its old id are rewritten. Items
// already in key form are left alone, so the pass is idempotent.
func (s *Store) migrateKeys(res *MigrateResult, prefix string, dryRun bool) error {
	locs, err := s.itemLocs()
	if err != nil {
		return err
	}
	skipped := map[string]bool{}
	for _, sk := range res.Skipped {
		skipped[sk.Path] = true
	}
	type cand struct {
		it   *Item
		stem string
	}
	var cands []cand
	for _, l := range locs {
		entries, err := os.ReadDir(l.dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || IsItemFilename(e.Name()) {
				continue
			}
			path := filepath.Join(l.dir, e.Name())
			if skipped[path] {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			stem := strings.TrimSuffix(e.Name(), ".md")
			it, perr := parseItem(data, stem, path)
			if perr != nil {
				res.Skipped = append(res.Skipped, MigrateSkip{Path: path, Reason: perr.Error()})
				continue
			}
			cands = append(cands, cand{it: it, stem: stem})
		}
	}
	if len(cands) == 0 {
		return nil
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].it.CreatedAt != cands[j].it.CreatedAt {
			return cands[i].it.CreatedAt < cands[j].it.CreatedAt
		}
		return cands[i].stem < cands[j].stem
	})
	high, err := s.maxNumber()
	if err != nil {
		return err
	}
	mapping := map[string]string{}
	for _, c := range cands {
		high++
		key := Key{Prefix: prefix, Number: high}
		oldPath := c.it.Path
		newPath := filepath.Join(filepath.Dir(oldPath), FormatFilename(high, Slug(c.stem)))
		c.it.FormerID = c.stem
		c.it.appendLog("migrate schema 2 → 3", "anthill", "", c.stem+" → "+key.String())
		mapping[c.stem] = key.String()
		ch := MigrateChange{ID: key.String(), Path: newPath, FormerID: c.stem, Changes: []string{
			"renamed " + filepath.Base(oldPath) + " → " + filepath.Base(newPath),
			"former-id " + c.stem,
		}}
		res.Changed = append(res.Changed, ch)
		if dryRun {
			continue
		}
		if err := s.relocateUnchecked(c.it, newPath); err != nil {
			return err
		}
	}
	res.Sequence = high
	if !dryRun {
		if err := s.bumpSequence(high); err != nil {
			return err
		}
	}
	refs, err := s.rewriteReferences(mapping, dryRun)
	if err != nil {
		return err
	}
	res.References = append(res.References, refs...)
	return nil
}

// rewriteReferences points every sessions-ledger record and escalation record
// whose item is a key of mapping at the mapped value. It returns one line per
// rewrite ("<file>: item <old> → <new>").
func (s *Store) rewriteReferences(mapping map[string]string, dryRun bool) ([]string, error) {
	var out []string
	ledger, err := session.NewStore(s.root).RewriteItems(mapping, dryRun)
	if err != nil {
		return nil, err
	}
	out = append(out, ledger...)
	esc, err := escalation.NewStore(s.root).RewriteItemRefs(mapping, dryRun)
	if err != nil {
		return nil, err
	}
	return append(out, esc...), nil
}

// countUnchanged sets res.Unchanged: every item file that no change entry
// names (by its pre-change path) and that was not skipped.
func (s *Store) countUnchanged(res *MigrateResult) error {
	total, err := s.countItemFiles()
	if err != nil {
		return err
	}
	touched := map[string]bool{}
	for _, c := range res.Changed {
		if c.FormerID != "" {
			touched[filepath.Join(filepath.Dir(c.Path), c.FormerID+".md")] = true
		} else {
			touched[c.Path] = true
		}
	}
	res.Unchanged = total - len(touched) - len(res.Skipped)
	if res.Unchanged < 0 {
		res.Unchanged = 0
	}
	return nil
}

// relocateUnchecked moves an item file to newPath without the persist gate:
// migration writes whatever the source held, and validate reports the gaps.
func (s *Store) relocateUnchecked(it *Item, newPath string) error {
	oldPath := it.Path
	it.Path = newPath
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
