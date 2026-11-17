---
title: 'Framework: item keys (ANT-74) in skills, templates and brief template'
summary: Replace slug-style item references in skills, templates (brief-template, bindings, README, workstreams) and INSTALLATION.md with the ANT-N key form; CHANGELOG entry with the consumer action (anthill migrate).
value: Skills, templates and adoption docs refer to items by slug examples; with PREFIX-N keys as ids every example, brief template line and CHANGELOG consumer action must say key (ANT-74) so agents write and recognise the new form
workstream: framework
status: done
dispatch: approved
disposition: auto
priority: high
reporter: supervisor
change-type: doc
risk: reversible
verify: grep over anthill-fw finds no slug-style example id in a backlog verb; CHANGELOG.md carries the keys entry; every edited skill reads self-contained
source: supervisor session 2026-11-17, ITEM_KEYS_SPEC.md
created-at: "2026-11-17T13:47:51Z"
triaged-at: "2026-11-17T13:47:51Z"
claimed-at: "2026-11-17T13:56:12Z"
claimed-by: fw-keys
session: 38acdb2e-2f54-4bd2-a2e8-29b22a9eec0f
delivered-at: "2026-11-17T14:00:01Z"
closed-at: "2026-11-17T14:00:29Z"
---

## Description

## Acceptance

## Triage
Split of the keys mission at triage: CLI side is backlog-item-keys-prefix-n-ids-ant-74-with-file; dispatched after the CLI phase lands so the re-vendor carries both.

## Evidence
**2026-11-17T14:00:01Z** — fw-keys
Branch item-keys in anthill-fw, pushed to origin; head f96774b6b0b56de00d4247ce3f0dfd80b9a34154 (5 commits on top of main f10dbcf). DoD grep: grep -rniE 'kebab|slug of the title|<id>\.md|schema: 2' (excluding .git, CHANGELOG.md) returns nothing (exit 1); grep -rn '<id>' returns nothing; spec grep '<id>\.md\|kebab slug' returns nothing. workstreams.md frontmatter: schema: 3 and 'key-prefix: <PREFIX>   # 2-5 uppercase letters; anthill init fills it' (angle-bracket placeholder per the template's convention; the spec wrote {{PREFIX}}). Self-contained rule checked against anthill-fw CLAUDE.md 'Writing conventions' (state what a thing is; contrast only in CHANGELOG.md / migration lines): all edited artifacts state the key scheme directly; the only contrast wording is in CHANGELOG.md ('Previously the id was the kebab slug...') and the INSTALLATION.md sync-downstream migration line. Cross-references: skill names and paths unchanged (bindings.md 'Key scheme' referenced from workstreams.md; INSTALLATION step 3 questions renumbered 7->8 for the new key-prefix question; nothing outside CHANGELOG history cites question numbers). CHANGELOG.md 'Item keys' entry with consumer action: anthill backlog migrate --key-prefix XXX, then anthill sync. Files edited (23): .anthill/README.md, .anthill/backlog/README.md, .anthill/backlog/bindings.md, .anthill/backlog/workstreams.md, .anthill/decisions.md, .anthill/dispatch/ledger.md, .anthill/escalations/LOG.md, .anthill/escalations/README.md, .anthill/resources.md, .anthill/supervisor/brief-template.md, .claude/skills/{autonomous,dispatch,dispatch-loop,dispatch-receive,escalate,expedite,rehydrate,submit,supervisor,triage}/SKILL.md, CLAUDE.template.md, INSTALLATION.md, CHANGELOG.md. Spec-gap calls: (1) placeholder <PREFIX> not {{PREFIX}}; (2) CLAUDE.template.md had no 'id scheme' invariant, so a 'Backlog schema and key prefix' ask-first bullet was added; (3) bindings.md command list gains migrate and renumber and a former-id schema row (spec section 3 verbs the skills' working summary should show); (4) repo CLAUDE.md (anthill-fw) untouched per config-file ownership. Note for the CLI worker: the template now ships schema: 3 with a placeholder key-prefix, so a scaffold that is not init'ed will fail key-prefix-invalid on first new rather than being stamped from schema 0; init must replace the placeholder line.

## Log
- 2026-11-17T13:47:51Z new by worker:38acdb2e
- 2026-11-17T13:47:51Z triage → framework by worker:38acdb2e
- 2026-11-17T13:47:51Z approve by worker:38acdb2e
- 2026-11-17T13:56:12Z claim by fw-keys (session 38acdb2e…)
- 2026-11-17T14:00:01Z deliver by fw-keys
- 2026-11-17T14:00:29Z close --done by worker:38acdb2e: Verified: grep clean on item-keys, workstreams schema 3 + key-prefix placeholder, bindings Key scheme self-contained, CHANGELOG consumer action; branch f96774b pushed, merge after CLI phase
