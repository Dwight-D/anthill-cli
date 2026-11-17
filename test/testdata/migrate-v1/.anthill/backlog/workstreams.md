---
sweep-order: bugs, framework, cli, dev, process
never-implicit:
change-types: doc, tooling, bugfix, audit, verify, new-command, new-flag, new-skill, contract-edit, rename, removal, design, subjective
never-auto-change-types: new-command, new-flag, new-skill, contract-edit, rename, removal, design, subjective
---

# Backlog workstreams — Anthill (framework + CLI)

The project's workstream definitions: what belongs in each, how its items are
triaged (profile), how they get implemented (dispatch route), and what evidence
closes them. Loaded by the `triage`/`dispatch`/`expedite` skills on invocation.

This installation coordinates work on **two sibling repositories**:
`anthill-fw/` (the framework's upstream template source — structured human
language, no build) and `anthill-cli/` (the Go companion CLI that embeds the
template and owns the backlog/escalation schemas). Every item names which repo
it lands in; an item spanning both is split at triage into one item per repo,
cross-referenced in `note`, with the framework side dispatched first (the CLI
embeds the framework template, so the template ref moves before the embed).

The frontmatter above is the machine-readable contract:
- **sweep-order** — the order bare `dispatch` walks workstreams.
- **never-implicit** — workstreams only ever dispatched deliberately (never
  swept by bare `dispatch`). Empty here: every stream is sweepable.
- **change-types** — the project's change-type vocabulary: the menu triage
  picks from. An item using a change-type outside this set is a
  `validate`/`doctor` **warning**, never a hard failure.
- **never-auto-change-types** — the subset that can never carry
  `disposition: AUTO` (permanent, cross-cutting, or taste-laden). Enforced as a
  hard `disposition-coherence` violation under `validate --strict` / `doctor`.

A workstream's identity is its **directory name** under
`.anthill/backlog/`. Every stream below must have a matching directory.

---

## framework

The framework itself, in `anthill-fw/`: the eleven general-tier skill
contracts (`.claude/skills/`), the `.anthill/` placeholder templates, the
adoption docs (`README.md`, `BOOTSTRAP.md`, `INSTALLATION.md`, `CLI_SPEC.md`),
and `CLAUDE.template.md`. The artifact is structured human language — editing
it is framework maintenance that propagates to every consumer installation on
their next sync, so changes are treated as upstream releases.

- **Triage profile:** improvement gates —
  - *Value gate:* benefit ÷ permanent cognitive cost for **every consumer**
    (a framework change taxes all installations, not just this one).
    Heuristics: recurring not one-off · distinct not redundant · painful
    workaround · composable · smallest footprint that delivers it.
  - *Safety gates* (for an AUTO recommendation): additive-or-reversible ·
    verifiable (a concrete `verify`) · bounded scope · unambiguous spec.
  - *Two-tier check:* does this belong in the general tier at all, or is it
    per-project config the `.anthill/` templates should teach instead?
  - *Self-containment check:* the changed artifact must still read with zero
    prior context, never defined by contrast with a previous design.
- **Never-auto:** adding/removing/renaming a skill; any `contract-edit` that
  changes a skill's invariants, roles, or cross-skill protocol; changes to the
  two-tier split (what is upstream-owned vs. project-owned); template schema
  changes. These cap at human review no matter how safe.
- **Dispatch route:** `dispatch` skill. Workers work inside `anthill-fw/` on
  its own branch discipline (see `resources.md`).
- **Evidence:** the edited artifact reads self-contained; template character
  preserved (placeholder quote blocks and `<angle-bracket>` fill-ins intact in
  `.anthill/` templates); cross-references between skills/docs still resolve;
  `CHANGELOG.md` updated when the change is a consumer-visible release; the
  item's own `verify`.

## cli

The product: the **Anthill CLI** in `anthill-cli/` (Go) — the schema owner for
backlog/escalation frontmatter and the bootstrap surface (`bootstrap`,
`scaffold`, `init`, `doctor`, `sync`, `version`, `validate`, the `backlog` and
`escalation` verbs). It embeds the framework template (`go:embed`, pinned ref)
— framework releases reach consumers through a CLI re-embed + release.

- **Triage profile:** improvement gates —
  - *Value gate:* benefit ÷ permanent cognitive cost (what the change adds to
    the surface everyone must learn — not implementation effort). Prefer the
    cheapest change type that works.
  - *Safety gates* (for an AUTO recommendation): additive-or-reversible ·
    verifiable (a concrete `verify`) · bounded scope · unambiguous spec.
  - *Surface dedup:* before adding a new command/subcommand/flag, check the
    existing CLI surface for a verb that already covers it.
  - *Contract check:* does the change keep the CLI conformant with
    `anthill-fw/CLI_SPEC.md`? A divergence is a `framework` item (change the
    spec) or a `bugs` item (fix the CLI), never silently absorbed.
- **Never-auto:** adding a new first-class command/subcommand or global flag;
  changing the backlog/escalation **frontmatter schema** or the **id scheme**
  (schema changes are permanent and cross-cutting); changing a command's
  default output format; bumping the embedded template ref (that is a release
  act).
- **Dispatch route:** `dispatch` skill. Workers work inside `anthill-cli/`.
- **Evidence:** `go build ./...` exit 0 and `go test ./...` exit 0, run in
  `anthill-cli/`; the item's `verify` test.

## dev

Development-process tooling for either repo: test harnesses, fixtures, CI
wiring, lint/format config, profiling and diagnostics — everything that speeds
the development loop itself.

- **Triage profile:** improvement gates as for `cli`, minus the CLI-surface
  dedup. Weight tooling value by how much it unblocks the agentic loop.
- **Never-auto:** changes to safety invariants or permission surfaces.
- **Dispatch route:** `dispatch` skill.
- **Evidence:** the headless test / exit code named in `verify` (for Go
  tooling, typically `go test ./...` exit 0 in `anthill-cli/`).

## process

How information and work flow through the project: docs, playbooks,
codification, backlog and Anthill configuration — **this installation's own
`.anthill/` config**, not the templates in `anthill-fw/` (those are `framework`
items). This project is both the framework's maintainer and a consumer of it:
a gap felt while *using* the harness here is filed as a `framework` item and
fixed at the source, never patched by locally editing a skill copy.

- **Triage profile:** improvement gates; plus the instruction-file rule —
  *reference material* goes to a scoped home that loads only when relevant;
  only a *standing behavioral directive* earns a place in an always-on file
  (CLAUDE.md), and the bar is high.
- **Never-auto:** edits to always-on instruction files (any CLAUDE.md).
- **Dispatch route:** `dispatch` skill (mostly `doc` change-type). A learning
  lands in its one durable home — never a second home for a fact that has one.
- **Evidence:** the doc/codification exists in its durable home and nothing
  else claims to own the same fact.

## bugs

Defects in intended existing behavior, in either repo. Routing rule:
restore-intended-behavior → here; capability/improvement work → the
component's workstream. A CLI behavior that contradicts `CLI_SPEC.md` is a bug
here; a spec that no longer says what we want is a `framework` item.

- **Triage profile:** light — the value gate auto-passes (correctness is its
  own value). Requires a reproduction and a headless `verify` (for `anthill-fw`
  doc defects, `verify` may be a concrete inspection: the named contradiction /
  broken reference is gone).
- **Never-auto:** behavior changes without a regression guard.
- **Dispatch route:** `dispatch` skill. Default `priority: high`.
- **Evidence:** the repro fails before the fix and passes after; for CLI
  fixes, `go build ./...` and `go test ./...` exit 0 in `anthill-cli/`.

---

## Judgment signals (accrued from triage decisions — read before classifying)

Starts empty. When a triage decision generalizes into a reusable rule, append
it here as one bullet so the next triage inherits it.

- **Split cross-repo items at triage.** One item per repo, cross-referenced in
  `note`; framework side first (the CLI embeds the template).
