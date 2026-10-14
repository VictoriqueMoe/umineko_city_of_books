[CmdletBinding()]
param(
    [ValidateSet('read', 'users')]
    [string]$Scenario = 'read',
    [string]$BaseUrl = 'http://localhost:4323',
    [string]$InternalUrl = 'http://localhost:9464',
    [string]$User = '',
    [string]$Pass = '',
    [int]$Users = 200,
    [string]$Ramp = '2m',
    [string]$Hold = '5m',
    [switch]$NoDashboard
)

if (-not (Get-Command k6 -ErrorAction SilentlyContinue))
{
    Write-Host 'k6 is not installed. Install it with: winget install k6 --source winget' -ForegroundColor Red
    exit 1
}

try
{
    Invoke-WebRequest -Uri "$BaseUrl/livez" -TimeoutSec 5 -UseBasicParsing | Out-Null
}
catch
{
    Write-Host "No server responding at $BaseUrl/livez" -ForegroundColor Red
    Write-Host "Start it with 'go run .' (binds :4323) or 'docker compose up' (host :2312)."
    exit 1
}

$env:BASE_URL = $BaseUrl
$env:INTERNAL_URL = $InternalUrl
$env:LOADTEST_USER = $User
$env:LOADTEST_PASS = $Pass

$script = 'read-endpoints.js'
if ($Scenario -eq 'users')
{
    $usersFile = Join-Path $PSScriptRoot 'users.json'
    if (-not (Test-Path $usersFile))
    {
        Write-Host "No $usersFile. Seed the load test database first: go run ./scripts/loadtest/seed -users $Users" -ForegroundColor Red
        exit 1
    }

    $env:USERS = "$Users"
    $env:RAMP = $Ramp
    $env:HOLD = $Hold
    $script = 'user-behaviour.js'
}

if ($NoDashboard)
{
    $env:K6_WEB_DASHBOARD = 'false'
}
else
{
    $env:K6_WEB_DASHBOARD = 'true'
    $env:K6_WEB_DASHBOARD_OPEN = 'true'
}

Write-Host "Load testing $BaseUrl with $script..." -ForegroundColor Cyan
k6 run (Join-Path $PSScriptRoot $script)

exit $LASTEXITCODE
