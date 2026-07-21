---
sweep-order: bugs, cli, dev, process
never-implicit:
change-types: doc, tooling, bugfix, audit, verify, new-command, new-flag, rename, removal, design, subjective
never-auto-change-types: new-command, new-flag, rename, removal, design, subjective
---

# Backlog workstreams — Anthill CLI

The project's workstream definitions: what belongs in each, how its items are
triaged (profile), how they get implemented (dispatch route), and what evidence
closes them. Loaded by the `triage`/`dispatch`/`expedite` skills on invocation.

The frontmatter above is the machine-readable contract:
- **sweep-order** — the order bare `dispatch` walks workstreams.
- **never-implicit** — workstreams only ever dispatched deliberately (never
  swept by bare `dispatch`). Empty here: every stream is sweepable.
- **change-types** — the project's change-type vocabulary: the menu triage
  picks from. The CLI does not hardcode it; an item using a change-type outside
  this set is a `validate`/`doctor` **warning**, never a hard failure.
- **never-auto-change-types** — the subset of change-types that can never carry
  `disposition: AUTO` (permanent, cross-cutting, or taste-laden). Enforced as a
  hard `disposition-coherence` violation under `validate --strict` / `doctor`.

A workstream's identity is its **directory name** under
`.anthill/backlog/`. Every stream below must have a matching directory.

---

## cli

The product: the **Anthill CLI** — the command-line tool that owns the backlog
and escalation schemas and the verbs the Anthill skills bind to. Its surface:
the `backlog` verbs (`new`, `list`, `set`, `next`, `claim`, `close`,
`validate`), the escalation verbs, per-item frontmatter invariant-checking, id
generation, and the semantics/defaults/output of each command. When it exists,
it becomes the **schema owner** named in `bindings.md`.

- **Triage profile:** improvement gates —
  - *Value gate:* benefit ÷ permanent cognitive cost (what the change adds to
    the surface everyone must learn — not implementation effort). Heuristics:
    recurring not one-off · distinct not redundant · painful workaround ·
    composable · smallest footprint that delivers it. Prefer the cheapest
    change type that works.
  - *Safety gates* (for an AUTO recommendation): additive-or-reversible ·
    verifiable (a concrete `verify`) · bounded scope · unambiguous spec.
  - *Surface dedup:* before adding a new command/subcommand/flag, check the
    existing CLI surface for a verb that already covers it.
- **Never-auto:** adding a new first-class command/subcommand or global flag;
  changing the backlog/escalation **frontmatter schema** or the **id scheme**
  (the CLI is the schema owner — schema changes are permanent and cross-cutting);
  changing a command's default output format. These cap at human review no
  matter how safe.
- **Dispatch route:** `dispatch` skill.
- **Evidence:** `go build ./...` exit 0; `go test ./...` exit 0; the item's
  `verify` test.

## dev

Development-process tooling: the test harness, fixtures, CI wiring, lint/format
config (`gofmt`/`go vet`), profiling and diagnostics — everything that speeds
the development loop itself.

- **Triage profile:** improvement gates as for `cli`, minus the CLI-surface
  dedup. Weight tooling value by how much it unblocks the agentic loop.
- **Never-auto:** changes to safety invariants or permission surfaces.
- **Dispatch route:** `dispatch` skill.
- **Evidence:** the headless test / exit code named in `verify` (typically
  `go test ./...` exit 0).

## process

How information and work flow through the project: docs, playbooks,
codification, backlog and Anthill configuration. Changes to Anthill
*mechanisms* themselves are rare and get flagged upstream to the framework
home (see `.anthill/framework.md`) rather than patched locally — local
divergence across installations is the failure mode to avoid.

- **Triage profile:** improvement gates; plus the instruction-file rule —
  *reference material* goes to a scoped home that loads only when relevant;
  only a *standing behavioral directive* earns a place in an always-on file
  (CLAUDE.md), and the bar is high.
- **Never-auto:** edits to always-on instruction files (CLAUDE.md).
- **Dispatch route:** `dispatch` skill (mostly `doc` change-type). A learning
  lands in its one durable home — never a second home for a fact that has one.
- **Evidence:** the doc/codification exists in its durable home and nothing
  else claims to own the same fact.

## bugs

Defects in intended existing behavior, regardless of component. Routing rule:
restore-intended-behavior → here; capability/improvement work → the
component's workstream.

- **Triage profile:** light — the value gate auto-passes (correctness is its
  own value). Requires a reproduction and a headless `verify`.
- **Never-auto:** behavior changes without a regression guard.
- **Dispatch route:** `dispatch` skill. Default `priority: high`.
- **Evidence:** the repro fails before the fix and passes after;
  `go build ./...` and `go test ./...` exit 0.

---

## Judgment signals (accrued from triage decisions — read before classifying)

Starts empty. When a triage decision generalizes into a reusable rule, append
it here as one bullet so the next triage inherits it.
