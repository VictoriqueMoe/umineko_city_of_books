[CmdletBinding()]
param(
    [string]$BaseUrl = 'http://localhost:4323',
    [string]$User = '',
    [string]$Pass = '',
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
$env:LOADTEST_USER = $User
$env:LOADTEST_PASS = $Pass

if ($NoDashboard)
{
    $env:K6_WEB_DASHBOARD = 'false'
}
else
{
    $env:K6_WEB_DASHBOARD = 'true'
    $env:K6_WEB_DASHBOARD_OPEN = 'true'
}

Write-Host "Load testing $BaseUrl..." -ForegroundColor Cyan
k6 run (Join-Path $PSScriptRoot 'read-endpoints.js')

exit $LASTEXITCODE
