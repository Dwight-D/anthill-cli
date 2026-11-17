# Backlog bindings — <YOUR PROJECT>

> **PROJECT-SPECIFIC TIER — TEMPLATE.** Swap for your own domain. Loaded by
> the backlog skills (`submit`, `triage`, `expedite`, `dispatch`,
> `dispatch-loop`) on invocation, alongside `workstreams.md`. The schema,
> lifecycle, and commands below are the framework's (the `anthill` CLI
> enforces them); keep them. Fill in the posture and parallelism for your
> environment. Delete this quote block once derived.

Environment bindings for the backlog mechanism. The full contract is the
`anthill` CLI's backlog specification (`docs/BACKLOG_V2_SPEC.md` in the CLI
repository); this file is the working summary the skills load.

**Schema owner: the `anthill` CLI.** It is the ONLY writer of item
frontmatter — every state change goes through one of its verbs, which guards
the transition, stamps it, and appends a `## Log` line. The body below the
frontmatter is plain markdown that humans and agents edit freely.

## Commands

```
anthill backlog new --title "…" --value "…" --reporter user|supervisor|worker [--summary "…"] [--source "…"] [--hint <ws>] [--priority …]
anthill backlog show <id>
anthill backlog list [--ready | --active | --untriaged | --all] [--status …] [--dispatch …] [--disposition …] [--reporter …] [--workstream …] [--sort rank|priority|age|updated|id] [--fields a,b,c]
anthill backlog next [--workstream <ws>]
anthill backlog triage <id> workstream=<ws> summary="…" change-type=… risk=… verify="…" disposition=… [priority=…]
anthill backlog approve <id>…          # user-gated
anthill backlog fast-track <id>…       # user-gated
anthill backlog unapprove <id>…
anthill backlog claim <id> | --next [--workstream <ws>] [--by <handle>] [--force]
anthill backlog deliver <id> --by <handle> --evidence "…" [--evidence-file <file>]
anthill backlog block <id> "<reason>"
anthill backlog unblock <id> [--release]
anthill backlog release <id>
anthill backlog park <id> "<reason>"
anthill backlog unpark <id>
anthill backlog close <id> --done ["…"] | --reject "…" | --discard "…" | --remove "…"
anthill backlog set <id> key=value …   # free keys only; workstream= moves the file
anthill backlog validate [--strict]
anthill backlog prune [--older-than <duration>] [--dry-run]
```

## Schema (per-item frontmatter) — framework-standard, keep as-is

| key | values | set by |
|---|---|---|
| `title` | one line; the id derives from it once | submitter |
| `summary` | one or two sentences: what the change is (required from `triaged` on) | submitter or triage |
| `value` | the pain it removes or the potential it unlocks | submitter |
| `workstream` | a directory under `.anthill/backlog/`; absent in `intake/` | triage |
| `status` | `new` · `triaged` · `in-progress` · `validation` · `done` · `blocked` · `parked` · `discarded` | CLI verbs only |
| `dispatch` | `proposed` · `approved` · `fast-tracked` (approved and pinned to the top of every queue) | user-gated verbs only |
| `disposition` | `auto` · `design` · `interactive` · `open` | triage (user may override) |
| `priority` | `high` · `normal` · `low` (default `normal`) | triage / user |
| `reporter` | `user` · `supervisor` · `worker` | submitter |
| `change-type` | the vocabulary in `workstreams.md` frontmatter (`change-types`) | triage |
| `risk` | `additive` · `reversible` · `behavior-change` · `subjective` | triage |
| `verify` | headless acceptance test, or `none` | triage / refined by worker |
| `source` | free text: where it came up | submitter |
| `hint` | non-binding workstream hint; stripped on triage | submitter |
| `note` | free text, no machine semantics | anyone, via `set` |
| `created-at` · `triaged-at` · `claimed-at` · `delivered-at` · `closed-at` | RFC3339 UTC stamps | CLI |
| `claimed-by` · `session` | claimant handle and harness session id | CLI on `claim` |

`disposition` coherence: `auto` requires a real `verify` (not empty or
`none`) and a change-type outside `never-auto-change-types`; `validate`
reports anything else as a violation.

Body sections, in order: `## Description` (the self-contained brief a cold
worker implements from), `## Acceptance` (what done looks like, expanding
`verify`), `## Triage` (triage's reasoning and value verdict), `## Evidence`
(appended by `deliver`), `## Log` (one line per transition, appended by the
CLI).

## Lifecycle

```
new ─triage─► triaged ─claim─► in-progress ─deliver─► validation ─close --done─► done
                 ▲                │  ▲                     │
                 │  release ◄─────┘  └──── close --reject ─┘
                 │                block ─► blocked ─unblock─► in-progress
                 └── park / unpark ──► parked          (unblock --release ─► triaged)
any ─close --discard─► discarded        any ─close --remove─► file deleted
```

**Ready** (dispatchable unattended) = `status: triaged`, `dispatch` approved
or fast-tracked, `disposition: auto`, and a real `verify`. `list --ready`,
`next`, and `claim --next` apply it; ranking is fast-tracked first, then
priority, then workstream sweep order, then oldest.

**Active** = `in-progress` or `validation` (`list --active`).

## Posture

**<propose only>** — triage proposes (`dispatch: proposed`) and records its
disposition; the user approves with `approve` / `fast-track`, directly or via
the `expedite` skill. No agent approves. Loosen per workstream only as the
gates earn trust.

## Who moves an item

Controllers (`dispatch`, `dispatch-loop`, `supervisor`, the user) own
`close`, `approve`, and `release`. A worker may only advance the item it
holds: `claim` on pickup when its brief hands it an id, `deliver`, `block`.
`deliver` is claimant-only — the worker passes the claim handle as `--by`.
Parallelism posture: **<serial | …>** — derived in `.anthill/resources.md`.
Escalations per the `escalate` skill (`.anthill/escalations/`).

## Id scheme

Filename = id = kebab slug of the title, ≤50 chars; unique across `intake/`,
all workstream directories, `done/`, and `discarded/`; numeric suffix on
collision; never changes once assigned, including across moves.

## Archive and changelog

`done/` keeps every finished item permanently — the project's history.
`discarded/` keeps discarded items until purged by hand with `prune`.
`.anthill/backlog/CHANGELOG.md` gets one line per closed item (done /
discarded / removed / purged, with a short reason), written by the CLI.
