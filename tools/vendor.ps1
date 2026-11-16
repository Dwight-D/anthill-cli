# Re-vendor the embedded framework template from the sibling anthill-fw repo
# and pin TemplateRef to that commit.
#
# Usage: .\tools\vendor.ps1            # vendor anthill-fw HEAD
#        .\tools\vendor.ps1 -Ref <sha> # vendor a specific commit
#
# Replaces internal/bootstrap/template/ with every tracked file at the ref,
# minus the framework home's own docs (README, INSTALLATION, CLAUDE, CLI_SPEC,
# BOOTSTRAP, CHANGELOG), which are fetched from upstream, not scaffolded.
# Refuses when the ref is not on origin (consumers resolve it upstream).
# Leaves the result uncommitted: review the diff, add a CHANGELOG entry,
# commit, then release with tools/release.ps1.
param([string]$Ref = 'HEAD')

$ErrorActionPreference = 'Stop'
$repo = Split-Path -Parent $PSScriptRoot
$fw = Join-Path (Split-Path -Parent $repo) 'anthill-fw'
$templateDir = Join-Path $repo 'internal\bootstrap\template'
$embedFile = Join-Path $repo 'internal\bootstrap\embed.go'
$excluded = @('README.md', 'INSTALLATION.md', 'CLAUDE.md', 'CLI_SPEC.md', 'BOOTSTRAP.md', 'CHANGELOG.md')

function Fail([string]$msg) {
    Write-Host "vendor: FAILED - $msg"
    exit 1
}

if (-not (Test-Path (Join-Path $fw '.git'))) { Fail "anthill-fw checkout not found at $fw" }
$sha = git -C $fw rev-parse --verify "$Ref^{commit}"
if ($LASTEXITCODE -ne 0) { Fail "ref '$Ref' not found in anthill-fw" }
git -C $fw fetch origin --quiet
if (-not (git -C $fw branch -r --contains $sha)) { Fail "commit $sha is not pushed to anthill-fw origin; push first" }

# Extract the tree at the ref into a temp dir, then swap it in.
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("anthill-vendor-" + [guid]::NewGuid())
New-Item -ItemType Directory $tmp | Out-Null
try {
    $tar = Join-Path $tmp 'payload.tar'
    git -C $fw archive --format=tar -o $tar $sha
    if ($LASTEXITCODE -ne 0) { Fail "git archive failed" }
    $out = Join-Path $tmp 'out'
    New-Item -ItemType Directory $out | Out-Null
    tar -xf $tar -C $out
    if ($LASTEXITCODE -ne 0) { Fail "tar extract failed" }
    foreach ($f in $excluded) { Remove-Item -Force -ErrorAction SilentlyContinue (Join-Path $out $f) }

    if (Test-Path $templateDir) { Remove-Item -Recurse -Force $templateDir }
    Move-Item $out $templateDir
}
finally { Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $tmp }

# Pin the ref.
$utf8 = New-Object System.Text.UTF8Encoding($false)
$src = [System.IO.File]::ReadAllText($embedFile, $utf8)
$new = $src -replace 'TemplateRef = "[0-9a-f]{40}"', ('TemplateRef = "{0}"' -f $sha)
if ($new -notmatch [regex]::Escape($sha)) { Fail "TemplateRef constant not found in embed.go" }
[System.IO.File]::WriteAllText($embedFile, $new, $utf8)

git -C $repo status --short -- internal/bootstrap
Write-Host ("vendor: OK - template vendored at {0}. Next: add a CHANGELOG entry, commit, run tools/release.ps1." -f $sha.Substring(0, 7))
