package backlog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// ErrNeedsMigration indicates the tree holds items written under an earlier
// schema (no `schema: 2` stamp in workstreams.md); `anthill backlog migrate`
// upgrades it.
var ErrNeedsMigration = errors.New("backlog is not on schema 2: run 'anthill backlog migrate' to upgrade it")

// validateForPersist is the per-write gate shared by every write. It rejects
// writes that would violate the schema at the point of write: missing
// always-required keys and illegal enum values. Stage completeness and
// cross-field coherence are certified by Validate.
func (s *Store) validateForPersist(it *Item) error {
	if strings.TrimSpace(it.Title) == "" {
		return invalid("title is required")
	}
	if strings.TrimSpace(it.Value) == "" {
		return invalid("value is required")
	}
	if it.Status == "" {
		return invalid("status is required")
	}
	for _, k := range []string{"status", "dispatch", "disposition", "priority", "reporter", "risk"} {
		if v := it.Field(k); v != "" && !EnumValues(k)[v] {
			return invalid("illegal %s %q", k, v)
		}
	}
	return nil
}

// Violation is one failed validation rule for one item (or file).
type Violation struct {
	ID      string `json:"id"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// String renders the violation as "<id>: <rule>: <message>".
func (v Violation) String() string { return v.ID + ": " + v.Rule + ": " + v.Message }

// ValidateResult is the outcome of a full-tree validation. Warnings are
// advisory (a change-type outside the project's declared vocabulary): they are
// reported but never flip OK unless strict promotes them.
type ValidateResult struct {
	OK         bool        `json:"ok"`
	Checked    int         `json:"checked"`
	Violations []Violation `json:"violations"`
	Warnings   []Violation `json:"warnings"`
}

func (r *ValidateResult) add(id, rule, format string, a ...any) {
	r.Violations = append(r.Violations, Violation{ID: id, Rule: rule, Message: fmt.Sprintf(format, a...)})
}

// dirKind classifies an item directory.
type dirKind int

const (
	kindIntake dirKind = iota
	kindWorkstream
	kindDone
	kindDiscarded
)

// liveWorkstreamStatuses are the statuses an item in a workstream dir may hold.
var liveWorkstreamStatuses = map[string]bool{
	StatusTriaged: true, StatusInProgress: true, StatusValidation: true,
	StatusBlocked: true, StatusParked: true,
}

// Validate certifies the backlog tree against the schema-2 rules: frontmatter,
// unknown-key, location, enum, required (by stage), disposition-coherence,
// hint-stripped, escalation-ref, stray-file, id-unique. strict promotes
// warnings to violations. A tree that needs migration returns
// ErrNeedsMigration.
func (s *Store) Validate(strict bool) (*ValidateResult, error) {
	need, err := s.NeedsMigration()
	if err != nil {
		return nil, err
	}
	if need {
		return nil, ErrNeedsMigration
	}
	res := &ValidateResult{OK: true, Violations: []Violation{}, Warnings: []Violation{}}
	declared, neverAuto := s.changeTypeConfig()
	idDirs := map[string][]string{}

	check := func(dir, label string, kind dirKind) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".md") {
				// Git housekeeping files keep an empty dir tracked; they are not
				// items. Editor temp files are skipped too.
				if name != ".gitkeep" && name != ".gitignore" && !strings.HasPrefix(name, ".") {
					res.add(name, "stray-file", "non-item file in %s/", label)
				}
				continue
			}
			id := strings.TrimSuffix(name, ".md")
			res.Checked++
			idDirs[id] = append(idDirs[id], label)
			data, rerr := os.ReadFile(filepath.Join(dir, name))
			if rerr != nil {
				return rerr
			}
			s.validateFile(res, data, id, label, kind, declared, neverAuto)
		}
		return nil
	}

	if err := check(s.intakeDir(), IntakeDir, kindIntake); err != nil {
		return nil, err
	}
	wss, err := s.Workstreams()
	if err != nil {
		return nil, err
	}
	for _, ws := range wss {
		if err := check(s.wsDir(ws), ws, kindWorkstream); err != nil {
			return nil, err
		}
	}
	if err := check(s.doneDir(), DoneDir, kindDone); err != nil {
		return nil, err
	}
	if err := check(s.discardedDir(), DiscardedDir, kindDiscarded); err != nil {
		return nil, err
	}

	for id, dirs := range idDirs {
		if len(dirs) > 1 {
			res.add(id, "id-unique", "id appears in multiple dirs: %s", strings.Join(dirs, ", "))
		}
	}
	if strict {
		res.Violations = append(res.Violations, res.Warnings...)
	}
	res.OK = len(res.Violations) == 0
	sortViolations(res.Violations)
	sortViolations(res.Warnings)
	return res, nil
}

func sortViolations(vs []Violation) {
	sort.SliceStable(vs, func(i, j int) bool {
		if vs[i].ID != vs[j].ID {
			return vs[i].ID < vs[j].ID
		}
		if vs[i].Rule != vs[j].Rule {
			return vs[i].Rule < vs[j].Rule
		}
		return vs[i].Message < vs[j].Message
	})
}

// validateFile checks one item file's bytes. label is the containing dir name.
func (s *Store) validateFile(res *ValidateResult, data []byte, id, label string, kind dirKind, declared, neverAuto map[string]bool) {
	fm, body, err := mdfile.Split(data)
	if err != nil {
		res.add(id, "frontmatter", "%s", err.Error())
		return
	}
	dec := yaml.NewDecoder(bytes.NewReader(fm))
	dec.KnownFields(true)
	var strictItem Item
	if err := dec.Decode(&strictItem); err != nil && !errors.Is(err, io.EOF) {
		res.add(id, "unknown-key", "%s", err.Error())
	}
	var it Item
	if err := yaml.Unmarshal(fm, &it); err != nil {
		res.add(id, "frontmatter", "%s", err.Error())
		return
	}
	it.ID = id
	it.Body = body

	// location: the directory and the item's workstream/status agree.
	switch kind {
	case kindIntake:
		if it.Workstream != "" {
			res.add(id, "location", "item in intake/ carries workstream %q", it.Workstream)
		}
		if it.Status != StatusNew && it.Status != StatusParked && statuses[it.Status] {
			res.add(id, "location", "item in intake/ has status %s (want new or parked)", it.Status)
		}
	case kindWorkstream:
		if it.Workstream != label {
			res.add(id, "location", "item in %s/ has workstream %q", label, it.Workstream)
		}
		if !liveWorkstreamStatuses[it.Status] && statuses[it.Status] {
			res.add(id, "location", "item in %s/ has status %s (want triaged, in-progress, validation, blocked or parked)", label, it.Status)
		}
	case kindDone:
		if it.Status != StatusDone && statuses[it.Status] {
			res.add(id, "location", "item in done/ has status %s", it.Status)
		}
	case kindDiscarded:
		if it.Status != StatusDiscarded && statuses[it.Status] {
			res.add(id, "location", "item in discarded/ has status %s", it.Status)
		}
	}

	// enum legality.
	if it.Status != "" && !statuses[it.Status] {
		res.add(id, "enum", "illegal status %q", it.Status)
	}
	for _, k := range []string{"dispatch", "disposition", "priority", "reporter", "risk"} {
		if v := it.Field(k); v != "" && !EnumValues(k)[v] {
			res.add(id, "enum", "illegal %s %q", k, v)
		}
	}
	if len(declared) > 0 && it.ChangeType != "" && !declared[it.ChangeType] {
		res.Warnings = append(res.Warnings, Violation{ID: id, Rule: "change-type-vocab",
			Message: fmt.Sprintf("change-type %q is outside the declared vocabulary", it.ChangeType)})
	}

	// required, by stage.
	for _, k := range requiredMissing(&it, kind) {
		res.add(id, "required", "missing %s for status %s", k, orDash(it.Status))
	}

	// disposition-coherence.
	if it.Disposition == "auto" {
		if !RealVerify(it.Verify) {
			res.add(id, "disposition-coherence", "disposition auto requires a real verify (not empty or none)")
		}
		if neverAuto[it.ChangeType] {
			res.add(id, "disposition-coherence", "disposition auto is illegal for never-auto change-type %q", it.ChangeType)
		}
	}

	// hint-stripped: a hint survives only in intake.
	if kind != kindIntake && it.Hint != "" {
		res.add(id, "hint-stripped", "item outside intake/ still carries a hint")
	}

	// escalation-ref: an "escalated: <file>" note points at an existing record.
	if f, ok := escalatedFile(it.Note); ok {
		p := filepath.Join(s.root, ".anthill", "escalations", f)
		if _, err := os.Stat(p); err != nil {
			res.add(id, "escalation-ref", "note references missing escalation record %q", f)
		}
	}
}

// requiredMissing lists the keys the item's stage requires but lacks.
func requiredMissing(it *Item, kind dirKind) []string {
	var missing []string
	need := func(keys ...string) {
		for _, k := range keys {
			if strings.TrimSpace(it.Field(k)) == "" {
				missing = append(missing, k)
			}
		}
	}
	need("title", "value", "status", "dispatch", "reporter", "created-at")
	switch it.Status {
	case StatusTriaged, StatusInProgress, StatusValidation, StatusBlocked:
		need(TriagedRequired...)
	case StatusParked:
		if kind == kindWorkstream {
			need(TriagedRequired...)
		}
	case StatusDone, StatusDiscarded:
		need("closed-at")
	}
	switch it.Status {
	case StatusInProgress, StatusBlocked:
		need("claimed-at")
	case StatusValidation:
		need("claimed-at", "delivered-at")
	}
	return missing
}

// escalatedFile extracts the "<file>" from a note of the form "escalated: <file>".
func escalatedFile(note string) (string, bool) {
	const prefix = "escalated:"
	i := strings.Index(note, prefix)
	if i < 0 {
		return "", false
	}
	rest := strings.TrimSpace(note[i+len(prefix):])
	if rest == "" {
		return "", false
	}
	if j := strings.IndexAny(rest, " \t"); j >= 0 {
		rest = rest[:j]
	}
	if !strings.HasSuffix(rest, ".md") {
		rest += ".md"
	}
	return rest, true
}
