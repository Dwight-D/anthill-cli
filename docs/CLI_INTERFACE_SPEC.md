# Anthill CLI — Interface Specification (PROPOSED)

Status: proposal for review. This document specifies the intended command
surface, output contracts, and invariant ownership of the `anthill` binary. No
Go implementation is prescribed here beyond what the existing bootstrap already
establishes (`spf13/cobra` root command, `internal/version`). Where a decision
is a judgment call, it is marked and the alternative is noted; the genuine forks
are collected in [Open questions](#7-open-questions-for-the-user).

Module: `github.com/Dwight-D/anthill-cli`. Binary: `anthill`.

---

## 1. Overview & principles

The Anthill CLI is the **schema owner** for a file-based backlog + escalation
system. It exists to make the frontmatter schema and the id scheme have exactly
one writer, so that every mutation is invariant-checked at the point of write
rather than trusted to be well-formed by convention.

Principles:

- **Files are the source of truth.** Every item is a markdown file with YAML
  frontmatter under `.anthill/backlog/`; every escalation is a markdown file
  under `.anthill/escalations/`. The CLI never introduces a second authority
  for a fact that a file already owns. A user can read, `git diff`, and
  hand-edit these files; the CLI's job is to keep writes well-formed, not to
  hide the data behind a database.
- **Schema-owner role.** Once the CLI exists, its `backlog` and `escalation`
  verbs become the ONLY sanctioned writer of item/escalation frontmatter. The
  id scheme, the enum-valued fields, the required-field-per-lifecycle-stage
  rules, and the file-location invariants belong to the CLI. Hand-editing stays
  possible (it is just text), but `anthill … validate` is what certifies a tree
  as well-formed.
- **Headless-first.** Agents drive this tool with no terminal attached. Every
  command is fully scriptable via flags and arguments; nothing requires
  interaction to complete. Every command emits a human-readable form by default
  and a stable machine form under `--json`. Any interactive TUI surface
  ([§5](#5-tui-surfaces-optional)) is an optional convenience layered over the
  same scriptable verbs — never the only way to do a thing.
- **Exit-code discipline.** Success is 0. Distinct non-zero codes classify
  failure so a caller can branch on the class (not-found vs conflict vs
  validation) without parsing text. See [§2](#2-global-conventions).
- **Propose-only posture is preserved by the tool.** The backlog is
  propose-only: triage recommends a `disposition`; a human sets
  `status: approved`. The CLI must not become a hole in that posture — see the
  approval fork in [Open questions](#7-open-questions-for-the-user).

---

## 2. Global conventions

### Binary and invocation

```
anthill [global flags] <group> <verb> [args] [flags]
```

Command groups: `backlog`, `escalation`, plus top-level lifecycle helpers
(`init`, `doctor`, `validate`, `version`).

### Global flags

| Flag | Type | Meaning |
|------|------|---------|
| `--root <dir>` | path | Directory containing `.anthill/`. Default: walk up from CWD to the nearest ancestor containing `.anthill/` (git-style discovery); error if none found. |
| `--json` | bool | Emit machine-readable JSON on stdout instead of the human form. |
| `--quiet`, `-q` | bool | Suppress non-essential human output (progress, confirmations). Errors still go to stderr; `--json` payloads still print. |
| `--no-color` | bool | Disable ANSI styling. Also auto-disabled when stdout is not a TTY or `NO_COLOR` is set. |
| `--version` | bool | Print `anthill <version>` and exit 0 (already wired on the root command). |
| `--help`, `-h` | bool | Command help. |

`--root` may also be supplied via `ANTHILL_ROOT`; the flag wins.

### stdout / stderr split

- **stdout** carries the *answer*: the requested data (human tables or
  `--json` payloads). A caller can pipe stdout without contamination.
- **stderr** carries everything else: progress notes, warnings, confirmations,
  and error messages. Under `--json`, errors are additionally emitted as a JSON
  object on stderr (see below) so machine callers get a structured failure.

### JSON envelope

Under `--json`, every command prints exactly one JSON value to stdout on
success. List-style commands print a JSON array of objects; single-item
commands print one object; mutating commands print the resulting object (or a
small result object). This is a stable contract — field additions are
backward-compatible; renames/removals are a schema change (never-auto).

On failure under `--json`, a structured error is written to **stderr**:

```json
{ "error": { "code": "not_found", "exit": 4, "message": "no item with id 'foo'", "id": "foo" } }
```

### Exit-code table

| Code | Name | When |
|------|------|------|
| `0` | ok | Command succeeded. |
| `1` | error | Unexpected/internal error (I/O failure, unhandled condition). |
| `2` | usage | Bad invocation: unknown flag/command, missing required arg, mutually exclusive flags. (Cobra usage errors map here.) |
| `3` | validation | A write was rejected because it would violate the schema, OR `validate` found a violation. The offending invariant is named on stderr. |
| `4` | not_found | A referenced id (item or escalation) or workstream does not exist. |
| `5` | conflict | Lock/claim conflict: the item is already claimed by another holder, or a concurrent write lost a compare-and-set. |
| `6` | precondition | The operation is illegal in the item's current state (e.g. closing an item that is not claimed, claiming a non-ready item without `--force`). |

Rationale for splitting `3/4/5/6`: these are the four failure classes a
dispatch/triage skill actually branches on — "the data is malformed",
"it's gone", "someone else has it", "it's not in a state where this makes
sense". Collapsing any pair would force text-parsing on the caller.

---

## 3. Command tree

Compact outline (proposed; `*` marks a subcommand not literally named in
`bindings.md` and therefore a never-auto surface addition to confirm):

```
anthill
├── init*                       scaffold a .anthill/ tree
├── doctor*                     environment + integrity health check
├── validate [--strict]         validate the whole tree (alias of backlog+escalation validate)
├── version                     print version (also --version on root)
├── backlog
│   ├── new                     create an intake item
│   ├── list                    list items (filters: --workstream --untriaged --ready --status)
│   ├── show* <id>              print one item
│   ├── set <id> key=value…     mutate frontmatter; workstream= moves the file
│   ├── next                    print the next dispatchable item (sweep order)
│   ├── claim <id> | --next     atomically mark an item in-progress
│   ├── close <id> --done | --discard "…" | --remove "…" | --block "…"
│   └── validate [--strict]     validate backlog items
└── escalation
    ├── raise                   create an escalation record
    ├── list                    list records (filters: --to --status --item)
    ├── show* <id>              print one record
    ├── answer <id> --decision "…"   append ## Decision, set status: answered
    └── apply <id>              append ## Applied, one-line LOG.md, delete the record
```

Notation below for each command: **Purpose · Args · Flags · Reads · Writes ·
Human output · `--json` output · Exit codes.**

---

### 3.1 `anthill backlog new`

- **Purpose.** Submit an intake item. Enforces the intake contract: a `title`
  and a `value` are the whole required ask.
- **Args.** none (all via flags).
- **Flags.** `--title <str>` (required), `--value <str>` (required),
  `--source <str>` (optional), `--hint <ws>` (optional, non-binding submitter
  hint), `--priority high|normal` (optional; triage normally sets this).
- **Reads.** `intake/` and all workstream dirs (to compute a unique id).
- **Writes.** `intake/<id>.md` with frontmatter `title`, `value`, optional
  `source`, optional `hint`, `status: idea`. No `workstream` key (absent while
  in intake).
- **Human output.** `created <id>` on stderr; the id on stdout.
- **`--json`.** The created item object (see [§4](#4-frontmatter--id-ownership)
  for the object shape), including its generated `id` and file `path`.
- **Exit codes.** 0; 2 (missing `--title`/`--value`); 3 (title slugifies to
  empty).

> Reconciliation: `bindings.md` sketches `--backlog <hint>` but the schema
> field is `hint`. This spec names the flag `--hint` to match the field.
> Flagged in [Open questions](#7-open-questions-for-the-user).

---

### 3.2 `anthill backlog list`

- **Purpose.** Enumerate items across intake + workstreams.
- **Args.** none.
- **Flags.**
  - `--workstream <ws>` — restrict to one workstream directory.
  - `--untriaged` — only items in `intake/` (no `workstream`).
  - `--ready` — only dispatchable items (`status: approved` + non-empty
    `verify`).
  - `--status <s>` — filter by lifecycle status (repeatable).
  - `--sort sweep|priority|id` — default `sweep` (workstream sweep-order, then
    priority, then id).
- **Reads.** `intake/`, workstream dirs, `workstreams.md` (for sweep order).
- **Writes.** none.
- **Human output.** A table: id · workstream · status · priority · title
  (truncated). Empty set → a note on stderr, exit 0.
- **`--json`.** Array of item objects, each including `ready: bool` (derived).
- **Exit codes.** 0; 4 (`--workstream` names a non-existent workstream).

---

### 3.3 `anthill backlog show` *(proposed helper)*

- **Purpose.** Print a single item's full frontmatter + body. Consumers
  (dispatch's "full body" claim output, triage's dedup read) need one item by
  id without shelling out to `cat`, and reading via the CLI guarantees the
  derived fields (`ready`, `path`) are computed the same way as everywhere else.
- **Args.** `<id>` (required).
- **Flags.** `--body/--no-body` (include markdown body; default include).
- **Reads.** the item file.
- **Writes.** none.
- **Human output.** The rendered item.
- **`--json`.** One item object, with `body` as a string field.
- **Exit codes.** 0; 4 (not found).

---

### 3.4 `anthill backlog set`

- **Purpose.** Mutate one or more frontmatter keys on an item, invariant-checked
  as a single atomic write. This is the triage workhorse.
- **Args.** `<id>` then one or more `key=value` pairs.
- **Flags.** none (all mutation is positional `key=value`).
- **Reads.** the item file; workstream dirs (to validate a `workstream=` target
  and re-check id uniqueness after a move).
- **Writes.** the item file. **File-move semantics:** setting
  `workstream=<ws>` moves the file from its current directory into
  `backlog/<ws>/` (git-friendly rename, id/filename unchanged). Setting
  `workstream=` while the item is in intake also strips any `hint` key (hint is
  removed on triage per schema). All other keys are updated in place.
- **Validation.** Every changed key is checked against the schema
  ([§4](#4-frontmatter--id-ownership)): enum fields must hold a legal value; a
  `workstream=` target must be a real workstream directory; `id`, `title`, and
  `value` are not settable via `set` (id is immutable; title/value edits are a
  deliberate carve-out — see open questions). Illegal write → exit 3, file
  untouched.
- **Human output.** `set <id>: <k>=<v> …` on stderr; on a move, also
  `moved <id> → <ws>/`.
- **`--json`.** The updated item object.
- **Exit codes.** 0; 2 (no `key=value` given, malformed pair); 3 (illegal
  value / immutable key / unknown workstream target — validation class); 4
  (item not found).

> Approval gate: whether `set <id> status=approved` is permitted at all, or
> must go through a human-only path, is a real fork —
> [Open questions](#7-open-questions-for-the-user).

---

### 3.5 `anthill backlog next`

- **Purpose.** Print the single next dispatchable item without claiming it, in
  sweep order. This is `dispatch`'s selection step.
- **Args.** none.
- **Flags.** `--workstream <ws>` — restrict to one stream (else full sweep
  order from `workstreams.md`, skipping `never-implicit` streams).
- **Reads.** workstream dirs, `workstreams.md`.
- **Writes.** none.
- **Human output.** The chosen item's id + title, or `no ready items` on
  stderr with exit 0.
- **`--json`.** The chosen item object, or `null` when none ready.
- **Exit codes.** 0 (including the empty case); 4 (`--workstream` unknown).

> Judgment: the empty case exits 0 with a `null`/note rather than a non-zero
> code, so a sweep loop terminates cleanly on "nothing ready" without treating
> it as an error. Alternative (exit 4 on empty) rejected because emptiness is a
> normal steady state, not a failure.

---

### 3.6 `anthill backlog claim`

- **Purpose.** Atomically take ownership of an item for implementation:
  transition it to `status: in-progress` so no other agent picks it up. This is
  the authorization boundary — `dispatch` claims, the worker never does.
- **Args.** `<id>` OR `--next`.
- **Flags.**
  - `--next` — claim the item `next` would select (mutually exclusive with a
    positional id).
  - `--workstream <ws>` — scope for `--next`.
  - `--force` — claim even if the item is not ready (a human naming an item is
    an explicit override of the readiness gate) or is already `in-progress`
    (reclaim an orphan).
- **Reads.** the item (or the sweep, for `--next`).
- **Writes.** the item file: `status: in-progress`, plus a `claimed` marker
  (see the claim-mechanism fork). Atomic compare-and-set: read current status,
  write only if it is still the expected value, else conflict.
- **Human output.** the claimed item's id + full body on stdout (dispatch
  packages this into the worker brief).
- **`--json`.** The claimed item object.
- **Exit codes.** 0; 4 (id not found; `--next` found nothing → see note); 5
  (already claimed by someone else, no `--force`); 6 (not ready and no
  `--force`).

> `--next` with an empty ready set: exit 4 (there is nothing to claim), distinct
> from `next`'s exit-0 empty, because `claim` was asked to produce a claim and
> could not. Flagged for confirmation.

---

### 3.7 `anthill backlog close`

- **Purpose.** Terminate or block a claimed item. Exactly one disposition flag
  is required.
- **Args.** `<id>`.
- **Flags** (mutually exclusive, one required):
  - `--done` — completed successfully.
  - `--discard "<reason>"` — decided not worth doing.
  - `--remove "<reason>"` — obsolete / superseded / rejected.
  - `--block "<reason>"` — cannot proceed right now (dependency, escalation).
- **Reads.** the item, `CHANGELOG.md`.
- **Writes.**
  - `--done` / `--discard` / `--remove` are **terminal**: delete the item file
    and append one line to `backlog/CHANGELOG.md`
    (`<date> <id> — done|discarded|removed: <reason/title>`).
  - `--block` is **non-terminal**: the file stays in place, `status: blocked`,
    and the reason is written to the `note` field (e.g.
    `escalated: <file>`). No CHANGELOG line — the item is not closed, it is
    parked. It is named under `close` because it is the terminating verb of a
    *dispatch attempt*, not of the item.
- **Human output.** `closed <id> (done)` / `blocked <id>: <reason>` on stderr.
- **`--json`.** `{ "id": …, "disposition": "done|discard|remove|block",
  "changelog": true|false }`.
- **Exit codes.** 0; 2 (no disposition flag, or more than one); 4 (not found);
  6 (item is not claimed / not in a closeable state — configurable, see fork).

> The done/discard/remove-vs-block asymmetry (terminal + CHANGELOG vs in-place
> annotate) is a deliberate reading of `bindings.md` and the `dispatch` skill
> ("blocked → block in place"). Confirm in open questions.

---

### 3.8 `anthill backlog validate`

- **Purpose.** Certify the backlog tree as schema-well-formed. Run at the end of
  a triage pass and in CI.
- **Args.** none.
- **Flags.** `--strict` (add the cross-field / consistency checks below).
- **Reads.** all item files, `workstreams.md`.
- **Writes.** none.
- **Checks (default).**
  1. Every item's frontmatter parses and carries the required keys for its
     lifecycle stage (intake: `title`, `value`, `status`; triaged: adds
     `workstream`, `change-type`, `risk`, `verify`, `value-verdict`,
     `disposition`).
  2. Every enum field holds a legal value.
  3. `id` == filename (minus `.md`); ids unique across `intake/` + all
     workstream dirs.
  4. An item's directory matches its `workstream` field (or it is in `intake/`
     with no `workstream`).
- **Checks (added by `--strict`).**
  5. Ready-consistency: `status: approved` ⇒ non-empty `verify`.
  6. Disposition coherence: `disposition: AUTO` ⇒ `verify` != `none` AND
     `change-type` is not a never-auto type for that workstream.
  7. No `hint` key survives on a triaged item.
  8. Any `escalated: <file>` note points at an existing escalation record.
  9. No stray files in workstream dirs that are not valid items.
- **Human output.** `ok: N items` or a per-item list of violations on stderr.
- **`--json`.** `{ "ok": bool, "checked": N, "violations": [ { "id", "check",
  "message" } ] }`.
- **Exit codes.** 0 (clean); 3 (one or more violations).

---

### 3.9 `anthill escalation raise`

- **Purpose.** Create a durable escalation record. The CLI owns the escalation
  frontmatter schema too, so controllers write records through it rather than
  hand-authoring frontmatter. (Ephemeral workers still do NOT write files —
  they return the record body inline and their spawner calls `raise`.)
- **Args.** none.
- **Flags.** `--to dispatcher|supervisor|user` (required), `--from <tier>`
  (required), `--item <id>` (optional), `--question <str>` (required, stored
  verbatim in `## Question`), `--context <str>` /
  `--options <str>` (optional initial section bodies), or `--body-file <path>`
  to supply the full markdown body.
- **Reads.** `escalations/`.
- **Writes.** `escalations/<yyyy-mm-dd>-<slug>.md` with frontmatter
  (`to`, `from`, optional `item`, `status: open`, `opened: <date>`) and the
  section skeleton (Question / Context & attempted remedies / Options &
  recommendation / Decision / Applied). If `--item` is given, does **not** by
  itself block the item — the caller blocks it via `backlog close --block`
  (single-writer discipline; the CLI does not do implicit cross-file cascades).
- **Human output.** `raised <file> (to: <tier>)`.
- **`--json`.** The created record object (frontmatter fields + `path`).
- **Exit codes.** 0; 2 (missing required flag); 3 (bad `to` value).

---

### 3.10 `anthill escalation list`

- **Purpose.** Sweep the escalation directory — the wake-up-protocol step every
  controller runs.
- **Args.** none.
- **Flags.** `--to <tier>` (records addressed to a tier), `--status
  open|answered|applied`, `--item <id>`.
- **Reads.** `escalations/`.
- **Writes.** none.
- **Human output.** table: file · to · from · status · item · question
  (truncated).
- **`--json`.** Array of record objects.
- **Exit codes.** 0.

---

### 3.11 `anthill escalation show` *(proposed helper)*

- **Purpose.** Print one full record (the receiving tier reads the verbatim
  Question + context).
- **Args.** `<id>` (the `<date>-<slug>` filename stem).
- **Reads.** the record file.
- **`--json`.** One record object, with each section body as a field.
- **Exit codes.** 0; 4 (not found).

---

### 3.12 `anthill escalation answer`

- **Purpose.** Record a decision on an open record.
- **Args.** `<id>`.
- **Flags.** `--decision "<str>"` (required; appended as `## Decision` with a
  one-line rationale).
- **Reads.** the record.
- **Writes.** appends `## Decision`, sets `status: answered`. The `## Question`
  is never touched (append-only invariant enforced by the CLI: it will refuse
  to answer a record whose Question would be altered).
- **`--json`.** The updated record object.
- **Exit codes.** 0; 4 (not found); 6 (record is not `open`).

---

### 3.13 `anthill escalation apply`

- **Purpose.** Close out an answered record: mark it applied, archive it,
  delete it.
- **Args.** `<id>`.
- **Flags.** `--note "<str>"` (optional, appended under `## Applied`).
- **Reads.** the record, `LOG.md`.
- **Writes.** appends `## Applied`, adds one line to `escalations/LOG.md`
  (`<date> <file> — <to>/<status>: <one-line outcome>`), then deletes the
  record file. Applying does not itself unblock a backlog item — the applier
  carries the decision into the work via `backlog set`/`claim` separately
  (single-writer discipline).
- **`--json`.** `{ "id": …, "applied": true, "logged": true }`.
- **Exit codes.** 0; 4 (not found); 6 (record is not `answered`).

---

### 3.14 `anthill init` *(proposed lifecycle helper)*

- **Purpose.** Scaffold a fresh `.anthill/` tree (directory map from
  `.anthill/README.md`): `backlog/{intake,cli,dev,process,bugs}`,
  `escalations/`, empty `CHANGELOG.md`/`LOG.md`, and stub config files. Makes
  onboarding a new installation a single command instead of hand-copying.
- **Flags.** `--root <dir>`, `--workstream <ws>` (repeatable, to seed
  non-default streams), `--force` (populate an existing dir without clobbering
  present files).
- **Writes.** the directory tree + empty runtime files. Never overwrites an
  existing config file.
- **Exit codes.** 0; 6 (target already initialized without `--force`).

---

### 3.15 `anthill doctor` *(proposed lifecycle helper)*

- **Purpose.** One-shot environment + integrity check: `.anthill/` is
  discoverable, required config files present, `workstreams.md` sweep-order
  names existing directories, `backlog validate --strict` clean, `escalation`
  records well-formed, no answered-but-unapplied records (the failure mode the
  escalate skill calls out). Read-only.
- **Flags.** `--strict` (fail on warnings too).
- **`--json`.** `{ "ok": bool, "checks": [ { "name", "ok", "detail" } ] }`.
- **Exit codes.** 0 (healthy); 3 (integrity problem).

---

## 4. Frontmatter & id ownership

### The item object

Canonical frontmatter (from `bindings.md`, kept verbatim as the schema):

| Key | Type / enum | Notes |
|-----|-------------|-------|
| `workstream` | dir name | absent in intake; set at triage (moves the file) |
| `title` | string | one line |
| `value` | string | the pain removed / potential unlocked |
| `source` | string | optional |
| `hint` | ws name | optional; submit-time only, removed on triage |
| `change-type` | `doc\|tooling\|bugfix\|audit\|verify\|new-command\|new-flag\|rename\|removal\|design\|subjective` | |
| `risk` | `additive\|reversible\|behavior-change\|subjective` | |
| `verify` | string | headless acceptance test, or `none` |
| `value-verdict` | `ADVANCE\|HOLD\|DISCARD — <why>` | |
| `disposition` | `AUTO\|REVIEW\|DISCARD` | AUTO needs non-`none` verify + non-never-auto change-type |
| `status` | `idea\|approved\|in-progress\|blocked\|parked\|done` | |
| `priority` | `high\|normal` | |
| `note` | string | free text; qualifiers: `expedited`, `needs-spec`, `remainder`, … |

Derived (computed, never stored): `id` (== filename stem), `path`, `ready`
(`status == approved && verify != "" && verify != "none"`).

### Invariant-checking on every write

`new`, `set`, and `claim`/`close` share one validation gate. Before persisting,
the CLI:

1. Parses the resulting frontmatter and rejects unknown keys (typo guard) and
   illegal enum values → exit 3, original file untouched (write to a temp file
   + atomic rename, so a rejected or crashed write never leaves a half-file).
2. Enforces stage-required keys (intake vs triaged, per
   [§3.8](#38-anthill-backlog-validate)).
3. Enforces id immutability and uniqueness.
4. On a `workstream=` change, performs the directory move as part of the same
   atomic operation.

### Id generation

- id = kebab **slug of the title**: lowercase; every run of non-alphanumeric
  characters → a single `-`; leading/trailing `-` stripped; truncated to **≤50
  chars** (truncate on a hyphen boundary where possible, never leaving a
  trailing hyphen).
- **Collision** (slug already used anywhere across `intake/` + workstream
  dirs): append a numeric suffix `-2`, `-3`, … choosing the lowest free
  integer. The suffix counts toward the 50-char budget (truncate the base
  further if needed).
- **Immutable**: the id is assigned once at `new` and never changes — not on
  `set`, not on the intake→workstream move (the move is a directory change, the
  filename is preserved), not on a title edit.

**`google/uuid` reconciliation.** The id scheme is deliberately a
human-readable title slug, not a UUID — ids appear in dispatch briefs,
CHANGELOG lines, and `git log`, where a slug is legible and a UUID is noise.
Recommendation: **do not use `google/uuid` for item or escalation ids.** Keep
the dependency only if a genuinely opaque internal token is ever needed (none is
in this spec); otherwise it can be dropped from `go.mod`. Escalation filenames
follow their own `<yyyy-mm-dd>-<slug>` scheme, also slug-based, also no UUID.

---

## 5. TUI surfaces (optional)

Recommendation: **defer the TUI for v1**; ship the scriptable surface first,
add a TUI only once the verbs are stable. Agents (the primary users) drive this
headless, so a TUI is a human-convenience layer, not a requirement. When added,
build it on `bubbletea`/`bubbles`/`lipgloss`, and hold to the rule that every
interactive action maps 1:1 to an existing non-interactive command:

| TUI surface | What it does | Scriptable equivalent |
|-------------|--------------|-----------------------|
| `anthill backlog browse` | scrollable, filterable list of items; drill into one | `backlog list` / `backlog show` |
| — set fields inline | edit `status`/`priority`/`workstream` on the selected item | `backlog set <id> k=v` |
| — claim/close from the list | pick an item and act | `backlog claim` / `backlog close` |
| `anthill triage review` | walk untriaged items, apply a classification per the workstream profile | a sequence of `backlog set` calls |
| `anthill escalation inbox` | browse records addressed to a tier, read Question, answer | `escalation list`/`show`/`answer` |

No TUI surface may perform a mutation that has no command-line equivalent — the
TUI is a viewport onto the verbs, never a privileged path.

---

## 6. Storage decision — sqlite index

**Recommendation: DEFER `modernc.org/sqlite`. Do not add it in v1.**

Reasoning:

- The corpus is tiny. A project backlog is dozens of items, not millions;
  reading and parsing every markdown file on each `list`/`next`/`validate` is a
  few milliseconds. There is no latency problem to solve.
- An index is a second source of truth. The whole design commits to files as
  the authority; a sqlite cache introduces a coherence obligation (invalidate
  on external `git pull`, hand-edit, branch switch) whose failure mode
  (stale/incorrect answers) is worse than the cost it removes.
- Pure-Go `modernc.org/sqlite` avoids cgo but not the coherence problem.

Add it only if a concrete need appears — e.g. cross-repo/global backlog
aggregation, or interactive TUI filtering over thousands of items — and even
then as a **rebuildable cache** (a `anthill index` command that derives the DB
from the files and can be blown away at any time), never as an authority. Until
then the files + in-memory scan are the store.

---

## 7. Open questions for the user

These are the taste-laden forks this proposal deliberately surfaces rather than
silently deciding:

1. **Approval gate through the CLI.** Should `backlog set <id> status=approved`
   be *allowed at all*, or must approval stay a human-only path (hand-edit or a
   dedicated gated verb)? The posture is propose-only; if the schema owner
   freely lets any caller set `approved`, an agent can self-approve and the
   gate is cosmetic. Options: (a) refuse `status=approved` via `set`, require a
   separate `approve` verb guarded by an explicit human-confirmation
   flag/interactive prompt; (b) allow it and rely on process discipline;
   (c) allow it but emit a loud stderr warning + require `--i-approve`.
   **Recommendation: (a)** — make the schema owner enforce the posture it
   documents.
    **Answer**: Separate approve verb
2. **Never-auto surface additions.** `show`, `init`, and `doctor` are new
   subcommands not literally in `bindings.md`'s target list, and adding a
   first-class subcommand is a never-auto change type. Approve these three as
   part of the v1 surface? **Recommendation: yes** — `show` is load-bearing for
   dispatch/triage, `init`/`doctor` are onboarding/health and read-only-ish.
    **Answer**: Yes, good additions
3. **`close --block` semantics.** Confirm block is non-terminal (file stays,
   `status: blocked`, no CHANGELOG line) while done/discard/remove are terminal
   (delete + CHANGELOG). This is my reading of `bindings.md` + the `dispatch`
   skill; worth an explicit yes.
    **Answer**: Confirmed
4. **Claim mechanism / locking.** With a **serial** parallelism posture, is an
   atomic compare-and-set on the `status` field sufficient, or do you want an
   explicit advisory lock (a `claimed-by`/`claimed-at` marker, a lockfile)?
   **Recommendation: CAS on status + a `claimed-at` timestamp field**, no
   lockfile — enough for serial, and orphan reclaim is `claim --force`. Revisit
   only if the posture goes parallel.
   **Answer**: Recommendation approved
5. **`--json` list shape.** A single JSON array (chosen here) vs newline-
   delimited JSON (NDJSON, one object per line) for streaming. **Recommendation:
   array** — the sets are small and an array is trivially `jq`-able; NDJSON only
   earns its keep at streaming scale.
    **Answer**: Array is fine
6. **`--hint` vs `--backlog` flag name.** `bindings.md` sketches
   `--backlog <hint>`; the schema field is `hint`. This spec uses `--hint`.
   Confirm the flag name (and whether `--backlog` should be a hidden alias).
    **Answer**: Support both with the hidden alias
7. **Title/value editability via `set`.** id is immutable, but may `title` and
   `value` be edited post-hoc via `set` (this spec allows it)? A title edit does
   **not** re-slug the id (id immutability wins), so the id can drift from the
   title. Accept that drift, or forbid title edits? **Recommendation: allow
   edits, accept drift** — the id is an opaque handle once assigned.
    **Answer**: Allow edits, accept drift
8. **`escalation` command group scope.** The escalate skill has controllers
   hand-authoring records today. Do you want the CLI to own escalation
   frontmatter (this spec's position — one schema owner for both backlog and
   escalations), or keep escalations convention-first and scope the CLI to
   backlog only? **Recommendation: CLI owns both** — same invariant-checking
   argument.
    **Answer**: Yes, CLI owns escalations too

---

## 8. Design rationale

- **One writer, checked at the write.** The entire value of a "schema owner" is
  that malformed state becomes unrepresentable through the sanctioned path.
  Hence: atomic temp-file+rename writes, validation before persist, and a
  `validate` command that certifies trees that were edited outside the tool.
- **Verbs match the consumers.** The command set is derived from what the
  `dispatch`/`triage`/`escalate` skills actually call — `list --ready`,
  `next`, `claim --next`, `close --done`, `set workstream=`, escalation
  `list`/`answer`/`apply`. The surface is shaped to those call sites, not to an
  abstract CRUD ideal; that is why, e.g., `next` and `claim --next` are
  distinct (select-without-taking vs take), matching dispatch's two-step select
  then claim.
- **Exit codes are an API.** The four failure classes (3/4/5/6) are exactly the
  branches a sweep loop needs; anything coarser forces text-parsing and anything
  finer is unused.
- **No implicit cross-file cascades.** `escalation raise --item` does not
  auto-block the item; `apply` does not auto-unblock it. Each file has one
  writer per operation, and the caller composes the two calls. This keeps every
  mutation independently auditable in `git diff` and avoids a half-applied
  cascade on interruption.
- **Files first, database never (yet).** The corpus is small and the coherence
  cost of a cache exceeds its benefit; deferring sqlite keeps the "files are
  truth" invariant literally true.
- **Headless is the default, TUI is a lens.** Because agents are the primary
  driver, the scriptable surface is complete on its own and any TUI is a strict
  viewport over the same verbs — no capability lives only behind the TUI.
```
