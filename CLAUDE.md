# Anthill CLI — agent instructions

This repo builds the **Anthill CLI** (Go): the command-line tool that owns the
backlog and escalation schemas and the verbs the Anthill harness skills bind
to, plus the bootstrap surface (`bootstrap`, `scaffold`, `init`, `doctor`,
`sync`, `version`, `validate`). It embeds the framework template from the
sibling `anthill-fw` repository (`go:embed`, pinned ref) — the CLI is
distributed via `go install`, separate from the framework files it installs.

Work on this repo is coordinated by the Anthill installation in the **parent
directory** (`../.anthill/`, workstream `cli`) — there is no installation
embedded here. The parent's CLAUDE.md carries the always-on harness
invariants; this file adds only repo specifics.

## Repo specifics

- **Contracts:** `docs/CLI_INTERFACE_SPEC.md` is the accepted v1
  backlog/escalation surface; `../anthill-fw/CLI_SPEC.md` is the
  bootstrapping-surface contract. A divergence between CLI behavior and spec
  is a bug (fix the CLI) or a spec change (a `framework` backlog item) —
  never silently absorbed.
- **Evidence for done:** `go build ./...` exit 0 and `go test ./...` exit 0,
  plus the item's own `verify`.
- **Schema irreversibles (ask first):** changing the backlog/escalation
  frontmatter schema or the id scheme once items exist — the CLI is the
  schema owner; schema changes are permanent and cross-cutting.
- **Release acts (ask first):** pushing a tag, bumping the embedded template
  ref (re-vendoring from `anthill-fw`). Committing to `main` is fine; bigger
  changes go branch-and-merge, and merges are ask-first unless instructed.
