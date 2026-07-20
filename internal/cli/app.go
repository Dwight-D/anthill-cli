package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Dwight-D/anthill-cli/internal/backlog"
	"github.com/Dwight-D/anthill-cli/internal/escalation"
)

// App holds global flags and the output streams shared by every command.
type App struct {
	rootFlag string
	json     bool
	quiet    bool
	noColor  bool

	out io.Writer // stdout: the answer (data / --json payloads)
	err io.Writer // stderr: progress, notes, errors
}

func newApp(out, errw io.Writer) *App {
	return &App{out: out, err: errw}
}

// resolveRoot returns the directory containing .anthill: the --root flag, then
// ANTHILL_ROOT, then a git-style walk up from CWD.
func (a *App) resolveRoot() (string, error) {
	if a.rootFlag != "" {
		return a.rootFlag, nil
	}
	if env := os.Getenv("ANTHILL_ROOT"); env != "" {
		return env, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", internalErr(err.Error())
	}
	dir := cwd
	for {
		if info, err := os.Stat(filepath.Join(dir, ".anthill")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", &Error{Exit: 4, Code: "not_found",
				Message: "no .anthill/ directory found from CWD upward; pass --root or set ANTHILL_ROOT"}
		}
		dir = parent
	}
}

// backlogStore resolves the root and returns a backlog store.
func (a *App) backlogStore() (*backlog.Store, error) {
	root, err := a.resolveRoot()
	if err != nil {
		return nil, err
	}
	return backlog.NewStore(root), nil
}

// escalationStore resolves the root and returns an escalation store.
func (a *App) escalationStore() (*escalation.Store, error) {
	root, err := a.resolveRoot()
	if err != nil {
		return nil, err
	}
	return escalation.NewStore(root), nil
}

// note writes a non-essential progress/confirmation line to stderr (suppressed
// under --quiet).
func (a *App) note(format string, args ...any) {
	if a.quiet {
		return
	}
	fmt.Fprintf(a.err, format+"\n", args...)
}

// answer writes a line of the requested human answer to stdout.
func (a *App) answer(format string, args ...any) {
	fmt.Fprintf(a.out, format+"\n", args...)
}

// emitJSON writes an indented JSON payload to stdout.
func (a *App) emitJSON(v any) error {
	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return internalErr(err.Error())
	}
	return nil
}

// reportError writes an error to stderr: a structured JSON object under --json,
// otherwise a plain "error: <message>" line.
func (a *App) reportError(e *Error) {
	if a.json {
		obj := map[string]any{
			"error": map[string]any{
				"code":    e.Code,
				"exit":    e.Exit,
				"message": e.Message,
			},
		}
		if e.ID != "" {
			obj["error"].(map[string]any)["id"] = e.ID
		}
		b, _ := json.Marshal(obj)
		fmt.Fprintln(a.err, string(b))
		return
	}
	fmt.Fprintf(a.err, "error: %s\n", e.Message)
}

// exitCode maps the error returned by root.Execute() to a process exit code,
// reporting it to stderr.
func (a *App) exitCode(err error) int {
	if err == nil {
		return 0
	}
	var e *Error
	if errors.As(err, &e) {
		a.reportError(e)
		return e.Exit
	}
	// Cobra's own errors (unknown flag/command, missing required arg) are usage.
	ue := usageErr(err.Error())
	a.reportError(ue)
	return ue.Exit
}
