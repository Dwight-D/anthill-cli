# Dispatch control flag

Updated: <date> (<who / session>)

> **RUNTIME ARTIFACT — durable control channel.** A pause/stop/resume order the
> dispatcher reads on EVERY wake-up drain, before selecting or advancing. It is
> durable so a control order survives message loss or cross-tier misrouting: a
> pause sent only as a chat message can be lost, and then never honored — this
> flag removes that failure. State first, signal second: the parent tier (or
> the user) sets the state here, THEN optionally pings; the dispatcher only
> reads it and acknowledges by landing state. Default state is `run`.

## State

- **state:** `run`
  <!-- run | pause | stop -->
- **set-by:** <who / session>
- **reason:** <why — shown in the dispatcher's stand-down / wrap-up>

## Semantics (how the dispatcher honors each state)

- **run** — normal operation.
- **pause** — finish landing the current item's state (or cleanly abort an
  in-flight spawn), stand down WITHOUT recycling: stay alive, keep draining on
  each wake-up, do not advance the ledger cursor until the state returns to
  `run`.
- **stop** — wrap up and terminate: unclaim anything in flight, land the ledger,
  send a one-line summary to the report-target, end.
