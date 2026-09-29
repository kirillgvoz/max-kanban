#Requires -Version 5.1
<#
.SYNOPSIS
  Пересборка и запуск production-контейнера kanban-backend (фронтенд вшит в бинарник через go:embed).
.DESCRIPTION
  1. Проверяет docker, kanban/.env (MAX_BOT_TOKEN задан, WEBHOOK_SECRET не дефолтный из примера) и исходники фронта.
  2. Проверяет, что в исходниках есть ожидаемые маркеры актуального UI (mobile-settings).
  3. Собирает образ kanban-backend через docker compose (пересборка обязательна: SPA вшита в бинарник).
  4. Пересоздаёт контейнер backend (postgres поднимается по depends_on).
  5. Ждёт /max-kanban/api/health со статусом ok + database ok.
  6. Проверяет, что отдаваемый CSS-бандл содержит .mobile-settings (то есть запущен свежий фронт, а не старый образ).
  Секреты из .env никогда не печатаются в лог.
.EXAMPLE
  .\scripts\rebuild.ps1
.EXAMPLE
  .\scripts\rebuild.ps1 -NoCache
#>
[CmdletBinding()]
param(
    [switch]$NoCache,
    [switch]$Pull,
    [int]$HealthTimeoutSeconds = 180
)

$ErrorActionPreference = 'Stop'
$kanbanRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $kanbanRoot

function Step([string]$msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Fail([string]$msg) { Write-Host "ОШИБКА: $msg" -ForegroundColor Red; exit 1 }

function Read-EnvFile([string]$path) {
    $cfg = @{}
    foreach ($line in (Get-Content -LiteralPath $path)) {
        $t = $line.Trim()
        if ($t -eq '' -or $t.StartsWith('#')) { continue }
        if ($t.StartsWith('export ')) { $t = $t.Substring(7) }
        $i = $t.IndexOf('=')
        if ($i -le 0) { continue }
        $k = $t.Substring(0, $i).Trim()
        $v = $t.Substring($i + 1).Trim().Trim('"', "'")
        if ($k -ne '') { $cfg[$k] = $v }
    }
    return $cfg
}

function Invoke-Compose {
    param([string[]]$ComposeArgs)
    & docker compose @ComposeArgs
    if ($LASTEXITCODE -ne 0) { Fail "docker compose $($ComposeArgs -join ' ') упал (exit $LASTEXITCODE)" }
}

# --- 0. Проверки окружения ---
Step 'Проверяю окружение...'
& docker info 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) { Fail 'docker недоступен (daemon не запущен?)' }
if (-not (Test-Path -LiteralPath (Join-Path $kanbanRoot '.env'))) {
    Fail 'kanban/.env не найден. Скопируй kanban/.env.example в kanban/.env и задай MAX_BOT_TOKEN + WEBHOOK_SECRET'
}
if (-not (Test-Path -LiteralPath (Join-Path $kanbanRoot 'frontend/package.json'))) {
    Fail 'frontend/ без package.json: исходники фронта потеряны, пересобирать нечего'
}

$cfg = Read-EnvFile (Join-Path $kanbanRoot '.env')
if ([string]::IsNullOrWhiteSpace($cfg['MAX_BOT_TOKEN'])) {
    Fail 'в kanban/.env пустой MAX_BOT_TOKEN - без него бот не сможет слать уведомления и отвечать в чатах'
}
if ([string]::IsNullOrWhiteSpace($cfg['WEBHOOK_SECRET'])) {
    Fail 'в kanban/.env пустой WEBHOOK_SECRET - MAX не сможет доставлять события вебхука'
}
if ($cfg['WEBHOOK_SECRET'] -eq 'taskflow_secret_change_me') {
    Write-Host 'ПРЕДУПРЕЖДЕНИЕ: WEBHOOK_SECRET равен значению из .env.example - для боевого контура задай свой' -ForegroundColor Yellow
}
Write-Host 'MAX_BOT_TOKEN задан, WEBHOOK_SECRET задан, образ=kanban-backend, порт=9300 (значения секретов не печатаю)'

# --- 1. Проверка исходников фронта ---
Step 'Проверяю маркеры актуального UI в исходниках...'
$expectedMarkers = @(
    @{ Path = 'frontend/src/components/MobileBoard.tsx'; Pattern = 'mobile-settings' },
    @{ Path = 'frontend/src/styles/index.scss'; Pattern = 'mobile-settings' }
)
foreach ($marker in $expectedMarkers) {
    if (-not (Select-String -LiteralPath (Join-Path $kanbanRoot $marker.Path) -Pattern $marker.Pattern -Quiet)) {
        Fail "маркер '$($marker.Pattern)' не найден в $($marker.Path) - подтяни свежие исходники фронта"
    }
}

# --- 2. Сборка образа ---
Step 'Собираю образ kanban-backend...'
$buildArgs = @('build', 'backend')
if ($NoCache) { $buildArgs += '--no-cache' }
if ($Pull) { $buildArgs += '--pull' }
Invoke-Compose $buildArgs

# --- 3. Пересоздание контейнера ---
Step 'Пересоздаю контейнер backend...'
Invoke-Compose @('up', '-d', 'backend')

# --- 4. Ожидание готовности ---
Step 'Жду /max-kanban/api/health...'
$deadline = (Get-Date).AddSeconds($HealthTimeoutSeconds)
$health = $null
while ((Get-Date) -lt $deadline) {
    $healthRaw = & curl.exe --max-time 20 -s http://localhost:9300/max-kanban/api/health
    if ($LASTEXITCODE -eq 0) {
        try { $health = $healthRaw | ConvertFrom-Json } catch { $health = $null }
        if ($health.status -eq 'ok' -and $health.database -eq 'ok') { break }
    }
    Start-Sleep -Seconds 5
}
if ($null -eq $health -or $health.status -ne 'ok' -or $health.database -ne 'ok') {
    & docker compose logs backend --tail 100
    Fail 'backend не отдал healthy-статус вовремя. Логи выше.'
}

# --- 5. Проверка, что запущен свежий фронт ---
Step 'Проверяю логи старта...'
$logs = (& docker compose logs backend --tail 50 2>&1) | Out-String
if ($logs -match 'Server listening') {
    Write-Host 'лог старта: Server listening - OK'
} else {
    Write-Host 'ПРЕДУПРЕЖДЕНИЕ: в логах нет "Server listening" - проверь, тот ли бинарник стартанул' -ForegroundColor Yellow
}

Step 'Проверяю отдаваемый CSS-бандл...'
$index = & curl.exe --max-time 20 -s http://localhost:9300/max-kanban/
if ($LASTEXITCODE -ne 0) { Fail 'не смог скачать index-страницу мини-приложения' }
$asset = ([regex]::Match($index, '/max-kanban/assets/[^"\s]+\.css')).Value
if (-not $asset) { Fail 'в index-странице нет ссылки на CSS-бандл' }
$css = & curl.exe --max-time 20 -s ("http://localhost:9300" + $asset)
if ($LASTEXITCODE -ne 0) { Fail 'не смог скачать CSS-бандл' }
if ($css -notmatch '\.mobile-settings') {
    Fail 'отдаваемый бандл без .mobile-settings - запущен старый образ, пересобери с -NoCache'
}
Write-Host 'CSS-бандл содержит .mobile-settings - запущен свежий фронт. OK'

Write-Host ''
Write-Host 'Готово: мини-приложение http://localhost:9300/max-kanban/ (прод: https://kirillgvoz.ru/max-kanban/)' -ForegroundColor Green
Write-Host 'Логи: docker compose logs -f backend'
Write-Host 'Статус: docker compose ps'
Write-Host 'Остановить: docker compose down'
