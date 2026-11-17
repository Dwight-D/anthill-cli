# Backlog — how to submit

Anyone (user, agent mid-task, outside submitter) can raise work here in
seconds. Provide the two things only a submitter knows:

- **title** — one line.
- **value** — the pain it removes or the potential it unlocks.

Optionally: a **summary** (one or two sentences: what the change is),
**source** (where it came up), and a **workstream hint** — the hint is
non-binding and a wrong or missing one costs nothing; triage assigns the
workstream authoritatively. (This project's workstreams are defined in
`workstreams.md`.) The **reporter** — `user`, `supervisor`, or `worker` —
records who filed it.

Intake command:
`anthill backlog new --title "…" --value "…" --reporter <who> [--summary "…"] [--source "…"] [--hint <ws>]`.
It lands the item in `intake/` as `status: new`, `dispatch: proposed`. If
the `anthill` CLI is not installed yet, write the file directly into
`intake/` following the schema in `bindings.md` — frontmatter with `title`,
`value`, `reporter`, optional `summary`/`source`/`hint`, `status: new`,
`dispatch: proposed`.

That is the whole ask. You do NOT decide risk, disposition, acceptance
tests, or approval — triage proposes and the user approves. Before adding, glance across the workstream directories for an
obvious duplicate on the same subsystem.

Everything else — workstream definitions and triage profiles
(`workstreams.md`), schema/commands (`bindings.md`) — is read by the
`triage`, `dispatch`, and `expedite` skills; submitters can ignore it.
Finished items are archived in `done/`; discarded ones in `discarded/`.

**Scope:** items here improve or produce this project's work. If it fits
nowhere, triage will discard it with a reason rather than let it rot.
