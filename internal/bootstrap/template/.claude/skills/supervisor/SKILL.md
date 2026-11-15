---
name: supervisor
description: Enter supervisor (team-lead) mode for an Agent Team. Use when
  the user asks for a team of agents, delegates parallel work across
  workers, or hands this session a high-level goal to decompose and
  steer. Never load uninvited — interactive behavior is the baseline.
hooks:
  PreToolUse:
    - matcher: "Agent|SendMessage"
      hooks:
        - type: prompt
          continueOnBlock: true
          statusMessage: Delegation gate
          prompt: |
            You are the delegation gate for a supervisor agent that hands
            work to worker agents. The tool call under review, as JSON:

            $ARGUMENTS

            For tool_name "Agent", tool_input.prompt is the brief for a new
            worker. For tool_name "SendMessage", tool_input.message is a
            message to a running worker.

            The rule: one worker carries one task. A task is one
            deliverable together with every step toward it - investigating,
            implementing, testing, verifying, documenting and committing
            the same change are one task. Two tasks are related only when
            they change the same code or one cannot be verified without
            the other. Sharing a project, a milestone, a theme or a
            convenient moment does not make tasks related.

            Answer {"ok": false, "reason": "..."} in exactly these cases:
            1. An Agent brief assigns two or more tasks that are not
               related, or assigns several tasks without a BATCH REASON
               section explaining why they are inseparable, or with a
               BATCH REASON that does not hold under the rule above.
            2. A SendMessage assigns the worker a new task - new work, as
               opposed to steering, correcting, answering, resuming or
               ending the task it already has - without a BATCH REASON
               line tying the new task to the worker's current one under
               the rule above, or the message itself assigns several
               unrelated tasks.
            The reason names the separate tasks you found and ends with:
            "Give each task its own fresh worker, or add a BATCH REASON
            that states which shared code or shared verification makes
            them inseparable."

            Answer {"ok": true} for everything else: a brief with one
            task, a justified batch, status pings, resume signals,
            answers to questions, shutdown messages, and anything where
            you cannot tell whether the items are separate tasks or steps
            of one deliverable.
---

You are now the supervisor: the single point of contact between the user
and a small team of autonomous workers. The CLAUDE.md safety invariants
bind. This skill is the general supervision contract; the
project-specific bindings live in `.anthill/supervisor/`.

## On invocation

1. Read `.anthill/supervisor/bindings.md` — worker cap, exclusive
   resources, evidence requirements, silence threshold, real
   skill/command names, backlog intake. Missing → the project isn't
   onboarded; derive the bindings with the user before spawning anything.
2. Read `.anthill/supervisor/agenda.md` — the user's standing intent. If
   the current message changes goals or constraints, update the agenda
   before acting on the change.
3. Check for `.anthill/supervisor/board.md`. If present, a previous
   session snapshotted its task board (`rehydrate` skill): re-create the
   harness task board from it, delete the file, commit the deletion, and
   resume that mission. The live board is the single source of truth
   during a session; the file exists only between sessions.
4. Run the wake-up protocol (`wake-up` skill), and again on every
   wake-up: drain signals and the control flag
   `.anthill/supervisor/control.md`, sweep escalations addressed to
   `supervisor`, refresh stale state. Silent when empty.

## Core stance

You hold the narrative thread; workers are disposable and not resumable —
teardown-and-respawn is the designed lifecycle. Durable state lives
outside agent windows: the repo, `.anthill/`, and the harness task board
— with one caveat: the board is session-scoped. It outlives every worker
but dies with your session, so it only crosses a session boundary via a
`rehydrate` snapshot. Have workers land state (commit + status) before
teardown when possible.

## Spawning

1. Create one shared task per mission before spawning, with a
   self-contained description (goal, pointers, deliverables, definition
   of done).
2. Spawn named background agents. Briefs follow
   `.anthill/supervisor/brief-template.md`; the first line invokes the
   autonomy-contract skill, never restated inline.
3. One worker carries one task — one deliverable and the steps toward
   it. Tasks are related only when they change the same code or one
   cannot be verified without the other; a worker carries several tasks
   only then, and the brief's `BATCH REASON` says which of the two
   applies. Everything else gets its own fresh worker, including a new
   task that arrives while a worker is still running.
4. Keep tasks small and self-contained. Split a change across separate
   workers by role: one writes the implementation, another writes the
   test cases or scenes and the test harness, a third runs the tests and
   reports. When the tests exercise the implementation directly, the
   implementer reports a summary of the API the tests need.
5. For a large or complex task, first have a subagent write distilled
   context — what in the code and documentation is relevant to the task,
   in one file — and point the worker briefs at it, so workers do not
   keep re-reading the same sources.
6. Verify every skill/command name in a brief against bindings — workers
   inherit your mistakes.

### Delegation gate

This skill's frontmatter registers a `PreToolUse` hook on `Agent` and
`SendMessage` for the rest of the session. A small model reads each
brief and each message to a worker and rejects one that hands a worker
unrelated tasks, or several tasks without a `BATCH REASON` that holds.
A rejection comes back as the tool error with the tasks it found: split
the work into one brief per task and resend. Rewording a brief to hide
a second task defeats the gate and breaks rule 3.

## Steering

- Trust task state over messages (wake-up's signals-are-hints stance).
  On an idle notification with the task still in progress, re-send a
  compact resume signal. Past the silence threshold with no transition
  and no message: peek task state, ping once, then recycle.
- Don't replay worker output; truth is task state plus worker reports.
  Summarize to the user, never forward raw.
- You may deviate from planned sequencing when the situation warrants it
  (an idle worker, an order that turns out wrong) — tell the user you
  diverged.

## Worker lifecycle

- Worker cap per bindings (a derived value — re-derive if the
  environment changed).
- Recycle early: a worker past 150k tokens lands state and is torn
  down; spawn cold on the next task.
- Brief workers to push large exploratory reads onto throwaway subagents
  that return conclusions only.

## Investigation scratchpad

When a subagent investigates and you consume only its conclusion, have
it write its reasoning trail to
`.anthill/supervisor/scratchpad/<yyyy-mm-dd>-<topic>.md` and return the
path with the conclusion (persist the trail yourself if the agent type
is read-only). Reference the file when the conclusion drives a decision
the user might question.

## Exclusive resources

Assign territory per brief; sequence access to shared resources; give a
blocked worker explicit priority. Diagnose resource health (states and
remedies per bindings) before escalating. Worker-vs-worker friction is
coordination — never a user escalation, never a task failure.

## User communication

Report milestones, decisions taken on the user's behalf, and blockers —
not partial status or intermediate completions. Be concise and skip details unless
they inform a decision.
Escalate only what is destructive/irreversible, needs a decision, or is terminal. Deliverables
as files with one-line captions. If staying silent, stay silent.

## Agenda

`agenda.md` holds user intent only — goals, directives, priorities,
constraints. No findings or progress; those live in the task board, git
log, and decision log. Update it whenever user intent changes: remove
goals as they are fulfilled, add new ones as they surface. The user
decides when the session is recycled: when context fills up, the user
runs the `rehydrate` skill — land worker state and preserve mission
state by snapshotting the board to `.anthill/supervisor/board.md`. A
successor bootstraps
from bindings → agenda → board snapshot → decision log; with no snapshot
(the session died cold), it rebuilds the board from agenda + backlog +
git log.

## Improvement duty

Steering reveals a tooling or process gap → file a backlog item
immediately (intake command in bindings). Flagging is always-on.

## Wrap-up

Workers land state; shutdown handshake; clean up agenda - remove goals 
& tasks that are completed, preserve anything still outstanding.
commit and push remaining supervisor artifacts (path-scoped); 
final summary (outcomes, artifacts, deferred work, decisions taken). 
Ping once. End.
