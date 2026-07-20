# Exclusive resources — Anthill CLI

The one home for the project's exclusive-resource inventory. Consumed by the
supervisor bindings (worker cap derivation, arbitration) and the dispatch tier
(parallelism posture). Keep each fact here and nowhere else.

An **exclusive resource** is any stateful thing two agents cannot use at once
without corrupting each other or the work. This project runs on a single dev
machine building a Go CLI; its one exclusive resource is the shared working
tree.

## Git checkout (shared, no worktree isolation)

One shared working tree and integration branch for concurrent agents; never
commit directly to `main`. Sharing discipline:

- **Branch model:** a shared integration/work branch; feature branches merge
  into it. `main` is release-only.
- **Staging scope:** path-scoped staging only — never `git add -A`. Each
  worker stages only the paths in its territory.
- **Commits:** tag commits with the task/item id; commit at every working
  state; push after each task-closing commit.
- **Races:** on push rejection, `git fetch` + rebase/retry; **never force-push**
  (force-push is an ask-first action per CLAUDE.md).

Health states and remedies (agents diagnose before escalating):
- **Dirty tree from another worker's in-flight paths** → treat as co-tenancy
  friction, not a blocker: stage only your own paths, report the overlap.
- **Push rejected (non-fast-forward)** → fetch + rebase your task-id commits,
  retry; escalate only if the rebase conflicts on your own territory.
- **Detached / wrong branch** → checkout the integration branch before work.

## Derived caps

Caps count **concurrent resource consumers, not agents** — a serial controller
plus its one live worker is one consumer.

- **Team worker cap: 2.** Derivation: a single shared checkout with no worktree
  isolation tolerates limited co-tenancy only when workers hold disjoint path
  territories; beyond two concurrent writers the merge/overlap risk and the
  supervisor's attention both degrade. Re-derive upward if git worktree
  isolation is adopted (each worker gets its own tree → the checkout stops
  being exclusive).
- **Dispatch parallelism: serial.** Derivation: one shared checkout, no
  worktree isolation, and no separate verification resource — so build/test
  evidence for one item can collide with another's edits. Dispatch one item to
  completion (implement + verify) before the next. Worktree isolation would
  allow `parallel-implement / serialized-verify`.
