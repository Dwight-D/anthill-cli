# Supervisor bindings — <YOUR PROJECT>

> **PROJECT-SPECIFIC TIER — TEMPLATE.** Swap this file's content for your
> own domain. The general supervisor contract lives in
> `.claude/skills/supervisor/SKILL.md` and loads this file on invocation.
> Derive every value below with the user. Delete this quote block once done.

Project-specific tier of the supervisor mechanism. The general contract
lives in `.claude/skills/supervisor/SKILL.md` and loads this file on
invocation.

## Worker cap and exclusive resources

The inventory (each exclusive resource + its health states, remedies,
leases) and the derived caps live in **`.anthill/resources.md`** — the one
home for those facts. Current derivation: **team worker cap <N>**, dispatch
parallelism **<serial | …>**. Caps count concurrent resource consumers,
not agents.

## Evidence requirements

<What counts as evidence per deliverable class, and which are mandatory.
The supervisor verifies evidence, not assertions. Examples of the shape:>
- <Deliverable class A>: <named artifact / render / count that must attach>.
- <Code changes>: <build/compile command> exit 0.
- Backlog items: the item's `## Acceptance` met and its `verify` passing,
  delivered under its `## Evidence`.

## Silence threshold

<e.g. ~30 min> with no task transition and no message during an active
mission → peek task state, ping once, then recycle the worker.

## Real names workers get wrong

<Exact skill/command names workers must use, and known confusions to
pre-empt — workers inherit the supervisor's mistakes. Fill from your
project's real skills/commands.>

## Backlog intake

<Your intake command — by default
`anthill backlog new --title "…" --value "…" --reporter supervisor`
(workers pass `--reporter worker`), per `.anthill/backlog/bindings.md`. A
title + value is the whole ask.>
Expedite (`fast-track`) and `approve` run only on the user's explicit
instruction — never self-initiated.

## Backlog dispatch under co-tenancy

Backlog workers skip ready items colliding with another live worker's
territory — leave them ready, report the skip. Hand skipped items to a
future session in the wrap-up report rather than rushing behavior changes
at the end.
