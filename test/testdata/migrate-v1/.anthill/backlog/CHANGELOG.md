# Improvement Changelog

One line per completed or discarded proposal. When an item in
`.anthill/backlog/` reaches `done` (or is discarded), delete the file and add
a line here. Full detail lives in git history and in the durable homes the
work landed in. Newest first.

## Done
- 2026-11-17 anthill-claude-language-sanitization-mod-to — done: shipped in anthill-claude 993f380 + follow-up; 5/5 node tests; manual check pending (new session, type 'reasoning trail')
- 2026-11-17 supervisor-scratchpad-wording-write-the-full — done: reworded in anthill-fw supervisor skill (findings + rationale), CHANGELOG entry added
- 2026-08-29 anthill-with-no-args-opens-the-tui — done: Shipped in v0.0.7: bare anthill launches the TUI on a TTY inside an installation, help otherwise
- 2026-08-29 interactive-tui — done: Shipped in v0.0.7: anthill tui with Backlog + Escalations modes, mode registry seam, keypress actions
- 2026-08-29 allow-closing-unanswered-escalation-add-remove-cmd — done: Allow closing unanswered escalation, add remove cmd
- 2026-08-29 fix-backlog-close — done: Fix backlog close

- 2026-07-20 — **stand-up-go-build-test-harness** (dev) — go.mod (module github.com/Dwight-D/anthill-cli), minimal cobra root command + real unit tests, CI workflow. `go build ./...` / `go test ./...` exit 0 verified. Bootstraps evidence-based done. (Migrated from the superseded `anthill-cli/` installation.)

## Discarded (triaged out, not done)
