package core

// Classifier + Rank Engine. kindOf assigns a coarse type/severity used for the
// colored dot; scoreStory produces a 0–100 importance used to surface the
// "ŞU ANDA ÖNEMLİ" stories and to order the feed. The factors mirror the product
// plan: breaking, multi-source corroboration, recency, repetition, hazard. The
// user-interest factor is added in a later phase.

import (
	"strings"
	"time"
)

// Kinds (also used as CSS classes in the UI).
const (
	kindAcil      = "acil"
	kindTeknoloji = "teknoloji"
	kindBilim     = "bilim"
	kindEkonomi   = "ekonomi"
	kindDunya     = "dunya"
	kindTurkiye   = "turkiye"
	kindGenel     = "genel"
)

func kindOf(c *Story) string {
	if c.Breaking || storyHasHazard(c) {
		return kindAcil
	}
	cat := strings.ToLower(c.Category)
	switch {
	case containsWord(cat, "teknoloji", "tech", "teknoloji̇"):
		return kindTeknoloji
	case containsWord(cat, "bilim", "science", "uzay", "space"):
		return kindBilim
	case containsWord(cat, "ekonomi", "economy", "finans", "finance", "piyasa"):
		return kindEkonomi
	case containsWord(cat, "türkiye", "turkiye", "turkey", "gündem"):
		return kindTurkiye
	case containsWord(cat, "dünya", "dunya", "world", "küresel", "uyarı",
		"avrupa", "europe", "asya", "asia", "ortadoğu", "ortadogu", "middle",
		"afrika", "africa", "amerika", "america"):
		return kindDunya
	default:
		return kindGenel
	}
}

func containsWord(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func storyHasHazard(c *Story) bool {
	for _, it := range c.items {
		if containsAny(strings.ToLower(it.Title), hazardTerms) {
			return true
		}
	}
	return false
}

// scoreStory returns a 0–100 importance score.
func scoreStory(c *Story) int {
	s := 0
	if c.Breaking {
		s += 38
	}
	if storyHasHazard(c) {
		s += 14
	}
	// independent-source corroboration (up to +42)
	ns := independentSourceCount(c)
	if ns > 6 {
		ns = 6
	}
	s += ns * 7
	// recency (up to +20)
	if c.HasDate {
		age := time.Since(c.Published)
		switch {
		case age <= 30*time.Minute:
			s += 20
		case age <= 2*time.Hour:
			s += 14
		case age <= 6*time.Hour:
			s += 8
		case age <= 24*time.Hour:
			s += 3
		}
	}
	// repetition / volume (up to +16)
	rep := c.Count
	if rep > 8 {
		rep = 8
	}
	s += rep * 2
	if s > 100 {
		s = 100
	}
	return s
}
