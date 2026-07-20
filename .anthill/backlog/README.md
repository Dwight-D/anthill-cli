# Backlog — how to submit

Anyone (user, agent mid-task, outside submitter) can raise work here in
seconds. Provide the two things only a submitter knows:

- **title** — one line.
- **value** — the pain it removes or the potential it unlocks.

Optionally: **source** (where it came up) and a **workstream hint** — the
hint is non-binding and a wrong or missing one costs nothing; triage
assigns the workstream authoritatively. (This project's workstreams are
defined in `workstreams.md`.)

Intake command: see `bindings.md`. If no schema-owner CLI is installed yet,
write the file directly into `intake/` following the schema there —
frontmatter with `title`, `value`, optional `source`, `status: idea`.

That is the whole ask. You do NOT decide risk, acceptance tests, or approval —
triage does. Before adding, glance across the workstream directories for an
obvious duplicate on the same subsystem.

Everything else — workstream definitions and triage profiles
(`workstreams.md`), schema/commands (`bindings.md`) — is read by the
`triage`, `dispatch`, and `expedite` skills; submitters can ignore it.

**Scope:** items here improve or produce this project's work. If it fits
nowhere, triage will reject it with a reason rather than let it rot.
