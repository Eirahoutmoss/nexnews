$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
Write-Host "NexNews yerel ceviri motoru baslatiliyor..." -ForegroundColor Cyan

docker compose up -d

Write-Host "Motor hazir olana kadar bekleniyor..." -ForegroundColor Yellow
$ok = $false
for ($i = 0; $i -lt 60; $i++) {
    try {
        $r = Invoke-WebRequest -Uri 'http://127.0.0.1:5000/health' -TimeoutSec 2 -UseBasicParsing
        if ($r.StatusCode -eq 200) { $ok = $true; break }
    } catch {}
    Start-Sleep -Seconds 2
}
if (-not $ok) {
    Write-Host "Ceviri motoru henuz hazir degil." -ForegroundColor Red
    docker compose ps
    exit 1
}

Write-Host "NexNews yerel ceviri motoru HAZIR: http://127.0.0.1:5000" -ForegroundColor Green
