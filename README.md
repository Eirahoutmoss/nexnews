# NexNews

Dünyanın nabzı, tek akışta. Dil ve kaynak bağımsız RSS/Atom haber toplayıcısı:
aynı olayı farklı kaynaklardan tek habere toplar, ajans metinlerini ayırt eder,
önem sırasına dizer ve "kim nasıl başlık attı?" karşılaştırması sunar.

## RC85 (masaüstü 1.0.0-rc85 · Android 0.2.0)

- Masaüstündeki RC83–RC84 çalışmaları (canlı TV kararlılığı, veri modu, olay
  takibi, sol menü) ortak `core/` çekirdeğine taşındı; Android da aynı özellikleri alır.
- Kanal listesi `core/assets/livetv.json` (325 kanal, 10 bölge). Kanal denetimi
  artık kullanıcıya sorularak yapılır ve **arka planda** çalışır (WebView2
  bağlamaları arayüz iş parçacığında çalıştığından eskiden pencere donuyordu).
- Canlı TV: kendi kanalını / M3U listeni ekleme, kanalı listeden kaldırma
  (Gizlenenler'den geri alma), Kendi kanallarım sekmesi.
- Sağ panel: kartlar yalnızca açıkken veri toplar; Akıllı Uyarılar varsayılan
  kapalı; Özet'te "Takip ettiklerim" kartı.
- Hakkında güncellendi (YouTube @bakis_acisi · Instagram @elem_ya).

### Android'i GitHub Actions olmadan derleme

```powershell
.\android-build.ps1            # dist\NexNews-<sürüm>.apk
.\android-build.ps1 -Install   # USB hata ayıklama açık telefona doğrudan kurar
```
İlk çalıştırmada Java 17, Android SDK/NDK ve gomobile kendiliğinden kurulur.

## Depo yapısı

| Klasör | İçerik |
|---|---|
| `core/` | Ortak Go çekirdeği: feed çekme, gruplama, sıralama, çeviri, depolama. Masaüstü ve Android aynı kodu kullanır. |
| `cmd/nexnews/` | Windows masaüstü uygulaması (Go + WebView2, tek `.exe`). Arayüz: `core/assets/index.html`. |
| `mobile/` | Android için gomobile köprüsü (`nexcore` paketi, JSON alışverişi). |
| `android/` | Android uygulaması (Kotlin + Jetpack Compose). |
| `translation-engine/` | İsteğe bağlı yerel çeviri motoru (Docker + LibreTranslate), masaüstü için. |

## Derleme

- **Masaüstü:** `.\build.ps1` (Windows, Go 1.24+) → `dist\NexNews.exe`. Hedef bilgisayarda Microsoft Edge WebView2 Runtime gerekir.
- **Android (yerel, önerilen):** `.\android-build.ps1` → `dist\NexNews-<sürüm>.apk`; `-Install` ile USB'deki telefona kurar.
  İlk çalıştırmada Java 17 (winget), Android SDK 35 + NDK ve gomobile kendiliğinden kurulur (~2–3 GB). İmza anahtarı
  `android-imza\nexnews-release.jks` (depo klasöründe ya da bir üst klasörde) bulunursa parolası sorulur ve APK kalıcı
  anahtarla imzalanır; bulunmazsa "deneme" APK'sı üretilir.
- **Android (elle):** `gomobile bind '-target=android/arm64,android/arm' -androidapi 24 '-javapkg=com.eirahoutmoss.nexnews.go' -o android/app/libs/nexcore.aar ./mobile`,
  sonra `cd android; .\gradlew.bat :app:assembleRelease`. (PowerShell'de virgüllü bağımsız değişkenler tırnak içinde yazılmalı.)
- **CI:** GitHub Actions (`.github/workflows/ci.yml`) aynı adımları çalıştırır; her `main` gönderiminden sonra APK ve `NexNews.exe`
  **Releases** sayfasında yayınlanır (Actions dakikası varsa).
- **Sürümler:** masaüstü `core/globals.go` (`appVersion`), Android `android/SURUM`; sürüm notları `android/surum-notlari/`.

### Android imza anahtarı (bir kez)

Depo ayarlarında (Settings → Secrets and variables → Actions) üç gizli değer tanımlanınca APK'lar kalıcı anahtarla imzalanır
ve güncellemeler kaldırıp kurmadan yüklenir:

- `NEXNEWS_KEYSTORE_B64` — `nexnews-release.jks` dosyasının base64 hali
- `NEXNEWS_KEYSTORE_PASSWORD` — anahtar deposu parolası
- `NEXNEWS_KEY_ALIAS` — `nexnews`

## Masaüstü

- Sürüm: **1.0.0-rc82**
- Ayarlar ve önbellek: `%APPDATA%\NexNews\` (`config.json`, `feed-cache.json`,
  `translations.json`, `favorites.json`, `history.json`, `follows.json`).

## RC84 değişiklikleri (masaüstü)

- Maçlar kartı: ESPN'den tarih aralığıyla (−2/+8 gün) çekilir; favoriler harf duyarsız eşleşir ve en üstte; ev sahibi önce; Türkçe durumlar; tanımsız `countdown` hatası giderildi; maç filtresi artık puan durumu sekmelerini etkilemiyor.

## RC83 değişiklikleri (masaüstü)

- Sağ panelde **Borsa Analiz** ve **Spor** ayrı sekmeler oldu.
- Borsa kartının açılmama hatası giderildi (başlıkta iki aç/kapa mekanizması birbirini iptal ediyordu; Akıllı Uyarılar kartında da aynısı vardı).
- Borsa Analiz: BIST endeksleri `XU100.IS`/`XU030.IS` üzerinden, günlük değişim önceki kapanıştan, "Piyasayı ilgilendiren gündem" başlıkları (indirilen haberlerden süzülür).
- Spor: Süper Lig (ESPN, olmazsa Wikipedia) ve Basketbol Süper Ligi (Wikipedia) puan durumları; 30 dk önbellek, kaynak bağlantısı, Wikipedia'nın "şu tarihe kadar güncel" notu.

## RC82 değişiklikleri (masaüstü)

- Radyo ve canlı TV, başka sayfalara geçildiğinde de çalmaya devam ediyor. TV
  sayfasından çıkınca yayın sağ altta küçük pencereye iner; radyo için alt
  köşede bir oynatıcı çubuğu kalır (duraklat, ses, sayfaya dön, kapat). Radyo
  ve TV aynı anda çalmaz; birini başlatınca diğeri durur. Widget modunda
  oynatıcı gizlenir ama ses sürer.

## RC81 değişiklikleri

### Güvenlik ve kararlılık
- Feed ve makale bağlantıları yalnızca `http/https` ise açılır (`file:`, UNC yolu,
  `ms-msdt:` gibi protokoller engellenir).
- Okuyucu, makaledeki `javascript:`/`data:` bağlantı ve görsellerini atar; göreli
  bağlantıları makale adresine göre çözer.
- Canlı TV aktarıcısı (`127.0.0.1:39765`) her çalıştırmada üretilen bir anahtar
  ister ve yerel ağ/özel IP adreslerine bağlanmayı reddeder.
- Resmî yayınlar ve radyo dizini artık arayüzü dondurmaz (arka planda yüklenir).
- Aynı feed birden çok kategorideyse bir kez indirilir; önbellekten açılışta
  tüm kategorilere geri yüklenir. Aynı olay üst bölümlerde iki kez görünmez.
- İndirme eşzamanlılığı sınırlı: toplam 16, aynı siteye en fazla 2 istek.
- Arayüzdeki tüm metin/öznitelik kaçışları tırnakları da kapsar.
- Yapılandırma ve önbellek dosyaları atomik yazılır. Spor takip listesi artık
  kaydediliyor; çeviri ayarı sürüm yükseltmesinde zorla açılmıyor.
- Testler Windows dışında da derlenip çalışıyor.

### Haber motoru
- Türkçe çekim ekleri gruplamayı bozmuyor ("deprem", "depremin", "depremde").
- Farklı dillerdeki aynı haber, çeviri üzerinden tek olayda birleşiyor.
- Ajans farkındalığı: AA, DHA, İHA, ANKA, Reuters, AP, AFP, dpa, EFE, ANSA metnini
  yayımlayan siteler ve aynı gövde metnini taşıyanlar tek kaynak sayılır.
- Olay panelinde "Kim nasıl başlık attı?": her kaynağın başlığı, yalnız o
  kaynağın kullandığı kelimeler işaretli.
- Çeviriler diske kaydedilir; ulaşılamayan çeviri sağlayıcısı 10 dk atlanır.
- Feed görselleri (media:thumbnail, media:content, enclosure, ilk `<img>`).
- "Farklı bir pencere": son iki haftada en az okunan bölge/konudan güçlü bir haber.
- Bildirimler için sessiz saatler.

### Arayüz
- Varsayılan pencere 1360×860; sağ panel 1100 px üstünde görünür ve
  Özet / Piyasa · Spor / Analiz / Olay sekmelerine ayrıldı. Yalnızca görünen
  sekme, pencere görünürken yenilenir.
- Haber kartında tek sinyal (son dakika ya da bağımsız kaynak sayısı); puan, güven
  ve evre ipucunda. Kırmızı yalnızca son dakikaya ayrıldı.
- En küçük yazı 11 px; etiketler Türkçe; menülerde çizgi ikonlar.
- Okunmamış sayıları sol menüde, okunan haberler soluk.
- Klavye: J/K gezin, O/Enter aç, R okuyucu, B tarayıcı, E olay, S favori, / ara,
  G ana sayfa, ? yardım, F5 yenile.
- Liste yoğunluğu: Rahat / Sıkı.
- Ayarlar bölümlere ayrıldı; kaynak listesinde her feed'in sağlık durumu.
- Açık temada okunabilirlik düzeltmeleri; sağ panel kartları tek tip.
- Sayfa sonunda kapanış kartı.

### Widget
- Zemin: Opak / Cam (akrilik) / Bulanık / Saydam. Yüzey opaklığı (%0–100) ve cam
  tonu ayarlanabilir; widget açıkken anında uygulanır (◐ düğmesi veya sağ tık).
- Windows 11'de köşeler sistem tarafından yuvarlanır (bulanıklık ve gölge korunur).
- Tek satır modu, fare üzerindeyken akış durur, sıra göstergesi.

### Ağ göstergesi
- Üst çubukta anlık bağlantı hızı ve NexNews'in son 5 dakikada çektiği veri.
  Hız, varsayılan olarak bilgisayarın etkin bağlantısının tamamıdır; tıklayınca
  yalnızca NexNews trafiğine geçer. Ayrıntılar (yükleme, oturum toplamı,
  bağdaştırıcı ve bağlantı hızı) ipucundadır.
