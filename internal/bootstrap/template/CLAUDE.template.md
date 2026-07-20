# <YOUR PROJECT> — agent instructions (TEMPLATE)

> **Rename to `CLAUDE.md` and derive with the user.** This is the always-on
> instruction file — it loads into every session, so keep it small. Reference
> material belongs in scoped homes (skills, `.anthill/` config, docs), not
> here. Only the Anthill-relevant minimum is templated below; add your
> project's own always-on content around it. Delete this quote block once done.

## Safety invariants (ask first / never)

The small always-on set that binds in EVERY mode, including autonomous and
supervised work. Derive per project — the categories:

- **Destructive / irreversible actions** — <e.g. deleting data, force-push,
  dropping/altering a database, publishing/releasing, sending outward-facing
  communication>: stop and ask, with a recommendation.
- **Config-file ownership** — <which files only the user edits: CI config,
  permission settings, this file, release manifests>.
- **<Domain-specific irreversibles>** — <anything in this project that can't
  be cheaply undone>.

Ask rules interrupt even under bypass permissions; these are the tier that
always stops.

## Modes are entered via skills, never ambient

Interactive, one-request-at-a-time behavior is the baseline. Elevated modes —
autonomy (`autonomous`), supervision (`supervisor`), dispatching
(`dispatch` / `dispatch-loop`) — are opt-in contracts entered ONLY by invoking
the skill. Never assume autonomy or spawn a team uninvited.

## Improvement flagging is always-on

Whenever you hit friction — a missing tool, a rough process, a recurring
papercut — file it to the backlog immediately. Submission is deliberately
dumb: a **title** + a **value** (the pain it removes or potential it unlocks)
is the whole ask; triage decides the rest. Flagging is always-on in every
mode; *implementing* is opt-in via `triage`/`dispatch`. See
`.anthill/backlog/README.md` for the intake command.

## Backlog scope

Work intake lives in `.anthill/backlog/`. Improvements and defects route to
the tool/process/bug workstreams; artifacts the project produces route to the
product (or a dedicated authoring) workstream. If it fits nowhere, triage
rejects it with a reason.
