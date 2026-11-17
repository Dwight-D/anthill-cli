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
and `bindings.md` (schema, commands, key scheme, posture). Read both on
invocation; if either is missing, the project isn't onboarded — derive them
with the user before triaging anything.

## Disposition — the key judgment

`disposition` says how the item is expected to be implemented:

| value | meaning |
|---|---|
| `auto` | A properly described, bounded change a fresh agent can one-shot from the item alone. Requires a real `verify` (not `none`) and a change-type outside the workstream's never-auto types. Only `auto` items become **ready** for unattended dispatch (once the user approves). |
| `design` | Needs a design session (spec, decisions, maybe a prototype) before anyone implements. That session ends in a re-disposition to `auto` or `interactive`, or a split. |
| `interactive` | Best done live with the user, iterating. |
| `open` | A problem worth remembering with no clear approach yet. Never dispatched; revisited by triage. |

Choose `auto` only when the `## Description` alone would let a cold worker
finish and `verify` is a real headless test. When in doubt, `design`.

## Per intake item (`anthill backlog list --untriaged`)

1. **Dedup** against all workstream directories and the recent CHANGELOG.
   Duplicate → `anthill backlog close <key> --discard "duplicate of ANT-12"`.
   Fold narrow items under an existing umbrella; split a bundle into pieces
   with independent readiness.
2. **Route.** Assign the workstream per the definitions; the submitter's
   hint is advisory only. A general capability buried inside a specific ask
   is extracted to its own item and routed separately from the carrier.
   Fits no workstream, or not worth doing →
   `anthill backlog close <key> --discard "<reason>"`, never leave it to rot.
   Worth doing but not now → `anthill backlog park <key> "<reason>"`.
3. **Classify per the workstream's triage profile** — the profile names the
   gates that apply (value, safety, feasibility, repro…), the change types,
   and the **never-auto** types. Set `change-type`, `risk`, `verify`,
   `priority`, `disposition`, and a `summary` when the submitter gave none.
   Never-auto change types never take `disposition: auto`, regardless of
   value or safety.
4. **Write the body** (plain markdown edits of the item file):
   - `## Description` — expand the submitter's words into the self-contained
     brief a cold worker implements from: context, what to change,
     constraints, pointers.
   - `## Acceptance` — what "done" looks like, expanding `verify`; the
     evidence a controller will check.
   - `## Triage` — why this workstream, disposition, and priority; the value
     verdict in prose.
5. **Record** with one call:
   `anthill backlog triage <key> workstream=<ws> summary="…" change-type=… risk=… verify="…" disposition=… [priority=…]`.
   It moves the item into its workstream as `status: triaged`. Triage
   proposes: `dispatch` stays `proposed` until the user approves
   (`approve` / `fast-track`), directly or via `expedite`.
6. **Re-scan** already-triaged items whose context changed — an approval
   can make an open sibling redundant; re-run dedup after approvals; revisit
   `open` items and re-disposition with
   `anthill backlog set <key> disposition=…` when an approach has emerged.

Finish a pass with `anthill backlog validate`.

## Acting vs surfacing

- **Act directly** on: clear discards, routing, mechanical splits and
  consolidations.
- **Surface to the user in batches** (cliff-notes, one workstream at a
  time): any item whose disposition turns on a decision only they should
  make — an open design choice, a genuinely borderline value call, or a
  footgun where every option is a band-aid (give those `disposition: design`
  rather than pick a patch).
- When a decision generalizes, append it to the **Judgment signals** in
  `workstreams.md` so the next triage inherits it.

## Boundaries

- **Propose only — never implement.** Building the change is `dispatch`.
- **Never approve.** `approve` and `fast-track` are the user's; triage leaves
  `dispatch` as it found it.
- Never-auto change types are hard caps, not suggestions.
- Deep product knowledge lives with the dispatch routes; triage only classifies.
