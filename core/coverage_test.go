package core

import (
	"testing"
	"time"
)

func TestCoverageBlindSpot(t *testing.T) {
	now := time.Now()
	mk := func(cat, title, src string) *Story {
		it := Item{Title: title, Link: "https://x/" + cat, Source: src, Category: cat, Published: now, HasDate: true}
		st := newStory(it, clusterTokens(title))
		finalizeStory(st)
		return st
	}
	a := mk("Türkiye", "Japonya'da 7,1 büyüklüğünde deprem: tsunami uyarısı verildi", "A")
	s := &Store{storiesByCat: map[string][]*Story{
		"Türkiye": {a},
		"Asya":    {mk("Asya", "Japonya deprem tsunami uyarısı 7,1 büyüklüğünde", "B")},
		"Afrika":  {mk("Afrika", "Nijerya seçimlerinde sonuçlar açıklandı", "C")},
		"Bilim":   {mk("Bilim", "Japonya deprem tsunami", "D")},
	}}
	cv := s.coverageFor(a)
	if cv == nil || len(cv.Covered) != 2 || len(cv.Silent) != 1 || cv.Silent[0] != "Afrika" {
		t.Fatalf("beklenmeyen kapsama: %+v", cv)
	}
}
