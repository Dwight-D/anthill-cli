---
name: expedite
description: Fast-lane a set of already-decided improvements into triaged +
  approved backlog items in one pass, flagged `expedited` for batch dispatch.
  Use when the user has made the calls (in a decisions doc, a description, or
  by naming existing item ids) and wants them queued for implementation. The
  user's invocation IS the approval; an agent must never self-expedite.
---

# expedite

Triage **and approve** in one pass, for work the user has already decided.
Normal triage only recommends (items stay `status: idea` awaiting a human
greenlight); invoking expedite IS that greenlight — items land
`status: approved`, `priority: high`, marked `expedited`, ready for
`dispatch expedited`.

Read `.anthill/backlog/workstreams.md` + `.anthill/backlog/bindings.md`
first; missing → not onboarded, derive them with the user.

**Authorization boundary.** Expedite writes the human approval. It runs only
when the user invokes it (or a brief the user authored explicitly calls it).
A subagent or autonomous worker must not — that would fabricate the approval
the propose-only posture exists to require. In doubt: use `triage` and stop.

## Input

One of: a reference doc with the user's decisions (point item bodies back at
it rather than re-typing rationale), a plain description/list, or existing
item ids being approved.

## Procedure

1. **Resolve or create each item** via the schema owner (new items through
   intake). Confirm one item per decision; ask when a decision maps to
   several or none.
2. **Assign workstream + full classification with triage rigor** — expedite
   changes who approves and how fast, never the gate logic. Never-auto
   change types still classify as REVIEW-grade even though the user's
   invocation approves them, and `verify` must be a real acceptance test the
   dispatching session can hold itself to.
3. **Approve and mark**: `status: approved`, `priority: high`, the token
   `expedited` first in `note` (qualifiers after `; ` — e.g.
   `expedited; needs-spec: research X first`).
4. **Record the decision in the body** — a short dated block linking the
   source and stating the chosen option + qualifiers.
5. **Split bundles** whose halves differ in readiness: approve and mark the
   decided half; the deferred half stays `idea` with a note, out of the
   batch.
6. Validate per bindings (when available) and report the batch: ids, one
   line each, and the handoff — run `dispatch expedited` to sweep them.

## Boundaries

- Approve, don't implement — `dispatch` builds them.
- User-invoked only.
- Deferred or undecided items are never expedited.
