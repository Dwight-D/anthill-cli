package backlog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Dwight-D/anthill-cli/internal/mdfile"
)

// CurrentSchema is the backlog schema version this CLI writes and validates.
const CurrentSchema = 3

// ErrKeyPrefixMissing indicates workstreams.md carries no key-prefix while
// the tree holds (or is about to hold) items. The CLI maps it to exit 6.
var ErrKeyPrefixMissing = errors.New("key-prefix missing: add key-prefix: XXX to .anthill/backlog/workstreams.md or run migrate --key-prefix XXX")

// ErrKeyPrefixInvalid indicates workstreams.md carries a key-prefix that is
// not 2 to 5 uppercase ASCII letters. The CLI maps it to exit 6.
var ErrKeyPrefixInvalid = errors.New("key-prefix invalid: must be 2 to 5 uppercase letters (edit .anthill/backlog/workstreams.md)")

func (s *Store) workstreamsPath() string { return filepath.Join(s.dir(), "workstreams.md") }

// sequencePath is the high-water-mark file of allocated item numbers.
func (s *Store) sequencePath() string { return filepath.Join(s.dir(), "sequence") }

// workstreamsMeta reads the frontmatter keys the schema owner cares about. A
// missing or unparseable file yields the zero value, never an error, except
// for an I/O failure other than not-exist.
func (s *Store) workstreamsMeta() (schema int, prefix string, err error) {
	data, err := os.ReadFile(s.workstreamsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return 0, "", nil
		}
		return 0, "", err
	}
	fm, _, ferr := mdfile.Split(data)
	if ferr != nil {
		return 0, "", nil
	}
	var meta struct {
		Schema    int    `yaml:"schema"`
		KeyPrefix string `yaml:"key-prefix"`
	}
	if yaml.Unmarshal(fm, &meta) != nil {
		return 0, "", nil
	}
	return meta.Schema, strings.TrimSpace(meta.KeyPrefix), nil
}

// SchemaVersion returns the `schema` stamp in workstreams.md frontmatter, or 0
// when absent (an unstamped tree).
func (s *Store) SchemaVersion() (int, error) {
	v, _, err := s.workstreamsMeta()
	return v, err
}

// KeyPrefix returns the `key-prefix` stamp in workstreams.md frontmatter: ""
// (nil error) when absent or still the template's unfilled placeholder
// (`<PREFIX>`, any angle-bracketed value), ErrKeyPrefixInvalid when present
// but malformed.
func (s *Store) KeyPrefix() (string, error) {
	_, p, err := s.workstreamsMeta()
	if err != nil {
		return "", err
	}
	if IsPrefixPlaceholder(p) {
		return "", nil
	}
	if p != "" && !ValidPrefix(p) {
		return p, ErrKeyPrefixInvalid
	}
	return p, nil
}

// IsPrefixPlaceholder reports whether v is the framework template's unfilled
// key-prefix placeholder (an angle-bracketed token such as `<PREFIX>`).
func IsPrefixPlaceholder(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">")
}

// prefix returns the installation prefix for rendering keys, tolerating a
// missing or malformed stamp (validate reports those): a malformed prefix is
// used as written so ids still round-trip; a missing one renders bare numbers.
func (s *Store) prefix() string {
	p, err := s.KeyPrefix()
	if err != nil && !errors.Is(err, ErrKeyPrefixInvalid) {
		return ""
	}
	return p
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
	return s.stampKey("schema", strconv.Itoa(CurrentSchema))
}

// StampKeyPrefix writes `key-prefix: <prefix>` into workstreams.md frontmatter
// the same way.
func (s *Store) StampKeyPrefix(prefix string) error {
	if !ValidPrefix(prefix) {
		return invalid("key prefix %q must be 2 to 5 uppercase letters", prefix)
	}
	return s.stampKey("key-prefix", prefix)
}

// stampKey sets one scalar frontmatter key in workstreams.md.
func (s *Store) stampKey(key, value string) error {
	path := s.workstreamsPath()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return mdfile.WriteAtomic(path, []byte(stampFrontmatterKey(string(data), key, value)))
}

// stampFrontmatterKey sets `key: value` inside content's frontmatter: an
// existing line for the key is replaced in place, otherwise the line is
// appended before the closing fence; content without frontmatter gains one.
func stampFrontmatterKey(content, key, value string) string {
	line := key + ": " + value
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
			if strings.HasPrefix(t, key+":") {
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

// readSequence returns the stored high-water mark of allocated numbers; a
// missing or unparsable file reads as 0.
func (s *Store) readSequence() int {
	data, err := os.ReadFile(s.sequencePath())
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// bumpSequence raises the stored high-water mark to n when n is higher.
func (s *Store) bumpSequence(n int) error {
	if n <= s.readSequence() {
		return nil
	}
	return mdfile.WriteAtomic(s.sequencePath(), []byte(fmt.Sprintf("%d\n", n)))
}

// maxNumber returns the highest item number in use: the highest number parsed
// from any item filename across every item directory, or the sequence
// high-water mark, whichever is greater.
func (s *Store) maxNumber() (int, error) {
	dirs, err := s.itemDirs(true)
	if err != nil {
		return 0, err
	}
	high := s.readSequence()
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if n, ok := ParseFilename(e.Name()); ok && n > high {
				high = n
			}
		}
	}
	return high, nil
}

// numberTaken reports whether any item directory holds a file numbered n.
func (s *Store) numberTaken(n int) (bool, error) {
	return s.numberTakenElsewhere(n, "")
}

// numberTakenElsewhere reports whether any item directory holds a file
// numbered n other than the file at except (empty: no exception).
func (s *Store) numberTakenElsewhere(n int, except string) (bool, error) {
	dirs, err := s.itemDirs(true)
	if err != nil {
		return false, err
	}
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return false, err
		}
		for _, e := range entries {
			if m, ok := ParseFilename(e.Name()); ok && m == n && filepath.Join(d, e.Name()) != except {
				return true, nil
			}
		}
	}
	return false, nil
}
