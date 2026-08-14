---
name: submit
description: File one item into the project backlog — a friction point, missing
  tool, defect, or improvement idea — in seconds. Use whenever you hit a
  papercut mid-task, spot a bug, or want to park an idea for later. Submission
  is deliberately dumb (a title + a value is the whole ask); triage decides
  workstream, risk, and approval. Never triages, approves, or implements.
---

# submit

Raise one piece of work into the backlog and move on. This is the intake side
of the backlog — the frictionless front door any agent (or the user) uses to
capture work without stopping to design it. It is always available, in every
mode, and requires no elevated permission.

Read `.anthill/backlog/README.md` (the submission contract) and
`.anthill/backlog/bindings.md` (the intake command for this environment). If
`bindings.md` is missing the project isn't onboarded — write the item file
directly into `.anthill/backlog/intake/` following the schema in `bindings.md`,
or surface that to the user.

## The whole ask

Provide only what the submitter uniquely knows:

- **title** — one line.
- **value** — the pain it removes or the potential it unlocks.

Optionally: **source** (where it came up) and a non-binding **hint** at a
workstream. A wrong or missing hint costs nothing — triage assigns the
workstream authoritatively.

## Procedure

1. **Dedup-glance.** Scan the workstream directories under
   `.anthill/backlog/` and the recent CHANGELOG for an obvious duplicate on
   the same subsystem. Found one → add a note to that item instead of a new
   one, or skip.
2. **Submit** via the intake command in `bindings.md`
   (`<owner> backlog new --title "…" --value "…" [--source "…"] [--hint …]`),
   which lands the item in `intake/` with `status: idea`. No schema-owner CLI
   installed → write the file directly into `intake/` with frontmatter
   `title`, `value`, optional `source`/`hint`, and `status: idea`.
3. **Report** the created id in one line and continue the original task. A
   mid-task submission is a capture, not a context switch.

## Boundaries

- **Submit only.** You do NOT set risk, `change-type`, `verify`, workstream,
  or approval — those are `triage`'s job, and approval follows the propose-only
  posture. Filling them in here fabricates decisions the pipeline exists to
  make.
- **Intake only.** Never touch queue state — no `claim`, `close`, `set`, or
  status changes. Those are the dispatch tier's (`dispatch`, `dispatch-loop`),
  which owns all queue writes.
- **One item per distinct thing.** A bundle of unrelated friction is several
  submissions; triage will split a bundle anyway, so file them apart.
- If it fits no workstream, submit it regardless — triage rejects with a
  reason rather than let it rot. Scope is "improves or produces this
  project's work."
