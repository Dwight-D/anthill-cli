# Dispatcher tier config — `.anthill/dispatch/`

The specific-tier home for the autonomous dispatcher (`dispatch-loop` skill),
mirroring `.anthill/supervisor/` for the supervisor tier. Two runtime
artifacts live here; both are **seeded on first use**, not pre-filled config.

- **`ledger.md`** — the **progress ledger**: the dispatcher's durable position
  in a *framed* batch (an ordered list of items + a reporting cadence + a
  report-target). Required only when the dispatcher is handed a framed batch;
  a bare ready-queue sweep needs no ledger (its position rehydrates from the
  backlog alone). The `dispatch-loop` skill writes and reads it; a cold
  successor rehydrates its cursor from it, losing no batch framing or ordering.

- **`control.md`** — the **control flag**: a durable pause/stop/resume signal
  the dispatcher reads on every wake-up drain. Durable so a pause survives
  message loss or cross-tier misrouting — a control order that exists only as a
  chat message can be lost, which is the failure the durable flag removes. The
  parent tier (or the user) writes it; the dispatcher only reads it and
  acknowledges by landing state.

Both are instances of the general controller-state model defined in the
`wake-up` skill (intent / position / control). They carry no project-specific
configuration — the dispatcher's actual bindings are in
`.anthill/backlog/bindings.md` and `.anthill/resources.md`.
