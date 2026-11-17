package backlog

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// CurrentSchema is the backlog schema version this CLI writes and validates.
const CurrentSchema = 2

func (s *Store) workstreamsPath() string { return filepath.Join(s.dir(), "workstreams.md") }

// SchemaVersion returns the `schema` stamp in workstreams.md frontmatter, or 0
// when absent (an unstamped tree).
func (s *Store) SchemaVersion() (int, error) {
	data, err := os.ReadFile(s.workstreamsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	fm, _, ferr := mdfile.Split(data)
	if ferr != nil {
		return 0, nil
	}
	var meta struct {
		Schema int `yaml:"schema"`
	}
	if yaml.Unmarshal(fm, &meta) != nil {
		return 0, nil
	}
	return meta.Schema, nil
}

// NeedsMigration reports whether the tree holds items but lacks the current
// schema stamp. An unstamped tree with no items has nothing to migrate.
func (s *Store) NeedsMigration() (bool, error) {
	v, err := s.SchemaVersion()
	if err != nil {
		return false, err
	}
	if v == CurrentSchema {
		return false, nil
	}
	n, err := s.countItemFiles()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// countItemFiles counts *.md files in every item directory, parseable or not.
func (s *Store) countItemFiles() (int, error) {
	dirs, err := s.itemDirs(true)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				n++
			}
		}
	}
	return n, nil
}

// StampSchema writes `schema: <CurrentSchema>` into workstreams.md frontmatter,
// editing the file textually so the rest of it is preserved byte for byte. An
// existing `schema:` line is replaced; a file without frontmatter gains one.
func (s *Store) StampSchema() error {
	path := s.workstreamsPath()
	line := "schema: " + strconv.Itoa(CurrentSchema)
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return mdfile.WriteAtomic(path, []byte(stampFrontmatter(string(data), line)))
}

// stampFrontmatter sets the schema line inside content's frontmatter.
func stampFrontmatter(content, line string) string {
	norm := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(norm, "\n")
	if len(lines) > 0 && strings.TrimRight(lines[0], " \t") == "---" {
		for i := 1; i < len(lines); i++ {
			t := strings.TrimRight(lines[i], " \t")
			if t == "---" {
				out := append([]string{}, lines[:i]...)
				out = append(out, line)
				out = append(out, lines[i:]...)
				return strings.Join(out, "\n")
			}
			if strings.HasPrefix(t, "schema:") {
				lines[i] = line
				return strings.Join(lines, "\n")
			}
		}
	}
	if strings.TrimSpace(norm) == "" {
		return "---\n" + line + "\n---\n"
	}
	return "---\n" + line + "\n---\n\n" + norm
}
