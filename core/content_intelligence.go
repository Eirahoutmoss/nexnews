package core

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type contentKind string

const (
	contentEvent    contentKind = "event"
	contentUpdate   contentKind = "update"
	contentAnalysis contentKind = "analysis"
	contentBulletin contentKind = "bulletin"
	contentLiveData contentKind = "live-data"
)

// classifyContent gives the feed item a semantic role. It is deliberately
// separate from rolling-bulletin suppression: classification describes what
// an item is, while filterEventItems decides whether it may enter Event
// Intelligence.
func classifyContent(it Item) contentKind {
	if cls := classifyRollingBulletin(it); cls != bulletinNone {
		if cls == bulletinMarket || cls == bulletinWeather || cls == bulletinSports || cls == bulletinPrayer {
			return contentLiveData
		}
		return contentBulletin
	}
	text := strings.ToLower(strings.TrimSpace(it.Title + " " + it.Summary))
	if text == "" {
		return contentEvent
	}
	if containsAnyText(text,
		"analiz", "analysis", "yorum", "uzman değerlendirmesi", "uzman degerlendirmesi",
		"ne anlama geliyor", "ne anlama geliyor?", "neden oldu", "neden oluyor",
		"arka plan", "dosya", "inceleme", "değerlendirme", "degerlendirme") {
		return contentAnalysis
	}
	if containsAnyText(text,
		"son gelişme", "son gelisme", "yeni gelişme", "yeni gelisme",
		"gelişme: ", "gelisme: ", "güncelleme", "guncelleme",
		"açıklama geldi", "aciklama geldi", "açıklamasında", "aciklamasinda") {
		return contentUpdate
	}
	return contentEvent
}

type contentIntelligenceSummary struct {
	Updated         string         `json:"updated"`
	RawItems        int            `json:"rawItems"`
	EventCandidates int            `json:"eventCandidates"`
	Suppressed      int            `json:"suppressed"`
	SuppressionPct  int            `json:"suppressionPct"`
	Classes         map[string]int `json:"classes"`
	TopClasses      []contentClass `json:"topClasses"`
	Kinds           map[string]int `json:"kinds"`
}

type contentClass struct {
	Class string `json:"class"`
	Count int    `json:"count"`
}

func (s *Store) contentIntelligenceJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := map[rollingBulletinClass]int{}
	kinds := map[contentKind]int{}
	raw := 0
	for _, entry := range s.cache {
		for _, it := range entry.Items {
			raw++
			kinds[classifyContent(it)]++
			if cls := classifyRollingBulletin(it); cls != bulletinNone {
				counts[cls]++
			}
		}
	}

	suppressed := 0
	top := make([]contentClass, 0, len(counts))
	labels := map[rollingBulletinClass]string{
		bulletinQuake:   "Deprem veri/listesi",
		bulletinMarket:  "Piyasa/fiyat",
		bulletinWeather: "Hava durumu",
		bulletinSports:  "Spor skor/listesi",
		bulletinPrayer:  "Namaz vakitleri",
		bulletinGeneric: "Genel veri/listesi",
	}
	for cls, n := range counts {
		suppressed += n
		top = append(top, contentClass{Class: labels[cls], Count: n})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].Count > top[j].Count })

	pct := 0
	if raw > 0 {
		pct = suppressed * 100 / raw
	}
	return json.Marshal(contentIntelligenceSummary{
		Updated:         time.Now().Format(time.RFC3339),
		RawItems:        raw,
		EventCandidates: raw - suppressed,
		Suppressed:      suppressed,
		SuppressionPct:  pct,
		Classes: map[string]int{
			"earthquake": counts[bulletinQuake],
			"market":     counts[bulletinMarket],
			"weather":    counts[bulletinWeather],
			"sports":     counts[bulletinSports],
			"prayer":     counts[bulletinPrayer],
			"generic":    counts[bulletinGeneric],
		},
		TopClasses: top,
		Kinds: map[string]int{
			"event":    kinds[contentEvent],
			"update":   kinds[contentUpdate],
			"analysis": kinds[contentAnalysis],
			"bulletin": kinds[contentBulletin],
			"liveData": kinds[contentLiveData],
		},
	})
}
