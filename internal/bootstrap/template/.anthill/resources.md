# Exclusive resources — <YOUR PROJECT>

> **PROJECT-SPECIFIC TIER — TEMPLATE.** Swap this file's content for your
> own domain. It is the ONE home for your project's exclusive-resource
> inventory; the supervisor bindings (worker-cap derivation, arbitration)
> and the dispatch tier (parallelism posture) both read it. Keep each fact
> here and nowhere else. Delete this quote block once derived.

The one home for the project's exclusive-resource inventory. Consumed by
the supervisor bindings (worker cap derivation, arbitration) and the
dispatch tier (parallelism posture).

An **exclusive resource** is any stateful thing two agents cannot use at
once without corrupting each other or the work: a shared checkout, a dev
server, an editor/IDE instance, a device, a database, a license seat. List
each one, its health states + remedies, and its lease/turn-taking rule.
Then derive the caps.

<!-- WORKED EXAMPLE (from the Nodachi template repo) — the SHAPE of an
     adaptation, not content to copy. Your resources will be different.

  ## Git checkout (shared, no worktree isolation in use)
  One shared integration branch for concurrent agents; never main.
  Path-scoped staging only (never `git add -A`), task-id-tagged commits,
  push after each task-closing commit; fetch+retry on races, never force.

  ## Unity Editor (single instance)
  Turn-taking on editor automation. Health: `nodev editor health` →
  healthy | busy | frozen | down.
  - BUSY (importing/compiling, responding): wait; focusing pumps it through.
  - Server-silent but idle: `nodev editor revive --holder <name>`.
  - DOWN: `nodev editor launch --holder <name>`.
  - FROZEN (not responding, sustained): `nodev editor kill …` then launch —
    killing stays ask-first.
  All launch/kill/revive are lease-gated.
-->

## <Resource 1 — e.g. shared checkout / build server / editor / device>

<Describe it. What makes it exclusive. The discipline for sharing it
(branch model, staging scope, lease/turn-taking rule).>

Health states and remedies (agents diagnose before escalating):
- <STATE> → <remedy / command>
- <STATE> → <remedy / command>

## <Resource 2 — if any>

<...>

## Derived caps

Caps count **concurrent resource consumers, not agents** — a serial
controller plus its one live worker is one consumer.

- **Team worker cap: <N>.** Derivation: <which resources constrain it +
  supervisor attention>. Re-derive whenever the environment changes (e.g.
  worktree isolation would raise it).
- **Dispatch parallelism: <serial | parallel-implement/serialized-verify |
  fully parallel>.** Derivation: <based on the resources above — worktree
  isolation? an exclusive verification resource?>.
