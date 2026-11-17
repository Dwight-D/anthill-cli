package backlog

import (
	"os"
	"os/user"
	"strings"
	"time"
)

// now is the clock used for every stamp; tests may replace it.
var now = func() time.Time { return time.Now().UTC() }

// Stamp returns the current time as an RFC3339 UTC string.
func Stamp() string { return now().Format(time.RFC3339) }

// Environment variables consulted for identity (§4, §5).
const (
	EnvSession  = "CLAUDE_CODE_SESSION_ID"
	EnvActor    = "ANTHILL_ACTOR"
	EnvReporter = "ANTHILL_REPORTER"
	EnvClaude   = "CLAUDECODE"
)

// ResolveReporter applies the §4 precedence: explicit flag, ANTHILL_REPORTER,
// then inference (CLAUDECODE set → worker, else user).
func ResolveReporter(flag string) string {
	if flag != "" {
		return flag
	}
	if v := strings.TrimSpace(os.Getenv(EnvReporter)); v != "" {
		return v
	}
	if os.Getenv(EnvClaude) != "" {
		return "worker"
	}
	return "user"
}

// ResolveSession applies the §5 precedence: explicit flag, then
// CLAUDE_CODE_SESSION_ID, else empty.
func ResolveSession(flag string) string {
	if flag != "" {
		return flag
	}
	return strings.TrimSpace(os.Getenv(EnvSession))
}

// ResolveActor applies the §5 claimed-by precedence: explicit handle,
// ANTHILL_ACTOR, `<role>:<session prefix>` when a harness session is known,
// else the OS user name.
func ResolveActor(by string) string {
	if by != "" {
		return by
	}
	if v := strings.TrimSpace(os.Getenv(EnvActor)); v != "" {
		return v
	}
	if sid := strings.TrimSpace(os.Getenv(EnvSession)); sid != "" {
		return ResolveReporter("") + ":" + shortID(sid)
	}
	if u, err := user.Current(); err == nil && u.Username != "" {
		name := u.Username
		if i := strings.LastIndexAny(name, `\/`); i >= 0 {
			name = name[i+1:]
		}
		return name
	}
	return "unknown"
}

// shortID is the 8-character prefix of a session id.
func shortID(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
