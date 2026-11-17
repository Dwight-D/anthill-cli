---
title: 'Backlog item keys: PREFIX-N ids (ANT-74) with file-based allocation'
summary: Sequential PREFIX-N item keys (ANT-74) become the id; files 0074-<slug>.md; file-based allocation (scan + sequence high-water mark + exclusive create); migrate v2→v3 with former-id; validate/doctor duplicate-key check; renumber verb.
value: 'Items are referred to by 48-char slugs that nothing recognises as anthill ids; a short self-describing key (ANT-74) is typeable, unambiguous next to Spark/GitHub #N, and lets the Claude plugin and other tools spot and act on item mentions. Proposal with options and plugin mockup: .anthill/supervisor/scratchpad/2026-11-17-item-keys-and-plugin-surface.md. Schema change: user-gated.'
workstream: cli
status: in-progress
dispatch: approved
disposition: design
priority: high
reporter: supervisor
change-type: contract-edit
risk: behavior-change
verify: go build ./... and go test ./... green in anthill-cli; new allocates ANT-N keys; show resolves key, bare number and former-id; migrate converts this installation's items; validate flags a duplicate key
source: supervisor session 2026-11-17, proposal 2026-11-17-item-keys-and-plugin-surface.md
created-at: "2026-11-17T13:18:51Z"
triaged-at: "2026-11-17T13:47:51Z"
claimed-at: "2026-11-17T13:55:51Z"
claimed-by: cli-keys
session: 38acdb2e-2f54-4bd2-a2e8-29b22a9eec0f
---

## Description

## Acceptance

## Triage
User approved 2026-11-17 the recommendation in .anthill/supervisor/scratchpad/2026-11-17-item-keys-and-plugin-surface.md (K1 ANT-74, K2 key as id, K3 escalations later, K4 read-only pane, K5 prompt-side deferred). Spec: anthill-cli/docs/ITEM_KEYS_SPEC.md.

## Evidence

## Log
- 2026-11-17T13:18:51Z new by worker:38acdb2e
- 2026-11-17T13:47:51Z triage → cli by worker:38acdb2e
- 2026-11-17T13:47:51Z approve by worker:38acdb2e
- 2026-11-17T13:55:51Z claim (forced) by cli-keys (session 38acdb2e…)
