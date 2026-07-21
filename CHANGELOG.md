# Changelog

Release notes for the Anthill CLI. Newest first.

## 0.0.5 — 2026-07-21

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
  payload file absent from the install** (`created` in `--json`) — a
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
  - the new `sync` general-tier skill — general-tier skill count is now
    **eleven**;
  - the updated `dispatch-loop`, `supervisor`, and `wake-up` skills;
  - the new `.anthill/dispatch/` subtree (`README.md`, plus the seed-on-first-use
    runtime artifacts `control.md` and `ledger.md`) that the dispatcher tier
    (`dispatch-loop`) reads. `anthill scaffold` therefore writes eleven skills
    and the `.anthill/dispatch/` tree; `doctor`'s structure check requires the
    `.anthill/dispatch/` directory.

## 0.0.4 — 2026-07-20

### Changed

- **Retire the `autonomous` skill's adaptation-region handling in `doctor` and
  `sync`.** The general tier is now byte-identical to upstream with no
  exceptions. `doctor` Section A checks every installed `.claude/skills/*` file
  byte-for-byte — the previous carve-out that exempted the `autonomous`
  proceed-list and decisions-log path is gone. `sync` is a plain verbatim
  re-copy: a changed skill is overwritten, and the only conflict class is a
  skill with an unexpected local edit (reported and left unchanged at exit 3
  unless `--force`). There is no longer any per-skill merge or
  adaptation-conflict class.

### Added

- **Re-embed the framework template at upstream ref `3797138`.** The pinned
  template now carries:
  - the new `submit` general-tier skill — general-tier skill count is now
    **ten**;
  - the refactored `autonomous` skill, which loads its project inputs from
    `.anthill/autonomy.md` at invocation instead of holding in-place adaptation
    regions;
  - the new `.anthill/autonomy.md` placeholder in the scaffold tree.

  `anthill scaffold` therefore writes ten skills and the `autonomy.md`
  placeholder; `anthill doctor` reports `autonomy.md` as an un-derived
  `.anthill/` file until the consumer fills it in, like other config
  placeholders.

### Migration — existing installs

On the first `anthill sync` after this CLI adopts the new ref, an install's
locally-derived `autonomous` skill (its old proceed-list) reconciles against the
refactored upstream skill:

1. Move your proceed-list and decisions-log path out of the old
   `.claude/skills/autonomous/SKILL.md` into a new `.anthill/autonomy.md`
   (derive it from the scaffolded placeholder).
2. Run `anthill sync` to add the `submit` skill and re-copy the `autonomous`
   skill verbatim. If your install already reports as current, the skill shows
   as an unexpected local edit — re-run with `--force` to overwrite it.

This is a relocation of configuration, not data loss: the proceed-list now lives
in the specific tier where it is upgrade-safe.
