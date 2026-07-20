package bootstrap

import (
	"os"
	"path/filepath"
	"time"
)

// ScaffoldStatus classifies a target path against the pristine template.
type ScaffoldStatus int

const (
	// StatusWrite: target absent — the file will be written.
	StatusWrite ScaffoldStatus = iota
	// StatusIdentical: target present and byte-identical to the template — safe
	// to skip (re-writing would be a no-op).
	StatusIdentical
	// StatusDiffers: target present and differs from the template — derived or
	// user-edited; refused unless --force.
	StatusDiffers
)

// ScaffoldEntry is one payload path with its classification against the target.
type ScaffoldEntry struct {
	// Path is the install-relative slash path (matches the payload path).
	Path   string
	Status ScaffoldStatus
}

// ClassifyScaffold walks the embedded payload and classifies each file against
// the target directory. It performs no writes.
func ClassifyScaffold(targetDir string) ([]ScaffoldEntry, error) {
	files, err := PayloadFiles()
	if err != nil {
		return nil, err
	}
	entries := make([]ScaffoldEntry, 0, len(files))
	for _, p := range files {
		tmpl, err := ReadTemplateFile(p)
		if err != nil {
			return nil, err
		}
		dest := filepath.Join(targetDir, filepath.FromSlash(p))
		existing, rerr := os.ReadFile(dest)
		status := StatusWrite
		if rerr == nil {
			switch {
			case filesEqual(existing, tmpl):
				status = StatusIdentical
			case p == frameworkRelPath && sameExceptSyncedThrough(existing, tmpl):
				// framework.md differs only by scaffold's own synced-through
				// stamp — an idempotent re-run, not a user derivation.
				status = StatusIdentical
			default:
				status = StatusDiffers
			}
		}
		entries = append(entries, ScaffoldEntry{Path: p, Status: status})
	}
	return entries, nil
}

// ScaffoldResult reports the outcome of an apply (or a dry-run plan).
type ScaffoldResult struct {
	Written []string // paths written (absent, or overwritten under force)
	Skipped []string // paths present and byte-identical
	Refused []string // paths present and differing (no force) — nothing written
	Ref     string   // the embedded template ref stamped into framework.md
}

// Scaffold writes the embedded payload into targetDir per the non-destructive
// rule: write if absent; skip if byte-identical; refuse if differing unless
// force. When force is set, differing files are overwritten (counted as
// written). On any successful write pass it stamps .anthill/framework.md
// synced-through with the embedded ref and today's UTC date.
//
// dryRun computes the manifest and writes nothing (Ref is still reported).
// When any path is refused (and not dryRun/force) nothing is written and the
// caller maps this to exit 3.
func Scaffold(targetDir string, force, dryRun bool) (*ScaffoldResult, error) {
	entries, err := ClassifyScaffold(targetDir)
	if err != nil {
		return nil, err
	}
	res := &ScaffoldResult{Ref: TemplateRef}
	var refused []string
	for _, e := range entries {
		switch e.Status {
		case StatusIdentical:
			res.Skipped = append(res.Skipped, e.Path)
		case StatusDiffers:
			if force {
				res.Written = append(res.Written, e.Path)
			} else {
				refused = append(refused, e.Path)
			}
		default: // StatusWrite
			res.Written = append(res.Written, e.Path)
		}
	}
	res.Refused = refused

	if dryRun {
		return res, nil
	}
	if len(refused) > 0 && !force {
		// Refusal is terminal: write nothing, leave the tree untouched.
		return res, nil
	}

	// Apply: write every path classified as written.
	writeSet := map[string]bool{}
	for _, p := range res.Written {
		writeSet[p] = true
	}
	for _, e := range entries {
		if !writeSet[e.Path] {
			continue
		}
		data, rerr := ReadTemplateFile(e.Path)
		if rerr != nil {
			return nil, rerr
		}
		dest := filepath.Join(targetDir, filepath.FromSlash(e.Path))
		if werr := atomicWrite(dest, data, fileModeFor(e.Path)); werr != nil {
			return nil, werr
		}
	}

	// Stamp framework.md synced-through (best-effort: only if present).
	if err := stampInstalledFramework(targetDir); err != nil {
		return nil, err
	}
	return res, nil
}

// stampInstalledFramework rewrites the installed .anthill/framework.md
// synced-through line with the embedded ref + today's UTC date. A missing
// framework.md or a missing marker is not an error (the payload always ships
// one, but a partial tree should not crash the scaffold).
func stampInstalledFramework(targetDir string) error {
	path := filepath.Join(targetDir, filepath.FromSlash(frameworkRelPath))
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	// Idempotent: if already stamped at the embedded ref, leave it untouched so
	// a re-scaffold is a true no-op (and preserves the original install date).
	if ref, rerr := ReadSyncedThroughRef(content); rerr == nil && ref == TemplateRef {
		return nil
	}
	date := time.Now().UTC().Format("2006-01-02")
	stamped, err := StampFramework(content, TemplateRef, date)
	if err != nil {
		if err == ErrNoSyncedThrough {
			return nil
		}
		return err
	}
	return atomicWrite(path, stamped, 0o644)
}
