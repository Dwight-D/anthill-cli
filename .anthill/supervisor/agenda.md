# Agenda

Updated: 2026-07-20 (install / derivation session)

> The supervisor's survival file: user intent ONLY — goals, directives,
> priorities, constraints as stated by the user. No findings, no progress, no
> reasoning (recreate those from the task board, git log, and decision log).
> Update it whenever user intent changes, before acting on the change.

## Standing goals

- Build the **Anthill CLI**: the command-line tool that owns the backlog and
  escalation schemas and provides the verbs the Anthill skills bind to
  (`backlog new/list/set/next/claim/close/validate`, escalation verbs, id
  generation, frontmatter invariant-checking). Written in Go.

## Directives

- 2026-07-20 — Installed the Anthill harness into this repo (dogfooding: the
  CLI being built is this harness's future schema owner). Posture is
  **propose-only** — triage recommends, the user approves.
- 2026-07-20 — Build pipeline commissioned (via `/supervisor`): (1) expedite
  the build+test harness dev item; (2) propose a CLI interface for user review;
  (3) on user go-ahead, parallel test-suite + implementation; (4) run the suite.
- 2026-07-20 — **CLI interface review (in progress).** Proposal at
  `docs/CLI_INTERFACE_SPEC.md`. User decisions so far: (a) CLI owns **both**
  backlog AND escalation records in v1; (b) approval is a **separate gated
  `approve` verb** — `set status=approved` is refused. Deferred to worker
  recommendation unless overridden: defer sqlite, drop uuid, `--block`
  non-terminal, CAS not lockfile, `--json` array, flag `--hint`, allow post-hoc
  `set title=`. User reviewed and **approved** the spec; **TUI deferred** to a
  later scope (planned, not v1). `docs/CLI_INTERFACE_SPEC.md` is the accepted
  v1 contract. v1 implemented on `integration` with a green black-box suite
  (not yet merged to `main` — pending user decision on merge/push).
- 2026-07-20 — **Tech stack for the CLI frontend:** mirror the user's other CLI
  — `spf13/cobra` (+`pflag`) for command structure, `charmbracelet/bubbletea`
  `bubbles` `lipgloss` for any interactive TUI, `modernc.org/sqlite` (pure-Go,
  no cgo) if an index/cache is needed, `google/uuid`. MCP SDK
  (`modelcontextprotocol/go-sdk`) available if a server surface is wanted.
  Backlog/escalation source of truth stays the markdown files in `.anthill/`.

## Constraints

- Single shared git checkout, no worktree isolation → team worker cap 2,
  dispatch serial (see `.anthill/resources.md`).
- No headless verification exists yet; standing up the Go build+test harness is
  the first `dev` backlog item and gates evidence-based done for everything else.
