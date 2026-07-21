# Backlog bindings — Anthill CLI

Environment bindings for the backlog mechanism. Loaded by the
`triage`/`dispatch`/`expedite` skills on invocation, alongside `workstreams.md`.

**Schema owner: convention-first (no CLI installed yet).** The Anthill CLI —
this project's own product — is the intended schema owner: once built, its
`backlog` verbs become the ONLY writer of item frontmatter, invariant-checking
every write and owning id generation. **Until it exists, operate
convention-first:** create and edit item files directly in `intake/` and the
workstream directories, following the schema below, with care. Swapping to the
CLI later is a one-line edit here (point the schema owner + commands at it), not
a change to any skill.

## Commands

Convention-first (no schema-owner CLI yet):

```
# submit → write a file into .anthill/backlog/intake/ with frontmatter:
#   title, value, optional source, status: idea
# triage → edit the item file: set workstream (move the file into that dir),
#   change-type, risk, verify, value-verdict, disposition
# approve → set status: approved (human only, or via the expedite skill)
# close → delete the item file, add one line to CHANGELOG.md
```

Target shape once the Anthill CLI lands (it becomes the schema owner above):

```
anthill backlog new --title "…" --value "…" [--source "…"] [--backlog <hint>]   # lands in intake/
anthill backlog list [--workstream <ws>] [--untriaged] [--ready]
anthill backlog set <id> workstream=<ws> key=value …    # workstream change moves the file
anthill backlog next [--workstream <ws>]                # default: sweep order
anthill backlog claim <id>|--next [--workstream <ws>] [--force]
anthill backlog close <id> --done|--discard "…"|--remove "…"|--block "…"
anthill backlog validate [--strict]
```

## Schema (per-item frontmatter) — framework-standard, keep as-is

```
workstream:   <set at triage; absent while the item is in intake/>
title:        <one line>
value:        <the pain it removes or the potential it unlocks>
source:       <where it came up>                                  # optional
hint:         <non-binding submitter hint at submit time; removed on triage> # optional
change-type:  project vocabulary declared in workstreams.md `change-types`;
              here: doc | tooling | bugfix | audit | verify | new-command |
              new-flag | rename | removal | design | subjective
              (out-of-vocabulary values warn, never fail)
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

Sender skills (`dispatch`, `dispatch-loop`) call the commands above and own all
queue state; workers (`dispatch-receive`) never write it. Parallelism posture:
**serial** — derived in `.anthill/resources.md`. Escalations per the `escalate`
skill (`.anthill/escalations/`).

## Id scheme

Filename = id = kebab slug of the title, ≤50 chars; unique across `intake/`
and all workstream directories; numeric suffix on collision; never changes
once assigned, including across the intake→workstream move.

## Changelog

`.anthill/backlog/CHANGELOG.md` — one line per closed item (done / discarded /
removed, with a short reason).
