// Package e2e_test is a black-box end-to-end suite for the anthill CLI.
//
// The suite drives a freshly built binary via os/exec and asserts on exit
// codes, stdout/stderr, parsed --json payloads, and resulting file state. It is
// STDLIB ONLY and never imports the implementation packages: everything is
// exercised through the compiled binary, exactly as an agent or script would.
//
// The contract under test is docs/CLI_INTERFACE_SPEC.md (including the user's
// binding answers in §7). Tests are written to the spec, not to any particular
// implementation; they are expected to fail until the implementation lands and
// to go green once it does.
package e2e_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binPath is the absolute path to the binary built once in TestMain.
var binPath string

func TestMain(m *testing.M) {
	code, err := buildAndRun(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

func buildAndRun(m *testing.M) (int, error) {
	dir, err := os.MkdirTemp("", "anthill-bin-")
	if err != nil {
		return 0, fmt.Errorf("mkdtemp: %w", err)
	}
	defer os.RemoveAll(dir)

	exe := "anthill"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	binPath = filepath.Join(dir, exe)

	wd, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("getwd: %w", err)
	}
	repoRoot := filepath.Dir(wd) // package dir is <repo>/test

	build := exec.Command("go", "build", "-o", binPath, "./cmd/anthill")
	build.Dir = repoRoot
	var berr bytes.Buffer
	build.Stderr = &berr
	if err := build.Run(); err != nil {
		return 0, fmt.Errorf("go build ./cmd/anthill failed: %v\n%s", err, berr.String())
	}

	return m.Run(), nil
}

// result is the captured outcome of one binary invocation.
type result struct {
	stdout string
	stderr string
	exit   int
}

// run invokes the binary with the given args, no --root injected. Use for
// version/help/unknown-command and init (which supplies its own --root).
func run(t *testing.T, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			t.Fatalf("run %v: process failed to start: %v", args, err)
		}
	}
	return result{stdout: so.String(), stderr: se.String(), exit: code}
}

// runIn invokes the binary with --root <root> prepended, so the command
// operates against a hermetic temp tree.
func runIn(t *testing.T, root string, args ...string) result {
	t.Helper()
	full := append([]string{"--root", root}, args...)
	return run(t, full...)
}

// ---- tree scaffolding --------------------------------------------------------

// mkTree builds a hermetic, well-formed .anthill/ tree under a fresh temp dir
// and returns the root (the directory that CONTAINS .anthill/). It scaffolds
// the tree directly (not via `anthill init`) so that a bug in init cannot
// cascade-fail every other command's tests; init has its own dedicated tests.
func mkTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, ".anthill")
	for _, d := range []string{
		"backlog/intake", "backlog/cli", "backlog/dev", "backlog/process",
		"backlog/bugs", "escalations",
	} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	writeRaw(t, filepath.Join(base, "backlog", "CHANGELOG.md"), "# Changelog\n")
	writeRaw(t, filepath.Join(base, "escalations", "LOG.md"), "# Escalation log\n")
	writeRaw(t, filepath.Join(base, "backlog", "workstreams.md"),
		"---\nsweep-order: bugs, cli, dev, process\nnever-implicit:\n---\n\n# Workstreams\n")
	return root
}

func writeRaw(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// ---- item fixtures -----------------------------------------------------------

// backlogDir is <root>/.anthill/backlog.
func backlogDir(root string) string { return filepath.Join(root, ".anthill", "backlog") }

// itemPath is the on-disk path of item <id> in workstream/intake dir <dir>.
func itemPath(root, dir, id string) string {
	return filepath.Join(backlogDir(root), dir, id+".md")
}

func changelogPath(root string) string { return filepath.Join(backlogDir(root), "CHANGELOG.md") }

func escalDir(root string) string { return filepath.Join(root, ".anthill", "escalations") }

func escalLogPath(root string) string { return filepath.Join(escalDir(root), "LOG.md") }

// writeItem writes a markdown item file with YAML frontmatter built from fields.
func writeItem(t *testing.T, root, dir, id string, fields map[string]string, body string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("---\n")
	for k, v := range fields {
		fmt.Fprintf(&b, "%s: %s\n", k, v)
	}
	b.WriteString("---\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	writeRaw(t, itemPath(root, dir, id), b.String())
}

// ideaFields is a minimal well-formed intake (untriaged) item.
func ideaFields(title string) map[string]string {
	return map[string]string{
		"title":  title,
		"value":  "removes a real pain",
		"status": "idea",
	}
}

// approvedFields is a well-formed triaged + approved (ready) item in workstream ws.
func approvedFields(title, ws string) map[string]string {
	return map[string]string{
		"workstream":    ws,
		"title":         title,
		"value":         "removes a real pain",
		"change-type":   "tooling",
		"risk":          "additive",
		"verify":        "go test ./...",
		"value-verdict": "ADVANCE — worth it",
		"disposition":   "AUTO",
		"status":        "approved",
		"priority":      "normal",
	}
}

// readFrontmatter parses the leading YAML frontmatter of a file into a map.
// It handles the simple `key: value` lines the CLI writes (values may be
// single/double quoted). Not a general YAML parser — sufficient for assertions.
func readFrontmatter(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	parts := strings.SplitN(s, "---", 3)
	if len(parts) < 3 {
		t.Fatalf("file %s has no frontmatter block:\n%s", path, s)
	}
	m := map[string]string{}
	for _, line := range strings.Split(parts[1], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		k := strings.TrimSpace(line[:i])
		v := strings.TrimSpace(line[i+1:])
		v = strings.Trim(v, "\"'")
		m[k] = v
	}
	return m
}

// ---- json helpers ------------------------------------------------------------

// jsonObj decodes a JSON object from s.
func jsonObj(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &m); err != nil {
		t.Fatalf("decode JSON object: %v\npayload: %q", err, s)
	}
	return m
}

// jsonArr decodes a JSON array of objects from s.
func jsonArr(t *testing.T, s string) []map[string]any {
	t.Helper()
	var a []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &a); err != nil {
		t.Fatalf("decode JSON array: %v\npayload: %q", err, s)
	}
	return a
}

// ---- assertions --------------------------------------------------------------

func wantExit(t *testing.T, r result, code int) {
	t.Helper()
	if r.exit != code {
		t.Fatalf("exit = %d, want %d\nstdout: %q\nstderr: %q", r.exit, code, r.stdout, r.stderr)
	}
}

func wantNonZero(t *testing.T, r result) {
	t.Helper()
	if r.exit == 0 {
		t.Fatalf("exit = 0, want non-zero\nstdout: %q\nstderr: %q", r.stdout, r.stderr)
	}
}

func wantContains(t *testing.T, haystack, needle, where string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("%s does not contain %q\ngot: %q", where, needle, haystack)
	}
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	t.Fatalf("stat %s: %v", path, err)
	return false
}

func wantFileGone(t *testing.T, path string) {
	t.Helper()
	if fileExists(t, path) {
		t.Fatalf("expected file to be deleted, but it exists: %s", path)
	}
}

func wantFilePresent(t *testing.T, path string) {
	t.Helper()
	if !fileExists(t, path) {
		t.Fatalf("expected file to exist: %s", path)
	}
}

func readAll(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
