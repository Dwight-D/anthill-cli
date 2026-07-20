package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
	"time"
)

// ScaffoldStatus classifies a target path against the pristine template.
type ScaffoldStatus int

const (
	// StatusWrite: target absent — the file will be written.
	StatusWrite ScaffoldStatus = iota
	// StatusIdentical: target present and byte-identical to the template (or, for
	// .gitignore, already carrying the framework block) — safe to skip.
	StatusIdentical
	// StatusDiffers: target present and differs from the template — derived or
	// user-edited; refused unless --force.
	StatusDiffers
	// StatusAppend: the target's .gitignore needs the framework block appended.
	// Install targets are assumed to already have a project .gitignore, so the
	// framework rules are merged in rather than overwriting or refusing.
	StatusAppend
)

// gitignoreRelPath is the payload path handled by append-merge rather than the
// write/skip/refuse rule.
const gitignoreRelPath = ".gitignore"

// Sentinel markers wrapping the framework's ignore rules inside a consumer's
// .gitignore. Their presence is what makes the append idempotent: a re-scaffold
// that finds the start marker leaves the file untouched.
const (
	gitignoreMarkerStart = "# >>> anthill scaffold (framework ignore rules) >>>"
	gitignoreMarkerEnd   = "# <<< anthill scaffold <<<"
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
		dest := filepath.Join(targetDir, filepath.FromSlash(p))
		existing, rerr := os.ReadFile(dest)

		// .gitignore is merged, never refused: a real install target almost
		// always has its own project .gitignore.
		if p == gitignoreRelPath {
			status := StatusAppend
			if rerr == nil && bytes.Contains(existing, []byte(gitignoreMarkerStart)) {
				status = StatusIdentical // framework block already present
			}
			entries = append(entries, ScaffoldEntry{Path: p, Status: status})
			continue
		}

		tmpl, err := ReadTemplateFile(p)
		if err != nil {
			return nil, err
		}
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

// ScaffoldResult reports the outcome of an apply (or a dry-run plan). Every list
// reflects what actually happened on disk (on a dry-run, what would happen).
type ScaffoldResult struct {
	Written []string // paths written (absent, or overwritten under force)
	Merged  []string // paths whose framework block was appended (.gitignore)
	Skipped []string // paths present and already satisfied (identical / block present)
	Refused []string // paths present and differing (no force) — left untouched
	Ref     string   // the embedded template ref stamped into framework.md
}

// Scaffold writes the embedded payload into targetDir per the non-destructive
// rule: write if absent; skip if byte-identical; append the framework block to
// an existing .gitignore; refuse (leave untouched) a differing file unless
// force. When force is set, differing files are overwritten (counted as
// written).
//
// Refusal is PER FILE, not terminal: every safe path (absent files, the
// .gitignore merge, and — under force — differing files) is still installed,
// and the result lists reflect exactly what was written. A refused path only
// signals a non-zero exit to the caller; it never rewinds the files that were
// written. .anthill/framework.md is stamped with the embedded ref unless it was
// itself refused.
//
// dryRun computes the manifest and writes nothing (Ref is still reported).
func Scaffold(targetDir string, force, dryRun bool) (*ScaffoldResult, error) {
	entries, err := ClassifyScaffold(targetDir)
	if err != nil {
		return nil, err
	}
	res := &ScaffoldResult{Ref: TemplateRef}
	frameworkRefused := false
	for _, e := range entries {
		switch e.Status {
		case StatusIdentical:
			res.Skipped = append(res.Skipped, e.Path)
		case StatusAppend:
			res.Merged = append(res.Merged, e.Path)
		case StatusDiffers:
			if force {
				res.Written = append(res.Written, e.Path)
			} else {
				res.Refused = append(res.Refused, e.Path)
				if e.Path == frameworkRelPath {
					frameworkRefused = true
				}
			}
		default: // StatusWrite
			res.Written = append(res.Written, e.Path)
		}
	}

	if dryRun {
		return res, nil
	}

	// Apply. Refused paths are simply not acted on — the rest still installs.
	for _, e := range entries {
		dest := filepath.Join(targetDir, filepath.FromSlash(e.Path))
		switch e.Status {
		case StatusAppend:
			tmpl, rerr := ReadTemplateFile(e.Path)
			if rerr != nil {
				return nil, rerr
			}
			existing, _ := os.ReadFile(dest) // absent → nil, merged into a fresh block
			if werr := atomicWrite(dest, mergeGitignore(existing, tmpl), fileModeFor(e.Path)); werr != nil {
				return nil, werr
			}
		case StatusWrite:
			if werr := writeTemplateFile(e.Path, dest); werr != nil {
				return nil, werr
			}
		case StatusDiffers:
			if force {
				if werr := writeTemplateFile(e.Path, dest); werr != nil {
					return nil, werr
				}
			}
		}
	}

	// Stamp framework.md synced-through unless it was refused (a differing,
	// user-derived framework.md is left exactly as found).
	if !frameworkRefused {
		if err := stampInstalledFramework(targetDir); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// writeTemplateFile copies one pristine payload file to dest.
func writeTemplateFile(slashPath, dest string) error {
	data, err := ReadTemplateFile(slashPath)
	if err != nil {
		return err
	}
	return atomicWrite(dest, data, fileModeFor(slashPath))
}

// mergeGitignore returns existing with the framework ignore block appended. If
// existing already contains the start marker it is returned unchanged
// (idempotent). An absent/empty existing yields just the block.
func mergeGitignore(existing, tmpl []byte) []byte {
	block := gitignoreMarkerStart + "\n" +
		string(bytes.TrimRight(tmpl, "\n")) + "\n" +
		gitignoreMarkerEnd + "\n"
	if len(bytes.TrimSpace(existing)) == 0 {
		return []byte(block)
	}
	if bytes.Contains(existing, []byte(gitignoreMarkerStart)) {
		return existing
	}
	out := make([]byte, 0, len(existing)+len(block)+2)
	out = append(out, existing...)
	if !bytes.HasSuffix(existing, []byte("\n")) {
		out = append(out, '\n')
	}
	out = append(out, '\n') // blank line separating the consumer's rules from ours
	out = append(out, block...)
	return out
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
