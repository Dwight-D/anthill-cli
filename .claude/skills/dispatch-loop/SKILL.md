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
verify evidence, record, and continue. Your durable state is the backlog,
`.anthill/escalations/`, and — while working a framed batch — the progress
ledger and control flag in `.anthill/dispatch/`; you can be killed and
respawned mid-queue with no handoff, so recycle freely.

On invocation read: `.anthill/backlog/workstreams.md`, `.anthill/backlog/
bindings.md`, `.anthill/resources.md` (parallelism posture derives from
the exclusive-resource inventory), `.anthill/dispatch/` (the progress
ledger + control flag, if present), and the `escalate` and `wake-up`
skills.

## Framed batch vs bare sweep

- **Bare sweep** — you work the ready backlog in sweep order. Position
  rehydrates from the backlog alone; no ledger. Silent operation.
- **Framed batch** — you were handed an ordered subset with a reporting
  cadence and a report-target (e.g. "these five, in order, report after
  each"). That framing lives nowhere in the backlog, so seed it into
  `.anthill/dispatch/ledger.md` (the ordered rows + the framing header)
  before starting; the ledger is then your position of record and a cold
  successor resumes the exact ordering, cadence, and report-target from it.

## Each wake-up, in order

You do not run a batch in one turn. **A wait is a turn boundary** (`wake-up`,
turn-boundary discipline): you spawn a worker and yield, and the worker's
completion re-wakes you to verify and advance. Each wake-up, in order:

1. **Wake up.** Run the wake-up protocol (`wake-up` skill): drain the mailbox
   AND the control flag `.anthill/dispatch/control.md` — `pause` → stand down,
   keep draining, do not advance until `run`; `stop` → wrap up and end. Sweep
   escalations addressed to `dispatcher` (absorb or annotate-and-raise; apply
   `answered` records you raised — unblock / re-approve / annotate the item,
   archive to LOG.md). Refresh any queue or ledger view the drain invalidated.
   Because you yield at every wait, a control order or answer lands here
   promptly instead of after the whole batch. Silent when empty.
2. **Orphan / liveness check** (first wake-up, after any recycle, and as the
   bounded-silence fallback when a worker-completion notification never
   arrives): reconcile against durable state — items claimed (`in-progress`)
   by runs no longer alive, and ledger rows stuck `in-progress` behind a
   finished or dead worker → verify from state, then close/block/unclaim. This
   is the safety net that keeps a lost completion notification from hanging the
   batch; correctness never depends on the notification arriving.
3. **Advance.** Pick the next unit: a framed batch → the first `pending`
   ledger row in order; a bare sweep → the next ready item per sweep order,
   skipping never-implicit workstreams and territory colliding with a live
   agent's (leave those ready; note the skip). Re-read the pick's live state
   via the schema owner before acting — another tier may have claimed, closed,
   or re-scoped it; on mismatch, drop it and re-select.
4. **Hand off, then yield.** Run one `dispatch` handoff: claim, mark the
   ledger row `in-progress` (framed batch), spawn the fresh worker. Then
   **end the turn — do
   not block waiting on the worker.** Serial posture: one live worker at a
   time. Parallel postures (worktrees available): up to N workers with
   disjoint territories, queueing on any exclusive verification resource's
   lease — posture and N per resources.md.
5. **On completion: verify, record, report.** Re-woken after the worker
   finishes (and after step 1's drain): verify its evidence (the `dispatch`
   verify step), close/block/escalate the item via the schema owner, and
   update the ledger row (done/blocked/escalated + a one-line pointer). If the
   batch's cadence is report-after-each, send that report now — it is this
   turn's ender. Otherwise stay silent. Message your parent unprompted **only**
   on: an escalation raised, the queue/batch running dry, or a terminal/
   systemic failure (e.g. a verification resource down, remedies exhausted). A
   framed batch's explicit reporting cadence overrides this silence default —
   the parent asked for those reports.

## Authority (absorb without escalating)

Skip/reorder within stated priorities, retry transient failures, block
items with reasons, file follow-up intake items, coordinate resource
turn-taking among your own workers, recycle workers. **Raise**: design
decisions, anything touching a never-auto boundary, destructive or
irreversible actions, systemic failures beyond the documented remedies.

## Yield vs recycle

Two cadences, not one. **Yield** — end the turn — at every wait (step 4);
cheap, same identity, and what keeps your window from growing across a batch
and your inbox drained. **Recycle** — tear down and cold-respawn — after
~15–20 items or sooner if your window grows heavy; expensive, so occasional.

On recycle, the successor rehydrates from durable state: a bare sweep from the
backlog alone; a framed batch from `.anthill/dispatch/ledger.md` (the ordering,
cadence, and report-target the backlog does not record) plus the backlog for
each row's live state. Wrap-up (voluntary or forced), in order: never leave an
item claimed (unclaim or block anything in flight), land the ledger, persist
any pending escalation record, one-line summary to your parent (items done/
blocked/escalated, queue depth remaining).
