[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$previousGoCache = $env:GOCACHE
$safeGoCache = Join-Path ([System.IO.Path]::GetTempPath()) "simple-erp-go-build"
New-Item -ItemType Directory -Force -Path $safeGoCache | Out-Null
$env:GOCACHE = $safeGoCache

function Invoke-Step {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][scriptblock]$Action
    )

    Write-Host "==> $Name"
    & $Action
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

try {
    Push-Location (Join-Path $root "backend")
    try {
        Invoke-Step "Go unit tests" { go test -mod=readonly -count=1 ./internal/... ./cmd/... }
        Invoke-Step "Go vet" { go vet -mod=readonly ./internal/... ./cmd/... }
        Invoke-Step "Integration package compile only" { go test -mod=readonly -tags=integration -run '^$' ./test/integration }
    }
    finally {
        Pop-Location
    }
}
finally {
    $env:GOCACHE = $previousGoCache
}

Write-Host "==> Changed Go formatting"
$changedGo = @(
    git -C $root diff --name-only -- "*.go"
    git -C $root ls-files --others --exclude-standard -- "*.go"
) | Sort-Object -Unique
if ($changedGo.Count -gt 0) {
    $unformatted = @(& gofmt -l ($changedGo | ForEach-Object { Join-Path $root $_ }))
    if ($unformatted.Count -gt 0) {
        throw "Changed Go files are not formatted: $($unformatted -join ', ')"
    }
}

Write-Host "==> Frontend static and unit checks"
$frontendBins = Join-Path $root "frontend/node_modules/.bin"
if ((Test-Path (Join-Path $frontendBins "eslint.cmd")) -and
    (Test-Path (Join-Path $frontendBins "tsc.cmd")) -and
    (Test-Path (Join-Path $frontendBins "vitest.cmd"))) {
    Push-Location (Join-Path $root "frontend")
    try {
        Invoke-Step "Frontend lint" { npm run lint }
        Invoke-Step "Frontend typecheck" { npm run typecheck }
        Invoke-Step "Frontend unit tests" { npm test -- --run }
    }
    finally {
        Pop-Location
    }
}
else {
    Write-Warning "SKIPPED frontend checks: local node_modules executables are unavailable. Run npm ci explicitly before this verifier."
}

Write-Host "==> BPMN XML well-formedness"
Get-ChildItem (Join-Path $root "docs/bpmn") -Filter "*.bpmn" | ForEach-Object {
    try {
        [xml](Get-Content -LiteralPath $_.FullName -Raw) | Out-Null
    }
    catch {
        throw "Invalid BPMN XML: $($_.FullName): $($_.Exception.Message)"
    }
}

Write-Host "==> Traceability structure"
$mapping = Join-Path $root "docs/PROCESS-MAPPING.md"
$taskIds = Get-ChildItem (Join-Path $root "docs/bpmn") -Filter "l2-*.bpmn" |
    ForEach-Object { Select-String -LiteralPath $_.FullName -Pattern '<bpmn:(?:userTask|serviceTask) id="([A-Z0-9-]+)"' -AllMatches } |
    ForEach-Object { $_.Matches } |
    ForEach-Object { $_.Groups[1].Value } |
    Sort-Object -Unique

if ($taskIds.Count -ne 39) {
    throw "Expected 39 stable BPMN task IDs, found $($taskIds.Count)"
}

$mappingLines = Get-Content -LiteralPath $mapping
foreach ($taskId in $taskIds) {
    $count = @($mappingLines | Where-Object { $_ -like "| $taskId |*" }).Count
    if ($count -ne 1) {
        throw "$taskId must have exactly one mapping row, found $count"
    }
}

$actionCount = @($mappingLines | Where-Object { $_ -like "| ACT-*" }).Count
if ($actionCount -ne 22) {
    throw "Expected 22 supporting action mappings, found $actionCount"
}

Write-Host "Traceability structure verified: 39 BPMN tasks and 22 supporting actions."
Write-Host "This script provides static and unit-level evidence only; it does not prove database, API, browser, or end-to-end behavior."
