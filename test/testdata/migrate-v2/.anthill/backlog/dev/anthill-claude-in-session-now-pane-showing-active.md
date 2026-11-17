---
title: 'anthill-claude: in-session ''Now'' pane showing active backlog items and their sessions'
summary: 'anthill-claude mod: docked Now pane (active + validation rows from the CLI''s --json), key chips under transcript rows, detail view on press, hover card, status line, /ant command.'
value: 'Phase 4b of Backlog v2: a mod Pane beside the chat rendering ''anthill backlog list --active --json'' + ''anthill session list --json'', refreshed on turn.complete, so a supervisor session sees what its workers hold without leaving Claude Code (the TUI Now mode covers the terminal case; this covers the in-session case)'
workstream: dev
status: triaged
dispatch: approved
disposition: auto
priority: high
reporter: user
change-type: tooling
risk: additive
verify: claude plugin validate and claude plugin test green in anthill-claude; pane renders list and detail from fixture JSON; chips drawn for ANT-N mentions in user and assistant rows
source: supervisor session 2026-11-17, BACKLOG_V2_SPEC.md §8.2
created-at: "2026-11-17T12:12:44Z"
triaged-at: "2026-11-17T13:47:51Z"
---

## Description
Phase 4b of Backlog v2: a mod Pane beside the chat rendering 'anthill backlog list --active --json' + 'anthill session list --json', refreshed on turn.complete, so a supervisor session sees what its workers hold without leaving Claude Code (the TUI Now mode covers the terminal case; this covers the in-session case)

## Acceptance

## Triage
Phase 4b of backlog v2; depends on the CLI key scheme (ANT-N) landing first. Mockup: claude.ai/artifact/52SoZaCYjZzR93xG8UFeV6; spec: anthill-cli/docs/ITEM_KEYS_SPEC.md §plugin.

## Evidence

## Log
- 2026-11-17T12:53:02Z migrate schema 1 → 2 by anthill
- 2026-11-17T13:47:51Z triage → dev by worker:38acdb2e
- 2026-11-17T13:47:51Z approve by worker:38acdb2e
