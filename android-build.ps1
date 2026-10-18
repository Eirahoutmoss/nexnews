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
# Windows PowerShell 5.1, 'Stop' kipinde stderr'e yazan her programı (java -version,
# go get, gradle…) hata sayıp durdurur. Bu yüzden 'Continue' kullanılır ve her
# programın sonucu çıkış koduyla (Check) denetlenir.
$ErrorActionPreference = 'Continue'
function Check($what) { if ($LASTEXITCODE -ne 0) { throw "$what başarısız (çıkış kodu $LASTEXITCODE)." } }
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
$gv = (go version) | Out-String
Write-Host $gv.Trim()
if ($gv -notmatch 'windows/amd64') {
    # gomobile, NDK derleyicilerini kendi mimarisine göre arar; 32 bit Go
    # 'prebuilt\windows' klasörüne bakar ama güncel NDK'larda yalnızca
    # 'prebuilt\windows-x86_64' vardır. Bu yüzden bu derleme için taşınabilir
    # 64 bit Go kullanılır (kurulu Go'ya dokunulmaz).
    $GoPortable = Join-Path $env:LOCALAPPDATA 'NexNewsBuild\go'
    $GoExe = Join-Path $GoPortable 'bin\go.exe'
    if (-not (Test-Path $GoExe)) {
        Write-Host 'Kurulu Go 32 bit; bu derleme için 64 bit Go indiriliyor (~70 MB, bir kez)…'
        $gz = Join-Path $env:TEMP 'go-windows-amd64.zip'
        Invoke-WebRequest 'https://go.dev/dl/go1.24.7.windows-amd64.zip' -OutFile $gz -UseBasicParsing -ErrorAction Stop
        $parent = Split-Path $GoPortable -Parent
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
        if (Test-Path $GoPortable) { Remove-Item $GoPortable -Recurse -Force }
        Expand-Archive $gz -DestinationPath $parent -Force -ErrorAction Stop
    }
    $env:GOROOT = $GoPortable
    $env:Path = (Join-Path $GoPortable 'bin') + ';' + $env:Path
    $env:GOTOOLCHAIN = 'local'
    $gv = (go version) | Out-String
    Write-Host ('Bu derleme için: ' + $gv.Trim())
    if ($gv -notmatch 'windows/amd64') { throw '64 bit Go hazırlanamadı.' }
}

# ---- 2. Java 17 ------------------------------------------------------------
# Proje jvmToolchain(17) istiyor; Gradle başka sürümle (21 dahil) çalışmaz.
# Sistemdeki Java'ya dokunulmaz, yalnızca bu derleme için JDK 17 seçilir.
Step 'Java 17 aranıyor'
function JavaMajor($jdkDir) {
    $exe = Join-Path $jdkDir 'bin\java.exe'
    if (-not (Test-Path $exe)) { return 0 }
    $out = cmd /c "`"$exe`" -version 2>&1" | Out-String
    if ($out -match 'version "(\d+)') { return [int]$Matches[1] }
    return 0
}
function FindJdk17 {
    $roots = @("$env:ProgramFiles\Eclipse Adoptium", "$env:ProgramFiles\Microsoft", "$env:ProgramFiles\Java", "$env:ProgramFiles\Zulu", "$env:ProgramFiles\Amazon Corretto", "$env:LOCALAPPDATA\Programs\Eclipse Adoptium")
    foreach ($r in $roots) {
        if (-not (Test-Path $r)) { continue }
        foreach ($d in (Get-ChildItem $r -Directory -ErrorAction SilentlyContinue)) {
            if ((JavaMajor $d.FullName) -eq 17) { return $d.FullName }
        }
    }
    if ($env:JAVA_HOME -and (JavaMajor $env:JAVA_HOME) -eq 17) { return $env:JAVA_HOME }
    return $null
}
$jdk = FindJdk17
if (-not $jdk) {
    Write-Host 'JDK 17 bulunamadı; Temurin 17 kuruluyor (winget). Mevcut Java sürümünüz etkilenmez…'
    winget install --id EclipseAdoptium.Temurin.17.JDK -e --silent --accept-source-agreements --accept-package-agreements | Out-Host
    $jdk = FindJdk17
}
if (-not $jdk) { throw 'JDK 17 kurulamadı. https://adoptium.net/temurin/releases/?version=17 adresinden "JDK 17 - Windows x64 .msi" kurup betiği yeniden çalıştırın.' }
$env:JAVA_HOME = $jdk
$env:Path = (Join-Path $jdk 'bin') + ';' + $env:Path
Write-Host "Bu derleme için JAVA_HOME = $jdk"

# ---- 3. Android SDK + NDK ---------------------------------------------------
Step 'Android SDK denetleniyor'
$sdkmanager = Join-Path $SdkDir 'cmdline-tools\latest\bin\sdkmanager.bat'
if (-not (Test-Path $sdkmanager)) {
    Write-Host "Android komut satırı araçları indiriliyor → $SdkDir"
    New-Item -ItemType Directory -Force -Path (Join-Path $SdkDir 'cmdline-tools') | Out-Null
    $zip = Join-Path $env:TEMP 'android-cmdline-tools.zip'
    try {
        Invoke-WebRequest 'https://dl.google.com/android/repository/commandlinetools-win-13114758_latest.zip' -OutFile $zip -UseBasicParsing -ErrorAction Stop
    } catch {
        throw 'Android komut satırı araçları indirilemedi. https://developer.android.com/studio#command-line-tools-only adresinden "commandlinetools-win" zip dosyasını indirip içindeki cmdline-tools klasörünü ' + (Join-Path $SdkDir 'cmdline-tools\latest') + ' olarak çıkarın ve betiği yeniden çalıştırın.'
    }
    $tmp = Join-Path $env:TEMP 'android-cmdline-tools'
    if (Test-Path $tmp) { Remove-Item $tmp -Recurse -Force }
    Expand-Archive $zip -DestinationPath $tmp -Force -ErrorAction Stop
    Move-Item (Join-Path $tmp 'cmdline-tools') (Join-Path $SdkDir 'cmdline-tools\latest') -Force -ErrorAction Stop
}
$env:ANDROID_HOME = $SdkDir
$env:ANDROID_SDK_ROOT = $SdkDir
$ndkDir = Join-Path $SdkDir "ndk\$NdkVersion"
if (-not (Test-Path (Join-Path $SdkDir 'platforms\android-35')) -or -not (Test-Path $ndkDir) -or -not (Test-Path (Join-Path $SdkDir 'build-tools\35.0.0'))) {
    Write-Host 'Lisanslar kabul ediliyor ve SDK bileşenleri kuruluyor (bir kez, ~2 GB)…'
    $yes = ('y' + [Environment]::NewLine) * 20
    $yes | & $sdkmanager --sdk_root=$SdkDir --licenses | Out-Null
    & $sdkmanager --sdk_root=$SdkDir 'platform-tools' 'platforms;android-35' 'build-tools;35.0.0' "ndk;$NdkVersion" | Out-Host
    Check 'Android SDK kurulumu'
}
$env:ANDROID_NDK_HOME = $ndkDir
Set-Content -Path (Join-Path $Android 'local.properties') -Value ("sdk.dir=" + ($SdkDir -replace '\\', '\\')) -Encoding ASCII

# ---- 4. Go çekirdeği → nexcore.aar (gomobile) -------------------------------
Step 'Go çekirdeği Android kütüphanesine çevriliyor (gomobile)'
Push-Location $Root
try {
    go install "golang.org/x/mobile/cmd/gomobile@$GomobileVersion"; Check 'gomobile kurulumu'
    go install "golang.org/x/mobile/cmd/gobind@$GomobileVersion"; Check 'gobind kurulumu'
    go get "golang.org/x/mobile/bind@$GomobileVersion"; Check 'x/mobile bağımlılığı'
    $gopathBin = Join-Path (go env GOPATH) 'bin'
    $env:Path = "$gopathBin;" + $env:Path
    go test ./core ./mobile
    if ($LASTEXITCODE -ne 0) { Write-Warning 'Go testlerinden biri başarısız oldu; derlemeye devam ediliyor.' }
    New-Item -ItemType Directory -Force -Path (Join-Path $Android 'app\libs') | Out-Null
    gomobile bind '-target=android/arm64,android/arm' -androidapi 24 '-javapkg=com.eirahoutmoss.nexnews.go' -trimpath '-ldflags=-s -w' -o (Join-Path $Android 'app\libs\nexcore.aar') ./mobile
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
