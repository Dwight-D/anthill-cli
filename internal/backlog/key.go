package backlog

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ErrInvalidID indicates an item reference that is neither a key, a bare
// number, nor a former id (ITEM_KEYS_SPEC.md §1). The CLI maps it to exit 2.
var ErrInvalidID = errors.New("invalid item id")

// Key is an item's id: an installation prefix and a sequential number
// (`ANT-74`). The prefix lives in workstreams.md (`key-prefix`); the number is
// allocated by Store.New and never reused.
type Key struct {
	Prefix string
	Number int
}

// String renders the canonical form, `ANT-74`. A key without a prefix (an
// installation whose workstreams.md lacks key-prefix) renders as the bare
// number, which every verb still resolves.
func (k Key) String() string {
	if k.Prefix == "" {
		return strconv.Itoa(k.Number)
	}
	return k.Prefix + "-" + strconv.Itoa(k.Number)
}

// FileStem is the zero-padded number that opens the item's filename: `0074`.
func (k Key) FileStem() string { return fmt.Sprintf("%04d", k.Number) }

// Less orders keys by number.
func (k Key) Less(o Key) bool { return k.Number < o.Number }

// Ref is a parsed item reference: either a Key or a former id (the pre-key
// slug recorded in the item's `former-id` field). Exactly one is set.
type Ref struct {
	Key    Key
	Former string
}

// IsKey reports whether the reference is a key (else a former id).
func (r Ref) IsKey() bool { return r.Former == "" }

var (
	prefixRe   = regexp.MustCompile(`^[A-Z]{2,5}$`)
	numberRe   = regexp.MustCompile(`^[1-9][0-9]*$`)
	formerRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,49}$`)
	filenameRe = regexp.MustCompile(`^([0-9]{4,})-([a-z0-9-]+)\.md$`)
)

// ValidPrefix reports whether p is a legal key prefix: 2 to 5 uppercase ASCII
// letters.
func ValidPrefix(p string) bool { return prefixRe.MatchString(p) }

// ParseKey parses a key reference: the canonical key (case-insensitive) or a
// bare number resolved with prefix. When prefix is empty any well-formed
// prefix is accepted. A former id is not a key; use ParseRef for the full
// accepted-input set.
func ParseKey(input, prefix string) (Key, error) {
	in := strings.TrimSpace(input)
	if numberRe.MatchString(in) {
		n, err := strconv.Atoi(in)
		if err != nil {
			return Key{}, invalidID(input)
		}
		return Key{Prefix: prefix, Number: n}, nil
	}
	p, num, ok := strings.Cut(in, "-")
	if !ok || !numberRe.MatchString(num) {
		return Key{}, invalidID(input)
	}
	up := strings.ToUpper(p)
	if !prefixRe.MatchString(up) {
		return Key{}, invalidID(input)
	}
	if prefix != "" && !strings.EqualFold(up, prefix) {
		return Key{}, invalidID(input)
	}
	n, err := strconv.Atoi(num)
	if err != nil {
		return Key{}, invalidID(input)
	}
	if prefix == "" {
		prefix = up
	}
	return Key{Prefix: prefix, Number: n}, nil
}

// ParseRef parses any accepted item reference (§1): a key, a bare number, or a
// former id (exact, lowercase). Anything else is ErrInvalidID.
func ParseRef(input, prefix string) (Ref, error) {
	if k, err := ParseKey(input, prefix); err == nil {
		return Ref{Key: k}, nil
	}
	in := strings.TrimSpace(input)
	if formerRe.MatchString(in) && strings.Trim(in, "0123456789") != "" {
		return Ref{Former: in}, nil
	}
	return Ref{}, invalidID(input)
}

func invalidID(input string) error {
	return fmt.Errorf("%w %q: expected ANT-74, 74, or a former id", ErrInvalidID, input)
}

// ParseFilename extracts the item number from an item filename of the form
// `NNNN-<slug>.md`. ok is false for any other name.
func ParseFilename(name string) (number int, ok bool) {
	m := filenameRe.FindStringSubmatch(name)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// IsItemFilename reports whether name has the `NNNN-<slug>.md` form.
func IsItemFilename(name string) bool {
	_, ok := ParseFilename(name)
	return ok
}

// FormatFilename builds an item filename from its number and title slug. An
// empty slug writes `NNNN-item.md`.
func FormatFilename(number int, slug string) string {
	if slug == "" {
		slug = "item"
	}
	return fmt.Sprintf("%04d-%s.md", number, slug)
}

// keyFromPath derives the item's key from its file path and the installation
// prefix; ok is false when the filename is not in item form.
func keyFromPath(path, prefix string) (Key, bool) {
	n, ok := ParseFilename(filepath.Base(path))
	if !ok {
		return Key{}, false
	}
	return Key{Prefix: prefix, Number: n}, true
}

// DefaultKeyPrefix derives an installation's key prefix from its root
// directory name: the first three ASCII letters, uppercased; fewer than two
// letters yield "ANT".
func DefaultKeyPrefix(root string) string {
	var b strings.Builder
	for _, r := range filepath.Base(filepath.Clean(root)) {
		if r < 128 && unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
			if b.Len() == 3 {
				break
			}
		}
	}
	if b.Len() < 2 {
		return "ANT"
	}
	return b.String()
}
