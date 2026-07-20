# Supervisor bindings — Anthill CLI

Project-specific tier of the supervisor mechanism. The general contract lives
in `.claude/skills/supervisor/SKILL.md` and loads this file on invocation.

## Launching an unattended supervisor session

```
.\tools\supervise.ps1 <mission…>      # Windows (this repo's primary env)
bash tools/supervise.sh <mission…>    # POSIX
```

The launcher starts the session with elevated permissions
(`--permission-mode bypassPermissions`) + remote steering + `/supervisor
<mission>`. Elevation must happen at launch — the harness forbids in-session
escalation, and subagents inherit the launched mode, so the whole
supervisor → dispatcher → worker tree runs elevated from this one flag. Do not
put `bypassPermissions` in `defaultMode` — keep elevation a per-launch decision
via the script.

## Worker cap and exclusive resources

The inventory (each exclusive resource + its health states, remedies, leases)
and the derived caps live in **`.anthill/resources.md`** — the one home for
those facts. Current derivation: **team worker cap 2**, dispatch parallelism
**serial**. Caps count concurrent resource consumers, not agents.

## Evidence requirements

The supervisor verifies evidence, not assertions. Per deliverable class:
- **CLI / Go code changes:** `go build ./...` exit 0 AND `go test ./...` exit 0
  (both mandatory).
- **Bug fixes:** a regression test that fails before the fix and passes after,
  plus the build/test above.
- **Backlog items:** the item's own `verify` field passing.
- **Docs / process:** the artifact exists in its one durable home and nothing
  else claims to own the same fact.

## Silence threshold

~30 min with no task transition and no message during an active mission → peek
task state, ping once, then recycle the worker.

## Real names workers get wrong

- Skills are invoked by exact name: `dispatch`, `dispatch-receive`,
  `triage`, `escalate`, `expedite`, `autonomous`, `supervisor`,
  `dispatch-loop`, `wake-up`. No `/anthill` prefix; no pluralization.
- The evidence commands are `go build ./...` and `go test ./...` — not `go
  build` / `go test` bare (the `./...` covers all packages).
- There is **no `anthill` CLI yet** — backlog/escalation writes are
  convention-first (edit the files directly per `.anthill/backlog/bindings.md`).
  Do not invent CLI verbs.

## Backlog intake

Write a file into `.anthill/backlog/intake/` per the schema in
`.anthill/backlog/bindings.md` — a `title` + `value` is the whole ask
(`status: idea`). Expedite runs only on the user's explicit instruction —
never self-initiated.

## Backlog dispatch under co-tenancy

Backlog workers skip ready items colliding with another live worker's territory
— leave them ready, report the skip. Hand skipped items to a future session in
the wrap-up report rather than rushing behavior changes at the end.
