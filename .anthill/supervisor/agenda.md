# Agenda

Updated: 2026-07-20 (install / derivation session)

> The supervisor's survival file: user intent ONLY — goals, directives,
> priorities, constraints as stated by the user. No findings, no progress, no
> reasoning (recreate those from the task board, git log, and decision log).
> Update it whenever user intent changes, before acting on the change.

## Standing goals

- Build the **Anthill CLI**: the command-line tool that owns the backlog and
  escalation schemas and provides the verbs the Anthill skills bind to
  (`backlog new/list/set/next/claim/close/validate`, escalation verbs, id
  generation, frontmatter invariant-checking). Written in Go.

## Directives

- 2026-07-20 — Installed the Anthill harness into this repo (dogfooding: the
  CLI being built is this harness's future schema owner). Posture is
  **propose-only** — triage recommends, the user approves.

## Constraints

- Single shared git checkout, no worktree isolation → team worker cap 2,
  dispatch serial (see `.anthill/resources.md`).
- No headless verification exists yet; standing up the Go build+test harness is
  the first `dev` backlog item and gates evidence-based done for everything else.
