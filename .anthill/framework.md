# Anthill — framework provenance & sync state

This installation's sync state against the upstream Anthill repository. The
upgrade procedure re-copies changed general-tier skills and bumps the
`synced-through` ref below; `anthill sync` automates it.

## This installation

- **Role:** consumer (installed the framework; does not maintain it).
- **Framework source:** the upstream Anthill repository
  (https://github.com/Dwight-D/anthill) — the framework's source of truth. Its
  `.claude/skills/` are the canonical general-tier texts.
- **synced-through:** 3797138ddd5eb6e89d083aa001156d4d28fefe18 (installed
  2026-07-20). This is the embedded template ref the `anthill` CLI pins;
  `anthill doctor` / `sync` read it as the baseline.

## Two-tier discipline (how this installation stays upgradeable)

The split is by **ownership**, not by directory. `anthill sync` reconciles every
upstream-owned file byte-identical to the embedded template; project-owned files
it never touches.

- **Upstream-owned (follows `sync`)** — never locally edit these; divergence
  across installations is the failure mode the split exists to prevent:
  - the `.claude/skills/` orchestration skills (`supervisor`, `autonomous`,
    `triage`, `submit`, `dispatch`, `dispatch-loop`, `dispatch-receive`,
    `expedite`, `escalate`, `wake-up`), byte-identical to upstream with no
    exceptions, and
  - the **framework-invariant** files: the `.anthill/` reference READMEs, the
    supervisor brief template, and the `tools/` launchers.

  The `autonomous` skill's project-specific inputs — its proceed-list and
  decisions-log path — live in `.anthill/autonomy.md` (project-owned config the
  skill loads at invocation), so filling them in is not a skill edit.
- **Project-owned (`sync` never touches)** — the adaptation config
  (`backlog/bindings.md`, `workstreams.md`, `autonomy.md`, `resources.md`,
  `supervisor/bindings.md`, this `framework.md`) and runtime state
  (`backlog/CHANGELOG.md`, `escalations/LOG.md`, `decisions.md`,
  `supervisor/agenda.md`). These arrive once at scaffold and are refused (left
  as-is) on any re-scaffold. All adaptation lives here.

## Flag upstream, don't fork

A gap or finding about the framework *itself* is filed upstream to the Anthill
repository (an issue or PR) — never fixed by editing a local skill. If you need
a mitigation NOW, put it in `.anthill/` config and name it in the upstream
report so the maintainer can supersede it.

## Sync downstream (periodically)

Compare `synced-through` above against the upstream Anthill repository's latest
release. For each newer release, apply its consumer action (usually: re-copy
the named upstream-owned files verbatim, or re-derive a named binding), then bump
`synced-through`. `anthill sync` automates the upstream-owned re-copy (skills +
framework-invariant files) and the bump.

## Sync log (framework changes applied here)

- 2026-07-20 — installed into the Anthill CLI repo from the `anthill-copy`
  template. General tier (9 skills) copied verbatim; `.anthill/` derived for a
  Go CLI (product workstream `cli`, git checkout as sole exclusive resource,
  propose-only posture). Sanctioned `autonomous` proceed-list re-derived for Go.
- 2026-07-20 — synced to ref 3797138. Added the `submit` skill (tenth
  general-tier skill) and re-copied the refactored `autonomous` skill verbatim;
  moved the Go proceed-list and decisions-log path into the new
  `.anthill/autonomy.md`. This retires the last in-skill adaptation region — the
  general tier is now byte-identical to upstream with no exceptions.
