---
name: triage
description: Triage the project backlog — dedup intake items, route them to
  workstreams, and classify them per each workstream's triage profile. Use
  when asked to triage/process the backlog, sort/prioritize proposals, or
  turn slim submissions into decided items. Classifies and proposes; it does
  NOT implement (that's dispatch).
---

# triage

General triage procedure. The project specifics live in `.anthill/backlog/`:
`workstreams.md` (workstream definitions, triage profiles, judgment signals)
and `bindings.md` (schema owner, commands, id scheme, posture). Read both on
invocation; if either is missing, the project isn't onboarded — derive them
with the user before triaging anything.

## Per intake item (`.anthill/backlog/intake/`)

1. **Dedup** against all workstream directories and the recent CHANGELOG.
   Duplicate → close with a reason. Fold narrow items under an existing
   umbrella; split a bundle into pieces with independent readiness.
2. **Route.** Assign the workstream per the definitions; the submitter's
   hint is advisory only. A general capability buried inside a specific ask
   is extracted to its own item and routed separately from the carrier.
   Fits no workstream → reject with a reason (close), never leave it to rot.
3. **Classify per the workstream's triage profile** — the profile names the
   gates that apply (value, safety, feasibility, repro…), the change types,
   and the **never-auto** types. Never-auto types cap at a REVIEW
   recommendation regardless of value or safety.
4. **Record via the schema owner** (bindings). Triage proposes: it writes
   the classification and a `disposition` recommendation; approval follows
   the posture (propose-only by default — `status` stays `idea` until a
   human approves, directly or via `expedite`).
5. **Re-scan** previously triaged items whose context changed — an approval
   can make an open sibling redundant; re-run dedup after approvals.

Finish a pass with the validation command from bindings (when available).

## Acting vs surfacing

- **Act directly** on: clear discards/rejects, routing, mechanical splits
  and consolidations.
- **Surface to the user in batches** (cliff-notes, one workstream at a
  time): any item whose disposition turns on a decision only they should
  make — an open design choice, a genuinely borderline value call, or a
  footgun where every option is a band-aid (escalate those to
  needs-investigation rather than pick a patch).
- When a decision generalizes, append it to the **Judgment signals** in
  `workstreams.md` so the next triage inherits it.

## Boundaries

- **Propose only — never implement.** Building the change is `dispatch`.
- Never-auto change types are hard caps, not suggestions.
- Deep product knowledge lives with the dispatch routes (e.g. node work →
  `create-node`); triage only classifies.
