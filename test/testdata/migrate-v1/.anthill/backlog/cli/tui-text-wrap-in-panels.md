---
workstream: cli
title: TUI text wrap in panels
value: Long text currently clips off screen with no way to read it; wrap text to pane width in the content/detail rendering so everything is readable
risk: reversible
verify: long body text wraps to pane width in the detail pane (no off-screen clipping) in a user test-drive; go test ./... green
status: in-progress
claimed-at: "2026-08-29T14:38:51Z"
---
