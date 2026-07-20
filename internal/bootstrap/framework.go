package bootstrap

import (
	"errors"
	"strings"
)

// frameworkRelPath is the install-relative path to the provenance file that
// carries the synced-through baseline.
const frameworkRelPath = ".anthill/framework.md"

// syncedThroughMarker is the line prefix scaffold stamps and sync/doctor read.
const syncedThroughMarker = "- **synced-through:**"

// ErrNoSyncedThrough is returned when framework.md has no synced-through line to
// stamp or read.
var ErrNoSyncedThrough = errors.New("no synced-through line in framework.md")

// StampFramework replaces the synced-through value in framework.md with the
// given ref and install date, producing "<ref> (installed <date>)". The
// pristine template ships a multi-line <angle-bracket> placeholder as the value;
// the whole placeholder span (from the marker line through the line that closes
// the '>' bracket) is collapsed into a single stamped line. If the value is
// already a single stamped line, only that line is replaced. Returns
// ErrNoSyncedThrough if the marker is absent.
func StampFramework(content []byte, ref, date string) ([]byte, error) {
	text := string(content)
	// Preserve the newline style already in the file.
	nl := "\n"
	if strings.Contains(text, "\r\n") {
		nl = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	start, end, ok := syncedThroughSpan(lines)
	if !ok {
		return nil, ErrNoSyncedThrough
	}
	stamped := syncedThroughMarker + " " + ref + " (installed " + date + ")"
	newLines := append([]string{}, lines[:start]...)
	newLines = append(newLines, stamped)
	newLines = append(newLines, lines[end+1:]...)
	return []byte(strings.Join(newLines, nl)), nil
}

// syncedThroughSpan locates the synced-through value's line span in lines. The
// value may be a single stamped line or the pristine multi-line
// <angle-bracket> placeholder (from the marker line through the line closing the
// '>' bracket). Returns the [start,end] inclusive line indices and whether the
// marker was found.
func syncedThroughSpan(lines []string) (start, end int, ok bool) {
	start = -1
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), syncedThroughMarker) {
			start = i
			break
		}
	}
	if start == -1 {
		return 0, 0, false
	}
	value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[start]), syncedThroughMarker))
	end = start
	if strings.Contains(value, "<") && !strings.Contains(value, ">") {
		for j := start + 1; j < len(lines); j++ {
			end = j
			if strings.Contains(lines[j], ">") {
				break
			}
		}
	}
	return start, end, true
}

// sameExceptSyncedThrough reports whether two framework.md byte slices are
// identical once the synced-through value span is neutralized in both. It lets
// scaffold treat an install whose framework.md differs from the pristine
// template *only* by scaffold's own synced-through stamp as unchanged
// (idempotent), while still refusing a framework.md the user has otherwise
// derived.
func sameExceptSyncedThrough(a, b []byte) bool {
	return neutralizeSyncedThrough(a) == neutralizeSyncedThrough(b)
}

// neutralizeSyncedThrough collapses the synced-through value span to a single
// bare marker line so unrelated framework.md content can be compared.
func neutralizeSyncedThrough(content []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	start, end, ok := syncedThroughSpan(lines)
	if !ok {
		return strings.Join(lines, "\n")
	}
	out := append([]string{}, lines[:start]...)
	out = append(out, syncedThroughMarker)
	out = append(out, lines[end+1:]...)
	return strings.Join(out, "\n")
}

// ReadSyncedThroughRef extracts the recorded ref token from framework.md. When
// the file has been stamped ("<ref> (installed <date>)") it returns "<ref>";
// when the synced-through value is still the pristine <angle-bracket>
// placeholder it returns "" (treated as un-stamped / manual). Returns
// ErrNoSyncedThrough if the marker line is absent.
func ReadSyncedThroughRef(content []byte) (string, error) {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, syncedThroughMarker) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(t, syncedThroughMarker))
		if value == "" || strings.HasPrefix(value, "<") {
			return "", nil // still a placeholder
		}
		// First whitespace-delimited token is the ref.
		if i := strings.IndexAny(value, " \t"); i >= 0 {
			return value[:i], nil
		}
		return value, nil
	}
	return "", ErrNoSyncedThrough
}
