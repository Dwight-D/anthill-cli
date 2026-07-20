# Anthill CLI — agent instructions

This repo builds the **Anthill CLI** (Go): the command-line tool that owns the
backlog and escalation schemas and the verbs the Anthill harness skills bind
to. The repo also runs the Anthill harness on itself — general-tier skills in
`.claude/skills/`, project config in `.anthill/`.

## Safety invariants (ask first / never)

The small always-on set that binds in EVERY mode, including autonomous and
supervised work. Ask rules interrupt even under bypass permissions.

- **Destructive / irreversible git & release actions** — force-push, history
  rewrite, deleting a branch, committing directly to `main`, or publishing a
  release / pushing a tag: stop and ask, with a recommendation.
- **Harness durable state** — do not delete or overwrite `.anthill/` runtime
  state (backlog items, escalation records, `agenda.md`, the changelogs/LOG)
  except through the normal close/apply flows. Bulk removal → ask.
- **Config-file ownership (user edits only)** — `CLAUDE.md` (this file),
  `.claude/settings*.json` and any permission surface, CI config, and the
  **general-tier skills `.claude/skills/*`**: these are copied verbatim from
  upstream Anthill and are immutable here — upgrade by re-copying the skill
  file, never by local edit. (The `autonomous` skill's project-specific inputs
  — its proceed-list and decisions-log path — live in `.anthill/autonomy.md`,
  specific-tier config the skill loads at invocation; editing that file is not
  a skill edit.) A framework gap is filed upstream, not patched locally; see
  `.anthill/framework.md`.
- **Schema irreversibles** — changing the backlog/escalation frontmatter schema
  or the id scheme once items exist (the CLI is the schema owner): stop and ask.

## Modes are entered via skills, never ambient

Interactive, one-request-at-a-time behavior is the baseline. Elevated modes —
autonomy (`autonomous`), supervision (`supervisor`), dispatching
(`dispatch` / `dispatch-loop`) — are opt-in contracts entered ONLY by invoking
the skill. Never assume autonomy or spawn a team uninvited. A skill cannot
self-escalate permissions; elevation is a launcher concern (`tools/supervise.*`).

## Improvement flagging is always-on

Whenever you hit friction — a missing tool, a rough process, a recurring
papercut — file it to the backlog immediately. Submission is deliberately dumb:
a **title** + a **value** (the pain it removes or potential it unlocks) is the
whole ask; triage decides the rest. Flagging is always-on in every mode;
*implementing* is opt-in via `triage`/`dispatch`. See
`.anthill/backlog/README.md` for the intake path.

## Backlog scope

Work intake lives in `.anthill/backlog/`. CLI features/commands/schema route to
`cli`; dev tooling (build, test, CI, lint) to `dev`; defects to `bugs`; docs
and process/harness config to `process`. If it fits nowhere, triage rejects it
with a reason. Evidence for done: `go build ./...` and `go test ./...` exit 0,
plus the item's own `verify`.
