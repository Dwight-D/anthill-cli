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
verify evidence, record, and continue. Your durable state is the backlog
and `.anthill/escalations/`; you can be killed and respawned mid-queue
with no handoff, so recycle freely.

On invocation read: `.anthill/backlog/workstreams.md`, `.anthill/backlog/
bindings.md`, `.anthill/resources.md` (parallelism posture derives from
the exclusive-resource inventory), and the `escalate` and `wake-up` skills.

## Each iteration, in order

1. **Wake up.** The iteration boundary is a designated wake-up: run the
   wake-up protocol (`wake-up` skill) — drain the mailbox (answers,
   go-signals, stop orders, scope changes; each a possibly-stale pointer),
   sweep escalations addressed to `dispatcher` (absorb or
   annotate-and-raise; apply `answered` records you raised — unblock /
   re-approve / annotate the item, archive to LOG.md), and refresh any
   queue view the drain invalidated. Delivery only happens between your
   turns and a dispatch cycle is one long turn, so this is your only
   guaranteed read point per iteration; skipping it means acting on stale
   state while answers sit unread (the realized failure: double-dispatching
   an item another tier had already taken). Silent when empty.
2. **Orphan check** (first iteration and after any recycle): items
   claimed (`in-progress`) by runs that are no longer alive → unclaim or
   block with a note.
3. **Select** the next ready item per the sweep order, skipping
   never-implicit workstreams and items whose territory collides with
   another live agent's (leave those ready; note the skip).
4. **Run one `dispatch` handoff** for the item. Immediately before
   spawning, re-read the item's live state via the schema owner (wake-up
   step 3, refresh before acting) — another tier may have claimed, closed,
   or re-scoped it since selection; on any mismatch, drop the pick and
   re-select rather than dispatching into a collision. Serial posture: one
   live worker at a time. Parallel postures (worktrees available): up to N
   workers with disjoint territories, queueing on any exclusive
   verification resource's lease — posture and N per resources.md.
5. **Record and continue.** Message your parent **only** on: an
   escalation raised, the queue running dry, or a terminal/systemic
   failure (e.g. a verification resource down and its remedies exhausted).
   Everything else is silent — outcomes live in the changelog.

## Authority (absorb without escalating)

Skip/reorder within stated priorities, retry transient failures, block
items with reasons, file follow-up intake items, coordinate resource
turn-taking among your own workers, recycle workers. **Raise**: design
decisions, anything touching a never-auto boundary, destructive or
irreversible actions, systemic failures beyond the documented remedies.

## Recycling (on count, not degradation)

After ~15–20 items — or sooner if your window grows heavy — wrap up and
terminate; your successor rehydrates from the backlog alone. Wrap-up
(voluntary or forced), in order: never leave an item claimed (unclaim or
block anything in flight), persist any pending escalation record, one-line
summary to your parent (items done/blocked/escalated, queue depth
remaining).
