---
name: dispatch
description: Send ONE backlog item to a fresh worker and shepherd it to
  closure — claim, package a self-contained brief, spawn, verify the
  delivered evidence, close/reject/block/escalate. The sender side of
  dispatching; use when asked to dispatch/work an item, a workstream's next
  item, or the fast-tracked batch. The worker side is dispatch-receive; the
  autonomous queue-working tier is dispatch-loop.
---

# dispatch (sender / handoff)

Read `.anthill/backlog/workstreams.md` (routes, evidence rules, sweep
order) and `.anthill/backlog/bindings.md` (schema, commands, ready rule) on
invocation.

Running dispatch **is** the authorization to implement what it selects;
the backlog is otherwise propose-only. **You are the controller of the
item**: you claim it, and you alone close, reject, block, release, and
persist escalations. The worker may only advance the item it holds — it
`deliver`s (and `block`s on a hard stop) its own item, nothing else.

## 1. Select

- `dispatch <item>` — the named item; a human naming it is the greenlight
  (overrides the readiness gate explicitly with `claim --force`, never
  silently).
- `dispatch <workstream>` — the highest-ranked ready item there.
- bare `dispatch` — the highest-ranked ready item in sweep order;
  never-implicit workstreams are skipped unless named.
- `dispatch fast-tracked` — loop the full cycle over
  `anthill backlog list --dispatch fast-tracked --ready`, best first; a
  blocked item is recorded and the sweep continues; end with a summary
  table.
- **Ready** = `status: triaged`, `dispatch` approved or fast-tracked,
  `disposition: auto`, and a real `verify` — the CLI derives it
  (`anthill backlog next [--workstream <ws>]` shows the pick without
  claiming). Skip items whose territory collides with another live agent's.
  State your pick and the one-line reason.

## 2. Claim

Pick the worker's name first; the claim handle is
`dispatcher:<worker-name>`. Claim atomically:

- `anthill backlog claim --next --by dispatcher:<worker-name> [--workstream <ws>]`
- `anthill backlog claim <id> --by dispatcher:<worker-name> [--force]`

Conflict (exit 5) → another tier took it; skip to the next candidate. Then
`anthill backlog show <id>` — its output goes into the brief.

## 3. Package and spawn

Spawn a **fresh subagent** per item with a self-contained brief:

- First line invokes the `dispatch-receive` skill (the worker contract —
  never restate it).
- The item id, the claim handle (`dispatcher:<worker-name>`), and the full
  `anthill backlog show <id>` output, plus any referenced spec paths.
- The workstream's **dispatch route** (e.g. node work follows
  `create-node`) and **evidence rules**, copied from workstreams.md.
- Territory: the item's footprint; co-tenancy/resource rules per
  `.anthill/resources.md` if other agents are live.
- The report format: `outcome: delivered | blocked | escalate`, evidence,
  decisions taken, proposed follow-ups, and for escalate the full record
  body inline.

## 4. Verify the delivery

On `outcome: delivered` the item sits in `status: validation` with the
worker's evidence under `## Evidence`. Re-read it with
`anthill backlog show <id>` and hold that evidence to the item's `## Acceptance`, its `verify`,
and the workstream's evidence rules — **evidence, not assertions**. Re-run
`verify` yourself when it is cheap.

## 5. Close out (never leave the item claimed)

- **verified** → `anthill backlog close <id> --done "<one line>"` (archives
  to `done/`, CHANGELOG line). File any follow-ups the worker proposed as new
  intake items (`submit`, `--reporter worker`). Codify learnings to their
  durable homes.
- **evidence missing or failing** →
  `anthill backlog close <id> --reject "<what is missing>"` (back to
  `in-progress`, same claim) and send the worker back once; a second
  failure → `anthill backlog block <id> "<reason>"`.
- **blocked** → the worker blocked it already, or
  `anthill backlog block <id> "<reason>"` from its report.
- **escalate** → persist the returned record to `.anthill/escalations/`
  per the `escalate` skill,
  `anthill backlog block <id> "escalated: <file>"`, signal your parent tier.
- **worker gone, nothing delivered** → `anthill backlog release <id>` (back
  to `triaged`) when no work landed; otherwise block with the reason.

Report the outcome and stop — one item per invocation, except the
fast-tracked sweep.

## Boundaries

- Never implement the item in this session — the fresh worker does; you
  are the sender.
- Never-auto change types are never `auto`, so they reach here only by a
  human naming them, and still follow their route's own contract.
- Classification belongs to `triage`; the queue-working tier is
  `dispatch-loop`.
