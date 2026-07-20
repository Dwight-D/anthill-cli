// Package mdfile parses and writes markdown files with YAML frontmatter, and
// provides atomic (temp-file + rename) persistence used by the backlog and
// escalation stores.
package mdfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoFrontmatter is returned when a file does not begin with a `---` fence.
var ErrNoFrontmatter = errors.New("no yaml frontmatter (missing leading '---')")

// ErrUnterminated is returned when the opening frontmatter fence has no match.
var ErrUnterminated = errors.New("unterminated yaml frontmatter (missing closing '---')")

// Split separates a markdown file into its raw YAML frontmatter bytes and the
// remaining markdown body. Line endings are normalised to "\n". The body has
// its leading blank line trimmed and trailing whitespace trimmed.
func Split(data []byte) (frontmatter []byte, body string, err error) {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		return nil, "", ErrNoFrontmatter
	}
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		return nil, "", ErrUnterminated
	}
	fm := strings.Join(lines[1:closeIdx], "\n")
	rest := strings.Join(lines[closeIdx+1:], "\n")
	rest = strings.TrimLeft(rest, "\n")
	rest = strings.TrimRight(rest, " \t\n")
	return []byte(fm), rest, nil
}

// Compose renders frontmatter + body back into a full markdown file. yamlBytes
// is expected to already end with a newline (yaml.Marshal does). The body, when
// non-empty, is separated from the closing fence by one blank line.
func Compose(yamlBytes []byte, body string) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(yamlBytes)
	if len(yamlBytes) == 0 || yamlBytes[len(yamlBytes)-1] != '\n' {
		b.WriteString("\n")
	}
	b.WriteString("---\n")
	if strings.TrimSpace(body) != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimRight(body, "\n"))
		b.WriteString("\n")
	}
	return []byte(b.String())
}

// WriteAtomic writes data to path via a temp file in the same directory followed
// by an atomic rename, so a rejected or crashed write never leaves a half-file.
func WriteAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file to %s: %w", path, err)
	}
	return nil
}
