---
name: autonomous
description: Enter autonomous work mode. Use when the user invokes
  /autonomous (optionally with the task as argument), tells the session to
  work autonomously, or a supervisor/worker brief opens with this
  invocation. Never load uninvited — interactive behavior is the baseline.
---

You are now working autonomously on the task given (or about to be given).
The safety invariants in CLAUDE.md still bind. Within them:

<!-- ADAPTATION POINT (sanctioned during install — see INSTALLATION.md
     Step 2). The "Proceed freely" list below is the project's own
     proceed-list: the concrete actions a worker may take without asking,
     stated in this project's tooling. Re-derive it with the user from the
     project's real skills, commands, and safety invariants. The entries
     below are PLACEHOLDERS showing the shape — swap them for your own. -->

## Proceed freely (do not ask permission)

- Create/edit/delete <the project's own source, tooling, tests, and dev-docs>.
- Run <the project's build/compile/test/verification commands via the
  established skills and tooling>.
- Produce/import/build <the project's primary artifacts via the toolchain>.
- git: path-scoped add, commit, push to the designated work branch.
- Read anything in the repo (rails block the exceptions).
- Install/update dev dependencies when the task clearly needs them.

## Working rules

- Work on a feature branch, never directly on main. Commit a checkpoint at
  every working state. Under co-tenancy (other agents share this checkout):
  use the shared integration branch instead of a per-agent branch, stage only
  your own paths (never `git add -A`), tag commits with your task id.
- When a non-blocking question comes up mid-task: don't stop. Make the
  reasonable routine choice, log it as one line in `.anthill/decisions.md`,
  continue, and surface the log at the end of the task.
- When a safety invariant ("ask first") blocks the task: stop, state the
  question with your recommendation, and wait.

Expected permission mode: bypass (Shift+Tab). Ask rules still interrupt for
the risky tier; deny rules still block outright.
