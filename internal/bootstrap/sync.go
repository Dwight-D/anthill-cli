package bootstrap

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SyncResult is the outcome of a sync (or dry-run plan): which general-tier
// skills were brought to the embedded ref, which were already current, and
// which could not be reconciled automatically.
type SyncResult struct {
	Updated   []string // skills re-copied verbatim to the embedded version
	Unchanged []string // skills already current (autonomous adaptations preserved)
	Conflicts []string // skills whose upstream change collides with a local edit / adaptation
	FromRef   string   // the install's synced-through ref before sync ("" if unstamped)
	ToRef     string   // the embedded template ref
}

// Sync brings installDir's general-tier skills to the embedded pinned template.
//
// Per-skill reconciliation:
//   - A skill whose files all match the embedded version is Unchanged.
//   - A skill that differs while the install's synced-through already equals the
//     embedded ref is treated as a local edit → Conflict (needs --force);
//     otherwise (the install is behind) a difference is an upstream update and
//     the skill is re-copied verbatim → Updated.
//   - The autonomous skill is region-aware: when the only differences fall inside
//     the sanctioned adaptation regions (proceed-list, decisions-log path) it is
//     Unchanged and its adaptations are preserved untouched. When the surrounding
//     text also differs, the upstream change cannot be merged around the
//     adaptations automatically → Conflict (needs --force, which overwrites
//     verbatim and drops the adaptations).
//
// On a clean apply the synced-through baseline is bumped to the embedded ref.
// With unresolved conflicts and no force, nothing is bumped and the caller maps
// this to exit 3. dryRun computes the plan and writes nothing.
func Sync(installDir string, dryRun, force bool) (*SyncResult, error) {
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
		bySkill[SkillNameOf(p)] = append(bySkill[SkillNameOf(p)], p)
	}

	fromRef := ""
	if content, rerr := os.ReadFile(filepath.Join(installDir, filepath.FromSlash(frameworkRelPath))); rerr == nil {
		if ref, ferr := ReadSyncedThroughRef(content); ferr == nil {
			fromRef = ref
		}
	}
	installClaimsCurrent := fromRef == TemplateRef

	res := &SyncResult{FromRef: fromRef, ToRef: TemplateRef}
	var toWrite []string // payload paths to copy verbatim (for Updated skills)

	for _, name := range names {
		var writes []string
		conflict := false
		for _, p := range bySkill[name] {
			tmpl, terr := ReadTemplateFile(p)
			if terr != nil {
				return nil, terr
			}
			dest := filepath.Join(installDir, filepath.FromSlash(p))
			installed, rerr := os.ReadFile(dest)
			missing := os.IsNotExist(rerr)
			if rerr != nil && !missing {
				return nil, rerr
			}

			isAutonomousSkill := name == AutonomousSkill && strings.HasSuffix(p, "/SKILL.md")
			switch {
			case missing:
				writes = append(writes, p)
			case isAutonomousSkill:
				if normalizeAutonomous(installed) == normalizeAutonomous(tmpl) {
					// Only sanctioned regions differ (if anything): preserve.
					continue
				}
				// Surrounding text diverged: unmergeable without guessing.
				if force {
					writes = append(writes, p)
				} else {
					conflict = true
				}
			case filesEqual(installed, tmpl):
				// current — nothing to do
			default:
				// Differs. Behind install → upstream update; current install → local edit.
				if installClaimsCurrent && !force {
					conflict = true
				} else {
					writes = append(writes, p)
				}
			}
		}
		switch {
		case conflict:
			res.Conflicts = append(res.Conflicts, name)
		case len(writes) > 0:
			res.Updated = append(res.Updated, name)
			toWrite = append(toWrite, writes...)
		default:
			res.Unchanged = append(res.Unchanged, name)
		}
	}
	sort.Strings(res.Updated)
	sort.Strings(res.Unchanged)
	sort.Strings(res.Conflicts)

	unresolved := len(res.Conflicts) > 0
	if dryRun {
		return res, nil
	}

	// Apply verbatim writes for Updated skills.
	for _, p := range toWrite {
		data, rerr := ReadTemplateFile(p)
		if rerr != nil {
			return nil, rerr
		}
		dest := filepath.Join(installDir, filepath.FromSlash(p))
		if werr := atomicWrite(dest, data, fileModeFor(p)); werr != nil {
			return nil, werr
		}
	}

	// Bump synced-through only when the install is fully reconciled.
	if !unresolved {
		if err := stampInstalledFramework(installDir); err != nil {
			return nil, err
		}
	}
	return res, nil
}
