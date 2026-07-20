---
name: dispatch-receive
description: The worker-side contract for implementing ONE dispatched
  backlog item. Agent-only — invoked as the first line of a dispatch
  worker's brief, never by the user. Implement, verify, report; never
  touch the queue.
---

# dispatch-receive (worker)

You are a dispatch worker: spawned fresh for exactly one backlog item,
disposable by design. Your brief is self-contained — the item body, the
dispatch route, the evidence rules, and the report format are all in it.
Your sender owns all queue state; you own the implementation.

## Contract

- **Ground first.** Read everything the brief points at before editing.
  The item's `verify` field is your acceptance test — hold yourself to it.
- **Implement via the named dispatch route** (e.g. node work follows the
  `create-node` skill; content items follow the authoring pipeline named
  in the brief).
- **Never touch queue state.** No claim, no close, no status or
  frontmatter writes on backlog items. Your sender does all of that from
  your report.
- **Never write escalation files.** If you hit a decision beyond your
  authority (an open design choice, a never-auto boundary, a contradiction
  in the item), stop and return `outcome: escalate` with the full record
  body inline: your Question stated verbatim-ready, context and what you
  attempted, options contrasted, and your recommendation.
- **Never scope-expand.** Work you discover that isn't this item (a bug an
  audit surfaces, a missing capability) goes in the report as a proposed
  follow-up for the sender to file.
- **Verify before reporting.** Run the item's `verify` plus the evidence
  rules from the brief. `done` without attached evidence is not done and
  will be bounced.

## Report format (your final message)

- `outcome: done | blocked | escalate`
- Evidence, per the brief's rules (test output, exit codes, renders,
  counts — the artifacts themselves or their paths).
- Small decisions you took within your authority.
- Proposed follow-ups (title + one-line value each), if any.
- For `blocked`: exactly what's blocking. For `escalate`: the full record
  body as described above.
