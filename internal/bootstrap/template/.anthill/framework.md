# Anthill — framework provenance & sync state

> **PROJECT-SPECIFIC — fill in on install.** This installation's sync state
> against the upstream Anthill repository. The upgrade procedure
> (INSTALLATION.md "Sync downstream") reads the `synced-through` ref below.

## This installation

- **Role:** consumer (installed the framework; does not maintain it).
- **Framework source:** the upstream Anthill repository — the framework's source
  of truth. Its `.claude/skills/` are the canonical general-tier texts; releases
  are tagged, and each tag is a version this install can track.
- **synced-through:** <the upstream release this install is current against — a
  tag/commit of the Anthill repository. The `anthill` CLI stamps this
  automatically at scaffold time; on a manual install, record the tag/commit you
  copied from.>
- **installed-with:** <`anthill <version>` if the CLI scaffolded this install,
  else `manual`. `anthill doctor` compares the general-tier skills against this
  ref to detect local divergence; `anthill sync` re-copies changed skills
  verbatim and bumps `synced-through`.>

## Two-tier discipline (how this installation stays upgradeable)

The split is by **ownership**, not by directory: some files are authored
upstream and must never diverge (they follow `sync`); the rest are yours to
adapt (`sync` never touches them).

- **Upstream-owned (follows `sync`)** — reconciled byte-identical to upstream on
  every sync, restored on `--force`:
  - the `.claude/skills/` orchestration skills (`supervisor`, `autonomous`,
    `triage`, `submit`, `dispatch`, `dispatch-loop`, `dispatch-receive`,
    `expedite`, `escalate`, `wake-up`), and
  - the **framework-invariant** files: the `.anthill/` reference READMEs, the
    supervisor brief template, and the `tools/` launchers.

  **Never locally edit an upstream-owned file** — divergence across
  installations is the failure mode the split exists to prevent. Upgrading =
  replacing these files.
- **Project-owned (yours; `sync` never touches)** — the adaptation config
  (`backlog/bindings.md`, `workstreams.md`, `autonomy.md`, `resources.md`,
  `supervisor/bindings.md`, this `framework.md`) and runtime state
  (`backlog/CHANGELOG.md`, `escalations/LOG.md`, `decisions.md`,
  `supervisor/agenda.md`). These arrive once at `scaffold` and are refused (left
  as-is) on any re-scaffold. (The `autonomous` skill's per-project inputs live
  in `autonomy.md`, project-owned config the skill loads at invocation — filling
  it in is not a skill edit.)

## Flag upstream, don't fork

A gap or finding about the framework *itself* is filed upstream to the Anthill
repository (an issue or PR) — never fixed by editing a local skill. If you need
a mitigation NOW, put it in `.anthill/` config and name it in the upstream
report so the maintainer can supersede it.

## Sync downstream (periodically)

Compare `synced-through` above against the upstream Anthill repository's latest
release. For each newer release, apply its consumer action (usually: re-copy the
named upstream-owned files verbatim, or re-derive a named binding), then bump
`synced-through`. `anthill sync` automates the upstream-owned re-copy (skills +
framework-invariant files) and the bump; project-owned bindings are re-derived
by hand when a release calls for it.

## Sync log (framework changes applied here)

- <date> — installed, tracking upstream ref <tag/commit>.
