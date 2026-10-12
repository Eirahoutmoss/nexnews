# NexNews Intelligence Research Workspace

**Durum:** İlk çalışan araştırma dikey dilimi. Özellik dalında geliştirilir, `main` dalına birleştirilmiş değildir.

## Bu dilim ne sağlar?

- Kendi araştırma sorusu ve bağlamıyla kalıcı araştırma dosyaları oluşturur.
- Her dosyada hipotezler, kaynaklı kanıt kayıtları ve açık sorular tutulur.
- Kanıtın gerçek URL'si, yayıncısı, yayın tarihi, erişilen içerik düzeyi, kaynak türü ve biliniyorsa ortak ajans/kopya kökeni kaydedilir.
- Bir kanıt hipotezi destekleyebilir, ona karşı çıkabilir veya yalnızca bağlam sağlayabilir. Her bağlantı gerekçeli olmalıdır.
- Bir hipotez "destek buldu" durumuna getirilecekse kayda bağlı destekleyici kanıt gerekir. "Zayıfladı" veya "reddedildi" için karşı kanıt; "sonuçsuz" için en az bir değerlendirilmiş kayıt gerekir.
- Kanıt kapsama raporu, hipotez başına destek/karşı kanıt/bağlam sayılarını ve veri boşluklarını gösterir. Farklı yayıncı alan adlarını bağımsız doğrulama saymaz.
- Çözülmüş araştırma sorusu, çözüm notu olmadan kapatılamaz. Soru tekrar açılabilir.
- NexNews Olay Dosyası üzerinden başlatılan araştırma o olay kimliğine bağlanabilir; olayın haber kayıtları kaynak metni, URL ve kapsam sınırları korunarak araştırma dosyasına aktarılır.
- Tekrarlanan kaynak içeriği ikinci kez eklenmez. Kaynağın başlığı/özeti veya kaynak kökeni değişirse yeni bir sürüm eklenir; önceki kayıt silinmez.
- Hedef seçimi ve örüntü analizi, araştırma dosyasında kaynakla desteklenen tesis evreni, geçerlilik dönemli sahiplik kayıtları ve olay/iddia durumları tutar.
- Karşılaştırma, yabancı katılımı bulunan/bulunmayan tesislerde kayıtlı hedefleme gözlemi bulunan tesis oranını betimsel olarak hesaplar. Yalnızca hedefleme iddialarını bu sayıya eklemez.
- Evren eksik beyan edilmişse veya sahiplik sınıflandırması tüm araştırma dönemini kanıtla kapsamıyorsa ana grup farkını sonuç olarak sunmaz; eksikleri ve payda uyarılarını açıklar.

## Windows'ta çalıştırma

Önce NexNews'i çalıştırıp haber önbelleğini güncelleyin. Depo kökünde, Intelligence veri tabanını senkronize edin:

```powershell
go run ./cmd/nexnews-intelligence -db "$env:APPDATA\NexNews\nexnews-intelligence.json"
```

Aynı dosyayı kullanarak arayüzü başlatın:

```powershell
go run ./cmd/nexnews-intelligence-ui -db "$env:APPDATA\NexNews\nexnews-intelligence.json" -addr 127.0.0.1:8787
```

Ana ekran: `http://127.0.0.1:8787`  
Araştırma Masası: `http://127.0.0.1:8787/research-workspace`

Bir olay dosyası açıldığında **Bu olay için araştırma başlat** eylemi kullanılabilir. Bu, araştırma formunu olay başlığı ve araştırma sorusu için başlangıç bilgileriyle doldurur ve olay bağlantısını korur.

NexNews önbelleğine yeni haberler geldiyse önce CLI senkronizasyonunu tekrar çalıştırın; ardından araştırma dosyasında **NexNews kaynaklarını yenile** seçeneğini kullanın. Senkronizasyon yalnızca mevcut Intelligence içe aktarım deposundaki olay/haber kayıtlarını kullanır.

## Araştırma kayıtlarını kullanma

### Kaynak ekleme

Kaynağın gerçekten erişilen metnini veya veri açıklamasını kaydedin. Yalnızca başlık/özet bulunduğunda tam metin olarak işaretlemeyin. Kaynağın özgünlüğü veya kopya zinciri bilinmiyorsa **Henüz bilinmiyor** seçeneğinde bırakın. Resmî açıklama, iddia kaydı veya birincil kaynak etiketi kendi başına doğruluk kararı değildir.

### Hipotez ve kanıt

Önce sınanabilir bir hipotez ekleyin. Kaynak eklerken o hipotezi seçebilir ve kanıtın yönünü gerekçesiyle kaydedebilirsiniz. Sistem aynı kaynağın hipotezi desteklediği ve ona karşı çıktığı kayıtları saklar; otomatik olarak tek bir hükme indirgemez.

### Hedef seçimi ve örüntü analizi

Araştırma Masası içindeki **Hedef seçimi ve örüntü analizi** paneli, kaynaklı envanter tabanlı bir karşılaştırma sağlar:

1. Bir araştırma dosyasında gerçek kaynakları kanıt kaydı olarak ekle.
2. Çalışma dönemini ve karşılaştırma evrenini tanımla. Evrenin eksiksiz olduğu kontrol edilmediyse kutuyu işaretleme.
3. Tesisleri tek tek ekle. Her tesis için uygunluk kanıtını, sahiplik sınıfını, sahiplik kaydının geçerlilik tarihlerini ve sahiplik kaynağını belirt.
4. Olay ve iddiaları ilgili tesise bağla. “Yalnızca hedefleme iddiası”, “saldırı girişimi bildirildi”, “önleme bildirildi”, “isabet bildirildi” ve “hasar bildirildi” durumlarını ayrı tut.
5. Çalışmayı kaydet ve karşılaştırma raporunu aç. Her grup için evren büyüklüğü, kayıtlı hedefleme gözlemi bulunan tesisler, yalnızca iddia bulunan tesisler, oranlar ve sınırlılıklar görünür.

Bu ilk analiz deterministik ve betimsel bir karşılaştırmadır. İstatistiksel anlamlılık, maruziyet süresi, stratejik değer, konum, savunma koşulları ve raporlama yanlılığı henüz modellenmez. Oran farkı saldırganın niyetini kanıtlamaz. Kanıtlı envanter ve olay listesi olmadan gerçek dünya hakkında sonuç üretilmez.

### Kanıt kapsama raporu

Rapor bir **kapsam ve eksik araştırma kontrolüdür**, otomatik doğrulama veya niyet çıkarımı değildir. Örneğin üç farklı alan adındaki haber, ortak bir ajans metnini yayımlamış olabilir. Rapor bunun bağımsız teyit olduğunu varsaymaz. Veri kümesi, yerel/saha haberciliği, birincil kaynak veya tam metin yoksa bu boşlukları açıkça belirtir.

## API sözleşmesi

Tüm uçlar aynı yerel Intelligence sürecinde sunulur ve önbelleklenmez.

- `GET /api/research-cases`: araştırma dosyalarının özetleri
- `POST /api/research-cases`: dosya oluşturma
- `GET /api/research-case?caseId=...`: bir dosyanın ayrıntıları
- `POST /api/research-hypothesis?caseId=...`: hipotez ekleme
- `POST /api/research-evidence?caseId=...`: kaynaklı kanıt ekleme
- `POST /api/research-question?caseId=...`: açık soru ekleme
- `POST /api/research-question-status?caseId=...&questionId=...`: soruyu açma veya çözüm notuyla kapatma
- `POST /api/research-hypothesis-status?caseId=...&hypothesisId=...`: hipotezin insan tarafından değerlendirilen durumunu değiştirme
- `POST /api/research-import?caseId=...`: bağlı NexNews olayının haberlerini yeni sürümleri koruyarak içe aktarma
- `GET /api/research-analysis?caseId=...`: hipotez bazında kanıt kapsam raporu
- `GET /api/target-selection-studies?caseId=...`: dosyaya bağlı hedef seçimi çalışmalarını listeleme
- `POST /api/target-selection-studies?caseId=...`: kaynak kanıtlarına bağlı tesis/olay envanterini kaydetme
- `GET /api/target-selection-analysis?caseId=...&studyId=...`: kaydedilmiş envanterin betimsel karşılaştırma raporu

Yazma uçları tek bir JSON gövdesi bekler ve 1 MiB üstünü reddeder.

## Şimdilik özellikle yapılmayanlar

Bu sürüm **internette otomatik arama yapan veya tam metinleri kendi başına indiren bir web araştırma ajanı değildir**. NexNews'e zaten aktarılmış haberleri içeri alabilir; yeni kaynaklar kullanıcı tarafından eklenebilir. Bağımsız arama sağlayıcıları, güvenli sayfa edinimi, yerel muhabir veri girişi/senkronizasyonu ve dış veri kümeleri bir sonraki entegrasyon dilimidir.

Hedef seçimi analizi için envanter ve kaynakları ilk sürümde araştırmacı girer; sistem dış internetten tesis listesi veya saldırı geçmişi bulup doğrulamaz. Otomatik arama, güvenli sayfa edinimi, yerel muhabir veri girişi/senkronizasyonu ve dış veri kümeleri bir sonraki entegrasyon dilimidir. Bu nedenle örnek ekranın doldurulması değil, gerçek kaynaklarla yeterli tarihsel veri toplanması tamamlanmadan gerçek dünya örüntüsü ileri sürülmemelidir.
