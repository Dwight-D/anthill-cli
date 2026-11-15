# Changelog

Release notes for the Anthill CLI. Newest first.

## 0.0.9 — 2026-11-15

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

## 0.0.7 Ã¢â‚¬â€ 2026-08-29

### Added

- **`anthill tui` ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â interactive full-screen browser** over the harness's
  durable state (Bubble Tea + Lip Gloss). Layout: mode tabs and `[key] action`
  hints in a top header, the selected record's frontmatter + body in a
  scrollable left content pane, the record list in a right nav pane, and a
  one-line status bar. `tab` switches pane focus; `ÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬ËœÃƒÂ¢Ã¢â‚¬Â Ã¢â‚¬Å“` move; destructive
  actions are y/n-confirm-gated in the status line.
- **Switchable TUI modes.** The TUI is organised as top-level modes behind a
  registry seam (`[m]` / number keys to switch). Two ship now:
  - **Backlog** ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â browse items; `a` approve, `d` close done, `x` discard,
    `X` remove. Actions map 1:1 to the backlog verbs and reload the list in
    place, preserving selection.
  - **Escalations** ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â browse records; `a` apply, `x` close, `X` remove.
    Free-text answering/raising stays on the scriptable verbs for now.
  The mode seam is documented in `docs/CLI_INTERFACE_SPEC.md` Ãƒâ€šÃ‚Â§5; adding a
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
  either argument order), with regression tests. Spec Ãƒâ€šÃ‚Â§3.7 updated to match.

## 0.0.6 ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â 2026-08-14

### Added

- **Re-embed the framework template at upstream ref `17a0ea2`.** The pinned
  template now carries:
  - the new `rehydrate` general-tier skill (supervisor session handoff via a
    task-board snapshot at `.anthill/supervisor/board.md`) ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â general-tier
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

## 0.0.5 ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â 2026-07-21

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
  payload file absent from the install** (`created` in `--json`) ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â a
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
  - the new `sync` general-tier skill ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â general-tier skill count is now
    **eleven**;
  - the updated `dispatch-loop`, `supervisor`, and `wake-up` skills;
  - the new `.anthill/dispatch/` subtree (`README.md`, plus the seed-on-first-use
    runtime artifacts `control.md` and `ledger.md`) that the dispatcher tier
    (`dispatch-loop`) reads. `anthill scaffold` therefore writes eleven skills
    and the `.anthill/dispatch/` tree; `doctor`'s structure check requires the
    `.anthill/dispatch/` directory.

## 0.0.4 ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â 2026-07-20

### Changed

- **Retire the `autonomous` skill's adaptation-region handling in `doctor` and
  `sync`.** The general tier is now byte-identical to upstream with no
  exceptions. `doctor` Section A checks every installed `.claude/skills/*` file
  byte-for-byte ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â the previous carve-out that exempted the `autonomous`
  proceed-list and decisions-log path is gone. `sync` is a plain verbatim
  re-copy: a changed skill is overwritten, and the only conflict class is a
  skill with an unexpected local edit (reported and left unchanged at exit 3
  unless `--force`). There is no longer any per-skill merge or
  adaptation-conflict class.

### Added

- **Re-embed the framework template at upstream ref `3797138`.** The pinned
  template now carries:
  - the new `submit` general-tier skill ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â general-tier skill count is now
    **ten**;
  - the refactored `autonomous` skill, which loads its project inputs from
    `.anthill/autonomy.md` at invocation instead of holding in-place adaptation
    regions;
  - the new `.anthill/autonomy.md` placeholder in the scaffold tree.

  `anthill scaffold` therefore writes ten skills and the `autonomy.md`
  placeholder; `anthill doctor` reports `autonomy.md` as an un-derived
  `.anthill/` file until the consumer fills it in, like other config
  placeholders.

### Migration ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â existing installs

On the first `anthill sync` after this CLI adopts the new ref, an install's
locally-derived `autonomous` skill (its old proceed-list) reconciles against the
refactored upstream skill:

1. Move your proceed-list and decisions-log path out of the old
   `.claude/skills/autonomous/SKILL.md` into a new `.anthill/autonomy.md`
   (derive it from the scaffolded placeholder).
2. Run `anthill sync` to add the `submit` skill and re-copy the `autonomous`
   skill verbatim. If your install already reports as current, the skill shows
   as an unexpected local edit ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â re-run with `--force` to overwrite it.

This is a relocation of configuration, not data loss: the proceed-list now lives
in the specific tier where it is upgrade-safe.
