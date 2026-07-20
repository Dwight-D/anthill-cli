# Autonomy contract config

The project-specific inputs to the autonomy contract for the Anthill CLI. The
portable contract — the safety invariants still bind, work on a branch,
log-and-continue vs. stop-and-ask, and the expected permission mode — lives in
the `autonomous` skill and is identical across installations. What is specific
to this project is the concrete list of routine actions below and where routine
decisions are logged. The `autonomous` skill loads this file on invocation.

## Proceed freely (do not ask permission)

- Create/edit/delete the Anthill CLI's Go source, tests, tooling, and dev-docs
  (everything except the files reserved in CLAUDE.md's config-file ownership).
- Run the Go toolchain: `go build ./...`, `go test ./...`, `go vet ./...`,
  `gofmt`, `go run`, and the project's own scripts.
- Build the CLI binary via `go build` and run it against fixtures/scratch data.
- git: path-scoped add, commit, push to the designated work branch.
- Read anything in the repo (rails block the exceptions).
- Add/update Go module dependencies (`go get`, `go mod tidy`) when the task
  clearly needs them.

## Decisions log

- **Path:** `.anthill/decisions.md` — the routine-choice log. When a
  non-blocking question comes up mid-task, the worker records the choice here as
  one line and continues, surfacing the log at the end of the task.
