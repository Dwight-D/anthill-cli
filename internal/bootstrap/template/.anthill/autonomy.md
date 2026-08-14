# Autonomy contract config — <YOUR PROJECT>

> **PROJECT-SPECIFIC TIER — TEMPLATE.** Swap this file's content for your own
> domain. It is the ONE home for what an autonomous worker in this project may
> do without asking. The `autonomous` skill loads it on invocation; the skill
> itself is portable and identical across installations. Keep each fact here
> and nowhere else. Delete this quote block once derived.

The project-specific inputs to the autonomy contract. The portable contract —
safety invariants still bind, work-on-a-branch, log-and-continue vs.
stop-and-ask, expected permission mode — lives in the `autonomous` skill and is
the same everywhere. What is project-specific is **the concrete list of actions
that count as routine here**, plus **where routine decisions get logged**.

## Proceed freely (do not ask permission)

The concrete actions a worker may take without asking, stated in this project's
own tooling and skills. Derive against the project's real skills, commands, and
safety invariants: every entry must be a routine action that the safety
invariants in the always-on file do NOT gate.

<!-- WORKED EXAMPLE (from the Nodachi template repo) — the SHAPE of a
     proceed-list, not content to copy. Your actions will be different.

  - Create/edit/delete C# source, tests, and dev-docs under the project tree.
  - Run `nodev build`, `nodev test`, and the editor-driven verification skills.
  - Import/build Unity assets and prefabs via the `nodev` toolchain.
  - git: path-scoped add, commit, push to the designated work branch.
  - Read anything in the repo (rails block the exceptions).
  - Install/update dev dependencies when the task clearly needs them.
-->

- <Create/edit/delete the project's own source, tooling, tests, and dev-docs.>
- <Run the project's build/compile/test/verification commands via the
  established skills and tooling.>
- <Produce/import/build the project's primary artifacts via the toolchain.>
- <git: path-scoped add, commit, push to the designated work branch.>
- <Read anything in the repo (rails block the exceptions).>
- <Install/update dev dependencies when the task clearly needs them.>

## Decisions log

- **Path:** `.anthill/decisions.md` — the routine-choice log. When a
  non-blocking question comes up mid-task, the worker records the choice here as
  one line and continues. (Change the path only if your project keeps this log
  elsewhere; the default is the framework convention.)
