package bootstrap

import (
	"regexp"
	"strings"
)

// AutonomousSkill is the one general-tier skill with sanctioned install-time
// adaptations, so its integrity check is region-aware rather than byte-exact.
const AutonomousSkill = "autonomous"

// decisionsLogRe matches a backtick-quoted decisions-log path (the sanctioned
// autonomous adaptation point besides the proceed-list), e.g.
// `.anthill/decisions.md` or a project's relocated equivalent.
var decisionsLogRe = regexp.MustCompile("`[^`]*decisions[^`]*\\.md`")

// adaptationCommentRe matches the "ADAPTATION POINT" HTML comment block that
// wraps the proceed-list placeholder in the pristine template. A real derivation
// removes it along with swapping the placeholder list (INSTALLATION.md Step 2),
// so it is part of the sanctioned proceed-list region and must be exempted too.
var adaptationCommentRe = regexp.MustCompile(`(?s)<!--\s*ADAPTATION POINT.*?-->`)

// normalizeAutonomous canonicalizes the two sanctioned adaptation regions of the
// autonomous SKILL.md so an install can be compared against the pristine
// template without the derived proceed-list or a relocated decisions-log path
// registering as an illegal edit.
//
// Region approach (pragmatic region-diff, per spec §4.4):
//   - Proceed-list: the body of the "## Proceed freely" section — every line
//     from that heading up to (but not including) the next "## " heading — is
//     dropped, so any derived contents there compare equal.
//   - Decisions-log path: any backtick-quoted `...decisions*.md` token is
//     replaced with a fixed sentinel, so relocating the log compares equal.
//
// Everything else in the file must still match the template byte-for-byte.
func normalizeAutonomous(b []byte) string {
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	// Drop the sanctioned ADAPTATION POINT comment (present in the template,
	// removed by a real derivation) so its absence is not an illegal edit.
	s = adaptationCommentRe.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	var out []string
	inProceed := false
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "## ") {
			// A "## " heading ends the proceed-freely region; enter the region
			// when the heading is "Proceed freely".
			inProceed = strings.HasPrefix(trimmed, "## Proceed freely")
			if inProceed {
				// Keep the heading itself as an anchor, drop its body.
				out = append(out, "## Proceed freely")
				continue
			}
		}
		if inProceed {
			continue // drop proceed-list body lines
		}
		out = append(out, decisionsLogRe.ReplaceAllString(ln, "`<DECISIONS-LOG>`"))
	}
	// Collapse blank-line runs (and trim ends) so whitespace left by the removed
	// comment / proceed-list body does not register as a difference.
	return collapseBlankLines(out)
}

// collapseBlankLines joins lines, squeezing any run of blank lines to one and
// trimming leading/trailing blanks.
func collapseBlankLines(lines []string) string {
	var out []string
	prevBlank := false
	for _, ln := range lines {
		blank := strings.TrimSpace(ln) == ""
		if blank && prevBlank {
			continue
		}
		out = append(out, ln)
		prevBlank = blank
	}
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// SkillFileMatches reports whether an installed skill file's bytes are
// acceptable against the pristine template bytes. For the autonomous skill's
// SKILL.md the comparison is region-aware (the sanctioned adaptations are
// exempted); every other skill file must be byte-identical.
func SkillFileMatches(payloadPath string, installed, template []byte) bool {
	if SkillNameOf(payloadPath) == AutonomousSkill && strings.HasSuffix(payloadPath, "/SKILL.md") {
		return normalizeAutonomous(installed) == normalizeAutonomous(template)
	}
	return filesEqual(installed, template)
}
