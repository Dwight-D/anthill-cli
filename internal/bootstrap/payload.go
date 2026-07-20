package bootstrap

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// PayloadFiles returns every regular file in the embedded template payload as a
// sorted list of slash-separated paths rooted at the payload top (e.g.
// ".claude/skills/autonomous/SKILL.md", "CLAUDE.template.md"). Directories are
// not included; empty payload directories are represented by their .gitkeep /
// .gitignore marker files.
func PayloadFiles() ([]string, error) {
	tfs := TemplateFS()
	var files []string
	err := fs.WalkDir(tfs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// PayloadDirs returns every directory in the embedded payload (excluding the
// "." root) as sorted slash paths. Used by doctor's structure check.
func PayloadDirs() ([]string, error) {
	tfs := TemplateFS()
	var dirs []string
	err := fs.WalkDir(tfs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != "." {
			dirs = append(dirs, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs)
	return dirs, nil
}

// ReadTemplateFile returns the pristine bytes of the payload file at slash-path
// p (rooted at the payload top).
func ReadTemplateFile(p string) ([]byte, error) {
	return fs.ReadFile(TemplateFS(), p)
}

// SkillFiles returns the payload files that belong to the general-tier skills
// (anything under ".claude/skills/"), sorted.
func SkillFiles() ([]string, error) {
	all, err := PayloadFiles()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range all {
		if strings.HasPrefix(p, ".claude/skills/") {
			out = append(out, p)
		}
	}
	return out, nil
}

// SkillNameOf returns the skill name for a payload path under
// ".claude/skills/<name>/...", or "" if the path is not a skill file.
func SkillNameOf(p string) string {
	const prefix = ".claude/skills/"
	if !strings.HasPrefix(p, prefix) {
		return ""
	}
	rest := p[len(prefix):]
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		return rest[:i]
	}
	return rest
}

// SkillNames returns the sorted set of general-tier skill names in the payload.
func SkillNames() ([]string, error) {
	files, err := SkillFiles()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	for _, p := range files {
		if n := SkillNameOf(p); n != "" && !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names, nil
}

// InsideGitRepo reports whether dir (or any ancestor) contains a ".git" entry
// (a directory for a normal repo, or a file for a worktree/submodule).
func InsideGitRepo(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	for {
		if _, err := os.Stat(filepath.Join(abs, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return false
		}
		abs = parent
	}
}

// filesEqual reports whether two byte slices are identical.
func filesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// atomicWrite writes data to path via a temp file + rename in the same
// directory, creating parent directories as needed. On Windows, rename fails if
// the destination exists, so an existing target is removed first.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".anthill-scaffold-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Best-effort mode set (a no-op meaningful only on Unix).
	_ = os.Chmod(tmpName, perm)
	if runtime.GOOS == "windows" {
		_ = os.Remove(path)
	}
	return os.Rename(tmpName, path)
}

// fileModeFor picks the on-disk mode for a scaffolded payload path: launcher
// shell scripts are executable, everything else is a normal file.
func fileModeFor(slashPath string) os.FileMode {
	if strings.HasPrefix(slashPath, "tools/") && strings.HasSuffix(slashPath, ".sh") {
		return 0o755
	}
	return 0o644
}
