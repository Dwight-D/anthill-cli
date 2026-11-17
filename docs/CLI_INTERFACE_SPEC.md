# Anthill CLI — Interface Specification (PROPOSED)

Status: proposal for review. This document specifies the intended command
surface, output contracts, and invariant ownership of the `anthill` binary. No
Go implementation is prescribed here beyond what the existing bootstrap already
establishes (`spf13/cobra` root command, `internal/version`). Where a decision
is a judgment call, it is marked and the alternative is noted; the genuine forks
are collected in [Open questions](#7-open-questions-for-the-user).

Module: `github.com/Dwight-D/anthill-cli`. Binary: `anthill`.

**Backlog contract.** The backlog item schema, its lifecycle, and every
`anthill backlog` verb are specified in [`BACKLOG_V2_SPEC.md`](BACKLOG_V2_SPEC.md)
(schema 2). This document carries the global conventions (§2), the command
tree (§3), the escalation record contract (§3.9–§3.15), the lifecycle helpers
(§3.16–§3.17), id generation (§4), and the TUI (§5).

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
├── backlog                     (contract: BACKLOG_V2_SPEC.md §6)
│   ├── new                     create an intake item (status new)
│   ├── list                    filter / sort / project items
│   ├── show <id>               print one item
│   ├── next                    print the highest-ranked ready item
│   ├── triage <id> k=v…        new → triaged, moves the file into its workstream
│   ├── approve|fast-track|unapprove <id>…   set dispatch (user-gated)
│   ├── claim <id> | --next     triaged → in-progress, stamps claimed-by / session
│   ├── deliver <id> --evidence "…"          in-progress → validation (claimant only)
│   ├── block <id> "reason" / unblock <id> [--release]
│   ├── release <id>            in-progress → triaged, drops the claim
│   ├── park <id> "reason" / unpark <id>
│   ├── close <id> --done|--reject|--discard|--remove ["reason"]
│   ├── set <id> key=value…     free fields only; workstream= moves the file
│   ├── validate [--strict]     list every violation as <id>: <rule>: <message>
│   ├── migrate [--dry-run]     upgrade items to schema 2 (idempotent)
│   └── prune [--older-than d] [--dry-run]   purge discarded/
└── escalation
    ├── raise                   create an escalation record
    ├── list                    list records (filters: --to --status --item)
    ├── show* <id>              print one record
    ├── answer <id> --decision "…"   append ## Decision, set status: answered
    ├── apply <id>              append ## Applied, one-line LOG.md, delete the record
    ├── close <id> ["reason"]   close an unanswered record: one-line LOG.md, delete
    └── remove <id> ["reason"]  remove a stale record (any status): one-line LOG.md, delete
```

Notation below for each command: **Purpose · Args · Flags · Reads · Writes ·
Human output · `--json` output · Exit codes.**

---

### 3.1–3.8 `anthill backlog …`

Specified in [`BACKLOG_V2_SPEC.md`](BACKLOG_V2_SPEC.md): the item schema (§1),
the lifecycle and its verbs (§2), `disposition` (§3), `reporter` (§4),
claimant identity (§5), and the full verb surface with flags (§6). Every
backlog verb follows the global conventions of §2 here: `--json` output,
stdout/stderr split, and the exit-code table.

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

### 3.14 `anthill escalation close`

- **Purpose.** Close an unanswered (open) record without a decision — the
  question became moot, was resolved out-of-band, or went stale. The
  raise → answer → apply lifecycle is for records that get a decision; `close`
  is the sanctioned exit for records that never will.
- **Args.** `<id> ["<reason>"]`. The reason is an optional second positional
  (mirroring `backlog close`); when omitted, the LOG.md line falls back to the
  record's question.
- **Reads.** the record, `LOG.md`.
- **Writes.** adds one line to `escalations/LOG.md`
  (`<date> <file> — <to>/closed: <reason/question>`), then deletes the record
  file. A record never vanishes silently.
- **Human output.** `closed <id>` on stderr.
- **`--json`.** `{ "id": …, "closed": true, "logged": true }`.
- **Exit codes.** 0; 4 (not found); 6 (record is not `open` — an answered
  record is closed with `apply`, or force-deleted with `remove`).

---

### 3.15 `anthill escalation remove`

- **Purpose.** Delete a stale record regardless of status. The destructive
  escape hatch for records that no lifecycle verb fits any more (superseded,
  duplicated, orphaned by a recycled session).
- **Args.** `<id> ["<reason>"]`. The reason is an optional second positional;
  when omitted, the LOG.md line falls back to the record's question.
- **Reads.** the record, `LOG.md`.
- **Writes.** adds one line to `escalations/LOG.md`
  (`<date> <file> — <to>/removed: <reason/question>`), then deletes the record
  file. A record never vanishes silently.
- **Human output.** `removed <id>` on stderr.
- **`--json`.** `{ "id": …, "removed": true, "logged": true }`.
- **Exit codes.** 0; 4 (not found).

---

### 3.16 `anthill init` *(proposed lifecycle helper)*

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

### 3.17 `anthill doctor` *(proposed lifecycle helper)*

- **Purpose.** One-shot environment + integrity check: `.anthill/` is
  discoverable, required config files present, `workstreams.md` sweep-order
  names existing directories, the backlog clean under the same rule set as
  `anthill backlog validate` (every violation listed under the
  `backlog-valid` check as `<id>: <rule>: <message>`), `escalation` records
  well-formed, no answered-but-unapplied records (the failure mode the
  escalate skill calls out). Read-only.
- **Flags.** `--strict` (fail on warnings too).
- **`--json`.** `{ "ok": bool, "checks": [ { "section", "name", "ok", "detail" } ],
  "violations": [ { "id", "rule", "message" } ] }`.
- **Exit codes.** 0 (healthy); 3 (integrity problem).

---

## 4. Frontmatter & id ownership

### The item object

The backlog item's frontmatter, body sections, derived fields, and the
per-write validation gate are specified in
[`BACKLOG_V2_SPEC.md`](BACKLOG_V2_SPEC.md) §1–§2. Every write is atomic
(temp file + rename), so a rejected or crashed write never leaves a
half-file.

### Id generation

The item id is a **key** `<PREFIX>-N` (`ANT-74`): the installation's
`key-prefix` from `.anthill/backlog/workstreams.md` and a number allocated
sequentially per installation and never reused. The file is
`NNNN-<slug>.md`, the slug being the cosmetic kebab tail of the title (≤40
chars, follows a title edit). Every verb accepts the key (case-insensitive),
the bare number, or the `former-id` of a migrated item, and always prints the
canonical key. Allocation (scan + `sequence` + exclusive create), resolution,
`renumber`, and the validate rules are the contract of
[`ITEM_KEYS_SPEC.md`](ITEM_KEYS_SPEC.md) §1–§3.

**`google/uuid` reconciliation.** The id scheme is deliberately a short,
human-readable key, not a UUID — ids appear in dispatch briefs, CHANGELOG
lines, and `git log`, where `ANT-74` is legible and a UUID is noise.
Recommendation: **do not use `google/uuid` for item or escalation ids.** Keep
the dependency only if a genuinely opaque internal token is ever needed (none is
in this spec); otherwise it can be dropped from `go.mod`. Escalation filenames
follow their own `<yyyy-mm-dd>-<slug>` scheme, slug-based, also no UUID.

---

## 5. TUI — `anthill tui`

`anthill tui` opens a full-screen interactive browser (Bubble Tea, alt-screen)
over the harness's durable state, rooted at the same resolved `.anthill` root
as every other command (`--root` / `ANTHILL_ROOT` / upward search). It is a
human-convenience layer: agents keep driving the scriptable verbs, and no TUI
surface may perform a mutation that has no command-line equivalent — the TUI
is a viewport onto the verbs, never a privileged path.

**Bare invocation.** Running `anthill` with no subcommand also opens the TUI,
provided both stdin and stdout are terminals and an `.anthill` root is
discoverable; otherwise it prints the root help and exits 0, so a headless or
misdirected caller is never left hanging on an interactive screen. `anthill
--help`, `anthill help`, `--version`, and unknown-command errors are untouched
by this default — they never reach the bare-invocation path.

### Modes

The TUI is organised around top-level **modes** — switchable screens sharing
one layout. Each mode supplies the nav-list rows, the detail text for the
selected row, and the keypress actions that operate on it; the root model owns
everything visual (layout, pane focus, selection, scrolling, confirm prompts,
refresh, mode switching). The seam is the `Mode` interface in `internal/tui`:
adding a mode (Escalations, Agents, ...) is one struct implementing it plus
one entry in the registry — no root-model changes. The header band shows the
mode tabs (`1:Backlog ...`) with the active one highlighted; `[m]` cycles
modes and number keys select one directly.

Implemented modes: **Backlog**, **Escalations**. Agents is an anticipated
next mode.

### Layout

Three bands: a header (mode tabs + `[key] action` hints), a main area — the
selected row's full detail in a scrollable content pane on the left, the
mode's row list on the right (Backlog: `id  workstream  status` in sweep
order; Escalations: `id  to  status` sorted by id) — and a one-line status
bar carrying action results, errors, and confirm prompts.

**Wrapping.** Detail-pane content is word-wrapped to the pane's inner width
before it enters the viewport — cell-accurate (wide runes count their display
width), preferring word boundaries and hard-breaking tokens wider than the
pane (URLs, hashes) — so no detail text is ever clipped off-screen. A
terminal-width change re-wraps in place, preserving the scroll offset. List
rows, the header bands, and the status line stay single-line and
cell-truncated by design; everything a list row truncates is fully readable
in the detail pane.

### Keys (Backlog mode)

| Key | Action | Scriptable equivalent |
|-----|--------|-----------------------|
| `↑/↓` `j/k` | move selection / scroll detail | — |
| `tab`, `enter`/`esc` | switch pane focus | — |
| `r` | reload from disk | `backlog list` |
| `a` | approve the selected item (dispatch approved) | `backlog approve <id>` |
| `d` | close as done (item in validation) | `backlog close <id> --done` |
| `x` | close as discard (confirm: `y`) | `backlog close <id> --discard` |
| `X` | remove (confirm: `y`) | `backlog close <id> --remove` |
| `m` / `1..9` | cycle / select mode | — |
| `q` / `ctrl+c` | quit | — |

### Keys (Escalations mode)

| Key | Action | Scriptable equivalent |
|-----|--------|-----------------------|
| `a` | answer the selected (open) record via the text overlay | `escalation answer <id> --decision <text>` |
| `p` | apply the selected (answered) record | `escalation apply <id>` |
| `x` | close an unanswered record (confirm: `y`) | `escalation close <id>` |
| `X` | remove regardless of status (confirm: `y`) | `escalation remove <id>` |

Navigation, refresh, mode, and quit keys are the root model's and identical in
every mode. Raising still needs structured input and stays on the scriptable
`escalation raise` verb.

**Text overlay.** A free-text action (`a` answer) opens a centered text-entry
overlay over the main area; while open it owns every key. `ctrl+s` submits
(`enter` inserts a newline in the multi-line textarea, so it cannot be the
submit key) and `esc` cancels; the header hint line swaps to the overlay's
keys while it is open. An empty or whitespace-only submission is rejected in
the status line and the overlay stays open. Submission runs the same domain
operation as the scriptable verb, then reloads in place — the answered record
shows its `answered` status and `## Decision` section in the detail pane.

Destructive actions (discard, remove, close) arm a y/n prompt in the status line;
any key other than `y` cancels. After every action the list reloads in place,
preserving the selection by item id where the row survives and clamping the
index otherwise. Preconditions are the verbs' own (approve requires `idea`,
done requires `in-progress`); violations surface in the status bar, exactly
as the CLI would exit non-zero.

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
    **Answer**: Allow edits, accept drift. (Under the key scheme the id is
    `ANT-N` and cannot drift; a title edit renames only the cosmetic slug tail
    of the filename — `ITEM_KEYS_SPEC.md` §2.1.)
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
