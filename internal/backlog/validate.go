package backlog

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// validateForPersist is the per-write gate shared by new/set/move/claim/close.
// It rejects writes that would violate the schema at the point of write:
// missing always-required keys, illegal enum values, malformed value-verdict.
// It does not require the full triaged field set (incremental triage writes are
// legal); completeness is certified by the Validate command.
func (s *Store) validateForPersist(it *Item, isNew bool) error {
	if strings.TrimSpace(it.Title) == "" {
		return invalid("title is required")
	}
	if strings.TrimSpace(it.Value) == "" {
		return invalid("value is required")
	}
	if it.Status == "" {
		return invalid("status is required")
	}
	if !statuses[it.Status] {
		return invalid("illegal status %q", it.Status)
	}
	if it.ChangeType != "" && !changeTypes[it.ChangeType] {
		return invalid("illegal change-type %q", it.ChangeType)
	}
	if it.Risk != "" && !risks[it.Risk] {
		return invalid("illegal risk %q", it.Risk)
	}
	if it.Disposition != "" && !dispositions[it.Disposition] {
		return invalid("illegal disposition %q", it.Disposition)
	}
	if it.Priority != "" && !priorities[it.Priority] {
		return invalid("illegal priority %q", it.Priority)
	}
	if it.ValueVerdict != "" && !validValueVerdict(it.ValueVerdict) {
		return invalid("illegal value-verdict %q (must start with ADVANCE|HOLD|DISCARD)", it.ValueVerdict)
	}
	return nil
}

// Violation is one failed validation check.
type Violation struct {
	ID      string `json:"id"`
	Check   string `json:"check"`
	Message string `json:"message"`
}

// ValidateResult is the outcome of a full-tree validation.
type ValidateResult struct {
	OK         bool        `json:"ok"`
	Checked    int         `json:"checked"`
	Violations []Violation `json:"violations"`
}

// Validate certifies the backlog tree as schema-well-formed. strict adds the
// cross-field consistency checks.
func (s *Store) Validate(strict bool) (*ValidateResult, error) {
	res := &ValidateResult{OK: true, Violations: []Violation{}}
	idDirs := map[string][]string{} // id -> dirs it appears in

	check := func(dir, ws string) error {
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
				// Git housekeeping files legitimately keep an otherwise-empty
				// workstream directory tracked; they are never backlog items and
				// must not count as stray files (the scaffolded template ships
				// .gitkeep in each empty stream dir).
				if ws != "" && strict && name != ".gitkeep" && name != ".gitignore" {
					res.add(&Violation{ID: name, Check: "stray-file",
						Message: fmt.Sprintf("non-item file %q in workstream dir %s", name, ws)})
				}
				continue
			}
			id := strings.TrimSuffix(name, ".md")
			path := filepath.Join(dir, name)
			res.Checked++
			idDirs[id] = append(idDirs[id], dirLabel(ws))
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			s.validateFile(res, data, id, ws, strict)
		}
		return nil
	}

	if err := check(s.intakeDir(), ""); err != nil {
		return nil, err
	}
	wss, err := s.Workstreams()
	if err != nil {
		return nil, err
	}
	for _, ws := range wss {
		if err := check(s.wsDir(ws), ws); err != nil {
			return nil, err
		}
	}

	// Uniqueness across intake + workstream dirs.
	var dupIDs []string
	for id, dirs := range idDirs {
		if len(dirs) > 1 {
			dupIDs = append(dupIDs, id)
		}
	}
	sort.Strings(dupIDs)
	for _, id := range dupIDs {
		res.add(&Violation{ID: id, Check: "id-unique",
			Message: fmt.Sprintf("id %q appears in multiple dirs: %s", id, strings.Join(idDirs[id], ", "))})
	}

	res.OK = len(res.Violations) == 0
	sort.Slice(res.Violations, func(i, j int) bool {
		if res.Violations[i].ID != res.Violations[j].ID {
			return res.Violations[i].ID < res.Violations[j].ID
		}
		return res.Violations[i].Check < res.Violations[j].Check
	})
	return res, nil
}

func dirLabel(ws string) string {
	if ws == "" {
		return "intake"
	}
	return ws
}

func (r *ValidateResult) add(v *Violation) { r.Violations = append(r.Violations, *v) }

// validateFile checks one item file's bytes against the schema.
func (s *Store) validateFile(res *ValidateResult, data []byte, id, ws string, strict bool) {
	fm, body, err := mdfile.Split(data)
	if err != nil {
		res.add(&Violation{ID: id, Check: "frontmatter", Message: err.Error()})
		return
	}
	// Unknown-key typo guard via strict YAML decoding.
	dec := yaml.NewDecoder(bytes.NewReader(fm))
	dec.KnownFields(true)
	var strictItem Item
	if err := dec.Decode(&strictItem); err != nil {
		res.add(&Violation{ID: id, Check: "unknown-key", Message: err.Error()})
	}

	var it Item
	if err := yaml.Unmarshal(fm, &it); err != nil {
		res.add(&Violation{ID: id, Check: "frontmatter", Message: err.Error()})
		return
	}
	it.ID = id
	it.Body = body

	// Directory / workstream coherence.
	if ws == "" {
		if it.Workstream != "" {
			res.add(&Violation{ID: id, Check: "location",
				Message: fmt.Sprintf("item in intake/ carries workstream %q", it.Workstream)})
		}
	} else if it.Workstream != ws {
		res.add(&Violation{ID: id, Check: "location",
			Message: fmt.Sprintf("item in %s/ has workstream %q", ws, it.Workstream)})
	}

	// Enum legality.
	if it.Status == "" || !statuses[it.Status] {
		res.add(&Violation{ID: id, Check: "enum", Message: fmt.Sprintf("illegal or missing status %q", it.Status)})
	}
	if it.ChangeType != "" && !changeTypes[it.ChangeType] {
		res.add(&Violation{ID: id, Check: "enum", Message: fmt.Sprintf("illegal change-type %q", it.ChangeType)})
	}
	if it.Risk != "" && !risks[it.Risk] {
		res.add(&Violation{ID: id, Check: "enum", Message: fmt.Sprintf("illegal risk %q", it.Risk)})
	}
	if it.Disposition != "" && !dispositions[it.Disposition] {
		res.add(&Violation{ID: id, Check: "enum", Message: fmt.Sprintf("illegal disposition %q", it.Disposition)})
	}
	if it.Priority != "" && !priorities[it.Priority] {
		res.add(&Violation{ID: id, Check: "enum", Message: fmt.Sprintf("illegal priority %q", it.Priority)})
	}
	if it.ValueVerdict != "" && !validValueVerdict(it.ValueVerdict) {
		res.add(&Violation{ID: id, Check: "enum", Message: fmt.Sprintf("illegal value-verdict %q", it.ValueVerdict)})
	}

	// Stage-required keys.
	missing := requiredMissing(&it, ws != "")
	for _, k := range missing {
		res.add(&Violation{ID: id, Check: "required", Message: fmt.Sprintf("missing required key %q for its stage", k)})
	}

	if strict {
		s.strictChecks(res, &it, id)
	}
}

func requiredMissing(it *Item, triaged bool) []string {
	var missing []string
	if strings.TrimSpace(it.Title) == "" {
		missing = append(missing, "title")
	}
	if strings.TrimSpace(it.Value) == "" {
		missing = append(missing, "value")
	}
	if it.Status == "" {
		missing = append(missing, "status")
	}
	if triaged {
		if it.Workstream == "" {
			missing = append(missing, "workstream")
		}
		if it.ChangeType == "" {
			missing = append(missing, "change-type")
		}
		if it.Risk == "" {
			missing = append(missing, "risk")
		}
		if it.Verify == "" {
			missing = append(missing, "verify")
		}
		if it.ValueVerdict == "" {
			missing = append(missing, "value-verdict")
		}
		if it.Disposition == "" {
			missing = append(missing, "disposition")
		}
	}
	return missing
}

// strictChecks are the added cross-field consistency checks under --strict.
func (s *Store) strictChecks(res *ValidateResult, it *Item, id string) {
	// 5. Ready-consistency.
	if it.Status == "approved" && (it.Verify == "" || it.Verify == "none") {
		res.add(&Violation{ID: id, Check: "ready-consistency",
			Message: "status is approved but verify is empty/none"})
	}
	// 6. Disposition coherence.
	if it.Disposition == "AUTO" {
		if it.Verify == "none" || it.Verify == "" {
			res.add(&Violation{ID: id, Check: "disposition-coherence",
				Message: "disposition AUTO requires a non-none verify"})
		}
		if neverAutoChangeTypes[it.ChangeType] {
			res.add(&Violation{ID: id, Check: "disposition-coherence",
				Message: fmt.Sprintf("disposition AUTO illegal for never-auto change-type %q", it.ChangeType)})
		}
	}
	// 7. No hint survives on a triaged item.
	if it.Workstream != "" && it.Hint != "" {
		res.add(&Violation{ID: id, Check: "hint-stripped",
			Message: "triaged item still carries a hint key"})
	}
	// 8. escalated: <file> note points at an existing escalation record.
	if f, ok := escalatedFile(it.Note); ok {
		p := filepath.Join(s.root, ".anthill", "escalations", f)
		if _, err := os.Stat(p); err != nil {
			res.add(&Violation{ID: id, Check: "escalation-ref",
				Message: fmt.Sprintf("note references missing escalation record %q", f)})
		}
	}
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
