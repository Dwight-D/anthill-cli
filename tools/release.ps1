# Release the anthill CLI: preflight, build+test, bump version, date the
# changelog, commit, tag, push, install, verify.
#
# Usage: .\tools\release.ps1 -Version 0.0.6
#
# Preflight refuses: dirty tree, not on main, behind origin/main, tag taken.
# Warns (but proceeds) if the embedded template ref differs from the sibling
# anthill-fw checkout's HEAD - re-vendor first if that is unintended.
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^\d+\.\d+\.\d+$')]
    [string]$Version
)

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot

function Fail([string]$msg) {
    Write-Host "release: FAILED - $msg"
    exit 1
}

# --- Preflight ---
if (git -C $repo status --porcelain) { Fail "working tree not clean" }
$branch = git -C $repo branch --show-current
if ($branch -ne 'main') { Fail "on branch '$branch', not main" }
git -C $repo fetch origin --quiet
if ([int](git -C $repo rev-list --count 'main..origin/main') -gt 0) {
    Fail "main is behind origin/main; pull first"
}
if (git -C $repo tag -l "v$Version") { Fail "tag v$Version already exists" }

# Embedded-ref staleness warning (the payload should usually match fw HEAD).
$embed = Get-Content (Join-Path $repo 'internal\bootstrap\embed.go') -Raw
if ($embed -match 'TemplateRef = "([0-9a-f]{40})"') {
    $ref = $Matches[1]
    $fw = Join-Path (Split-Path -Parent $repo) 'anthill-fw'
    if (Test-Path (Join-Path $fw '.git')) {
        $fwHead = git -C $fw rev-parse HEAD
        if ($fwHead -ne $ref) {
            Write-Warning ("embedded template ref {0} != anthill-fw HEAD {1}; re-vendor before releasing if unintended" -f $ref.Substring(0, 12), $fwHead.Substring(0, 12))
        }
    }
}

# --- Build + test ---
Push-Location $repo
try {
    go build ./...
    if ($LASTEXITCODE -ne 0) { Fail "go build failed" }
    go test ./...
    if ($LASTEXITCODE -ne 0) { Fail "go test failed" }
}
finally { Pop-Location }

# --- Bump version constant ---
$utf8 = New-Object System.Text.UTF8Encoding($false)
$verFile = Join-Path $repo 'internal\version\version.go'
$src = Get-Content $verFile -Raw
$new = $src -replace 'const Version = "[^"]+"', ('const Version = "{0}"' -f $Version)
if ($new -eq $src) { Fail "version constant not found in version.go" }
[System.IO.File]::WriteAllText($verFile, $new, $utf8)

# --- Date the changelog (Unreleased -> version + date, em-dash house style) ---
$clFile = Join-Path $repo 'CHANGELOG.md'
$cl = Get-Content $clFile -Raw
if ($cl -match '## Unreleased') {
    $emdash = [char]0x2014
    $today = Get-Date -Format 'yyyy-MM-dd'
    $cl = $cl -replace '## Unreleased', ("## {0} {1} {2}" -f $Version, $emdash, $today)
    [System.IO.File]::WriteAllText($clFile, $cl, $utf8)
}
else {
    Write-Warning "no '## Unreleased' section in CHANGELOG.md - releasing without a changelog cut"
}

# --- Commit, tag, push ---
git -C $repo add internal/version/version.go CHANGELOG.md
git -C $repo commit -q -m ("release: bump version to {0}" -f $Version)
if ($LASTEXITCODE -ne 0) { Fail "commit failed" }
git -C $repo tag "v$Version"
git -C $repo push origin main "v$Version"
if ($LASTEXITCODE -ne 0) { Fail "push failed" }

# --- Install + verify ---
Push-Location $repo
try {
    go install ./cmd/anthill
    if ($LASTEXITCODE -ne 0) { Fail "go install failed" }
}
finally { Pop-Location }

$reported = (anthill --version | Select-Object -First 1)
if ($reported -notmatch [regex]::Escape($Version)) {
    Fail ("installed binary reports '{0}', expected {1}" -f $reported, $Version)
}
Write-Host ("release: OK - v{0} tagged and pushed; installed binary reports: {1}" -f $Version, $reported)
