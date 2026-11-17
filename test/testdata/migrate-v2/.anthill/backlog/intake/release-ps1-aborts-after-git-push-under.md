---
title: release.ps1 aborts after 'git push' under PowerShell 5.1 (git stderr treated as a terminating error)
value: 'The 0.1.0 release stopped right after pushing: with $ErrorActionPreference=Stop, git''s normal stderr progress (''To github.com:…'') became a NativeCommandError and the script never ran go install or the version check, leaving the release half-done until finished by hand. Capture git output (e.g. 2>&1 | Out-String or --quiet, or check $LASTEXITCODE only) so the push step cannot false-fail.'
status: new
dispatch: proposed
priority: normal
reporter: worker
source: supervisor session 2026-11-17, 0.1.0 release
hint: dev
created-at: "2026-11-17T12:52:51Z"
---

## Description

## Acceptance

## Triage

## Evidence

## Log
- 2026-11-17T12:52:51Z new by worker:efcb0f1d
