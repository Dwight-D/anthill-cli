---
name: escalate
description: Raise, route, answer, or apply an escalation — a durable
  decision request traveling up the agent tiers (worker → dispatcher →
  supervisor → user). Load when you must raise a question beyond your
  authority, when a subagent returns an escalation record, when answering
  one, or when applying an answered one.
---

# escalate

Escalations are durable records in `.anthill/escalations/`, one file per
open escalation, addressed by the `to:` frontmatter field. Record format:
frontmatter (`to` / `from` / `item` (optional) / `status` / `opened`) +
sections: **Question** (verbatim, never edited), **Context & attempted
remedies** (append-only, dated blocks), **Options & recommendation**,
**Decision**, **Applied**. Lifecycle: `open → answered → applied` →
archived (one line in `LOG.md`, file deleted). Filename:
`<yyyy-mm-dd>-<slug>.md`.

The record is the truth; signals are best-effort nudges. Harness channels
can flatten — a nested agent's message may route to the top session,
skipping tiers, and a deep agent may be unreachable by name for a reply.
The `to:` field, not the signal path, determines who owns a record.

## Raising

- **Ephemeral agent** (dispatch worker, subagent): do NOT write files.
  Return your report with `outcome: escalate` and the full record body
  inline; your spawner persists it.
- **Controller tier**: write the record to `.anthill/escalations/`,
  addressed to your parent tier (worker → dispatcher → supervisor → user).
  If item-bound: block the backlog item with note `escalated: <file>`.
  Then signal your parent through your normal channel — one line + the
  record path, never the content.
- State the question with full decision context: background, the catch,
  options contrasted with implications, your recommendation with the
  reason.

## Receiving (any controller, on EVERY wake-up)

This is step 2 of the wake-up protocol (`wake-up` skill): on every wake-up
— invocation, incoming message, loop-iteration boundary, silence-check,
rehydration — sweep `.anthill/escalations/` for records addressed to you
before other work, silently when empty. For each record `to:` you: absorb it if it falls within your
authority (answer it — below); otherwise append your own assessment and
attempted remedies (dated, append-only — never edit the Question),
re-address the same file one tier up, and signal upward. If you are
signaled about a record addressed to a tier below you (flattened channel),
leave it for its owner's sweep unless it is urgent and within your
authority. Never silently ignore a record; if you decline to act, write
why.

## Answering

Append `## Decision` with the call and a one-line rationale; set
`status: answered`. Don't apply it yourself unless you also own the
underlying work.

## Applying (the tier that owns the work; mandatory)

On your next tick, for each `answered` record you raised or inherited:
carry the decision into the work (unblock / re-approve / annotate the
backlog item; whatever the decision directs), append `## Applied`, add one
line to `LOG.md`, delete the record. An `answered` record found by
anyone's sweep is actionable by the finder — answered-but-unapplied is the
failure mode to hunt.

## Hard rules

- **Lost-message invariant** (home: `wake-up` skill): the record and the
  sweeps carry the obligation; signals only shorten latency. Every hop must
  stay correct if its signal is lost, duplicated, or arbitrarily late.
- The originating **Question travels verbatim** through every tier.
- Signals carry a pointer, never the content.
- A crashed tier's open records belong to its successor.
- Chain verification: a new installation (or topology change) is tested
  with a canary — a deliberately unimplementable approved item whose
  escalation must reach the user with the Question intact.
