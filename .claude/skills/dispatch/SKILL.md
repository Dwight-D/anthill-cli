---
name: dispatch
description: Send ONE backlog item to a fresh worker and shepherd it to
  closure — claim, package a self-contained brief, spawn, verify the
  report's evidence, close/block/escalate. The sender side of dispatching;
  use when asked to dispatch/work an item, a workstream's next item, or
  the expedited batch. The worker side is dispatch-receive; the autonomous
  queue-working tier is dispatch-loop.
---

# dispatch (sender / handoff)

Read `.anthill/backlog/workstreams.md` (routes, evidence rules, sweep
order) and `.anthill/backlog/bindings.md` (schema-owner commands, claim
semantics) on invocation; missing → the project isn't onboarded, derive
them with the user first.

Running dispatch **is** the authorization to implement what it selects;
the backlog is otherwise propose-only. **Queue state belongs to you, the
sender**: you claim, close, block, and persist escalations. The worker
never touches the queue — that split is what makes workers disposable.

## 1. Select

- `dispatch <item>` — the named item; a human naming it is the greenlight
  (overrides the readiness gate explicitly, never silently).
- `dispatch <workstream>` — highest-priority ready item there.
- bare `dispatch` — sweep order from workstreams.md, skipping
  never-implicit workstreams.
- `dispatch expedited` — loop the full cycle over every `expedited` item,
  highest priority first; a blocked item is recorded and the sweep
  continues; end with a summary table.
- Ready = `status: approved` + non-empty `verify`. Skip items whose
  territory collides with another live agent's. State your pick and the
  one-line reason.

## 2. Claim

Atomically, via the schema owner. Conflict → skip to the next candidate.

## 3. Package and spawn

Spawn a **fresh subagent** per item with a self-contained brief:

- First line invokes the `dispatch-receive` skill (the worker contract —
  never restate it).
- The item id and full body (the claim output), plus any referenced spec
  paths.
- The workstream's **dispatch route** (e.g. node work follows
  `create-node`) and **evidence rules**, copied from workstreams.md.
- Territory: the item's footprint; co-tenancy/resource rules per
  `.anthill/resources.md` if other agents are live.
- The report format: `outcome: done | blocked | escalate`, evidence,
  decisions taken, proposed follow-ups, and for escalate the full record
  body inline.

## 4. Verify the report

Hold the report to the item's `verify` and the workstream's evidence
rules — **evidence, not assertions**. Missing or failing evidence → send
the worker back once with what's missing, or block the item with the
reason.

## 5. Close out (never leave the item claimed)

- **done** → close via the schema owner (delete + changelog line). File
  any follow-ups the worker proposed as new intake items. Codify learnings
  to their durable homes.
- **blocked** → block in place with the reason.
- **escalate** → persist the returned record to `.anthill/escalations/`
  per the `escalate` skill, block the item with `escalated: <file>`,
  signal your parent tier.

Report the outcome and stop — one item per invocation, except the
expedited sweep.

## Boundaries

- Never implement the item in this session — the fresh worker does; you
  are the sender.
- Never-auto change types reach here only with explicit human approval,
  and still follow their route's own contract.
- Classification belongs to `triage`; the queue-working tier is
  `dispatch-loop`.
