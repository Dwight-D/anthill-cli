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

- **General tier** = the `.claude/skills/` orchestration skills (`supervisor`,
  `autonomous`, `triage`, `submit`, `dispatch`, `dispatch-loop`,
  `dispatch-receive`, `expedite`, `escalate`, `wake-up`). These are
  byte-identical to upstream — no exceptions. **Never locally edit a
  general-tier skill** — divergence across installations is the failure mode
  the two-tier split exists to prevent. Upgrading = replacing skill files.
  (The `autonomous` skill's per-project inputs live in `.anthill/autonomy.md`,
  specific-tier config the skill loads at invocation — not a skill edit.)
- **Specific tier** = everything under `.anthill/` — bindings, workstreams,
  resources, the launcher. All adaptation lives here.

## Flag upstream, don't fork

A gap or finding about the framework *itself* is filed upstream to the Anthill
repository (an issue or PR) — never fixed by editing a local skill. If you need
a mitigation NOW, put it in `.anthill/` config and name it in the upstream
report so the maintainer can supersede it.

## Sync downstream (periodically)

Compare `synced-through` above against the upstream Anthill repository's latest
release. For each newer release, apply its consumer action (usually: re-copy the
named general-tier skills verbatim, or re-derive a named binding), then bump
`synced-through`. `anthill sync` automates the skill re-copy and the bump.

## Sync log (framework changes applied here)

- <date> — installed, tracking upstream ref <tag/commit>.
