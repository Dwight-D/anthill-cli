---
name: sync
description: Bring an installation's general-tier skills up to the CLI's embedded
  template ref via `anthill sync`, reconciling any conflict without discarding
  the project's config. Use when asked to "sync anthill", "update the framework",
  or after an `anthill` CLI upgrade.
---

# sync

Advance this installation's `.claude/skills/` to the framework ref the `anthill`
CLI embeds, and record the new ref in `.anthill/framework.md`. The general tier
is byte-identical to upstream by design, so syncing is a **whole-file verbatim
re-copy** of changed skills — never a per-skill merge. All per-project
adaptation lives under `.anthill/` (the proceed-list in `.anthill/autonomy.md`,
bindings, resources, workstreams), which `anthill sync` never touches. That
separation is what lets a changed skill be simply overwritten.

## Procedure

1. **Upgrade the CLI first.** The template ref is embedded in the `anthill`
   binary and only advances when the binary does — an outdated CLI syncs the
   install to a stale pin. Upgrade `anthill` to its latest release (however it
   was installed), then `anthill version` — record the embedded template ref;
   this is what the install will be current against when done.
2. **Dry-run.** `anthill sync --dry-run` reports the per-skill diff.
   - **0 conflicts** → `anthill sync` applies the re-copy and bumps
     `synced-through`. Go to step 5.
   - **Conflicts** → step 3. A conflict is a general-tier skill that differs
     from the embedded version by a **local edit** — an illegal divergence,
     since general-tier skills are never locally edited. Do NOT reflexively
     `--force` past it before step 3.
3. **Reconcile each conflicted skill at its source.** The resolution is always
   the same: the skill returns to the pristine upstream text, because a skill is
   never a home for project config.
   a. **Rescue any trapped config first.** Inspect the local edit. If it carries
      project-specific content that belongs in `.anthill/` (e.g. a legacy install
      that filled a proceed-list *inside* the `autonomous` skill), move that
      content to its `.anthill/` home — the proceed-list into
      `.anthill/autonomy.md` — before overwriting. Verify the config file holds
      it, then continue.
   b. **Confirm the remainder is drift.** Anything else in the diff (a
      personalized `description`, reflowed prose, a stray edit) is accidental and
      is discarded.
   c. **Accept upstream.** `anthill sync --force` overwrites the conflicted skill
      with the embedded version (it prints the diff first — read it). Or revert
      the file to the embedded text by hand.
4. **Re-verify.** `anthill sync --dry-run` → expect 0 conflicts → `anthill sync`.
5. **Confirm health.** `anthill doctor` — `skill-integrity` and `sync-status`
   should be green. If `framework.md` predates the CLI and `doctor` warns
   `no synced-through line`, rewrite its `synced-through` to the CLI's git-ref
   format (the ref from step 1) by hand.
6. **Apply non-mechanical consumer actions.** Read the upstream `CHANGELOG.md`
   entries newer than the install's previous ref. `sync` re-copies skills but
   does not automate config changes a release calls for — a re-derived binding,
   a new `.anthill/` file to fill in. Apply those, or surface them to the user.

## `--force` is precise, not a big hammer

`--force` overwrites a locally-edited skill with the embedded version, showing
the diff first. It is safe once step 3a has confirmed nothing project-specific is
trapped in the skill — sanctioned adaptation never lives in a skill, so
overwriting a general-tier skill only discards drift. Never run it blind: read
the diff, rescue trapped config, then force. When in doubt, reconcile by hand.

## Boundaries

- **General tier only.** `sync` moves `.claude/skills/*`. It never edits
  `.anthill/` config, backlog items, escalation records, or queue state.
- **No local skill authorship.** The only edit `sync` makes to a skill is
  reverting it to the exact upstream text. You never hand-write skill content to
  "resolve" a conflict — a framework gap is flagged upstream, not forked here.
- **Config is rescued, never invented.** Step 3a only relocates content that
  already exists in the install; it does not fabricate proceed-list entries or
  bindings.
