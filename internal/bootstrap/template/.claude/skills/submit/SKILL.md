---
name: submit
description: File one item into the project backlog — a friction point, missing
  tool, defect, or improvement idea — in seconds. Use whenever you hit a
  papercut mid-task, spot a bug, or want to park an idea for later. Submission
  is deliberately dumb (a title + a value is the whole ask); triage decides
  workstream, risk, and disposition. Never triages, approves, or implements.
---

# submit

Raise one piece of work into the backlog and move on. This is the intake side
of the backlog — the frictionless front door any agent (or the user) uses to
capture work without stopping to design it. It is always available, in every
mode, and requires no elevated permission.

Read `.anthill/backlog/README.md` (the submission contract) and
`.anthill/backlog/bindings.md` (the schema and commands for this environment).
If `bindings.md` is missing the project isn't onboarded — surface that to the
user.

## The whole ask

Provide only what the submitter uniquely knows:

- **title** — one line.
- **value** — the pain it removes or the potential it unlocks.

Optionally: a **summary** (one or two sentences: what the change is), a
**source** (where it came up), and a non-binding **hint** at a workstream. A
wrong or missing hint costs nothing — triage assigns the workstream
authoritatively, and writes the summary when the submitter cannot.

Always: the **reporter** — who filed it, per the tier you run in:

- `user` — an interactive session filing at the user's request (the user
  filed it; the agent typed it), or the user themself.
- `supervisor` — a supervisor session filing on its own initiative.
- `worker` — any worker: a supervisor's worker, a dispatch worker, an
  autonomous worker.

## Procedure

1. **Dedup-glance.** Scan the workstream directories under
   `.anthill/backlog/` and the recent CHANGELOG for an obvious duplicate on
   the same subsystem. Found one → add a line to that item's
   `## Description` instead of a new one, or skip.
2. **Submit** via the intake command:
   `anthill backlog new --title "…" --value "…" --reporter <tier> [--summary "…"] [--source "…"] [--hint <ws>]`.
   It lands the item in `intake/` with `status: new`, `dispatch: proposed`,
   and the body skeleton. No `anthill` installed → write the file directly
   into `intake/` following the schema in `bindings.md` (`title`, `value`,
   `reporter`, optional `summary`/`source`/`hint`, `status: new`,
   `dispatch: proposed`).
3. **Report** the created id in one line and continue the original task. A
   mid-task submission is a capture, not a context switch.

## Boundaries

- **Submit only.** You do NOT set risk, `change-type`, `verify`,
  `disposition`, workstream, or dispatch approval — those are `triage`'s job,
  and approval is the user's. Filling them in here fabricates decisions the
  pipeline exists to make.
- **Intake only.** Never move an item's state — no `claim`, `deliver`,
  `close`, `set`, or any other transition. Item state belongs to the
  controllers (`dispatch`, `dispatch-loop`, `supervisor`) and to the worker
  holding a claimed item.
- **One item per distinct thing.** A bundle of unrelated friction is several
  submissions; triage will split a bundle anyway, so file them apart.
- If it fits no workstream, submit it regardless — triage discards with a
  reason rather than let it rot. Scope is "improves or produces this
  project's work."
