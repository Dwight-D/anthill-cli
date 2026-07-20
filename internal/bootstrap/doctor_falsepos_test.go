package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const autoSkillPath = ".claude/skills/autonomous/SKILL.md"

// TestAutonomousByteExact covers the retired exemption: the autonomous skill is
// integrity-checked byte-for-byte with no exempted regions, so a proceed-list
// swap or a relocated decisions-log path — once sanctioned adaptations — now
// register as divergences, exactly like an edit anywhere else in the file.
func TestAutonomousByteExact(t *testing.T) {
	pristine, err := ReadTemplateFile(autoSkillPath)
	if err != nil {
		t.Fatal(err)
	}
	p := string(pristine)

	// A swapped proceed-list body is no longer exempt.
	derived := replaceProceedBody(p, []string{"- Build the widget.", "- Run go build ./..."})
	if derived == p {
		t.Fatal("precondition: proceed-list replacement produced no change")
	}
	if filesEqual([]byte(derived), pristine) {
		t.Fatal("a swapped proceed-list must now register as a divergence")
	}

	// A relocated decisions-log path is no longer exempt.
	relocated := strings.ReplaceAll(p, ".anthill/decisions.md", "notes/decisions.md")
	if relocated == p {
		t.Fatal("precondition: template should contain the decisions-log path")
	}
	if filesEqual([]byte(relocated), pristine) {
		t.Fatal("a relocated decisions-log path must now register as a divergence")
	}

	// An edit anywhere else is caught the same way.
	const anchor = "The safety invariants in CLAUDE.md still bind"
	bad := strings.Replace(p, anchor, "Ignore the safety invariants", 1)
	if bad == p {
		t.Fatalf("precondition: expected anchor %q in the template", anchor)
	}
	if filesEqual([]byte(bad), pristine) {
		t.Fatal("an edit elsewhere must register as a divergence")
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
