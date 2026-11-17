# Changelog

Release notes for the Anthill CLI. Newest first.

## Unreleased

- Embedded framework template re-vendored at anthill-fw `f10dbcf` (backlog v2 skills and templates; supervisor reporting/agenda hygiene). Consumer action: `anthill backlog migrate`, then `anthill sync`, then re-derive `.anthill/backlog/bindings.md` and `.anthill/supervisor/brief-template.md`.

### Changed â€” backlog schema 2 (breaking)

The backlog follows `docs/BACKLOG_V2_SPEC.md`.

- **Item schema.** New fields `summary`, `dispatch`
  (`proposed|approved|fast-tracked`), `reporter` (`user|supervisor|worker`),
  `created-at`, `triaged-at`, `claimed-by`, `session`, `delivered-at`,
  `closed-at`. `status` is `new|triaged|in-progress|validation|done|blocked|
  parked|discarded`; `disposition` is `auto|design|interactive|open`;
  `priority` gains `low`. `value-verdict` is dropped (its rationale lives in
  the body's `## Triage`). `ready` = triaged + dispatch approved/fast-tracked
  + disposition auto + a real verify; `active` = in-progress or validation.
- **Body sections.** New items get `## Description`, `## Acceptance`,
  `## Triage`, `## Evidence`, `## Log`; every transition appends a `## Log`
  line, `deliver` appends to `## Evidence`.
- **Lifecycle verbs.** `triage`, `deliver`, `block`/`unblock [--release]`,
  `release`, `park`/`unpark`, `fast-track`, `unapprove`; `approve` sets
  dispatch and takes several ids (no `--yes`); `claim` stamps `claimed-by`
  (`--by`, `ANTHILL_ACTOR`, role:session, OS user) and `session`
  (`--session`, `CLAUDE_CODE_SESSION_ID`); `close` takes
  `--done|--reject|--discard|--remove` (`--block` moved to `block`). Done
  items are archived in `backlog/done/`, discarded items in
  `backlog/discarded/`; `anthill backlog prune` purges `discarded/`.
- **`set`** accepts only the free fields; `status=` and `dispatch=` are
  refused (exit 3).
- **`new`** gains `--summary`, `--description`/`--description-file`,
  `--reporter`, and `--priority low`.
- **`list`** gains filters for every axis, `--search`, the `--active` and
  `--all` presets, `--sort rank|priority|age|updated|id`, `--reverse`,
  `--limit`, `--fields`. `next` and `claim --next` pick by rank.
- **`validate` / `doctor`** share one rule set and list every violation as
  `<id>: <rule>: <message>` (`--json`: a `violations` array). `--strict`
  promotes the change-type vocabulary warning to a violation.

**Consumer action:** after upgrading the CLI, run `anthill backlog migrate`
(preview with `--dry-run`). It rewrites every item to schema 2 in place,
idempotently, and stamps `schema: 2` into `backlog/workstreams.md`; until
then `validate` refuses the tree and points at `migrate`. Then run
`anthill backlog validate` and complete the triage fields it lists.

### Added â€” sessions ledger and the Now board

- **`anthill session link|touch|list|prune`** maintain the runtime-only
  ledger `.anthill/sessions/<session>[.<agent>].json` (untracked; the first
  write adds a `.gitignore`) correlating harness sessions with the items they
  work. `link --session --item [--agent --handle --role --transcript --cwd
  --verb]` creates or updates a record and stamps `last-seen` (an unknown
  item exits 4); `touch` bumps `last-seen`; `list [--item] [--active]`
  shows session, agent, handle, item, role, last-seen age, and transcript;
  `prune [--older-than <dur|Nd>] [--dry-run]` deletes records not seen
  within the threshold (default `7d`). Every verb honours `--json`.
- **`backlog claim` writes the ledger record itself** when a session is known
  (`--session` or `CLAUDE_CODE_SESSION_ID`), so correlation needs no hook;
  `deliver`, `close`, `block`, and `release` bump the caller's record for the
  item when one exists.
- **TUI Now mode** (`3`): a single-pane board of active items (id, summary,
  status, claimed-by, session, last-seen age, workstream) refreshed every 2 s,
  a block of `validation` items awaiting a controller, and ready / proposed /
  blocked counts. `enter` shows the item, `d` closes it as done, `j` rejects
  it with a reason, `r` releases it, `o` shows the transcript path.

## 0.0.10 â€” 2026-11-16

### Changed

- **Re-embed the framework template at upstream ref `dec42bb`.** The pinned
  `supervisor` skill's delegation gate judges whether work belongs on one
  worker by the information it shares (the same artifacts, one judgment over
  a shared set of items, or output chained to input), counts tasks by result
  rather than step, and rejects a missing `BATCH REASON` only when the brief
  shows no shared information. `.anthill/supervisor/brief-template.md` states
  the `BATCH REASON` format. `anthill sync` re-copies both files, plus the
  `dispatch` and `triage` skill cleanups at the same ref.

### Added

- **`tools/vendor.ps1`** re-vendors `internal/bootstrap/template/` from the
  sibling `anthill-fw` checkout (HEAD or `-Ref <sha>`) and pins `TemplateRef`;
  it refuses a commit not yet pushed to the framework's origin.

### Fixed

- **`tools/release.ps1` reads `CHANGELOG.md` as UTF-8.** Windows PowerShell
  read it as ANSI and re-encoded every non-ASCII character on each release;
  the corrupted dashes and symbols in earlier entries are restored.

## 0.0.9 â€” 2026-11-15

### Changed

- **Re-embed the framework template at upstream ref `40393e8`.** The pinned
  `supervisor` skill carries the role-split and distilled-context delegation
  rules and a 150k-token worker recycle threshold; `supervisor` and
  `rehydrate` make recycling the supervisor session user-initiated.
  `anthill sync` re-copies both skills.

## 0.0.8 â€” 2026-11-15

### Added

- **TUI: answer open escalations in place.** A free-text overlay in the
  Escalations mode records the answer on the selected open escalation.
- **TUI: word-wrapped detail pane.** Record content wraps to the pane width.

### Changed

- **Re-embed the framework template at upstream ref `ae961d6`.** The pinned
  template now carries the supervisor delegation gate: the `supervisor`
  skill's frontmatter registers a `PreToolUse` prompt hook on
  `Agent|SendMessage` that rejects briefs and messages handing a worker
  unrelated tasks, and `.anthill/supervisor/brief-template.md` names one
  task in `GOAL` and gains a `BATCH REASON` section. `anthill sync` re-copies
  both files.

## 0.0.7 â€” 2026-08-29

### Added

- **`anthill tui` â€” interactive full-screen browser** over the harness's
  durable state (Bubble Tea + Lip Gloss). Layout: mode tabs and `[key] action`
  hints in a top header, the selected record's frontmatter + body in a
  scrollable left content pane, the record list in a right nav pane, and a
  one-line status bar. `tab` switches pane focus; `â†‘â†“` move; destructive
  actions are y/n-confirm-gated in the status line.
- **Switchable TUI modes.** The TUI is organised as top-level modes behind a
  registry seam (`[m]` / number keys to switch). Two ship now:
  - **Backlog** â€” browse items; `a` approve, `d` close done, `x` discard,
    `X` remove. Actions map 1:1 to the backlog verbs and reload the list in
    place, preserving selection.
  - **Escalations** â€” browse records; `a` apply, `x` close, `X` remove.
    Free-text answering/raising stays on the scriptable verbs for now.
  The mode seam is documented in `docs/CLI_INTERFACE_SPEC.md` Â§5; adding a
  mode is one struct plus one registry append.
- **`escalation close <id> ["reason"]` and `escalation remove <id>
  ["reason"]`.** An unanswered (stale) escalation can now be closed, and any
  record removed, without going through `answer`/`apply`. Both write a LOG.md
  audit line before deleting the record; the reason defaults to the record's
  question. No frontmatter schema change.

### Changed

- **Bare `anthill` opens the TUI.** With no subcommand, an interactive
  terminal inside an installation launches `anthill tui`; non-TTY invocations
  and invocations outside any `.anthill` root print help with exit 0.
  `--help`, `help`, `--version`, and unknown-command handling are unchanged.
- Backlog approve/close transitions and sweep ordering moved from the command
  layer into the `internal/backlog` domain package so the CLI and TUI share
  one transition path. Command output and exit codes are unchanged.

### Fixed

- **`backlog close` disposition flags no longer swallow the item id.**
  `--done`/`--discard`/`--remove`/`--block` were string flags requiring a
  value, so `close <id> --remove` failed and `close --remove <id>` misread
  the id as the reason. Dispositions are now pure boolean flags with the
  reason as an optional second positional (`close <id> --discard ["reason"]`,
  either argument order), with regression tests. Spec Â§3.7 updated to match.

## 0.0.6 â€” 2026-08-14

### Added

- **Re-embed the framework template at upstream ref `17a0ea2`.** The pinned
  template now carries:
  - the new `rehydrate` general-tier skill (supervisor session handoff via a
    task-board snapshot at `.anthill/supervisor/board.md`) â€” general-tier
    skill count is now **twelve**;
  - the rewritten `wake-up`, `supervisor`, and `dispatch-loop` skills
    (concision pass; yield-by-cause rule; no-destructive-orphan-recovery;
    a missing control flag reads as `run`) and the updated `autonomous`
    skill (branch discipline deferred to project config);
  - the updated `supervisor/brief-template.md` (spawner-injected reply
    handle, durable-state-before-signal) and `dispatch/README.md`;
  - the `.anthill/` template updates that shipped upstream alongside the
    project-owned change-type vocabulary (declared `change-types` /
    `never-auto-change-types` frontmatter in `workstreams.md`), previously
    missing from the vendored payload.

  Payload files are normalized to LF line endings.

## 0.0.5 â€” 2026-07-21

### Changed

- **`change-type` vocabulary is now project-owned, not a hardcoded enum.** The
  CLI no longer ships a fixed accepted set and no longer rejects an unknown
  `change-type` on write. A project declares its vocabulary in `workstreams.md`
  frontmatter (`change-types`); an item using a value outside the declared set
  is a **warning** at `validate`/`doctor` (a new advisory `change-type-vocab`
  check that never fails), never a hard violation. With no declared vocabulary,
  any `change-type` is accepted. The AUTO-forbidden subset moves to
  `never-auto-change-types` in the same frontmatter, so disposition-coherence is
  project-declared too. `validate` gains a `warnings` array in `--json`.
- **`sync` scope widened beyond skills.** In addition to the general-tier
  skills, `sync` now reconciles the **framework-invariant** non-skill files (the
  `.anthill/` reference READMEs, the supervisor brief template, and the `tools/`
  launchers) byte-identical to the embedded template. It also **creates any
  payload file absent from the install** (`created` in `--json`) â€” a
  non-destructive write that carries new upstream scaffold files and whole
  subtrees (e.g. a newly added `.anthill/` mechanism or required structural dir)
  to a sync-upgraded install without a re-scaffold. Existing project-derived
  config and runtime state are never overwritten; `.gitignore`, the disposable
  `CLAUDE.template.md` starter, and per-workstream files are excluded from
  creation. This closes the gap where new upstream scaffold content could only
  arrive via a re-scaffold.

### Added

- **Re-embed the framework template at upstream ref `8fa21a2`.** The pinned
  template now carries:
  - the new `sync` general-tier skill â€” general-tier skill count is now
    **eleven**;
  - the updated `dispatch-loop`, `supervisor`, and `wake-up` skills;
  - the new `.anthill/dispatch/` subtree (`README.md`, plus the seed-on-first-use
    runtime artifacts `control.md` and `ledger.md`) that the dispatcher tier
    (`dispatch-loop`) reads. `anthill scaffold` therefore writes eleven skills
    and the `.anthill/dispatch/` tree; `doctor`'s structure check requires the
    `.anthill/dispatch/` directory.

## 0.0.4 â€” 2026-07-20

### Changed

- **Retire the `autonomous` skill's adaptation-region handling in `doctor` and
  `sync`.** The general tier is now byte-identical to upstream with no
  exceptions. `doctor` Section A checks every installed `.claude/skills/*` file
  byte-for-byte â€” the previous carve-out that exempted the `autonomous`
  proceed-list and decisions-log path is gone. `sync` is a plain verbatim
  re-copy: a changed skill is overwritten, and the only conflict class is a
  skill with an unexpected local edit (reported and left unchanged at exit 3
  unless `--force`). There is no longer any per-skill merge or
  adaptation-conflict class.

### Added

- **Re-embed the framework template at upstream ref `3797138`.** The pinned
  template now carries:
  - the new `submit` general-tier skill â€” general-tier skill count is now
    **ten**;
  - the refactored `autonomous` skill, which loads its project inputs from
    `.anthill/autonomy.md` at invocation instead of holding in-place adaptation
    regions;
  - the new `.anthill/autonomy.md` placeholder in the scaffold tree.

  `anthill scaffold` therefore writes ten skills and the `autonomy.md`
  placeholder; `anthill doctor` reports `autonomy.md` as an un-derived
  `.anthill/` file until the consumer fills it in, like other config
  placeholders.

### Migration â€” existing installs

On the first `anthill sync` after this CLI adopts the new ref, an install's
locally-derived `autonomous` skill (its old proceed-list) reconciles against the
refactored upstream skill:

1. Move your proceed-list and decisions-log path out of the old
   `.claude/skills/autonomous/SKILL.md` into a new `.anthill/autonomy.md`
   (derive it from the scaffolded placeholder).
2. Run `anthill sync` to add the `submit` skill and re-copy the `autonomous`
   skill verbatim. If your install already reports as current, the skill shows
   as an unexpected local edit â€” re-run with `--force` to overwrite it.

This is a relocation of configuration, not data loss: the proceed-list now lives
in the specific tier where it is upgrade-safe.
