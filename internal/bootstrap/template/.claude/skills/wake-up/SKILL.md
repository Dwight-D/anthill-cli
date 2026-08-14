---
name: wake-up
description: The controller wake-up protocol — run at every point a
  controller-tier agent regains control (invocation, incoming message,
  loop-iteration boundary, silence-check, rehydration) before doing other
  work. Agent-only; referenced by supervisor, dispatch-loop, and escalate.
  Never a user-facing action.
---

# wake-up (controller protocol)

Harness messages are best-effort: they arrive late (only between your
turns), in bursts, and sometimes at the wrong tier (a nested agent's
message may land in the top session instead of its spawner). No protocol
may depend on delivery.

## The lost-message invariant (binds always)

**A protocol must stay correct if any signal is lost, duplicated, or
arbitrarily late. Signals may shorten latency; they may never carry an
obligation that exists nowhere else.**

- **Sender: state first, signal second.** Write the durable artifact
  (escalation record, item status, task state) before signaling. The
  signal is a pointer to the artifact, never the content.
- **Receiver: signals are hints.** A message tells you where to look, not
  what is true. Act on what durable state says now.
- **Absence of a signal is not information.** A worker that hasn't
  reported may be alive, finished, or dead — read durable state to find
  out, and never take destructive recovery action (e.g. resetting paths)
  on the strength of a missing or ambiguous signal.
- When unreliability bites, the fix is this invariant — not a "check your
  messages" step bolted onto one workflow.

## The protocol (on EVERY wake-up, before other work)

A wake-up is any point you regain control: invocation, an incoming
message, a loop-iteration boundary, a silence check, rehydration after a
respawn. Each time, in order:

1. **Drain.** Read pending messages and your tier's durable control flag.
2. **Sweep escalations.** Handle `.anthill/escalations/` records addressed
   to your tier per the `escalate` skill.
3. **Refresh.** Re-read any shared state you are about to act on; if it
   no longer matches what you remembered, re-decide.
4. **Empty is silent.** No log line, no report. The cost is a mailbox
   read and a directory listing.

## Durable control (pause / stop)

A pause or stop that exists only as a chat message can be lost and never
honored, so it is durable state: the parent (or user) writes the tier's
control flag (e.g. `.anthill/dispatch/control.md`,
`.anthill/supervisor/control.md`), then optionally pings. The controller
reads the flag on every drain: `pause` → stand down but keep draining
until `run` returns; `stop` → wrap up and end. A missing flag file means
`run`.

## Turn-boundary discipline

Messages land only between turns, so a controller that stays in one long
turn starves its own inbox — it cannot receive a pause order, an answer,
or a report while it works. Yield by cause, not cadence:

- **Yield when you are out of work** — typically the moment you have
  spawned a worker and can only wait. Never block in-turn waiting on a
  worker: a wait IS a turn boundary, and taking it drains your inbox for
  free. Land your position durably first, so your re-woken self (or a
  successor) picks up from state.
- **Never yield merely to say something.** Acknowledgements, decision
  notes, and status belong in your durable files, where the parent reads
  them without a turn being spent. Yielding twice with no durable state
  change in between is bookkeeping, not waiting.

A lost completion signal must not hang you: on the next drain or silence
check, re-derive worker status from durable state and proceed.

**Yield** (end the turn, same identity) is cheap — take it at every wait.
**Recycle** (tear down, cold-respawn) is expensive — do it on count or a
heavy window. Either way, a successor rehydrates from durable state.
