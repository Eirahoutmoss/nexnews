# NexNews Android APK — Windows'ta yerel derleme (GitHub Actions gerekmez)
#
# Kullanım (PowerShell, depo klasöründe):
#   .\android-build.ps1                 # derle
#   .\android-build.ps1 -Install        # derle ve USB ile bağlı telefona kur (adb)
#
# İlk çalıştırmada gerekenler kendiliğinden kurulur (yaklaşık 2–3 GB indirme):
#   - Java 17 (winget ile Temurin)          - Android komut satırı araçları
#   - Android SDK 35, build-tools, NDK        - gomobile / gobind
# Sonraki derlemeler birkaç dakika sürer. Go 1.24+ kurulu olmalı.
param(
    [switch]$Install,
    [string]$Keystore = "",           # boşsa ..\android-imza\nexnews-release.jks ya da .\android-imza\... aranır
    [string]$SdkDir = "$env:LOCALAPPDATA\Android\Sdk"
)
$ErrorActionPreference = 'Stop'
try { [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false) } catch {}
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Android = Join-Path $Root 'android'
$Dist = Join-Path $Root 'dist'
New-Item -ItemType Directory -Force -Path $Dist | Out-Null
$GomobileVersion = 'v0.0.0-20260203041319-574ceaa2f723'
$NdkVersion = '27.2.12479018'

function Step($t) { Write-Host ''; Write-Host "==> $t" -ForegroundColor Cyan }

# ---- 1. Go -----------------------------------------------------------------
Step 'Go denetleniyor'
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go bulunamadı (https://go.dev/dl). Kurduktan sonra PowerShell''i yeniden açın.' }
go version

# ---- 2. Java 17 ------------------------------------------------------------
Step 'Java 17 denetleniyor'
$javaOk = $false
if ($env:JAVA_HOME -and (Test-Path (Join-Path $env:JAVA_HOME 'bin\java.exe'))) {
    $v = & (Join-Path $env:JAVA_HOME 'bin\java.exe') -version 2>&1 | Out-String
    if ($v -match 'version "(17|21)') { $javaOk = $true }
}
if (-not $javaOk) {
    $cand = Get-ChildItem 'C:\Program Files\Eclipse Adoptium' -Directory -ErrorAction SilentlyContinue | Where-Object { $_.Name -match '^jdk-(17|21)' } | Select-Object -First 1
    if (-not $cand) {
        Write-Host 'Java 17 kuruluyor (winget)…'
        winget install --id EclipseAdoptium.Temurin.17.JDK -e --accept-source-agreements --accept-package-agreements | Out-Host
        $cand = Get-ChildItem 'C:\Program Files\Eclipse Adoptium' -Directory -ErrorAction SilentlyContinue | Where-Object { $_.Name -match '^jdk-17' } | Select-Object -First 1
    }
    if (-not $cand) { throw 'Java 17 bulunamadı. https://adoptium.net adresinden Temurin 17 JDK kurun.' }
    $env:JAVA_HOME = $cand.FullName
}
$env:Path = (Join-Path $env:JAVA_HOME 'bin') + ';' + $env:Path
Write-Host "JAVA_HOME = $env:JAVA_HOME"

# ---- 3. Android SDK + NDK ---------------------------------------------------
Step 'Android SDK denetleniyor'
$sdkmanager = Join-Path $SdkDir 'cmdline-tools\latest\bin\sdkmanager.bat'
if (-not (Test-Path $sdkmanager)) {
    Write-Host "Android komut satırı araçları indiriliyor → $SdkDir"
    New-Item -ItemType Directory -Force -Path (Join-Path $SdkDir 'cmdline-tools') | Out-Null
    $zip = Join-Path $env:TEMP 'android-cmdline-tools.zip'
    try {
        Invoke-WebRequest 'https://dl.google.com/android/repository/commandlinetools-win-13114758_latest.zip' -OutFile $zip
    } catch {
        throw 'Android komut satırı araçları indirilemedi. https://developer.android.com/studio#command-line-tools-only adresinden "commandlinetools-win" zip dosyasını indirip içindeki cmdline-tools klasörünü ' + (Join-Path $SdkDir 'cmdline-tools\latest') + ' olarak çıkarın ve betiği yeniden çalıştırın.'
    }
    $tmp = Join-Path $env:TEMP 'android-cmdline-tools'
    if (Test-Path $tmp) { Remove-Item $tmp -Recurse -Force }
    Expand-Archive $zip -DestinationPath $tmp -Force
    Move-Item (Join-Path $tmp 'cmdline-tools') (Join-Path $SdkDir 'cmdline-tools\latest') -Force
}
$env:ANDROID_HOME = $SdkDir
$env:ANDROID_SDK_ROOT = $SdkDir
$ndkDir = Join-Path $SdkDir "ndk\$NdkVersion"
if (-not (Test-Path (Join-Path $SdkDir 'platforms\android-35')) -or -not (Test-Path $ndkDir) -or -not (Test-Path (Join-Path $SdkDir 'build-tools\35.0.0'))) {
    Write-Host 'Lisanslar kabul ediliyor ve SDK bileşenleri kuruluyor (bir kez, ~2 GB)…'
    $yes = ('y' + [Environment]::NewLine) * 20
    $yes | & $sdkmanager --sdk_root=$SdkDir --licenses | Out-Null
    & $sdkmanager --sdk_root=$SdkDir 'platform-tools' 'platforms;android-35' 'build-tools;35.0.0' "ndk;$NdkVersion" | Out-Host
}
$env:ANDROID_NDK_HOME = $ndkDir
Set-Content -Path (Join-Path $Android 'local.properties') -Value ("sdk.dir=" + ($SdkDir -replace '\\', '\\')) -Encoding ASCII

# ---- 4. Go çekirdeği → nexcore.aar (gomobile) -------------------------------
Step 'Go çekirdeği Android kütüphanesine çevriliyor (gomobile)'
Push-Location $Root
try {
    go install "golang.org/x/mobile/cmd/gomobile@$GomobileVersion"
    go install "golang.org/x/mobile/cmd/gobind@$GomobileVersion"
    go get "golang.org/x/mobile/bind@$GomobileVersion"
    $gopathBin = Join-Path (go env GOPATH) 'bin'
    $env:Path = "$gopathBin;" + $env:Path
    go test ./core ./mobile
    if ($LASTEXITCODE -ne 0) { Write-Warning 'Go testlerinden biri başarısız oldu; derlemeye devam ediliyor.' }
    New-Item -ItemType Directory -Force -Path (Join-Path $Android 'app\libs') | Out-Null
    gomobile bind -target=android/arm64,android/arm -androidapi 24 -javapkg=com.eirahoutmoss.nexnews.go -trimpath -ldflags="-s -w" -o (Join-Path $Android 'app\libs\nexcore.aar') ./mobile
    if ($LASTEXITCODE -ne 0) { throw 'gomobile bind başarısız.' }
} finally { Pop-Location }

# ---- 5. İmza anahtarı --------------------------------------------------------
Step 'İmza anahtarı'
if (-not $Keystore) {
    foreach ($c in @((Join-Path $Root 'android-imza\nexnews-release.jks'), (Join-Path (Split-Path $Root -Parent) 'android-imza\nexnews-release.jks'))) {
        if (Test-Path $c) { $Keystore = $c; break }
    }
}
$signed = $false
if ($Keystore -and (Test-Path $Keystore)) {
    $pw = $env:NEXNEWS_KEYSTORE_PASSWORD
    if (-not $pw) {
        $sec = Read-Host "Anahtar parolası ($Keystore)" -AsSecureString
        $pw = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec))
    }
    $env:NEXNEWS_KEYSTORE_FILE = $Keystore
    $env:NEXNEWS_KEYSTORE_PASSWORD = $pw
    if (-not $env:NEXNEWS_KEY_ALIAS) { $env:NEXNEWS_KEY_ALIAS = 'nexnews' }
    $signed = $true
    Write-Host "Kalıcı anahtarla imzalanacak: $Keystore"
} else {
    Write-Host 'Anahtar bulunamadı — APK deneme (debug) anahtarıyla imzalanacak.' -ForegroundColor Yellow
}

# ---- 6. APK -----------------------------------------------------------------
Step 'APK derleniyor (Gradle)'
$ver = (Get-Content (Join-Path $Android 'SURUM') -Raw).Trim()
$env:NEXNEWS_VERSION_NAME = $ver
$env:NEXNEWS_VERSION_CODE = [string]([int]((Get-Date).ToUniversalTime() - [datetime]'2026-01-01').TotalMinutes)
Push-Location $Android
try {
    .\gradlew.bat :app:assembleRelease --no-daemon
    if ($LASTEXITCODE -ne 0) { throw 'Gradle derlemesi başarısız (hata çıktısını yukarıda görebilirsiniz).' }
} finally { Pop-Location }
$name = if ($signed) { "NexNews-$ver.apk" } else { "NexNews-$ver-deneme.apk" }
$apk = Join-Path $Dist $name
Copy-Item (Join-Path $Android 'app\build\outputs\apk\release\app-release.apk') $apk -Force
Write-Host ''
Write-Host "APK HAZIR: $apk" -ForegroundColor Green

# ---- 7. Telefona kur (isteğe bağlı) -------------------------------------------
if ($Install) {
    $adb = Join-Path $SdkDir 'platform-tools\adb.exe'
    Step 'Telefona kuruluyor (USB hata ayıklama açık olmalı)'
    & $adb devices
    & $adb install -r $apk
    if ($LASTEXITCODE -ne 0) {
        Write-Host 'Kurulum başarısız. İmza farklıysa önce telefondaki eski NexNews''i kaldırın: adb uninstall com.eirahoutmoss.nexnews' -ForegroundColor Yellow
    }
} else {
    Write-Host 'Telefona aktarmak için: APK dosyasını telefona kopyalayıp açın ya da  .\android-build.ps1 -Install  komutunu kullanın.'
}
