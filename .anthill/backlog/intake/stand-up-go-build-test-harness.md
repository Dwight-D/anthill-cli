---
title: Stand up the Go build + test harness
value: Evidence-based done is unenforceable until a headless build+test exists — every cli/dev/bugs item's evidence contract (`go build ./...` / `go test ./...` exit 0) depends on this. It is the bootstrap that unblocks the whole dispatch loop.
source: Anthill install (2026-07-20) — flagged per INSTALLATION.md Step 3 (no verification existed at install).
hint: dev
status: idea
---

Initialize the Go module and a minimal but real build+test setup so the
evidence commands named across `.anthill/` actually run:

- `go.mod` (module path + Go version).
- A buildable `./...` (at minimum a `cmd/anthill` main package or a stub
  package) so `go build ./...` exits 0.
- At least one real `_test.go` so `go test ./...` exits 0 meaningfully.
- Optional but recommended: `gofmt`/`go vet` wired, and a CI workflow running
  both commands.

Acceptance / verify: `go build ./...` exit 0 and `go test ./...` exit 0 from a
clean checkout.
