---
title: TUI text wrap in panels
summary: Wrap long text to pane width in the TUI content/detail rendering so nothing clips off screen.
value: Long text currently clips off screen with no way to read it; wrap text to pane width in the content/detail rendering so everything is readable
workstream: cli
status: triaged
dispatch: approved
disposition: auto
priority: normal
reporter: user
change-type: tooling
risk: reversible
verify: long body text wraps to pane width in the detail pane (no off-screen clipping) in a user test-drive; go test ./... green
created-at: "2026-11-17T11:10:30Z"
---

## Description
Long text currently clips off screen with no way to read it; wrap text to pane width in the content/detail rendering so everything is readable

## Acceptance

## Triage

## Evidence

## Log
- 2026-11-17T12:53:03Z migrate schema 1 → 2 by anthill
- 2026-11-17T12:54:08Z release by worker:efcb0f1d: was claimed by -
