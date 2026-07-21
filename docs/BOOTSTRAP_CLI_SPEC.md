# Anthill CLI — Bootstrapping & Integrity Surface (Specification)

Status: PROPOSED. Source of requirements: the "Anthill Bootstrapping" mechanism
node in the framework home. This document is the implementable contract for the
**install-time / integrity** role of the `anthill` binary. The **runtime
schema-owner** role (`backlog`, `escalation`) is specified separately in
`docs/CLI_INTERFACE_SPEC.md`; the two share one binary but are conceptually
distinct.

## 1. Overview — two roles of one binary

The single `anthill` binary carries two responsibilities:

1. **Bootstrapping / integrity** (this spec): `bootstrap`, `scaffold`,
   `version`, `doctor`, `sync`. Getting a project from zero to a scaffolded
   Anthill install, and keeping the general-tier skills byte-identical to a
   pinned upstream release over time. It makes the two-tier discipline ("copy
   the general-tier skills verbatim, never locally edit them") *enforceable by
   the tool* rather than merely remembered.
2. **Schema ownership** (`docs/CLI_INTERFACE_SPEC.md`): `backlog`, `escalation`.
   The sole writer of backlog/escalation record frontmatter at runtime.

The bootstrapping surface automates only the **mechanical** half of adoption
(verbatim skill copy + placeholder scaffold). The **judgment** half — deriving
the workstream taxonomy, resource inventory, never-auto types, caps, sweep
order; authoring the always-on file; the verification round-trip — stays with
the agent and the user. The tool never makes a derivation decision.

## 2. Binary name and invocation

The binary is **`anthill`**. It is built from `./cmd/anthill` (module
`github.com/Dwight-D/anthill-cli`), so `go install
github.com/Dwight-D/anthill-cli/cmd/anthill@<tag>` installs a command named
`anthill`. All commands are invoked `anthill <command>` (e.g. `anthill
bootstrap`, `anthill scaffold`). Where upstream prose or `BOOTSTRAP.md` shows
`anthill-cli <command>`, that is the repository name leaking into the invocation
example and is corrected to `anthill <command>` (tracked as an upstream
documentation fix, not a local behavior change).

## 3. Provenance — the embedded template

`scaffold`, `sync`, `doctor` (integrity), and the ref reported by `version` all
read a **pinned copy of the upstream framework template**, compiled into the
binary with `go:embed`. This guarantees the verbatim copy (byte-identical from a
fixed release) and lets `scaffold`/`doctor` work offline.

### Payload

Vendored into the CLI repo under `internal/bootstrap/template/` and embedded
whole. Contents (the mechanical half of an install):

- `.claude/skills/` — the nine general-tier skills, **pristine** (no project
  adaptations; in particular the `autonomous` skill carries its upstream
  placeholder proceed-list and decisions-log path).
- `.anthill/` — the placeholder config tree with template quote-blocks and
  `<angle-bracket>` fill-ins intact, and empty runtime directories
  (`intake/`, workstream dirs, `escalations/`, `supervisor/scratchpad/`) carrying
  their `.gitkeep`/`.gitignore` files.
- `CLAUDE.template.md` — the always-on-file starter.
- `tools/` — the launcher scripts (`supervise.sh` / `.ps1`).
- `.gitignore` — the framework's ignore rules.

Embedding directive (validated): `//go:embed all:template` — the `all:` prefix
is required because the payload is almost entirely dotfiles and dot-directories
(`.claude`, `.anthill`, `.gitignore`), which a bare `//go:embed` silently
excludes.

### Pinned ref

The vendored payload corresponds to exactly one **tagged upstream release**. The
binary records that ref as a build-stamped constant (the embedded-template ref)
alongside its own version. Every `scaffold` writes that ref, plus the install
date, into the new install's `.anthill/framework.md` `synced-through` field —
the baseline `sync` and `doctor` compare against.

**Pinned source (recorded):**
- Upstream template repo: `https://github.com/Dwight-D/anthill`
- Pinned ref: `ccd6807ba30d85632ebd1435145b3d0d50789567`
- Canonical `BOOTSTRAP.md` (fetchable, for `bootstrap` to print):
  `https://raw.githubusercontent.com/Dwight-D/anthill/main/BOOTSTRAP.md`
  (human view: `https://github.com/Dwight-D/anthill/blob/main/BOOTSTRAP.md`)

The payload vendored at that ref lives in `internal/bootstrap/template/` (33
files: the nine pristine skills, the `.anthill/` placeholder tree, `tools/`,
`CLAUDE.template.md`, `.gitignore`). The pinned ref and the BOOTSTRAP URL are
Go constants in `internal/bootstrap` (`TemplateRef`, `BootstrapDocURL`). The CLI
repo's CI re-vendors `internal/bootstrap/template/` from this source on each
tagged CLI release and updates `TemplateRef`.

Note the payload deliberately excludes the framework home's own `README.md`,
`INSTALLATION.md`, `CLAUDE.md`, `CLI_SPEC.md`, and `BOOTSTRAP.md` — those are
authoritative docs fetched from upstream, not scaffolded into a consumer.

## 4. Commands

Global conventions are shared with the runtime surface (`docs/CLI_INTERFACE_SPEC.md`
§2): `--json` for machine output, the exit-code table (0 ok · 1 internal ·
2 usage · 3 validation · 4 not-found · 5 conflict · 6 precondition), and the
stdout/stderr split. The bootstrapping commands are **idempotent and
non-destructive**: re-running converges, and nothing overwrites derived config
or user edits without an explicit `--force`.

### 4.1 `anthill bootstrap [--open]`

Pure redirect; the headline entrypoint. Prints the canonical `BOOTSTRAP.md` URL
plus a compact agent-directed preamble ("You are installing Anthill. Read this
document, run `anthill scaffold`, then drive the derivation session with the
user."). Zero side effects — safe to run in any directory, in or out of a repo.

- **Flags.** `--open` opens the URL in the platform browser (`start` on Windows,
  `open` on macOS, `xdg-open` on Linux) instead of printing.
- **`--json`.** `{ "entrypoint": "<url>", "preamble": "<text>" }`.
- **Exit codes.** 0 always (1 only if `--open` fails to launch a browser).
- Does **not** embed the install procedure — the fetched Markdown is
  authoritative so instructions cannot drift from the binary.

### 4.2 `anthill scaffold [--into <dir>] [--force] [--dry-run]`

The mechanical install. Into `<dir>` (default: current directory) writes, from
the embedded pinned template: the nine general-tier skills verbatim, the
`.anthill/` placeholder tree (quote-blocks intact) with empty runtime dirs,
`CLAUDE.template.md`, `tools/`, and `.gitignore`; then stamps
`.anthill/framework.md` `synced-through` with the embedded ref + install date.

- **Preconditions.** Must run inside a git repository (exit 6 otherwise).
- **Non-destructive rule.** For each target path: write if absent; rewrite if
  present **and byte-identical to the pinned template** (an un-derived
  placeholder or an unmodified skill — safe); **refuse** (exit 6) if present and
  **differs** from the template (already derived or user-edited) unless
  `--force`. `--force` overwrites differing files.
- **Flags.** `--into <dir>` target root; `--force` overwrite differing files;
  `--dry-run` compute and print the manifest without writing.
- **Output.** A written / skipped(identical) / refused(differs) manifest, then
  the next agent step ("now derive `.anthill/` with the user — see
  `INSTALLATION.md` Steps 3–6"). `--json`: `{ "written": [...], "skipped":
  [...], "refused": [...], "ref": "<tag>" }`.
- **Exit codes.** 0 success (including a clean `--dry-run`); 3 if any path was
  refused (differs, no `--force`); 6 not in a git repo.

### 4.3 `anthill version`

Prints the CLI's own version and the embedded upstream template ref (tag/commit)
— the value an agent records as `synced-through` on a manual install.

- **`--json`.** `{ "version": "<cli-version>", "template_ref": "<tag>" }`.
- **Exit codes.** 0.

### 4.4 `anthill doctor [--strict]`

One `doctor` covering both roles, reported as two labeled sections. Read-only.

**Section A — install integrity** (this spec):
- **skill integrity** — each installed `.claude/skills/*` is byte-identical to
  the embedded pinned version. Any diff is flagged as an illegal local edit to a
  general-tier skill (the exact divergence the two-tier split prevents). The two
  sanctioned `autonomous` adaptation points — the proceed-list and the
  decisions-log path — are recognized and exempted.
- **structure** — the expected `.anthill/` tree is present.
- **derivation status** — which `.anthill/` files still hold template
  quote-blocks / `<angle-bracket>` fill-ins (i.e. remain un-derived). Reported
  as information, not a hard failure, except under `--strict`.
- **sync status** — installed `synced-through` vs the embedded ref (up-to-date /
  behind).

**Section B — runtime data integrity** (from `docs/CLI_INTERFACE_SPEC.md`):
`.anthill/` discoverable, required config present, `workstreams.md` sweep-order
names existing directories, `backlog validate --strict` clean, escalation
records well-formed, no answered-but-unapplied escalations.

- **Flags.** `--strict` — exit non-zero on any skill diff, any remaining
  placeholder, or any Section B failure (CI use).
- **`--json`.** `{ "ok": bool, "checks": [ { "section", "name", "ok",
  "detail" } ] }`.
- **Exit codes.** 0 healthy; 3 on an integrity problem (skill diff, structure
  gap, Section B failure; plus remaining placeholders under `--strict`).
- Serves `INSTALLATION.md` Step 6 and ongoing discipline.

### 4.5 `anthill sync [--dry-run] [--force]`

Realizes `INSTALLATION.md`'s "Sync downstream". Diffs the installed
**upstream-owned files** against the (newer) embedded version, re-copies changed
ones verbatim, and bumps `.anthill/framework.md` `synced-through` to the embedded
ref.

- **Scope — the reconciled units.** Two kinds, both compared byte-for-byte with
  no exceptions:
  1. the general-tier skills under `.claude/skills/**` (labeled by skill name), and
  2. the **framework-invariant** non-skill files (labeled by payload path): the
     `.anthill/` reference READMEs (`.anthill/README.md`,
     `.anthill/backlog/README.md`, `.anthill/escalations/README.md`), the
     supervisor brief template and scratchpad README, and the `tools/`
     launchers (`supervise.ps1`, `supervise.sh`). These are authored upstream,
     identical across every install, and hold no per-project fill-in, so they
     ride upstream exactly like a skill.
- **What sync never touches.** Project-derived config (`workstreams.md`,
  `backlog/bindings.md`, `autonomy.md`, `resources.md`, `supervisor/bindings.md`,
  `framework.md`, `CLAUDE.template.md`) and runtime state (`backlog/CHANGELOG.md`,
  `escalations/LOG.md`, `decisions.md`, `supervisor/agenda.md`). Overwriting
  either would clobber per-install content, so they are excluded by design and
  only ever arrive/change via `scaffold` (a one-time, refuse-if-derived install).
- **Conflict rule (uniform per unit).** A unit that differs while the install
  already claims the embedded ref is an unexpected local edit → reported as a
  conflict and left unchanged (exit 3) unless `--force`. A unit that differs on a
  behind install is an upstream update → re-copied verbatim.
- **Flags.** `--dry-run` show the diff without applying; `--force` apply even
  when a unit has an unexpected local edit (overwrites it — the diff is shown
  first).
- **`--json`.** `{ "updated": [...], "unchanged": [...], "conflicts": [...],
  "from_ref": "<old>", "to_ref": "<new>" }`. List entries are skill names or
  framework-invariant file paths.
- **Exit codes.** 0 applied (or clean `--dry-run`); 3 on an unresolved
  conflict without `--force`.

### 4.6 `esc` alias

The runtime escalation group is `anthill escalation …`. Register `esc` as a
hidden alias so the node's shorthand (`esc`) resolves. No behavior change.

## 5. What stays out of scope

- The derivation session, always-on-file authoring, and the verification
  round-trip — agent + user judgment, driven from `INSTALLATION.md` Steps 3–6.
- Any permission elevation — mode gating is a launcher concern
  (`tools/supervise.*`); the CLI never escalates a session.
- Authoring or patching general-tier skills — the CLI copies and verifies them;
  framework changes originate upstream and propagate by re-embedding at a new
  tagged release.

## 6. Implementation waves (dependency-ordered)

- **Wave 0 (no embed needed).** `bootstrap` (needs only the `BOOTSTRAP.md` URL);
  the `esc` alias; register empty `scaffold`/`sync` command stubs under the root
  so the surface is discoverable. Add `template_ref` plumbing to `version` as a
  build-stamped constant (value "unset" until Wave 1).
- **Wave 1 (embed).** Vendor the pinned upstream template into
  `internal/bootstrap/template/`; wire `//go:embed all:template`; record the
  pinned ref; make `version` report it. Requires the external input in §3.
- **Wave 2.** `scaffold` (write/skip/refuse manifest, git-repo precondition,
  framework.md stamping) + its tests against a temp repo.
- **Wave 3.** `doctor` Section A (skill byte-identity with the `autonomous`
  exemptions, structure, derivation-status, sync-status) merged with the
  existing Section B; `--strict`.
- **Wave 4.** `sync` (skill-level diff/apply, adaptation-region preservation,
  conflict reporting).

Each wave's evidence: `go build ./...` and `go test ./...` exit 0, plus a
black-box test exercising the command against a temp tree (matching the existing
`test/` suite's style).

## 7. Integration with the existing binary

- The new commands are added under the existing cobra root in `internal/cli/`;
  a new `internal/bootstrap/` package holds the embed, the template-write logic,
  and the skill-diff logic.
- `doctor` already exists with Section B checks; this spec extends it with
  Section A and the two-section reporting — the existing runtime checks are
  retained unchanged as Section B.
- No change to the `backlog`/`escalation` runtime surface beyond the additive
  `esc` alias.
