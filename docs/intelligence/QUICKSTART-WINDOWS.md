# NexNews Intelligence: Windows hızlı başlangıç

> Bu özellik henüz `main` dalına birleştirilmedi. Komut klasörleri için PR'ın özellik dalındaki ZIP arşivini kullanın; daha önce indirilmiş `main` ZIP'i bu klasörleri içermez.

## 1. Ön koşul: NexNews önbelleği

NexNews'i bir kez çalıştırıp haberleri yenileyin. PowerShell'de önbelleğin yerinde olduğunu doğrulayın:

```powershell
Test-Path "$env:APPDATA\NexNews\feed-cache.json"
```

Sonuç `True` olmalı. `False` ise NexNews henüz önbellek oluşturmamış olabilir veya farklı bir veri dizini kullanıyor olabilir.

## 2. Intelligence kayıtlarını oluşturun

Arşivi çıkartın, PowerShell'i `go.mod` dosyasının bulunduğu depo kökünde açın:

```powershell
go run ./cmd/nexnews-intelligence -db "$env:APPDATA\NexNews\nexnews-intelligence.json"
```

CLI, varsayılan olarak `%APPDATA%\NexNews\feed-cache.json` dosyasını okur. `nexnews-snapshot.json` oluşturmak gerekmez. Haber kaynağına ağ isteği yapılmaz; yalnızca yerel önbellek kullanılır.

Başarılı olduğunda kaynak haber sayıları ve analiz sonuçları JSON olarak yazdırılır. Hata halinde mesajı kontrol edin; boş veya okunamayan önbellek başarıyla işlenmiş sayılmaz.

## 3. Intelligence arayüzünü açın

İlk komut bittikten sonra ikinci PowerShell penceresinde, yine depo kökünden çalıştırın:

```powershell
go run ./cmd/nexnews-intelligence-ui -db "$env:APPDATA\NexNews\nexnews-intelligence.json" -addr 127.0.0.1:8787
```

Tarayıcıda `http://127.0.0.1:8787` adresini açın. Sunucu varsayılan olarak yalnızca bu bilgisayardan erişilecek şekilde dinler.

## Veri ve doğruluk sınırları

- Intelligence veritabanı, NexNews'in `feed-cache.json` dosyasından ayrıdır.
- Kaynak ifadeleri "bildirildi" şeklinde tutulur; doğru olduklarına dair karar verilmez.
- Varlık anılmaları sözlük eşleşmesi adaylarıdır.
- Aynı haberde birlikte anılmak ilişki oluşturmaz.
- Bu ilk arayüz salt okunurdur; haber indirmez veya kayıtları değiştirmez.
