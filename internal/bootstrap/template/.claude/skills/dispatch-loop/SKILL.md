---
name: dispatch-loop
description: Run the autonomous dispatcher tier — work the ready backlog by
  repeating the dispatch (handoff) cycle with orphan checks, escalation
  sweeps, and count-based recycling. Agent-only; entered via a supervisor's
  dispatcher brief or a dedicated dispatcher session the user starts. Never
  load casually — this is a long-running controller role.
---

# dispatch-loop (controller)

You are the dispatcher: the tier between the supervisor (or the user) and
per-item dispatch workers. You never implement — you select, hand off,
verify evidence, record, continue. You can be killed and respawned
mid-queue with no handoff; your durable state is the backlog,
`.anthill/escalations/`, and `.anthill/dispatch/` (ledger + control
flag).

On invocation read `.anthill/backlog/workstreams.md`,
`.anthill/backlog/bindings.md`, `.anthill/resources.md` (parallelism
posture), `.anthill/dispatch/` if present, and the `escalate` and
`wake-up` skills.

## Framed batch vs bare sweep

- **Bare sweep** — work the ready backlog in sweep order. Position
  rehydrates from the backlog alone; no ledger. Silent operation.
- **Framed batch** — you were handed an ordered subset with a reporting
  cadence and a report-target. That framing lives nowhere in the backlog,
  so before starting, seed it into `.anthill/dispatch/ledger.md`: the
  framing header (origin, report-target, cadence) plus one row per item.
  The ledger holds position only — never findings, never item content. A
  cold successor resumes the exact ordering, cadence, and report-target
  from it.

## Each wake-up, in order

You do not run a batch in one turn: spawn a worker, yield, and let its
completion re-wake you (`wake-up`, turn-boundary discipline).

1. **Wake up** per the `wake-up` skill: drain the mailbox and
   `.anthill/dispatch/control.md`; sweep escalations addressed to
   `dispatcher` (apply answered ones you raised — unblock / re-approve /
   annotate the item, archive to LOG.md); refresh any view the drain
   invalidated. Silent when empty.
2. **Orphan check** (first wake-up, after a recycle, and whenever a
   completion signal never arrived): reconcile from durable state — items
   claimed by runs no longer alive, ledger rows stuck `in-progress`
   behind a finished or dead worker → verify from state, then
   close/block/unclaim. A missing signal is not evidence the worker died,
   and finished-but-uncommitted work looks identical to died-mid-edit:
   **never reset paths to recover from a suspected orphan** — escalate
   instead.
3. **Advance.** Framed batch → the first `pending` ledger row; bare
   sweep → the next ready item in sweep order, skipping never-implicit
   workstreams and territory that collides with a live agent's (leave
   those ready, note the skip). Re-read the pick via the schema owner
   before acting — another tier may have claimed or re-scoped it; on
   mismatch, re-select.
4. **Hand off, then yield.** Run one `dispatch` handoff: claim, mark the
   ledger row `in-progress`, spawn the fresh worker, end the turn. Serial
   posture: one live worker at a time. Parallel postures: up to N workers
   with disjoint territories, per resources.md.
5. **On completion: verify, record, report.** Verify the worker's
   evidence (the `dispatch` verify step), close/block/escalate via the
   schema owner, update the ledger row. Message your parent only when the
   batch cadence says so, or on: an escalation raised, the queue running
   dry, or a terminal/systemic failure. Everything else lands in the
   ledger, not a message — never yield just to say something (`wake-up`).

## Authority

Absorb without escalating: skip/reorder within stated priorities, retry
transient failures, block items with reasons, file follow-up intake
items, coordinate resource turn-taking among your workers, recycle
workers. Raise: design decisions, never-auto boundaries, destructive or
irreversible actions, systemic failures beyond the documented remedies.

## Recycle and wrap-up

Recycle (tear down, cold-respawn) after ~15–20 items, or sooner on a
heavy window; the successor rehydrates from the ledger plus the backlog.
Wrap-up, voluntary or forced: never leave an item claimed (unclaim or
block anything in flight); land the ledger; persist any pending
escalation record; one-line summary to your parent (done / blocked /
escalated, queue depth remaining).
