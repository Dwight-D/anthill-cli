# Backlog bindings — <YOUR PROJECT>

> **PROJECT-SPECIFIC TIER — TEMPLATE.** Swap for your own domain. Loaded by
> the `triage`/`dispatch`/`expedite` skills on invocation, alongside
> `workstreams.md`. The schema below is the framework's; keep it. Fill in
> the schema owner and commands for your environment. Delete this quote block
> once derived.

Environment bindings for the backlog mechanism.

**Schema owner: <the Anthill CLI / your project's CLI / convention>.** When a
schema-owning tool is installed, it is the ONLY writer of item frontmatter —
it invariant-checks every write and owns id generation. Until then, operate
convention-first: write/edit item files directly following the schema below,
with care.

## Commands

> Fill these in for your schema owner. Shape (from the specified Anthill CLI):

```
<owner> backlog new --title "…" --value "…" [--source "…"] [--backlog <hint>]   # lands in intake/
<owner> backlog list [--workstream <ws>] [--untriaged] [--ready]
<owner> backlog set <id> workstream=<ws> key=value …    # workstream change moves the file
<owner> backlog next [--workstream <ws>]                # default: sweep order
<owner> backlog claim <id>|--next [--workstream <ws>] [--force]
<owner> backlog close <id> --done|--discard "…"|--remove "…"|--block "…"
<owner> backlog validate [--strict]
```

## Schema (per-item frontmatter) — framework-standard, keep as-is

```
workstream:   <set at triage; absent while the item is in intake/>
title:        <one line>
value:        <the pain it removes or the potential it unlocks>
source:       <where it came up>                                  # optional
hint:         <non-binding submitter hint at submit time; removed on triage> # optional
change-type:  <your domain's change-type vocabulary — e.g. doc | default |
              option | tooling | bugfix | audit | verify | new-<primitive> |
              rename | removal | design | subjective>
risk:         additive | reversible | behavior-change | subjective
verify:       <headless acceptance test, or "none">
value-verdict: ADVANCE | HOLD | DISCARD — <one-line why>
disposition:  AUTO | REVIEW | DISCARD    # AUTO needs a non-"none" verify and
                                         # must not be a never-auto change type
status:       idea | approved | in-progress | blocked | parked | done
priority:     high | normal
note:         <free text; qualifiers: expedited, needs-spec, remainder, …>
```

Ready (dispatchable) = `status: approved` with a non-empty `verify`.
Posture: **propose only** — triage recommends via `disposition`; a human sets
`status: approved` (directly, or via the `expedite` skill). Loosen per
workstream only as the gates earn trust.

## Dispatch tier

Sender skills (`dispatch`, `dispatch-loop`) call the commands above and own
all queue state; workers (`dispatch-receive`) never write it. Parallelism
posture: **<serial | …>** — derived in `.anthill/resources.md`. Escalations
per the `escalate` skill (`.anthill/escalations/`).

## Id scheme

Filename = id = kebab slug of the title, ≤50 chars; unique across `intake/`
and all workstream directories; numeric suffix on collision; never changes
once assigned, including across the intake→workstream move.

## Changelog

`.anthill/backlog/CHANGELOG.md` — one line per closed item (done / discarded /
removed, with a short reason).
