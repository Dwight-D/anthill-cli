package bootstrap

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
// OK when all its files are present and match (byte-identical, except the
// autonomous skill's sanctioned adaptation regions, which are exempted). A
// missing or diverging file marks the skill not-OK — the exact local-edit
// drift the two-tier split prevents.
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
			if !SkillFileMatches(p, installed, tmpl) {
				problems = append(problems, "local edit in "+p)
			}
		}
		r := SkillResult{Name: name, OK: len(problems) == 0}
		if r.OK {
			if name == AutonomousSkill {
				r.Detail = "matches pinned template (sanctioned adaptations exempted)"
			} else {
				r.Detail = "byte-identical to pinned template"
			}
		} else {
			r.Detail = strings.Join(problems, "; ")
		}
		results = append(results, r)
	}
	return results, nil
}

// CheckStructure returns the expected .anthill/ payload directories that are
// missing under installDir (empty slice when the tree is complete).
func CheckStructure(installDir string) ([]string, error) {
	dirs, err := PayloadDirs()
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, d := range dirs {
		if !strings.HasPrefix(d, ".anthill/") && d != ".anthill" {
			continue
		}
		info, err := os.Stat(filepath.Join(installDir, filepath.FromSlash(d)))
		if err != nil || !info.IsDir() {
			missing = append(missing, d)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

// placeholderRe matches an un-filled <angle-bracket> template placeholder (an
// opening '<' followed by a letter and no nested '<' or newline before '>').
var placeholderRe = regexp.MustCompile(`<[A-Za-z][^<>\n]*>`)

// DerivationStatus returns the install-relative paths of .anthill/ markdown
// files that still hold template quote-blocks ("PROJECT-SPECIFIC") or
// <angle-bracket> fill-ins — i.e. remain un-derived. Informational: reported by
// doctor but only a hard failure under --strict.
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
		if strings.Contains(s, "PROJECT-SPECIFIC") || placeholderRe.MatchString(s) {
			rel, _ := filepath.Rel(installDir, path)
			undrived = append(undrived, filepath.ToSlash(rel))
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
