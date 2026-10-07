package core

import "strings"

// rollingBulletinClass identifies pages that behave like continuously updated
// data/listing pages rather than a report about a distinct event. These items
// are kept in the raw feed cache for traceability, but are excluded from event
// clustering and timelines so repeated SEO/list updates cannot manufacture a
// fake event history.
type rollingBulletinClass string

const (
	bulletinNone    rollingBulletinClass = ""
	bulletinQuake   rollingBulletinClass = "earthquake"
	bulletinMarket  rollingBulletinClass = "market"
	bulletinWeather rollingBulletinClass = "weather"
	bulletinSports  rollingBulletinClass = "sports"
	bulletinPrayer  rollingBulletinClass = "prayer"
	bulletinGeneric rollingBulletinClass = "generic"
)

func classifyRollingBulletin(it Item) rollingBulletinClass {
	text := strings.ToLower(strings.TrimSpace(it.Title + " " + it.Summary))
	if text == "" {
		return bulletinNone
	}

	// Earthquake feeds are the most important case for the current timeline
	// problem. A real quake report is allowed through; recurring "last quakes"
	// lists are not.
	if containsAnyText(text, "son depremler", "deprem listesi", "son depremler listesi", "afad son depremler", "kandilli son depremler") {
		return bulletinQuake
	}
	if containsAnyText(text, "son dakika") &&
		containsAnyText(text, "nerede oldu", "kaç büyüklüğünde", "hangi bölgede", "afad son", "kandilli son") &&
		containsAnyText(text, "deprem", "sarsıntı", "sarsinti") {
		return bulletinQuake
	}

	// Market/rate pages are snapshots of a changing data table, not events.
	if containsAnyText(text,
		"altın fiyatları", "altin fiyatlari", "dolar ne kadar", "euro ne kadar",
		"döviz kurları", "doviz kurlari", "canlı döviz", "canli doviz",
		"güncel döviz", "guncel doviz", "gram altın ne kadar", "gram altin ne kadar") {
		return bulletinMarket
	}

	// Sports scoreboards and fixture tables should not create event timelines.
	if containsAnyText(text, "canlı skor", "canli skor", "maç sonuçları", "mac sonuclari", "puan durumu", "fikstür") {
		return bulletinSports
	}

	if containsAnyText(text, "hava durumu", "5 günlük hava", "5 gunluk hava", "7 günlük hava", "7 gunluk hava", "haftalık hava", "haftalik hava") {
		return bulletinWeather
	}

	if containsAnyText(text, "namaz vakitleri", "imsak vakti", "iftar vakti") {
		return bulletinPrayer
	}

	// Generic list/snapshot language only counts when it is paired with a
	// recurring-data marker. This deliberately avoids suppressing ordinary
	// articles that merely contain words such as "son dakika" or "liste".
	if containsAnyText(text, "güncel liste", "guncel liste", "son durum", "anlık durum", "anlik durum") &&
		containsAnyText(text, "saat", "bugün", "bugun", "bugünün", "bugunun", "liste", "verileri", "verileri") {
		return bulletinGeneric
	}

	return bulletinNone
}

func containsAnyText(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func filterEventItems(items []Item) []Item {
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if classifyRollingBulletin(it) != bulletinNone {
			continue
		}
		out = append(out, it)
	}
	return out
}
