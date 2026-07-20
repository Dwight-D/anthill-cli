---
name: wake-up
description: The controller wake-up protocol — run at every point a
  controller-tier agent regains control (invocation, incoming message,
  loop-iteration boundary, silence-check, rehydration) before doing other
  work. Agent-only; referenced by supervisor, dispatch-loop, and escalate.
  Never a user-facing action.
---

# wake-up (controller protocol)

Harness signal channels are best-effort. Delivery is **delayed** (messages
land only between your turns), **bursty** (a long turn accumulates a
backlog that arrives all at once), and can **flatten across tiers** (a
nested agent's message may route to the top session, skipping its spawner,
leaving the deep agent unreachable by name for a reply). Any protocol that
assumes timely, ordered, correctly-routed delivery inherits these
failures.

## The lost-message invariant (binds always)

**Every cross-agent protocol must remain correct if any signal is lost,
duplicated, or delivered arbitrarily late. Signals may only shorten
latency; they may never carry an obligation that exists nowhere else.**

- **Sender side: state first, signal second.** Write the durable artifact
  (escalation record, backlog/item status, task-board state, agenda)
  *before* signaling. The signal carries a one-liner plus a pointer to the
  artifact, never the content. A message whose loss would change outcomes
  is a bug in the mechanism that sent it.
- **Receiver side: signals are hints.** A message tells you *where to
  look*, not *what is true*. Act on what the durable state says now, not on
  what the message said when it was sent.
- **Never patch delivery per-workflow.** When unreliability bites, the fix
  is this invariant and the protocol below — not a "check your messages in
  step N" line bolted onto one workflow. That patches one workflow, leaves
  every other exposed, and fires as noise on every empty iteration.

## The protocol (on EVERY wake-up, before other work)

A **wake-up** is any point where you regain control: skill invocation, an
incoming message, a loop-iteration boundary, a silence-check, rehydration
after respawn. On each, in order:

1. **Drain signals.** Read all pending messages/notifications. Take from
   each only what it points at; discard any assumption about its freshness
   or ordering.
2. **Sweep escalations.** Process `.anthill/escalations/` records addressed
   to your tier per the `escalate` skill (absorb / annotate-and-raise /
   apply answered ones).
3. **Refresh before acting.** Any pending action derived from a
   shared-state view older than this wake-up must be re-derived from the
   durable source first (re-read the queue item, the task board, the
   agenda). On mismatch, re-decide — never proceed on the stale view.
4. **Empty is silent.** This is a read, not a task. An empty drain and
   sweep produce no log line, no report, no task, no artifact. The cost is
   a mailbox read plus a directory listing.

## Loop shaping (latency only)

A loop that runs many iterations inside one long turn starves its own
inbox — the iteration-start drain is its only guaranteed read point.
Prefer many short turns over one long turn where the harness supports
re-waking; where it doesn't, keep iterations short and treat the iteration
boundary as the designated wake-up. This shapes latency only; correctness
is already guaranteed by the invariant.
