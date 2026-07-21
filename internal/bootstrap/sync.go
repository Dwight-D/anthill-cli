package bootstrap

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SyncResult is the outcome of a sync (or dry-run plan): which reconciled units
// were brought to the embedded ref, which were already current, and which could
// not be reconciled automatically. A unit is a general-tier skill (labeled by
// name) or a framework-invariant non-skill file (labeled by its payload path).
type SyncResult struct {
	Updated   []string // units re-copied verbatim to the embedded version
	Unchanged []string // units already byte-identical to the embedded version
	Conflicts []string // units with an unexpected local edit (need --force to overwrite)
	Created   []string // payload files absent from the install, written verbatim (safe: nothing to clobber)
	FromRef   string   // the install's synced-through ref before sync ("" if unstamped)
	ToRef     string   // the embedded template ref
}

// claudeTemplateRelPath is the always-on-file starter. Unlike the in-place
// derive targets, it is consumed into a DIFFERENT file (the user's CLAUDE.md)
// and then legitimately removed — so recreating it would resurrect an
// un-derived starter, not restore a missing structural file.
const claudeTemplateRelPath = "CLAUDE.template.md"

// createBlocklistPath reports whether a payload path must be skipped by sync's
// create-if-absent pass — i.e. creating it (even when absent) is NOT safe:
//   - .gitignore is scaffold-merged into the consumer's own file, never written
//     standalone;
//   - CLAUDE.template.md is a disposable derive-source (its output is the user's
//     CLAUDE.md, at a different path);
//   - files under a project-defined backlog workstream dir
//     (.anthill/backlog/<ws>/… with <ws> != intake) would resurrect a stream a
//     derivation renamed or dropped (the template ships example streams like
//     product/ that a real install replaces).
//
// The line between "safe to create" and "disposable derive-source" is currently
// hardcoded here; upstream anthill publishing a per-file sync-class schema would
// let the CLI derive it instead (filed: Dwight-D/anthill#3, recorded in
// .anthill/framework.md).
func createBlocklistPath(p string) bool {
	if p == gitignoreRelPath || p == claudeTemplateRelPath {
		return true
	}
	const backlogPrefix = ".anthill/backlog/"
	if rest, ok := strings.CutPrefix(p, backlogPrefix); ok {
		if i := strings.IndexByte(rest, '/'); i >= 0 && rest[:i] != "intake" {
			return true
		}
	}
	return false
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
// A second pass creates any OTHER payload file that is absent from the install
// (Created) — a non-destructive write that carries new upstream scaffold files
// and subtrees to a sync-upgraded install. See createBlocklistPath for the
// paths excluded from creation.
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
	// Create-if-absent pass. Any other payload file missing from the install is
	// written verbatim. A creation cannot clobber anything (the target is
	// absent), so it is always safe — this is how new upstream scaffold files and
	// whole subtrees (e.g. a newly added .anthill/ mechanism, or a required
	// structural dir) reach an install that upgrades via sync instead of a
	// re-scaffold. Files already covered by a reconciled unit are handled above;
	// createBlocklistPath excludes the paths that are unsafe to create.
	unitPaths := map[string]bool{}
	for _, u := range units {
		for _, p := range u.paths {
			unitPaths[p] = true
		}
	}
	payload, err := PayloadFiles()
	if err != nil {
		return nil, err
	}
	var toCreate []string
	for _, p := range payload {
		if unitPaths[p] || createBlocklistPath(p) {
			continue
		}
		dest := filepath.Join(installDir, filepath.FromSlash(p))
		_, serr := os.Stat(dest)
		if os.IsNotExist(serr) {
			toCreate = append(toCreate, p)
		} else if serr != nil {
			return nil, serr
		}
	}
	res.Created = toCreate

	sort.Strings(res.Updated)
	sort.Strings(res.Unchanged)
	sort.Strings(res.Conflicts)
	sort.Strings(res.Created)

	unresolved := len(res.Conflicts) > 0
	if dryRun {
		return res, nil
	}

	// Apply verbatim writes for Updated units and create-if-absent files.
	for _, p := range append(toWrite, toCreate...) {
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
