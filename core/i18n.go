package core

import _ "embed"

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
