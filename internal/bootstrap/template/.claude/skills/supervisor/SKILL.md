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

            The rule: one worker carries one task. A task is one result
            - a change, a report, a decision - together with every step
            toward it. Count results, not steps: a brief listing many
            steps that end in one result is one task.

            Judge relatedness by information, not by code. Information
            is the artifacts work reads or writes: source files,
            documents, logs, backlog items, records, test output. Work
            belongs on one worker when any of these holds:
            - the parts read or change largely the same artifacts, so
              splitting them makes several workers re-read the same
              material;
            - the parts form one judgment over shared information -
              surveying, classifying, comparing, ranking and summarizing
              the same set of items is one task;
            - one part's output is the next part's input, or one part
              cannot be verified without the other.
            Parts are unrelated when they draw on different information
            and produce independent results, such as fixing a parser bug
            and rewording the release notes. Sharing a project, a
            milestone, a theme or a convenient moment does not relate
            them.

            Answer {"ok": false, "reason": "..."} only in these cases:
            1. An Agent brief combines work that is unrelated under the
               rule.
            2. A SendMessage hands a running worker new work - as opposed
               to steering, correcting, answering, resuming or ending
               its current work - that is unrelated to that work under
               the rule.
            A brief or message carrying several parts states in a BATCH
            REASON section which information they share. A missing or
            thin BATCH REASON is not by itself grounds to reject: when
            the brief shows the parts share information, answer ok.
            Reject over a missing BATCH REASON only when the brief gives
            no sign that the parts share anything.

            The reason names each unrelated part and the information it
            draws on, and ends with: "Give each unrelated part its own
            fresh worker, or add a BATCH REASON naming the artifacts the
            parts share and the single result they feed, e.g. 'BATCH
            REASON: all steps read the open backlog and the playtest
            logs and feed one triage report; split, each worker would
            re-read both.'"

            Answer {"ok": true} for everything else: one task, multi-step
            work toward one result, a batch over shared information,
            status pings, resume signals, answers to questions, shutdown
            messages, and anything where you cannot tell whether the
            parts are related.
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
3. One worker carries one task — one result (a change, a report, a
   decision) and every step toward it. Work is related when it draws on
   the same information: it reads or changes largely the same artifacts
   (files, documents, logs, backlog items, records), forms one judgment
   over a shared set of items, or feeds one part's output into the next.
   Related work stays on one worker — splitting it makes several workers
   re-read the same material and fragments one judgment. Unrelated work
   gets its own fresh worker, including a new task that arrives while a
   worker is still running.
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
work drawing on unrelated information.

Every brief carries a `BATCH REASON` section, written on the first
send — the gate sees each call alone, never an earlier attempt:

- One task: `BATCH REASON: none`.
- Several parts: name the artifacts the parts share and the single
  result they feed, e.g. `BATCH REASON: all steps read the open backlog
  and the playtest logs and feed one triage report; split, each worker
  would re-read both.`
- A new task sent to a running worker: a `BATCH REASON:` line naming
  the information it shares with the worker's current task.

A rejection comes back as the tool error naming the unrelated parts:
give each its own brief, or — when the parts do share information the
brief failed to show — add the `BATCH REASON` and resend. Rewording a
brief to hide a second task defeats the gate and breaks rule 3.

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
