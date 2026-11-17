# Backlog v2 — task model, lifecycle, and CLI contract

Status: **PROPOSED (draft 2026-11-17)** — awaiting the user's decisions listed in
§10. Nothing here is implemented. Once accepted this document becomes the
backlog contract that `anthill` (schema owner) enforces and the framework
skills (`submit`, `triage`, `expedite`, `dispatch`, `dispatch-receive`,
`dispatch-loop`, `escalate`, `supervisor`, `autonomous`, `rehydrate`) follow.
The bootstrap contract (`anthill-fw/CLI_SPEC.md`) and the escalation record
contract (`CLI_INTERFACE_SPEC.md` §5) are unaffected.

Design goals, in the user's words: work goes through the item with an
accurate state; items are rich enough to differ from one another; the queue
is queryable along several independent axes (who filed it, has the user
approved or fast-tracked it, can an agent one-shot it, what is live right
now); every state change goes through the CLI so it is typed and stamped;
and the stamps are enough to build a "what is being worked on, by which
session" dashboard.

---

## 1. The item

One markdown file per item. YAML frontmatter is the typed record, owned and
written only by the CLI. The body is structured markdown with named sections;
humans and agents may edit the body freely, the CLI edits only the sections
it owns.

### 1.1 Frontmatter

| key | type | set by | meaning |
|---|---|---|---|
| `title` | string, required | submitter | One line. The id (filename stem) is derived from it once and never changes. |
| `summary` | string | submitter or triage | One or two sentences: what the change is. Shown in every list view. Required from `triaged` onward. |
| `value` | string, required | submitter | The pain it removes or the potential it unlocks. The triage currency. |
| `workstream` | enum (project dirs) | triage | Absent while in `intake/`; equals the containing directory afterwards. |
| `status` | enum §2 | CLI transitions | Where the item is in its lifecycle. |
| `dispatch` | `proposed` \| `approved` \| `fast-tracked` | user-gated verbs | The user's go/no-go, independent of status. `fast-tracked` = approved *and* pinned to the top of every queue. |
| `disposition` | `auto` \| `design` \| `interactive` \| `open` | triage (user may override) | How we expect to implement it — §3. |
| `priority` | `high` \| `normal` \| `low` | triage / user | Ordering within equal dispatch tier. Default `normal`. |
| `reporter` | `user` \| `supervisor` \| `worker` | submitter | Who filed it — §4. |
| `change-type` | project vocabulary (`workstreams.md`) | triage | What kind of change; out-of-vocabulary values warn. |
| `risk` | `additive` \| `reversible` \| `behavior-change` \| `subjective` | triage | Reversibility class. |
| `verify` | string | triage / refined by worker | Headless acceptance test, or `none`. Required for `disposition: auto`. |
| `source` | string | submitter | Free text: where it came up (a session, a doc, a conversation). |
| `hint` | string | submitter | Non-binding workstream hint; stripped when the item leaves `intake/`. |
| `note` | string | anyone via `set` | Free text. Carries no machine semantics. |
| `created-at` | RFC3339 UTC | CLI on `new` | |
| `triaged-at` | RFC3339 UTC | CLI on `triage` | |
| `claimed-at` | RFC3339 UTC | CLI on `claim` | Cleared on release. |
| `claimed-by` | string | CLI on `claim` | Handle of the claimant — §5. Cleared on release. |
| `session` | string | CLI on `claim` | Harness session id of the claimant, when known — §5. |
| `delivered-at` | RFC3339 UTC | CLI on `deliver` | |
| `closed-at` | RFC3339 UTC | CLI on `close` | |

`id`, `path`, `body`, and the derived booleans below are never serialised.

Derived:
- **ready** — `status == triaged && dispatch != proposed && disposition == auto && verify not in ("", "none")`. The only definition of "an agent may pick this up unattended".
- **active** — `status in (in-progress, validation)`.

### 1.2 Body sections

The body is markdown under fixed `##` headings. Unknown headings are preserved
verbatim. The CLI creates the skeleton on `new` and appends to `## Evidence`
and `## Log`; it never rewrites prose.

```
## Description
<full description: context, what to change, constraints, pointers. The
 self-contained brief a cold worker implements from.>

## Acceptance
<what "done" looks like, expanding `verify`; the evidence a controller will
 check.>

## Triage
<triage's reasoning: why this workstream / disposition / priority; the
 value verdict in prose.>

## Evidence
<appended by `deliver --evidence`: what was run, exit codes, artifacts.>

## Log
<one line per CLI transition, appended by the CLI:
 `- 2026-11-17T10:03:12Z claim by supervisor:worker-cli (session 2f1c…)`>
```

### 1.3 Layout

```
.anthill/backlog/
  workstreams.md          project vocabulary (frontmatter) + stream profiles
  bindings.md             how this project runs the mechanism
  CHANGELOG.md            one line per closed item
  intake/                 status new, no workstream
  <workstream>/           triaged … validation, blocked, parked
  closed/                 done and discarded items, kept for history (§10 D1)
```

---

## 2. Lifecycle (`status`)

```
            triage                 claim              deliver            close --done
  new ─────────────► triaged ─────────► in-progress ─────► validation ─────────► done
                       ▲   ▲               │ ▲               │
                       │   │   block       │ │ release       │ close --reject
                       │   └───────────────┘ └───────────────┘ (back to in-progress)
                       │          blocked ──── unblock ───┘
                       └──── park / unpark ──── parked
  any ─── close --discard ───► discarded      any ─── close --remove ───► (file deleted)
```

| from | to | verb | guard |
|---|---|---|---|
| `new` | `triaged` | `triage <id> workstream=… …` | required triage fields present (§6.3); moves the file into the workstream dir; stamps `triaged-at` |
| `triaged` | `in-progress` | `claim` | `ready`, or `--force`; CAS on status; stamps `claimed-at`, `claimed-by`, `session` |
| `in-progress` | `validation` | `deliver --evidence "…"` | claimant only unless `--force`; stamps `delivered-at`; appends Evidence |
| `validation` | `done` | `close --done ["…"]` | controller/user; moves to `closed/`; stamps `closed-at`; CHANGELOG line |
| `validation` | `in-progress` | `close --reject "…"` | evidence insufficient; keeps claim; appends Log |
| `in-progress` | `blocked` | `block "<reason>"` | reason appended to Log, `note` untouched |
| `blocked` | `in-progress` | `unblock` | resumes the same claim; `--release` instead returns it to `triaged` |
| `in-progress` | `triaged` | `release` | drops claim fields (worker died, item reassigned) |
| `triaged`/`new` | `parked` | `park "<reason>"` | |
| `parked` | prior status | `unpark` | |
| any | `discarded` | `close --discard "…"` | moves to `closed/`; CHANGELOG line |
| any | — | `close --remove "…"` | filed in error; file deleted; CHANGELOG line |

Rules:
- `set status=…` is refused. Status moves only through the verbs above, so
  every transition is guarded, stamped, and logged.
- `set dispatch=…` is refused. Dispatch moves only through `approve`,
  `fast-track`, `unapprove`.
- `--force` exists on `claim`, `deliver`, `close --done` for the human who
  knows better; it is logged as forced.
- Workstream changes after triage go through `set workstream=…` (moves the
  file) as today.

### 2.1 `dispatch` is orthogonal

`dispatch` answers "has the user said go?"; `status` answers "where is it?".
A `new` item can be `fast-tracked` (the user wants it badly before triage
even looks); a `triaged` item can sit `proposed` for months. `approve` and
`fast-track` are user-gated: the `expedite` skill may call them only because
the user's invocation is the approval; no agent self-approves.

---

## 3. `disposition` — how we expect to implement it

| value | meaning | consequence |
|---|---|---|
| `auto` | A properly described, bounded change an agent can one-shot from the item alone. | Eligible for unattended dispatch (`ready`) once approved. Must have a non-`none` `verify` and a change-type outside `never-auto-change-types`. |
| `design` | Needs a design session (spec, decisions, maybe a prototype) before anyone implements. | Not `ready`. The design session's output is a spec and a re-disposition to `auto` or `interactive`, or a split into several items. |
| `interactive` | Expected to be done in a live, iterative back-and-forth with the user. | Not `ready`. Surfaces in "pick a session topic" views. |
| `open` | A reminder of a problem worth solving; no clear approach yet. | Not `ready`; never dispatched; periodically revisited by triage. |

Triage sets it; the user may override with `set disposition=…`. Coherence is
validated: `auto` without a real `verify` or with a never-auto change-type is
a hard violation.

---

## 4. `reporter` — who filed it

`user`, `supervisor`, or `worker`. Precedence on `new`:

1. `--reporter <value>` flag (the skills always pass it: `submit` passes what
   the tier it runs in dictates; a supervisor session passes `supervisor`;
   a worker brief passes `worker`).
2. `ANTHILL_REPORTER` environment variable (set by launchers such as
   `tools/supervise.*`).
3. Inference: `CLAUDECODE` set in the environment → `worker`; otherwise
   `user`.

An interactive session filing an item at the user's request passes
`--reporter user`: the user filed it, the agent typed it.

---

## 5. Claimant identity and session correlation

On `claim` the CLI records:

- `claimed-by` — `--by <handle>` if given; else `ANTHILL_ACTOR` env; else
  `<reporter-style role>:<CLAUDE_CODE_SESSION_ID prefix>`; else the OS user
  name.
- `session` — `--session <id>` if given; else `CLAUDE_CODE_SESSION_ID` from
  the environment (Claude Code exports it into every tool subprocess,
  verified 2026-11-17); else empty.

Subagents spawned inside a session (the supervisor's workers, a dispatcher's
workers) run with the **parent's** session id and a distinct agent id; the
agent id is not in the environment. So for a worker, `session` identifies the
controlling session and `claimed-by` (the agent name the brief assigns)
identifies the worker inside it. That pair is the correlation key.

Everything else about a session lives outside the item, in a runtime-only
ledger the CLI owns:

```
.anthill/sessions/<session-id>[.<agent-id>].json      (untracked)
{ "session": "…", "agent": "<agent id or empty>", "handle": "cli-worker",
  "item": "<id>", "role": "worker", "transcript": "<path>", "cwd": "…",
  "started": "…", "last-seen": "…" }
```

Written by `anthill session link` and refreshed by `anthill session touch`,
both called from the Claude Code side (§8.2), never by the skills. Transcript
locations, verified 2026-11-17: a session's transcript is
`~/.claude/projects/<cwd-slug>/<session_id>.jsonl`; its subagents' are
`<session_id>/subagents/agent-<agentId>.jsonl` next to it, with a
`.meta.json` carrying the agent's name. A dashboard therefore needs only the
session id to enumerate every transcript that touched an item, and the
handle to pick the right one. Liveness = `last-seen` age, with the transcript
file's mtime as a fallback.

---

## 6. CLI surface

Every verb: `--json` structured output, exit codes 0 ok · 1 internal ·
2 usage · 3 validation · 4 not-found · 5 conflict · 6 precondition.

### 6.1 Intake

```
anthill backlog new --title "…" --value "…" [--summary "…"] [--description "…"|--description-file f]
                    [--source "…"] [--hint <ws>] [--reporter user|supervisor|worker] [--priority …]
```
Creates `intake/<id>.md` with `status: new`, `dispatch: proposed`,
`created-at`, the body skeleton, and the reporter per §4. Prints the id.

### 6.2 Reading

```
anthill backlog show <id>                       full record + body
anthill backlog list [filters] [--sort …] [--limit n] [--fields a,b,c]
anthill backlog next [--workstream ws]          highest-ranked ready item
```

Filters, all repeatable and AND-ed: `--status`, `--dispatch`, `--disposition`,
`--reporter`, `--priority`, `--workstream`, `--change-type`, `--risk`,
`--claimed-by`, `--session`, `--search <substring over title/summary/value>`.
Presets: `--ready` (derived), `--active` (in-progress + validation),
`--untriaged` (status new), `--closed` (include `closed/`).

Sort keys: `rank` (default: fast-tracked → priority → workstream sweep-order
→ oldest first), `priority`, `age`, `updated`, `id`; `--reverse`.

Examples the design must satisfy:
```
anthill backlog list --ready                               agent-implementable, best first
anthill backlog list --reporter user --dispatch fast-tracked
anthill backlog list --active --fields id,summary,claimed-by,session,claimed-at
anthill backlog list --disposition interactive --dispatch approved   topics for a live session
anthill backlog list --disposition design --sort priority            design sessions to schedule
```

### 6.3 Triage

```
anthill backlog triage <id> workstream=<ws> change-type=… risk=… verify="…" disposition=… [priority=…] [summary="…"]
```
One atomic transition `new → triaged`: validates the full triaged field set,
moves the file, strips `hint`, stamps `triaged-at`, appends Log. Triage
reasoning goes in the `## Triage` body section (the skill writes it before
calling the verb, or passes `--triage-note "…"`). Refusing an item at triage
is `close --discard`; deferring is `park`.

### 6.4 Approval (user-gated)

```
anthill backlog approve <id>…        dispatch → approved
anthill backlog fast-track <id>…     dispatch → fast-tracked
anthill backlog unapprove <id>…      dispatch → proposed
```
Multiple ids per call. No `--yes` ceremony: the verb is the gate.

### 6.5 Working

```
anthill backlog claim <id>|--next [--workstream ws] [--by h] [--session s] [--force]
anthill backlog deliver <id> --evidence "…" [--evidence-file f] [--force]
anthill backlog block <id> "<reason>"
anthill backlog unblock <id> [--release]
anthill backlog release <id>
anthill backlog park <id> "<reason>" / unpark <id>
anthill backlog close <id> --done ["…"] | --reject "…" | --discard "…" | --remove "…"
anthill backlog set <id> key=value …    free fields only: title summary value source hint
                                         note priority disposition change-type risk verify
                                         workstream (moves file)
```

### 6.6 Health and migration

```
anthill backlog validate [--strict]     per-item violations, rule names, ids
anthill doctor                           same rules, same listing (never a bare count)
anthill backlog migrate [--dry-run]      v1 → v2 (§9), idempotent
anthill session link|touch|list|prune    §5
```

---

## 7. How the skills use it

Only the deltas that matter for the contract; each skill's own text carries
the full procedure.

- **submit** — `new` with `--reporter` per tier and a `--summary` when the
  submitter can give one. Still deliberately dumb: title + value is enough.
- **triage** — writes `## Description` (expanding the submitter's words into a
  cold-worker brief when it can), `## Acceptance`, `## Triage`, then one
  `triage` call. `disposition` is the key judgment: `auto` only when the
  description alone would let a fresh worker finish and `verify` is real.
- **expedite** — `fast-track` (not `approved` + a note token). The user's
  invocation is the gate.
- **dispatch / dispatch-loop** (sender) — `claim --next --by dispatcher:<name>`
  → brief carries the id and `show` output → worker `deliver`s → sender
  verifies `## Evidence` against `## Acceptance` and `verify` → `close --done`
  or `close --reject`. Senders still own `close`, `block`, `release`.
- **dispatch-receive** (worker) — may call `deliver` (and `block` for a hard
  stop) on *its own item only*; never `claim`, `close`, `approve`, `set`.
- **supervisor** — brief template gains an `ITEM` section: the backlog id the
  task implements, or `none` with a one-line reason; the spawning step says
  "file one if the work will outlive this session". The brief instructs the
  worker to `claim <id> --by <agent-name>` on pickup and `deliver` on finish;
  the supervisor closes after verifying evidence. The task board row carries
  the id. `rehydrate` snapshots the id per row so a cold successor can
  `list --active` and rebuild.
- **autonomous** — the generic worker contract gains: "if the brief names an
  item, `claim` it before the first edit and `deliver` before reporting".
- **escalate** — `block` with the escalation file in the reason; apply =
  `unblock`.

The invariant becomes: **controllers own close, approve, and release; a
worker may only advance the item it holds (`claim` on pickup when the brief
hands it one, `deliver`, `block`).** The CLI enforces it mechanically with
the claimant check on `deliver`.

---

## 8. Dashboard

### 8.1 TUI "Now" mode (`anthill` → mode 3)

Single-pane board polled every 2 s (`tea.Tick` → reload): one row per
active item — id · summary · status · claimed-by · session (short) ·
last-seen age · workstream; a second block for `validation` items awaiting a
controller; a footer with counts (ready / proposed / blocked). Actions:
`enter` show, `d` close --done (confirm), `j` close --reject (input),
`r` release (confirm), `o` open transcript path in `$EDITOR`/`less`.

### 8.2 Claude Code plugin surface (`anthill-claude`)

`anthill-claude` is today a per-user plugin with one hooks module and no
classic hooks. Two mechanisms, which can coexist in its `hooks/hooks.json`:

- **Classic PostToolUse hook** (works in every mode, including `-p` and
  non-mod consumers): matcher `Bash` (and a `PowerShell` twin on Windows)
  with `"if": "Bash(anthill backlog *)"`; stdin gives `session_id`,
  `transcript_path`, `cwd`, `tool_input.command`, `tool_response.stdout`.
  The script parses the verb and id from the command and runs
  `anthill session link --session … --transcript … --cwd … --item … --verb …`.
  It does not see the agent id.
- **Mod `tool.call` handler** in `register.tsx`: sees `e.agentId` (set for
  subagents, undefined for the main session) and the same tool args, so it
  can key the ledger on `(session_id, agent_id)`, and it can render a
  `Pane` beside the chat showing `anthill backlog list --active --json` +
  `anthill session list --json`, refreshed on `turn.complete`.

Proposal: ship the classic hook first (correlation for everyone, including
headless dispatchers), add the mod pane second (richer, in-session). Both
call the same CLI verb so the ledger has one writer.

---

## 9. Migration (v1 → v2)

This is the one section allowed to speak in terms of the previous design.

`anthill backlog migrate` rewrites every item in place, idempotently, and
stamps `schema: 2` into `workstreams.md` frontmatter so `validate` knows which
rules apply. Mapping:

| v1 | v2 |
|---|---|
| `status: idea` in `intake/` | `status: new` |
| `status: idea` in a workstream dir | `status: triaged` |
| `status: approved` | `status: triaged`, `dispatch: approved` |
| `note` beginning with `expedited` | `dispatch: fast-tracked`; token removed from note |
| `status: in-progress` / `blocked` / `parked` | unchanged status; `dispatch: approved` |
| `status: done` (never stored in practice) | moved to `closed/` |
| `disposition: AUTO` | `auto` |
| `disposition: REVIEW` | `auto` if `verify` is real and change-type is not never-auto, else `design` |
| `disposition: DISCARD` | `open`, `dispatch: proposed` (nothing is deleted by migration) |
| `value-verdict: X — why` | removed from frontmatter; `why` appended to `## Triage` |
| `note: needs-spec` qualifier | `disposition: design`; token removed |
| `priority` absent | `normal` |
| `reporter` absent | `user` (overridable with `--default-reporter`) |
| `created-at` absent | file's git first-commit date if available, else the migration time |
| empty body | the §1.2 skeleton, `## Description` seeded with `value` |

Consumer action after upgrading the CLI: run `anthill backlog migrate`, then
`anthill sync` for the skills, then re-derive `bindings.md` from the new
template (`sync` never touches `.anthill/`).

---

## 10. Decisions needed from the user

- **D1 Closed items: archive or delete?** Proposal: `closed/` keeps done and
  discarded items with their Log and Evidence (history for the dashboard and
  for "what did we do last month"); `--remove` still deletes. Alternative:
  keep today's delete-on-close and rely on `CHANGELOG.md`.
- **D2 Status set.** Proposal: `new · triaged · in-progress · validation ·
  done · blocked · parked · discarded`. Alternative names: `validation` →
  `review`.
- **D3 Disposition names.** Proposal: `auto · design · interactive · open`.
- **D4 Drop `value-verdict`.** Its three outcomes are now actions (triage /
  park / discard) and its reasoning lives in `## Triage`. Keep it only if
  you want HOLD-vs-ADVANCE queryable.
- **D5 Reporter enum.** Proposal: `user · supervisor · worker` with the §4
  inference. Add `dispatcher` as a fourth value?
- **D6 Priority.** Proposal: add `low` (three levels). Alternative: keep
  `high · normal`.
- **D7 Verb names.** `triage`, `deliver`, `block`/`unblock`, `release`,
  `park`/`unpark`, `fast-track`, `unapprove`, `close --reject`, `migrate`,
  `session link|touch|list|prune`. Any you want renamed before they exist?
- **D8 Worker writes.** Confirm the relaxed invariant in §7: workers may
  `claim` (when the brief hands them an id), `deliver`, and `block` their own
  item; controllers keep `close`, `approve`, `release`.
- **D9 Phasing.** Proposal in §11; the CLI phase is a minor release (0.1.0)
  because the schema changes.
- **D10 Session ledger writer.** Classic PostToolUse hook first, mod pane
  second (§8.2), both feeding `anthill session link`. Alternative: mod only.

---

## 11. Phasing

1. **CLI 0.1.0** (`anthill-cli`, one branch): schema + enums + transitions +
   Log/Evidence sections + `triage`/`deliver`/`block`/`release`/`park`/
   `fast-track`/`unapprove`/`migrate` + list filters/sort + `validate`/`doctor`
   per-item listing + `claimed-by`/`session` stamping. Tests: e2e per verb,
   transition table, migrate fixtures from this installation's and
   battlehounds' real items. Update `CLI_INTERFACE_SPEC.md` to v2.
2. **Framework** (`anthill-fw`, one branch): skills per §7, templates
   (`bindings.md`, `README.md`, `workstreams.md`, `brief-template.md`,
   `supervisor/bindings.md`, `dispatch/ledger.md`, `escalations/README.md`),
   `INSTALLATION.md` step 6, `CLAUDE.template.md` + fw `CLAUDE.md` invariant
   wording, `CHANGELOG.md` with the consumer action. Then re-vendor into the
   CLI (user-gated ref bump) and `anthill sync` this installation; migrate
   its items.
3. **Sessions + TUI Now mode** (`anthill-cli` 0.2.0): `session` verbs, hook
   script shipped under `tools/hooks/`, Now mode.
4. **Plugin panel** (`anthill-claude`): hook registration + panel.
