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
anthill backlog show <key>
anthill backlog list [--ready | --active | --untriaged | --all] [--status …] [--dispatch …] [--disposition …] [--reporter …] [--workstream …] [--sort rank|priority|age|updated|id] [--fields a,b,c]
anthill backlog next [--workstream <ws>]
anthill backlog triage <key> workstream=<ws> summary="…" change-type=… risk=… verify="…" disposition=… [priority=…]
anthill backlog approve <key>…          # user-gated
anthill backlog fast-track <key>…       # user-gated
anthill backlog unapprove <key>…
anthill backlog claim <key> | --next [--workstream <ws>] [--by <handle>] [--force]
anthill backlog deliver <key> --by <handle> --evidence "…" [--evidence-file <file>]
anthill backlog block <key> "<reason>"
anthill backlog unblock <key> [--release]
anthill backlog release <key>
anthill backlog park <key> "<reason>"
anthill backlog unpark <key>
anthill backlog close <key> --done ["…"] | --reject "…" | --discard "…" | --remove "…"
anthill backlog set <key> field=value … # free fields only; workstream= moves the file
anthill backlog validate [--strict]
anthill backlog prune [--older-than <duration>] [--dry-run]
anthill backlog migrate [--key-prefix <PREFIX>] [--dry-run]
anthill backlog renumber <key> [--to <N>]
```

`<key>` is the item's key (see *Key scheme* below); every verb that takes
one accepts the key, the bare number, or the item's former id. Examples:

```
anthill backlog show ANT-74
anthill backlog claim ANT-74 --by dispatcher:worker-a
anthill backlog deliver 74 --by dispatcher:worker-a --evidence "…"
anthill backlog close ant-74 --done "shipped"
```

## Schema (per-item frontmatter) — framework-standard, keep as-is

| key | values | set by |
|---|---|---|
| `title` | one line, the human label; the slug in the item's filename derives from it and is cosmetic | submitter |
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
| `former-id` | an additional id the item answers to in every verb; present only on items that `migrate` renumbered | CLI on `migrate` only |

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
holds: `claim` on pickup when its brief hands it a key, `deliver`, `block`.
`deliver` is claimant-only — the worker passes the claim handle as `--by`.
Parallelism posture: **<serial | …>** — derived in `.anthill/resources.md`.
Escalations per the `escalate` skill (`.anthill/escalations/`).

## Key scheme

An item's id is its **key**: `PREFIX-N` — the installation's prefix, a
hyphen, and a positive integer without padding (`ANT-74`, `BH-3`).

- **Prefix:** 2–5 uppercase ASCII letters, set once per installation as
  `key-prefix` in the frontmatter of `workstreams.md` (`anthill init`
  seeds it). Changing it later is a config edit that leaves every number
  in place; it is discouraged because keys written in prose, ledger rows,
  and escalation records under the old prefix stop resolving.
- **Number:** allocated sequentially by the CLI on `new`, never reused —
  `.anthill/backlog/sequence` holds the highest number ever allocated and
  is tracked in git with the items. A key never changes across moves
  between directories; only `renumber` (after a branch-merge collision
  gave two items the same number) assigns a new one.
- **Filename:** `NNNN-<slug>.md` — the number zero-padded to at least four
  digits, a dash, and the title slug (lowercase, dash-separated, ≤40 chars; cosmetic
  — `set title=…` renames the file, the number stays). The key is read
  from the leading digits plus the prefix; it is unique across `intake/`,
  all workstream directories, `done/`, and `discarded/` (`validate`
  reports a duplicate number as `id-unique`).
- **Accepted input** wherever a verb takes a key: the key itself,
  case-insensitive (`ant-74`); the bare number (`74`); or the item's
  `former-id`. The CLI prints the canonical form (`ANT-74`) everywhere.

## Referring to items

Write the key in prose exactly as the CLI prints it — `ANT-74`, uppercase
prefix, no padding — in briefs, ledger rows, escalation records, commit
messages, and reports, so humans and tools recognise it; the Claude Code
plugin draws a chip for every key it finds in a transcript row.

## Archive and changelog

`done/` keeps every finished item permanently — the project's history.
`discarded/` keeps discarded items until purged by hand with `prune`.
`.anthill/backlog/CHANGELOG.md` gets one line per closed item (done /
discarded / removed / purged, with a short reason), written by the CLI.
