# Anthill CLI

`anthill` is the command-line tool for the **Anthill** agent-harness framework.
It owns the backlog and escalation schemas, scaffolds the framework into a
repository, and keeps an installation's general-tier skills verbatim-synced
with upstream. It is the mechanical half of Anthill: the durable state and
verbs that the harness's orchestration skills bind to.

## What Anthill is

Anthill is a framework for running autonomous and supervised agent workstreams
against a codebase. It has two tiers:

- **General tier** — portable orchestration skills under `.claude/skills/`
  (`supervisor`, `autonomous`, `triage`, `submit`, `dispatch`, `dispatch-loop`,
  `dispatch-receive`, `expedite`, `escalate`, `wake-up`). These are
  byte-identical across every installation; you upgrade them by re-copying, never
  by local edit.
- **Specific tier** — everything under `.anthill/`: the backlog, escalation
  records, workstream and resource bindings, autonomy config, and the launcher.
  All project adaptation lives here.

This CLI is the schema owner and install manager for both tiers. The framework
itself is maintained upstream at
[Dwight-D/anthill](https://github.com/Dwight-D/anthill); this repository builds
the binary and pins (embeds) a specific upstream template ref that `scaffold`,
`doctor`, and `sync` read as their baseline.

## Relationship to the upstream framework

- The framework's source of truth is the upstream Anthill repository. Its
  `.claude/skills/` are the canonical general-tier texts and its `.anthill/`
  tree is the canonical placeholder skeleton.
- This CLI **embeds a pinned copy** of that template (see
  `internal/bootstrap/template/`) at the ref in `internal/bootstrap/embed.go`.
  `anthill scaffold` writes it; `anthill doctor` checks an install against it;
  `anthill sync` brings an install up to it.
- A gap in the framework is filed **upstream** (an issue or PR), never patched
  by editing a local skill. Adopting a newer framework release means re-vendoring
  the embedded template and bumping the pinned ref, then consumers run
  `anthill sync`.

This repository also runs the Anthill harness **on itself** — its own
`.claude/skills/` and `.anthill/` are a live installation.

## Installation

Requires Go 1.26+.

```sh
# Install the binary onto your PATH
go install github.com/Dwight-D/anthill-cli/cmd/anthill@latest

# …or build from a checkout
git clone https://github.com/Dwight-D/anthill-cli
cd anthill-cli
go build -o anthill ./cmd/anthill
```

Verify:

```sh
anthill version
```

## Installing the framework into a repository

From inside the target git repository:

```sh
# 1. Print the authoritative install guide (fetched fresh, never embedded)
anthill bootstrap

# 2. Write the embedded framework template into the repo
anthill scaffold

# 3. Derive the .anthill/ tree with the guide, then check health
anthill doctor
```

`scaffold` writes the ten general-tier skills, the `.anthill/` placeholder tree,
`CLAUDE.template.md`, `tools/`, and `.gitignore`, and stamps the pinned template
ref into `.anthill/framework.md`. It never overwrites a file you have derived
without `--force`. After scaffolding you derive the specific-tier placeholders
(workstreams, resources, bindings, `.anthill/autonomy.md`) — `anthill doctor`
lists which files still hold template placeholders.

## Everyday commands

| Command | Purpose |
| --- | --- |
| `anthill scaffold` | Write the embedded framework template into a git repo. |
| `anthill doctor [--strict]` | Read-only health check: skill integrity, structure, derivation status, sync status, and runtime data integrity. |
| `anthill sync [--dry-run] [--force]` | Bring installed general-tier skills up to the embedded ref; re-copies changed skills verbatim and bumps `synced-through`. |
| `anthill backlog …` | Create, list, triage, claim, and close backlog items. |
| `anthill escalation …` | Raise, list, answer, and apply escalation records. |
| `anthill validate` | Validate the whole tree (backlog + escalations). |
| `anthill version` | Print the CLI version and the embedded template ref. |

Global flags: `--json` (machine-readable output), `--root <dir>` (locate the
`.anthill/` tree), `--quiet`, `--no-color`.

## Keeping an install current

When the embedded ref advances (a new framework release is vendored into this
CLI), a consumer upgrades their skills with:

```sh
anthill sync --dry-run   # preview the skill-level diff
anthill sync             # re-copy changed skills verbatim, bump synced-through
```

Every skill is compared byte-for-byte with no exceptions. A skill carrying an
unexpected local edit is reported as a conflict and left unchanged; re-run with
`--force` to overwrite it (the diff is shown first).

## Development

```sh
go build ./...
go test ./...
```

Both must exit 0. See `CLAUDE.md` for the repository's agent-work conventions
and safety invariants.
