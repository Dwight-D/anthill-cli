---
name: rehydrate
description: Snapshot a supervisor session's working state to durable files
  so a fresh session resumes it by invoking the supervisor skill. Use when
  the user invokes /rehydrate or asks to recycle or hand off the supervisor
  session. User-initiated — never load uninvited.
---

# rehydrate (supervisor handoff)

The harness task board lives inside the session and dies with it. This
skill bounces everything a successor needs into durable files, so
recycling a supervisor session is as routine as recycling a worker:
run `/rehydrate`, end the session, start a fresh one, invoke
`/supervisor` — it resumes automatically.

## Procedure

1. **Land workers.** Have each live worker commit its state and report,
   then tear it down. A worker holding a backlog item delivers or blocks
   it first. Never leave a worker running past the snapshot.
2. **Snapshot the task board** to `.anthill/supervisor/board.md`: the
   mission framing (goal, origin), then one row per task — title, backlog
   item key (or `none`), board status (pending / in-progress / blocked /
   done), and a one-line pointer (branch, file, blocking reason). Position only — findings live in
   their durable homes, not here.
3. **Sweep the rest of the durable set.** Agenda current (user intent
   only)? Routine choices in the decisions log? Pending escalation
   records persisted? Plans or decisions that exist nowhere else → a
   Notes section at the end of the board file.
4. **Commit** the supervisor artifacts, path-scoped.
5. **Confirm** in one line: what was snapshotted, and that `/supervisor`
   in a fresh session resumes it.

## Resume (what the supervisor skill does with the snapshot)

The `supervisor` skill checks for `.anthill/supervisor/board.md` on
invocation. If present, it re-creates the harness task board from the
file, then deletes the file and commits the deletion. The live board is
the single source of truth during a session; the file exists only
between sessions — never maintain both in parallel.

A session that dies without running rehydrate leaves no snapshot. The
successor falls back to agenda + `anthill backlog list --active` (every
claimed item, with `claimed-by` and `session`) + git log + decision log and
rebuilds the board from those.
