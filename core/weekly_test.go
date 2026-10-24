package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWeeklyRecordAndSummary(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	weekMu.Lock()
	weekData = nil
	weekMu.Unlock()
	now := time.Now()
	mk := func(cat, title string, score, src int) *Story {
		return &Story{Title: title, Link: "https://x/" + title, Category: cat, Score: score, IndependentSources: src, FirstSeen: now, Published: now, HasDate: true}
	}
	s := &Store{storiesByCat: map[string][]*Story{
		"Avrupa":             {mk("Avrupa", "a", 90, 4), mk("Avrupa", "b", 40, 1)},
		"Asya":               {mk("Asya", "c", 70, 2)},
		"⚡ Küresel Uyarılar": {mk("⚡ Küresel Uyarılar", "d", 99, 1)},
	}}
	recordWeekly(s)
	b, _ := weeklyJSON()
	var o weeklyOut
	_ = json.Unmarshal(b, &o)
	if o.Total != 2 || o.Top[0].Title != "a" || len(o.Regions) != 2 {
		t.Fatalf("beklenmeyen özet: %s", b)
	}
}
