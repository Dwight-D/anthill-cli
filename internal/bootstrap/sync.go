package bootstrap

import (
	"os"
	"path/filepath"
	"sort"
)

// SyncResult is the outcome of a sync (or dry-run plan): which reconciled units
// were brought to the embedded ref, which were already current, and which could
// not be reconciled automatically. A unit is a general-tier skill (labeled by
// name) or a framework-invariant non-skill file (labeled by its payload path).
type SyncResult struct {
	Updated   []string // units re-copied verbatim to the embedded version
	Unchanged []string // units already byte-identical to the embedded version
	Conflicts []string // units with an unexpected local edit (need --force to overwrite)
	FromRef   string   // the install's synced-through ref before sync ("" if unstamped)
	ToRef     string   // the embedded template ref
}

// syncUnit is one reconcilable payload unit: a display label (a skill name or a
// file path) and the payload file paths it covers.
type syncUnit struct {
	label string
	paths []string
}

// syncedUnits returns everything sync reconciles against the embedded template,
// in report order: the general-tier skills (one unit per skill, labeled by
// name), then the framework-invariant non-skill files (one unit per file,
// labeled by its path).
func syncedUnits() ([]syncUnit, error) {
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
	var units []syncUnit
	for _, name := range names {
		units = append(units, syncUnit{label: name, paths: bySkill[name]})
	}
	for _, p := range FrameworkInvariantFiles() {
		units = append(units, syncUnit{label: p, paths: []string{p}})
	}
	return units, nil
}

// Sync brings installDir's synced units (general-tier skills plus the
// framework-invariant non-skill files) to the embedded pinned template.
//
// Per-unit reconciliation is byte-exact and uniform — there are no adaptation
// regions or merges:
//   - A unit whose files all match the embedded version is Unchanged.
//   - A unit that differs while the install's synced-through already equals the
//     embedded ref is treated as an unexpected local edit → Conflict (needs
//     --force to overwrite); otherwise (the install is behind) a difference is an
//     upstream update and the unit is re-copied verbatim → Updated.
//
// On a clean apply the synced-through baseline is bumped to the embedded ref.
// With unresolved conflicts and no force, nothing is bumped and the caller maps
// this to exit 3. dryRun computes the plan and writes nothing.
func Sync(installDir string, dryRun, force bool) (*SyncResult, error) {
	units, err := syncedUnits()
	if err != nil {
		return nil, err
	}

	fromRef := ""
	if content, rerr := os.ReadFile(filepath.Join(installDir, filepath.FromSlash(frameworkRelPath))); rerr == nil {
		if ref, ferr := ReadSyncedThroughRef(content); ferr == nil {
			fromRef = ref
		}
	}
	installClaimsCurrent := fromRef == TemplateRef

	res := &SyncResult{FromRef: fromRef, ToRef: TemplateRef}
	var toWrite []string // payload paths to copy verbatim (for Updated units)

	for _, u := range units {
		var writes []string
		conflict := false
		for _, p := range u.paths {
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

			switch {
			case missing:
				writes = append(writes, p)
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
			res.Conflicts = append(res.Conflicts, u.label)
		case len(writes) > 0:
			res.Updated = append(res.Updated, u.label)
			toWrite = append(toWrite, writes...)
		default:
			res.Unchanged = append(res.Unchanged, u.label)
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
