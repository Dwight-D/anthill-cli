#!/bin/bash
# Launch an unattended supervisor session: bypass permissions + Remote Control
# + the supervisor skill invoked with the given mission text.
#
#   bash tools/supervise.sh Ship the X pass, then sweep the expedited batch
#
# Elevation must happen at launch — the harness forbids in-session
# escalation by design, and subagents inherit this session's mode, so the
# whole supervisor->dispatcher->worker tree runs elevated from this one flag.
#
# Smoke-test note (first run): if your CLI version treats the argument after
# --remote-control as a session NAME rather than the prompt, insert a name
# argument (e.g. "supervisor") before the prompt string.
cd "$(dirname "$0")/.." || exit 1
exec claude --permission-mode bypassPermissions --remote-control "/supervisor $*"
