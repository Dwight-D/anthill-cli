---
name: expedite
description: Fast-lane a set of already-decided improvements into triaged,
  fast-tracked backlog items in one pass, pinned to the top of every queue for
  batch dispatch. Use when the user has made the calls (in a decisions doc, a
  description, or by naming existing item ids) and wants them queued for
  implementation. The user's invocation IS the approval; an agent must never
  self-expedite.
---

# expedite

Triage **and approve** in one pass, for work the user has already decided.
Normal triage only proposes (items keep `dispatch: proposed` awaiting the
user's go); invoking expedite IS that go — items land
`dispatch: fast-tracked`: approved and pinned to the top of every queue, so
the `auto` ones are swept first by `dispatch fast-tracked`.

Read `.anthill/backlog/workstreams.md` + `.anthill/backlog/bindings.md`
first; missing → not onboarded, derive them with the user.

**Authorization boundary.** Expedite writes the user's approval. It runs only
when the user invokes it (or a brief the user authored explicitly calls it).
A subagent or autonomous worker must not — that would fabricate the approval
the propose-only posture exists to require. In doubt: use `triage` and stop.

## Input

One of: a reference doc with the user's decisions (point item bodies back at
it rather than re-typing rationale), a plain description/list, or existing
item ids being approved.

## Procedure

1. **Resolve or create each item.** New items go through intake:
   `anthill backlog new --title "…" --value "…" --summary "…" --reporter user`.
   Confirm one item per decision; ask when a decision maps to several or
   none.
2. **Triage with full rigor** each item still `status: new` — expedite
   changes who approves and how fast, never the gate logic. Follow `triage`
   steps 3–5: write `## Description`, `## Acceptance`, `## Triage`, then
   `anthill backlog triage <id> workstream=<ws> summary="…" change-type=… risk=… verify="…" disposition=… [priority=…]`.
   Never-auto change types still never take `disposition: auto` even though
   the user's invocation approves them, and an `auto` item's `verify` must be
   a real acceptance test the dispatching session can hold itself to. A
   decision that still needs a spec takes `disposition: design`.
3. **Record the decision in the body** — a short dated block under
   `## Triage` linking the source and stating the chosen option +
   qualifiers.
4. **Fast-track** the decided set in one call:
   `anthill backlog fast-track <id>…`.
5. **Split bundles** whose halves differ in readiness: fast-track the
   decided half; the deferred half keeps `dispatch: proposed` with the
   reason in its `## Triage`, out of the batch.
6. `anthill backlog validate`, then report the batch: ids, one line each
   with the disposition, and the handoff — run `dispatch fast-tracked` to
   sweep the `auto` ones; `design` / `interactive` ones wait for a session
   with the user.

## Boundaries

- Approve, don't implement — `dispatch` builds them.
- User-invoked only.
- Deferred or undecided items are never fast-tracked.
