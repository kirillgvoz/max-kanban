[CmdletBinding()]
param(
    [switch]$NoCache,
    [switch]$Pull,
    [int]$HealthTimeoutSeconds = 180
)

$ErrorActionPreference = 'Stop'
$kanbanRoot = Split-Path -Parent $PSScriptRoot
Set-Location $kanbanRoot

if (-not (Test-Path -LiteralPath (Join-Path $kanbanRoot '.env'))) {
    throw 'Missing kanban/.env. Copy kanban/.env.example to kanban/.env and set deployment values first.'
}

$expectedMarkers = @(
    @{ Path = 'frontend/src/components/MobileBoard.tsx'; Pattern = 'mobile-settings' },
    @{ Path = 'frontend/src/styles/index.scss'; Pattern = 'mobile-settings' }
)
foreach ($marker in $expectedMarkers) {
    if (-not (Select-String -LiteralPath (Join-Path $kanbanRoot $marker.Path) -Pattern $marker.Pattern -Quiet)) {
        throw "Expected marker '$($marker.Pattern)' was not found in $($marker.Path). Pull the latest frontend changes first."
    }
}

function Invoke-Compose {
    param([string[]]$ComposeArgs)
    & docker compose @ComposeArgs
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose $($ComposeArgs -join ' ') failed with exit code $LASTEXITCODE."
    }
}

$buildArgs = @('build', 'backend')
if ($NoCache) {
    $buildArgs += '--no-cache'
}
if ($Pull) {
    $buildArgs += '--pull'
}
Invoke-Compose $buildArgs
Invoke-Compose @('up', '-d', 'backend')

$deadline = (Get-Date).AddSeconds($HealthTimeoutSeconds)
while ((Get-Date) -lt $deadline) {
    $healthRaw = & curl.exe --max-time 20 -s http://localhost:9300/max-kanban/api/health
    if ($LASTEXITCODE -eq 0) {
        $health = $healthRaw | ConvertFrom-Json
        if ($health.status -eq 'ok' -and $health.database -eq 'ok') {
            break
        }
    }
    Start-Sleep -Seconds 5
}
if ($health.status -ne 'ok' -or $health.database -ne 'ok') {
    & docker compose logs backend --tail 100
    throw 'Backend did not report healthy status and database availability in time.'
}

$index = & curl.exe --max-time 20 -s http://localhost:9300/max-kanban/
if ($LASTEXITCODE -ne 0) {
    throw 'Could not download the served frontend index page.'
}
$asset = ([regex]::Match($index, '/max-kanban/assets/[^"\s]+\.css')).Value
if (-not $asset) {
    throw 'Could not find the served frontend CSS bundle.'
}
$css = & curl.exe --max-time 20 -s ("http://localhost:9300" + $asset)
if ($LASTEXITCODE -ne 0) {
    throw 'Could not download the served frontend CSS bundle.'
}
if ($css -notmatch '\.mobile-settings') {
    throw 'The running container is serving a frontend bundle without the mobile settings control.'
}

Write-Host 'Rebuild complete: backend is healthy and the served bundle contains the mobile settings control.'
