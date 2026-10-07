# NexNews Yerel Çeviri Motoru

NexNews RC58, başlık çevirisinde önce `127.0.0.1:5000` üzerindeki self-hosted LibreTranslate/Argos motorunu kullanır. Böylece public çeviri servislerinin rate-limit/CORS/erişim sorunları ana yol olmaktan çıkar.

## Gereksinim

- Windows 10/11
- Docker Desktop

## Başlatma

PowerShell:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\start.ps1
```

Motor hazır olduğunda:

```text
NexNews yerel ceviri motoru HAZIR: http://127.0.0.1:5000
```

Test:

```powershell
.\test.ps1
```

Beklenen JSON içinde `translatedText` alanıdır.

## NexNews

Ardından `dist\NexNews.exe` dosyasını çalıştırın. Başlık çevirisi etkinse NexNews önce yerel motora bağlanır. Yerel motor kapalıysa eski public provider fallback'i korunur.

## Durdurma

```powershell
.\stop.ps1
```

Model dosyaları Docker volume içinde tutulmadığından container yeniden oluşturulduğunda model indirme süreci tekrar yaşanabilir. İlk başlangıç bu nedenle sonraki başlangıçlardan daha uzun sürebilir.
