package core

// Arayüz sayfası (RC95): eskiden tek bir 4.600 satırlık index.html idi.
// Artık assets/ui/ altında bölümlere ayrılmış dosyalardan (stiller, çeviri
// katmanı, genel bakış, canlı TV, radyo, ayarlar…) oluşur. Kabuk dosyadaki
// "@@dosya@@" satırları çalışırken o dosyanın içeriğiyle değiştirilir; çıkan
// sayfa eski tek dosyayla bayt bayt aynıdır.

import (
	"embed"
	"regexp"
	"strings"
	"sync"
)

//go:embed assets/ui
var uiFS embed.FS

var (
	indexOnce  sync.Once
	indexCache string
	includeRe  = regexp.MustCompile(`^@@([A-Za-z0-9._\-]+)@@$`)
)

func assembleIndex() string {
	shell, err := uiFS.ReadFile("assets/ui/index.html")
	if err != nil {
		return "<html><body>NexNews arayüzü bulunamadı.</body></html>"
	}
	lines := strings.Split(string(shell), "\n")
	var b strings.Builder
	for i, ln := range lines {
		if m := includeRe.FindStringSubmatch(ln); m != nil {
			part, err := uiFS.ReadFile("assets/ui/" + m[1])
			if err == nil {
				ln = strings.TrimSuffix(string(part), "\n")
			}
		}
		b.WriteString(ln)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// indexPage: birleştirilmiş sayfa (bir kez kurulur).
func indexPage() string {
	indexOnce.Do(func() { indexCache = assembleIndex() })
	return indexCache
}

// PreviewPage: geliştirme önizlemesi için yer tutucuları doldurulmuş sayfa
// (logo ve arka plan resmi hariç).
func PreviewPage() string {
	h := indexPage()
	h = strings.ReplaceAll(h, "__NEXLIVETV__", string(liveTVJSON()))
	h = strings.ReplaceAll(h, "__NEXI18N__", string(i18nSource()))
	h = strings.ReplaceAll(h, "__NEXOWN__", string(ownershipJSON()))
	return h
}
