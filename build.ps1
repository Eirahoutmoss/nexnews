$ErrorActionPreference = 'Stop'
# Windows PowerShell 5.1: keep Turkish/Unicode console output readable.
try { [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false) } catch {}
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Dist = Join-Path $Root 'dist'
New-Item -ItemType Directory -Force -Path $Dist | Out-Null

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go bulunamadı. Go kurulumu sonrası PowerShell yeniden açılmalı.'
}

Push-Location $Root
try {
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'

    # İsteğe bağlı exe ikonu: assets\nexnews.ico varsa gömülür.
    $Ico  = Join-Path $Root 'cmd\nexnews\assets\nexnews.ico'
    $Syso = Join-Path $Root 'cmd\nexnews\rsrc_windows_amd64.syso'
    Remove-Item $Syso -Force -ErrorAction SilentlyContinue
    if (Test-Path $Ico) {
        Write-Host 'Generating icon resource ...'
        go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico $Ico -o $Syso
        if ($LASTEXITCODE -ne 0) { throw 'Icon resource generation failed.' }
    } else {
        Write-Host 'İkon yok (assets\nexnews.ico) — ikonsuz derleniyor.'
    }

    Write-Host 'Building NexNews.exe ...'
    # -H=windowsgui: açılışta konsol penceresi çıkmasın.
    go build -trimpath -ldflags='-s -w -H=windowsgui' -o (Join-Path $Dist 'NexNews.exe') './cmd/nexnews'
    if ($LASTEXITCODE -ne 0) { throw 'NexNews.exe build failed.' }

    Write-Host ''
    Write-Host 'BUILD OK'
    Write-Host (Join-Path $Dist 'NexNews.exe')
    Write-Host 'Note: Microsoft Edge WebView2 Runtime is required on the target PC.'
}
finally { Pop-Location }
