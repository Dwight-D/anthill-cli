package bootstrap

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SkillResult is one general-tier skill's integrity verdict against the pinned
// template.
type SkillResult struct {
	Name   string
	OK     bool
	Detail string
}

// CheckSkillIntegrity compares every installed general-tier skill under
// installDir/.claude/skills against the embedded pinned template. Each skill is
// OK when all its files are present and byte-identical to the pinned version —
// there are no exempted regions. A missing or diverging file marks the skill
// not-OK — the exact local-edit drift the two-tier split prevents.
func CheckSkillIntegrity(installDir string) ([]SkillResult, error) {
	names, err := SkillNames()
	if err != nil {
		return nil, err
	}
	files, err := SkillFiles()
	if err != nil {
		return nil, err
	}
	bySkill := map[string][]string{}
	for _, p := range files {
		n := SkillNameOf(p)
		bySkill[n] = append(bySkill[n], p)
	}

	results := make([]SkillResult, 0, len(names))
	for _, name := range names {
		var problems []string
		for _, p := range bySkill[name] {
			tmpl, terr := ReadTemplateFile(p)
			if terr != nil {
				return nil, terr
			}
			dest := filepath.Join(installDir, filepath.FromSlash(p))
			installed, rerr := os.ReadFile(dest)
			if rerr != nil {
				if os.IsNotExist(rerr) {
					problems = append(problems, "missing "+p)
					continue
				}
				return nil, rerr
			}
			if !filesEqual(installed, tmpl) {
				problems = append(problems, "local edit in "+p)
			}
		}
		r := SkillResult{Name: name, OK: len(problems) == 0}
		if r.OK {
			r.Detail = "byte-identical to pinned template"
		} else {
			r.Detail = strings.Join(problems, "; ")
		}
		results = append(results, r)
	}
	return results, nil
}

// CheckStructure returns the expected .anthill/ skeleton directories that are
// missing under installDir (empty slice when the tree is complete). It checks
// only the STABLE skeleton; the per-workstream backlog subdirectories
// (bugs/dev/process/product/…) are project-defined — a derivation renames the
// product stream and may add or drop others — so they are excluded here and
// validated instead by Section B's sweep-order check (every sweep-order name has
// a directory).
func CheckStructure(installDir string) ([]string, error) {
	dirs, err := PayloadDirs()
	if err != nil {
		return nil, err
	}
	const backlogPrefix = ".anthill/backlog/"
	var missing []string
	for _, d := range dirs {
		if !strings.HasPrefix(d, ".anthill/") && d != ".anthill" {
			continue
		}
		// Skip project-defined workstream dirs — direct children of
		// .anthill/backlog/ other than the fixed intake/ queue.
		if strings.HasPrefix(d, backlogPrefix) {
			rest := d[len(backlogPrefix):]
			if rest != "intake" && !strings.Contains(rest, "/") {
				continue
			}
		}
		info, err := os.Stat(filepath.Join(installDir, filepath.FromSlash(d)))
		if err != nil || !info.IsDir() {
			missing = append(missing, d)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

// templateMarkers are the sentinel phrases the derivation session removes from a
// template file (the PROJECT-SPECIFIC / RUNTIME-ARTIFACT quote-blocks and the
// <YOUR PROJECT> name placeholder). Their presence — NOT the presence of any
// <angle-bracket> token — is what marks a file as un-derived. Legitimate
// angle-bracket content survives derivation: schema field descriptions
// (`<one line>`), record/filename templates (`<yyyy-mm-dd>-<slug>.md`), and the
// per-use worker-brief skeleton (`<files/folders>`) are permanent format
// documentation, not fill-ins, so they must not be flagged.
var templateMarkers = []string{
	"PROJECT-SPECIFIC",
	"RUNTIME ARTIFACT",
	"<YOUR PROJECT>",
}

// DerivationStatus returns the install-relative paths of .anthill/ markdown
// files that still hold an un-derived template quote-block (see templateMarkers)
// — i.e. remain un-derived. Informational: reported by doctor but only a hard
// failure under --strict.
func DerivationStatus(installDir string) ([]string, error) {
	anthillDir := filepath.Join(installDir, ".anthill")
	var undrived []string
	err := filepath.WalkDir(anthillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		s := string(data)
		for _, marker := range templateMarkers {
			if strings.Contains(s, marker) {
				rel, _ := filepath.Rel(installDir, path)
				undrived = append(undrived, filepath.ToSlash(rel))
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(undrived)
	return undrived, nil
}

// SyncState reports an install's synced-through ref against the embedded ref.
type SyncState struct {
	InstalledRef string // recorded ref ("" if unstamped / manual)
	EmbeddedRef  string
	UpToDate     bool
	Detail       string
}

// SyncStatus reads installDir/.anthill/framework.md synced-through and compares
// it to the embedded TemplateRef.
func SyncStatus(installDir string) (SyncState, error) {
	st := SyncState{EmbeddedRef: TemplateRef}
	path := filepath.Join(installDir, filepath.FromSlash(frameworkRelPath))
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			st.Detail = "framework.md not found"
			return st, nil
		}
		return st, err
	}
	ref, rerr := ReadSyncedThroughRef(content)
	if rerr != nil {
		st.Detail = "no synced-through line in framework.md"
		return st, nil
	}
	st.InstalledRef = ref
	switch {
	case ref == "":
		st.Detail = "synced-through unstamped (manual install)"
	case ref == TemplateRef:
		st.UpToDate = true
		st.Detail = "up-to-date (" + shortRef(ref) + ")"
	default:
		st.Detail = "behind: install " + shortRef(ref) + " vs embedded " + shortRef(TemplateRef)
	}
	return st, nil
}

// shortRef abbreviates a 40-char commit ref for display.
func shortRef(ref string) string {
	if len(ref) > 12 {
		return ref[:12]
	}
	return ref
}
