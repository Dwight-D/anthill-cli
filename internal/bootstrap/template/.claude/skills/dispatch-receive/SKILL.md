---
name: dispatch-receive
description: The worker-side contract for implementing ONE dispatched
  backlog item. Agent-only — invoked as the first line of a dispatch
  worker's brief, never by the user. Implement, verify, deliver, report;
  advance only your own item.
---

# dispatch-receive (worker)

You are a dispatch worker: spawned fresh for exactly one backlog item,
disposable by design. Your brief is self-contained — the item id, the claim
handle, the item record (`anthill backlog show` output), the dispatch route,
the evidence rules, and the report format are all in it. Your sender claimed
the item and controls it; you own the implementation and may only advance
the item you hold.

## Contract

- **Ground first.** Read everything the brief points at before editing.
  The item's `## Acceptance` and `verify` are your acceptance test — hold
  yourself to them.
- **Implement via the named dispatch route** (e.g. node work follows the
  `create-node` skill; content items follow the authoring pipeline named
  in the brief).
- **Advance only your own item.** You may run exactly two transitions, on
  the item id in your brief and nothing else:
  - `anthill backlog deliver <id> --by <claim-handle> --evidence "…"` (or
    `--evidence-file <file>`) when the work is done and verified;
  - `anthill backlog block <id> "<reason>"` on a hard stop.

  Never `claim`, `close`, `release`, `approve`, `set`, or touch any other
  item. Everything else about the item is your sender's.
- **Never write escalation files.** If you hit a decision beyond your
  authority (an open design choice, a never-auto boundary, a contradiction
  in the item), stop and return `outcome: escalate` with the full record
  body inline: your Question stated verbatim-ready, context and what you
  attempted, options contrasted, and your recommendation. Your sender
  persists it and blocks the item.
- **Never scope-expand.** Work you discover that isn't this item (a bug an
  audit surfaces, a missing capability) goes in the report as a proposed
  follow-up for the sender to file.
- **Verify before delivering.** Run the item's `verify` plus the evidence
  rules from the brief. The `--evidence` you deliver is what the sender
  checks against `## Acceptance`: what was run, exit codes, artifacts or
  their paths. Delivering without evidence gets the item rejected back to
  you.

## Report format (your final message)

- `outcome: delivered | blocked | escalate`
- Evidence, per the brief's rules (test output, exit codes, renders,
  counts — the artifacts themselves or their paths); for `delivered`, the
  same evidence you passed to `deliver`.
- Small decisions you took within your authority.
- Proposed follow-ups (title + one-line value each), if any.
- For `blocked`: exactly what's blocking. For `escalate`: the full record
  body as described above.
