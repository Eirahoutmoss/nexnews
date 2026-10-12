# NexNews Intelligence Foundation (v0.1)

**Durum:** Uygulama öncesi veri ve analiz sözleşmesi  
**Hedef:** NexNews verisini kullanan fakat bağımsız ürün değeri ve kendi analiz katmanı olan bir Intelligence ürünü kurmak.  
**Temel kural:** İlişki üretmek zorunda değiliz. Kanıt yoksa ilişki gösterilmez.

## 1. Ürün sınırı

NexNews haber toplama, akış, okuma, temel gruplama ve kaynak karşılaştırma ürünüdür. NexNews Intelligence ise haberleri ve ileride bağlanabilecek güvenilir veri kaynaklarını kullanarak olay dosyaları, iddialar, kanıtlar, zaman içindeki değişimler ve gerekçelendirilmiş ilişkiler sunan ayrı bir analiz ürünüdür.

Intelligence mevcut NexNews verisini yeniden kullanabilir; ancak kendi veri sözleşmesine, analiz kurallarına, kalıcı kayıtlarına ve API/arayüz sınırına sahip olmalıdır. Aynı Go deposunda geliştirilmesi, sonradan bağımsız çalıştırılabilmesini engellememelidir.

## 2. Bağlayıcı doğruluk kuralları

1. **Birlikte geçmek ilişki değildir.** İki varlığın aynı başlıkta, aynı haberde, aynı gün veya aynı kategoride anılması aralarında ilişki kurmaya yetmez.
2. **Kelime benzerliği kanıt değildir.** Başlık token benzerliği yalnızca olay adayı bulmaya yardımcı olabilir; olaylar arasında semantik ilişki kuramaz.
3. **Her ilişki kanıt taşır.** Her yayımlanmış ilişki, kaynağa bağlı bir kanıt parçası, ilişkiyi destekleyen açık ifade ve denetlenebilir gerekçe içermelidir.
4. **Kanıtsız ilişki yoktur.** Eşik karşılanmıyorsa ilişki gizlenir. Sistem boş ilişki haritası gösterebilir.
5. **İddia, gerçek olarak etiketlenmez.** Haberin iddiayı aktarması, iddianın doğru olduğu anlamına gelmez. Kaynak sayısı tek başına doğruluk kanıtı değildir.
6. **Kaynak bağımsızlığı kontrol edilir.** Ajans metnini veya aynı metni yeniden yayımlayan siteler ayrı bağımsız doğrulama sayılmaz.
7. **Çelişki silinmez.** Birbiriyle uyuşmayan anlatımlar ayrı iddialar ve kanıtlar olarak tutulur.
8. **Kaynağa geri dönüş zorunludur.** Her iddia ve kanıt; haber URL'si, yayıncı ve zaman bilgisine geri götürülebilmelidir.
9. **Belirsizlik görünürdür.** Yalnızca başlık/özet düzeyinde verilerle yapılan analiz, tam metin incelenmiş gibi sunulamaz.
10. **Analiz denetlenebilir olmalıdır.** Hangi girdi, kural/model sürümü ve işlem sonucunda kayıt oluşturulduğu saklanır.

## 3. Mevcut depoyla ilgili mimari not

Mevcut NexNews deposunda olay gruplama, bağımsız kaynak sayımı ve ajans farkındalığı için yeniden kullanılabilecek parçalar vardır. Ancak core/entity_intelligence.go aynı olayda birlikte görülen varlık çiftlerinden ilişki üretmektedir; core/event_graph.go ise başlık token benzerliği eşiğinden olaylar arası related kenarları oluşturabilmektedir. Bu davranışlar Intelligence'ın kanıtlanmış semantik ilişkisi olarak kullanılmayacaktır.

Mevcut Story kümesi olay adayı olarak kabul edilebilir; ancak kümeleme sonucu mutlak gerçek değildir. core.Item verisi çoğu akışta başlık, özet, yayıncı, URL ve tarihten oluştuğu için içeriğin erişim seviyesi kaydedilmelidir.

İlk fazda ortak masaüstü/Android core çekirdeğinin yeni bağımlılıklarla ağırlaştırılmaması; Intelligence veri sözleşmesinin ve analiz kodunun mantıksal olarak ayrılması tercih edilir. Haber önbelleği mevcut biçiminde kalmalı, Intelligence kayıtları ayrı bir depoda tutulmalıdır.

## 4. Temel veri varlıkları

### Source
Yayıncı veya birincil veri sağlayıcı kimliği. Yayıncı ile ajans/orijinal kaynak ayrı tutulur.

Alanlar: id, name, canonicalDomain, sourceType (publisher / primary / official_dataset / other), identityBasis, createdAt.

Kaynak tipi bir doğruluk puanı değildir. Resmî kaynak etiketi, kaynağın her ifadesinin doğru olduğu anlamına gelmez.

### Article
Mevcut NexNews öğesinin kaynak referanslı görünümü.

Alanlar: id, canonicalUrl, publisherSourceId, title, summary, publishedAt (bilinmiyorsa null), retrievedAt, language, contentAvailability (title_only / title_summary / full_text), contentHash, syndicationOriginId, eventCandidateIds.

Kaynak metni ile Intelligence'ın ürettiği özet veya çıkarım birbirine karıştırılmaz.

### Event
Belirli bir gelişmeyi zaman içinde izleyen olay dosyası. Olay kimliği, geçici kümeleme kimliğinden ayrı ve kalıcı olmalıdır.

Alanlar: id, title, scope, status, firstObservedAt, lastObservedAt, articleIds, claimIds, timelineEntryIds, coverageLimitations, createdByRuleVersion, updatedAt.

Durumlar: emerging, developing, stabilizing, archived, needs_review. Durum, doğruluk hükmü değildir.

### Entity
Kişi, kurum, şirket, yer, mevzuat, ürün veya başka bir adlandırılmış nesne.

Alanlar: id, canonicalName, entityType, aliases, identityStatus (candidate / resolved / ambiguous), resolutionEvidenceIds.

Sadece sözlükte bir kelime geçti diye varlığın kesin tanımlandığı varsayılmaz. Kısa ve çok anlamlı adlarda belirsizlik desteklenir.

### Claim
Bir kaynak veya aktör tarafından ileri sürülen, doğruluğu ayrıca değerlendirilebilen atomik ifade.

Alanlar: id, subjectEntityId, predicate, objectText veya objectEntityId, polarity, claimText, attributedTo, firstSeenAt, eventId, evidenceIds, status, statusReason.

Durumlar: reported, corroborated, contested, retracted, unverified, unknown. Corroborated yalnızca bağımsız ve iddianın kendisini destekleyen kanıtlar varsa kullanılabilir. MVP'de otomatik ve mutlak “kesin doğrulandı” hükmü yoktur.

### Evidence
İddiayı destekleyen, çürüten veya bağlam sağlayan belirli kaynak parçası.

Alanlar: id, articleId, claimId, stance (supports / contradicts / context / mentions), quote, startOffset/endOffset (varsa), sourceUrl, capturedAt, contentLevel, extractionMethod, extractionVersion.

Alıntı yalnızca gerçekten erişilen metinden alınır. Başlık/özet dışındaki metin erişilebilir değilse kanıt o sınırı aşamaz.

### Relation
İki varlık veya iki olay arasında ileri sürülen, türü belirlenmiş ve kanıtla temellendirilmiş bağ.

Alanlar: id, subjectId, predicate, objectId, relationStatus, evidenceIds, rationale, validFrom, validTo, createdBy, ruleVersion, reviewStatus.

Durumlar: candidate, supported, contested, rejected, expired. Yalnızca supported ilişkiler standart ilişki haritasında gösterilebilir. Adaylar kanıtlanmış gerçekmiş gibi sunulamaz.

### TimelineEntry
Olayla ilgili yeni gözlem, açıklama, ölçüm, karar veya durum değişikliği. İlk görülme zamanı ile olayın gerçekleşme zamanı ayrı tutulur; zaman kesin değilse hassasiyet belirtilir.

### AnalysisRun
Her otomatik işlem için girdi kimliklerini, kural/model sürümünü, tamamlanma durumunu ve uyarıları kaydeder. Yanlış çıkarım kaynağına ve sürümüne kadar izlenebilir.

## 5. İzin verilen ilişki türleri

İlişki türleri kapalı bir kayıt defterinden seçilir. MVP'de yalnızca doğrudan metinsel veya yapılandırılmış kanıtı olan türler işlenir:

- announced_by, said_by, issued_by
- participates_in, member_of
- located_in
- owns, operates, controls
- responds_to
- supports_claim, contradicts_claim
- updates_event, same_event_as (yalnızca olay eşleştirme sürecinde)
- caused_by veya leads_to: yalnızca açıkça ileri sürülmüş nedensellik ifadesi ve kaynağıyla; korelasyon veya zaman sırası yeterli değildir.

Yeni ilişki türü eklenmeden önce anlamı, gerekli kanıtı, yanlış pozitif riski ve testleri tanımlanmalıdır.

### Yasaklar

- Varlıkların aynı belgede geçmesinden varlık-varlık ilişkisi çıkarmak.
- Aynı ülke, kategori, konum, zaman veya konu etiketinden ilişki çıkarmak.
- Benzer kelimelerden related_to bağı üretmek.
- Nedenselliği korelasyondan veya ardışık olaylardan çıkarmak.
- Modelin ürettiği açıklamayı kaynak kanıtı gibi kaydetmek.
- Kaynak sayısından tek başına doğru sonucu üretmek.
- Görselliği doldurmak için tahmini kenarlar eklemek.

“Birlikte anıldı” durumu bir haber içi mention kaydı olarak tutulabilir; bu, iki varlık arasında ilişki oluşturmaz.

## 6. İlk analiz hattı

1. **Ingest:** NexNews öğe/olay verisini salt-okunur adaptörle al.
2. **Normalize:** URL, zaman, yayıncı, dil, içerik seviyesi ve metin parmak izini düzenle; kaynak referansını koru.
3. **Deduplicate:** aynı URL/içerik/ajans kopyalarını grupla; her yayıncı ve kaynak kökeni bilgisini kaybetme.
4. **Event candidate:** mevcut Story kümesini başlangıç adayı olarak kullan. Belirsiz eşleşmelerde birleştirme zorunlu değildir.
5. **Extract:** varlık ve atomik iddia adaylarını çıkar; her adayın kaynak ve kanıt konumu bulunmalı.
6. **Evaluate evidence:** bağımsız kaynak/kopya zincirini dikkate al; destekleyen ve çelişen ifadeleri ayrı kaydet.
7. **Qualify relation:** kapalı ilişki türü, kanıt ve gerekçe kurallarını geçmeyen kenarı yayımlama.
8. **Persist:** kaynak/kanıt/iddia/ilişki kayıtlarını ayrı Intelligence deposuna yaz; şema ve analiz sürümünü tut.
9. **Present:** olay dosyası, kaynaklar, iddialar, çelişkiler ve zaman çizelgesini göster. İlişki haritası yalnızca nitelikli ilişkiler varsa görünür.

İlk sürümde ücretli bir LLM API'si veya uzak servis zorunlu olmamalıdır. Model tabanlı çıkarım ileride eklenirse yalnızca aday üretir; kanıt şartını kaldırmaz.

## 7. MVP kapsamı

### Dahil
- NexNews Story/Item kayıtlarından olay dosyası.
- Kaynak ve bağımsız köken zinciri.
- Makale/özet düzeyinin açıkça belirtilmesi.
- Kaynağa bağlı iddia ve kanıt kayıtları.
- Olay zaman çizelgesi ve yeni bilgi/tekrar ayrımı.
- Desteklenen, çelişen ve belirsiz iddiaları ayrı gösterme.
- Analiz sonucunun gerekçesi ve sürüm izi.
- İlişki yoksa boş/ilişkisiz görünüm.

### İlk sürüme dahil değil
- Kanıtsız genel bilgi grafiği.
- Otomatik jeopolitik veya siyasi “büyük resim” bağlantıları.
- Kesin gelecek tahmini veya kanıtsız nedensellik.
- Haber metninde olmayan bilgiyi kanıt gibi gösterme.
- Ücretli LLM API'sine zorunlu bağımlılık.
- NexNews'in mevcut haber akışını bozacak değişiklikler.

## 8. Veri saklama ve uyumluluk

- Mevcut feed-cache.json biçimi kırılmamalı.
- Intelligence deposu bağımsız şema sürümüne ve atomik yazıma sahip olmalı.
- İlk uygulamada depolama adaptör arayüzü bulunmalı; başlangıç deposu mevcut masaüstü/Android bağımlılık kısıtlarını bozmamalı.
- Her kayıt oluşturulma/güncellenme zamanı ve analiz kuralı/model sürümünü tutmalı.
- Kaynak içeriği nedeniyle kanıt artık doğrulanamıyorsa kayıt sessizce korunmamalı; erişilememe uyarısı eklenmeli.
- Her gösterilen iddia ve ilişki orijinal kaynağa götürmeli.

## 9. MVP kabul ölçütleri

1. Alakasız iki haber aynı kişi/ülke/kelimeyi içerse bile aralarında semantik ilişki oluşmaz.
2. İki varlık tek belgede geçiyor ama ilişkiyi söyleyen kanıt yoksa ilişki kenarı oluşmaz.
3. Aynı ajans metnini yeniden yayımlayan beş site, beş bağımsız doğrulama sayılmaz.
4. Bir kaynak destekleyip diğeri karşı çıkıyorsa iki kanıt da korunur ve iddia contested olabilir.
5. Yalnızca başlık/özet erişilen öğe tam metin analiz edilmiş gibi işaretlenmez.
6. Her supported ilişki için kaynak alıntısı ve gerekçe vardır.
7. İlişki kurulamıyorsa sistem ilişkisiz sonucu verebilir.
8. Aynı girdi ve kural sürümü deterministik aşamada aynı sonucu verir.
9. NexNews masaüstü ve Android ortak çekirdeği bozulmaz.
10. İlişki çıkarımında yanlış pozitifleri engelleyen negatif testler bulunur.

## 10. Uygulama sırası

**Faz A - Sözleşme ve test fikstürleri:** Veri sözleşmesini sabitle; pozitif ve negatif ilişki örnekleri oluştur.

**Faz B - Veri modeli ve adapter:** NexNews Item/Story verisini salt-okunur biçimde Intelligence dokümanlarına eşle; ayrı depoyu oluştur; haber önbelleğine dokunma.

**Faz C - Kanıt/iddia:** Kaynak metni sınırlarını koruyarak iddia adayları ve kanıt parçaları üret; bağımsızlık ve çelişki testlerini tamamla.

**Faz D - İlişki kapısı:** İlişki kayıt defteri, kanıt gereksinimleri ve negatif testler hazır olmadan ilişki haritasını açma.

**Faz E - Bağımsız ürün arayüzü:** Olay Dosyası, Kanıtlar, Zaman Çizelgesi, Çelişkiler ve Kaynaklar. Harita yalnızca doğrulanabilir kenarlar olduğunda gösterilir.

Her faz ayrı derlenebilir/test edilebilir değişiklik olarak teslim edilir. Faz test edilmeden sonraki faza geçilmez.
