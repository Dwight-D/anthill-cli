# Anthill — framework provenance & sync state

> **PROJECT-SPECIFIC — fill in on install.** This installation's sync state
> against the Anthill framework home. The upgrade procedure (INSTALLATION.md
> "Sync downstream") reads the `synced-through` date below.

## This installation

- **Role:** consumer (installed the framework; does not maintain it).
- **Framework source:** the Anthill framework home (Spark graph — hub node
  "Anthill", its mechanism nodes, the Installation guide, and the propagation
  pair Anthill Feedback + Anthill Changelog). Where the source repo is
  reachable, its `.claude/skills/` are the canonical general-tier texts.
- **synced-through:** <the upstream framework ref this install is current
  against — a tag/commit of the framework repo when scaffolded by the `anthill`
  CLI (it stamps this automatically); otherwise the Anthill Changelog date for a
  manual install>
- **installed-with:** <`anthill <version>` if the CLI scaffolded this install,
  else `manual`. `anthill doctor` compares the general-tier skills against this
  ref to detect local divergence; `anthill sync` re-copies changed skills
  verbatim and bumps `synced-through`.>

## Two-tier discipline (how this installation stays upgradeable)

- **General tier** = the `.claude/skills/` orchestration skills (`supervisor`,
  `autonomous`, `triage`, `dispatch`, `dispatch-loop`, `dispatch-receive`,
  `expedite`, `escalate`, `wake-up`). These are byte-identical to upstream
  (except the two sanctioned `autonomous` adaptation points). **Never
  locally edit a general-tier skill** — divergence across installations is
  the failure mode the two-tier split exists to prevent. Upgrading = replacing
  skill files.
- **Specific tier** = everything under `.anthill/` — bindings, workstreams,
  resources, the launcher. All adaptation lives here.

## Flag upstream, don't fork

A gap or finding about the framework *itself* is filed to the framework home
as one `Anthill Feedback: <slug>` node — never fixed by editing a local skill.
If you need a mitigation NOW, put it in `.anthill/` config and name it in the
feedback item so the maintainer can supersede it.

## Sync downstream (periodically, and when the user relays framework news)

Read Anthill Changelog entries newer than `synced-through` above; apply each
entry's consumer action (usually: re-copy the named skills verbatim from the
source repo, or re-derive a named binding); bump `synced-through`.

## Sync log (framework changes applied here)

- <date> — installed at Anthill Changelog `synced-through: <date>`.
