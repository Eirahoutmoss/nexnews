package core

import _ "embed"

// Arayüz çevirileri (RC92): Türkçe arayüz metni → dil → çeviri, ve Türkçe ülke
// adı → ISO kodu (ülke adları sayfada Intl.DisplayNames ile yazılır).
//
//go:embed assets/i18n.json
var i18nJSON []byte
