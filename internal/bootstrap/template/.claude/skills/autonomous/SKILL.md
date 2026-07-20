---
name: autonomous
description: Enter autonomous work mode. Use when the user invokes
  /autonomous (optionally with the task as argument), tells the session to
  work autonomously, or a supervisor/worker brief opens with this
  invocation. Never load uninvited — interactive behavior is the baseline.
---

You are now working autonomously on the task given (or about to be given).
The safety invariants in CLAUDE.md still bind. Within them:

Load `.anthill/autonomy.md` on invocation — it holds this project's
**proceed-list** (the concrete actions that count as routine here, stated in
the project's own tooling) and its **decisions-log path**. Missing → the
project isn't onboarded; derive it with the user before proceeding
autonomously.

## Proceed freely (do not ask permission)

The actions listed in `.anthill/autonomy.md`'s proceed-list — the project's
routine actions that its safety invariants do not gate. Take them without
asking. Anything outside that list and not clearly routine follows the
log-and-continue vs. stop-and-ask rules below.

## Working rules

- Work on a feature branch, never directly on main. Commit a checkpoint at
  every working state. Under co-tenancy (other agents share this checkout):
  use the shared integration branch instead of a per-agent branch, stage only
  your own paths (never `git add -A`), tag commits with your task id.
- When a non-blocking question comes up mid-task: don't stop. Make the
  reasonable routine choice, log it as one line in the decisions log named in
  `.anthill/autonomy.md` (`.anthill/decisions.md` by default), continue, and
  surface the log at the end of the task.
- When a safety invariant ("ask first") blocks the task: stop, state the
  question with your recommendation, and wait.

Expected permission mode: bypass (Shift+Tab). Ask rules still interrupt for
the risky tier; deny rules still block outright.
