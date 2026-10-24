package core

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

// Arayüz çevirileri (RC92): Türkçe arayüz metni → dil → çeviri, ve Türkçe ülke
// adı → ISO kodu (ülke adları sayfada Intl.DisplayNames ile yazılır).
//
//go:embed assets/i18n.json
var i18nJSON []byte

// i18nSource: uzaktan güncellenmiş çeviriler varsa onlar, yoksa gömülü olanlar.
func i18nSource() []byte {
	if b := updatedAsset("i18n.json"); b != nil {
		return b
	}
	return i18nJSON
}

// ---- Go tarafı çeviri (RC95) ----
// Sayfadan geçmeyen metinler (masaüstü bildirimleri, pencere başlığı, dosya
// iletişim kutuları) ve bağlama göre anlamı değişen sözcükler (hava durumu
// "Açık" ≠ tema "Açık") için. Sözlük sayfayla aynı i18n.json'dur; bağlama özel
// anahtarlar "hava:", "gün:", "yön:" önekiyle tutulur.

var (
	i18nOnce sync.Once
	i18nDict map[string]map[string]string
)

func loadI18nDict() {
	var f struct {
		Langs map[string]map[string]string `json:"langs"`
	}
	_ = json.Unmarshal(i18nSource(), &f)
	i18nDict = f.Langs
}

// uiLangCode: arayüz dili (ayar ya da sistem); sözlüğü yoksa "en", Türkçe ise "tr".
func uiLangCode() string {
	if cfg == nil {
		return "tr" // ayar yüklenmeden (testler) metinler olduğu gibi kalır
	}
	cfg.mu.RLock()
	l := cfg.UILang
	cfg.mu.RUnlock()
	if l == "" {
		l = systemLang()
	}
	l = strings.ToLower(l)
	if i := strings.IndexAny(l, "-_"); i > 0 {
		l = l[:i]
	}
	if l == "" || l == "tr" {
		return "tr"
	}
	i18nOnce.Do(loadI18nDict)
	if _, ok := i18nDict[l]; !ok {
		return "en"
	}
	return l
}

// uiT: Türkçe arayüz metninin seçili dildeki karşılığı (yoksa olduğu gibi).
func uiT(s string) string { return uiTK(s, s) }

// uiTK: bağlama özel anahtarla çeviri; bulunamazsa Türkçe metin döner.
func uiTK(key, tr string) string {
	l := uiLangCode()
	if l == "tr" {
		return tr
	}
	i18nOnce.Do(loadI18nDict)
	for _, lang := range []string{l, "en"} {
		if v, ok := i18nDict[lang][key]; ok && v != "" {
			return v
		}
		if key != tr {
			if v, ok := i18nDict[lang][tr]; ok && v != "" && !strings.Contains(key, ":") {
				return v
			}
		}
	}
	return tr
}
