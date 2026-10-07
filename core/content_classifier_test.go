package core

import (
	"testing"
	"time"
)

func TestRollingBulletinClassifier(t *testing.T) {
	cases := []struct {
		name  string
		title string
		want  rollingBulletinClass
	}{
		{"quake list", "Son dakika deprem mi oldu? Az önce deprem nerede oldu? İstanbul, Ankara, İzmir ve il il AFAD son depremler 06 Ekim 2026", bulletinQuake},
		{"quake list short", "AFAD son depremler listesi", bulletinQuake},
		{"real quake", "Adıyaman'da 4,2 büyüklüğünde deprem meydana geldi", bulletinNone},
		{"market", "Altın fiyatları bugün ne kadar? Gram altın ve çeyrek altın", bulletinMarket},
		{"weather", "İstanbul 5 günlük hava durumu", bulletinWeather},
		{"sports", "Galatasaray maç sonucu ve puan durumu", bulletinSports},
		{"prayer", "İstanbul namaz vakitleri bugün", bulletinPrayer},
		{"breaking report", "Son dakika: Deprem sonrası ekipler bölgeye sevk edildi", bulletinNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyRollingBulletin(Item{Title: tc.title})
			if got != tc.want {
				t.Fatalf("classifyRollingBulletin()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestRollingBulletinsDoNotEnterTimeline(t *testing.T) {
	now := time.Now()
	items := []Item{
		{Title: "Son dakika deprem mi oldu? AFAD son depremler 06 Ekim 2026", Link: "bulletin", Source: "NTV Son Dakika", Published: now, HasDate: true},
		{Title: "Son dakika Adana'da deprem mi oldu? AFAD son depremler listesi", Link: "bulletin2", Source: "NTV Son Dakika", Published: now.Add(-2 * time.Hour), HasDate: true},
		{Title: "Adıyaman'da 4,2 büyüklüğünde deprem meydana geldi", Link: "event", Source: "AP", Published: now.Add(-3 * time.Hour), HasDate: true},
		{Title: "Adıyaman'da 4,2 büyüklüğünde deprem sonrası ekipler sahada", Link: "event2", Source: "BBC", Published: now.Add(-2 * time.Hour), HasDate: true},
	}
	filtered := filterEventItems(items)
	if len(filtered) != 2 {
		t.Fatalf("filtered items=%d want 2", len(filtered))
	}

	cl := clusterItems(items)
	if len(cl) != 1 {
		t.Fatalf("got %d clusters, want 1", len(cl))
	}
	if cl[0].Count != 2 {
		t.Fatalf("event count=%d want 2", cl[0].Count)
	}
	for _, it := range cl[0].items {
		if classifyRollingBulletin(it) != bulletinNone {
			t.Fatalf("rolling bulletin leaked into event timeline: %q", it.Title)
		}
	}
}

func TestContentIntelligenceKinds(t *testing.T) {
	cases := []struct {
		name  string
		title string
		want  contentKind
	}{
		{"event", "Adıyaman'da 4,2 büyüklüğünde deprem meydana geldi", contentEvent},
		{"update", "Depremde yeni gelişme: ekipler bölgeye sevk edildi", contentUpdate},
		{"analysis", "Uzman değerlendirmesi: Deprem neden oldu?", contentAnalysis},
		{"bulletin", "AFAD son depremler listesi", contentBulletin},
		{"live data", "Altın fiyatları bugün ne kadar?", contentLiveData},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyContent(Item{Title: tc.title})
			if got != tc.want {
				t.Fatalf("classifyContent()=%q want %q", got, tc.want)
			}
		})
	}
}
