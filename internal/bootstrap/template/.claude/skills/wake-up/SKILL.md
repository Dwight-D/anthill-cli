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

## Controller state model

Everything a controller relies on across turns lives in durable state outside
its window, in three categories with distinct trust rules:

- **Intent** — what the user wants (goals, directives, constraints).
  Human-owned, authoritative, changed only when the user changes it (the
  supervisor's `agenda.md`).
- **Position** — where the work is: task-board state, backlog item statuses,
  and a controller's cursor through a framed batch (a *progress ledger*).
  Volatile. Re-derived from its authoritative source on every wake-up (refresh
  before acting); never trusted from a prior turn's read.
- **Control** — a durable pause/stop/resume flag for the controller itself,
  read on every drain (see *Durable control* below).

A **progress ledger** captures the one piece of position no other durable
source records: a controller's place in a *framed* unit of work — an ordered
list of units plus its framing (report-target, reporting cadence, origin). A
cold successor rehydrates its cursor from the ledger, losing no ordering or
framing. Bare unordered queue-work needs no ledger — its position rehydrates
from the queue itself. The ledger holds position only: never a unit's content
(that lives in the unit), never findings.

## The protocol (on EVERY wake-up, before other work)

A **wake-up** is any point where you regain control: skill invocation, an
incoming message, a loop-iteration boundary, a silence-check, rehydration
after respawn. On each, in order:

1. **Drain signals.** Read all pending messages/notifications, and your tier's
   durable control flag (state, not a message — it gates whether you advance at
   all; see *Durable control*). Take from each only what it points at; discard
   any assumption about its freshness or ordering.
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

## Durable control (pause / stop)

A control order that exists only as a chat message can be lost or misrouted
(cross-tier flattening) and then never honored. So a controller's pause/stop is
durable state, not a message: the parent (or user) writes the control flag —
state first — then optionally pings. The controller reads the flag on every
wake-up drain, before selecting or advancing work, and acknowledges by landing
state: `pause` stands down without tearing down and keeps draining until `run`
returns; `stop` runs wrap-up and ends. A lost ping is harmless — the next drain
reads the flag. The flag lives in the controller's tier config home (e.g.
`.anthill/dispatch/control.md`, `.anthill/supervisor/control.md`).

## Turn-boundary discipline

Messages land only between your turns, so a turn boundary is your only
inbox-drain point. A loop that runs many iterations inside one long turn
starves its own inbox — and for a controller that spawns workers, that is a
correctness failure, not just latency: while blocked in-turn waiting on a
worker it cannot drain, so it cannot honor a pause, acknowledge its parent, or
report between units. The realized failure: a dispatcher told to work a batch
one at a time, reporting between each, ran the whole batch in a single turn and
never yielded — no reports, and pause orders that never reached it.

So a worker-spawning controller **never blocks in-turn waiting on a worker. A
wait is a turn boundary.** Land position (ledger/board) durably, end the turn,
and let the worker's completion notification re-wake you to verify and advance.
Every wait then becomes a drain point for free.

Re-wake is itself a best-effort signal — a lost completion notification must
not hang the controller. Back it with a bounded-silence fallback: on a periodic
tick (or the next drain), re-derive worker liveness from durable state (the
orphan/liveness check) and proceed. Correctness never depends on the
notification arriving; it only shortens latency.

Distinguish two cadences: **yield** (end the turn, same controller identity —
cheap, at every wait) from **recycle** (tear down and cold-respawn — expensive,
on count or a heavy window). Yield often; recycle occasionally; a cold
successor rehydrates from durable state either way.
