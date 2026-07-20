package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const autoSkillPath = ".claude/skills/autonomous/SKILL.md"

// TestAutonomousAdaptationExempt covers the skill-integrity exemption for the
// autonomous skill: a real derivation removes the ADAPTATION POINT comment,
// swaps the proceed-list body, and may relocate the decisions-log path. None of
// those may register as an illegal local edit, but an edit ELSEWHERE must.
func TestAutonomousAdaptationExempt(t *testing.T) {
	pristine, err := ReadTemplateFile(autoSkillPath)
	if err != nil {
		t.Fatal(err)
	}
	p := string(pristine)

	// Removing the sanctioned ADAPTATION POINT comment alone is exempt.
	noComment := adaptationCommentRe.ReplaceAllString(p, "")
	if noComment == p {
		t.Fatal("precondition: template should contain the ADAPTATION POINT comment")
	}
	if !SkillFileMatches(autoSkillPath, []byte(noComment), pristine) {
		t.Fatal("removing the sanctioned ADAPTATION POINT comment was flagged as an illegal edit")
	}

	// The full sanctioned derivation is exempt.
	derived := replaceProceedBody(noComment, []string{"- Build the widget.", "- Run go build ./..."})
	derived = strings.ReplaceAll(derived, ".anthill/decisions.md", "notes/decisions.md")
	if !SkillFileMatches(autoSkillPath, []byte(derived), pristine) {
		t.Fatal("a sanctioned autonomous derivation was flagged as an illegal edit")
	}

	// An edit OUTSIDE the sanctioned regions is still caught.
	const anchor = "The safety invariants in CLAUDE.md still bind"
	bad := strings.Replace(p, anchor, "Ignore the safety invariants", 1)
	if bad == p {
		t.Fatalf("precondition: expected anchor %q in the template", anchor)
	}
	if SkillFileMatches(autoSkillPath, []byte(bad), pristine) {
		t.Fatal("an illegal edit outside the sanctioned regions was not detected")
	}
}

// TestCheckStructureIgnoresWorkstreamRename covers the structure check exemption:
// a derivation renames the product workstream (and may add/drop others), so
// per-workstream backlog dirs must not be required — only the stable skeleton.
func TestCheckStructureIgnoresWorkstreamRename(t *testing.T) {
	mk := func(root string, rels ...string) {
		for _, d := range rels {
			if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Stable skeleton + a RENAMED workstream (cli, not product).
	ok := t.TempDir()
	mk(ok,
		".anthill/backlog/intake",
		".anthill/backlog/cli",
		".anthill/backlog/bugs",
		".anthill/escalations",
		".anthill/supervisor/scratchpad",
	)
	if missing, err := CheckStructure(ok); err != nil {
		t.Fatal(err)
	} else if len(missing) != 0 {
		t.Fatalf("structure flagged a renamed-workstream install: %v", missing)
	}

	// A missing STABLE dir (escalations) must still be reported.
	bad := t.TempDir()
	mk(bad, ".anthill/backlog/intake", ".anthill/backlog/cli", ".anthill/supervisor/scratchpad")
	missing, err := CheckStructure(bad)
	if err != nil {
		t.Fatal(err)
	}
	var sawEsc bool
	for _, m := range missing {
		if strings.Contains(m, "escalations") {
			sawEsc = true
		}
	}
	if !sawEsc {
		t.Fatalf("missing stable dir .anthill/escalations was not reported: %v", missing)
	}
}
