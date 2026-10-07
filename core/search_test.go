package core

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseWindow(t *testing.T) {
	cases := []struct{ q, label, rest string }{
		{"deprem türkiye son 24 saat", "son 24 saat", "deprem türkiye"},
		{"yapay zeka son 7 gün", "son 7 gün", "yapay zeka"},
		{"bugün deprem", "bugün", "deprem"},
		{"deprem", "", "deprem"},
	}
	for _, c := range cases {
		_, label, rest := parseWindow(c.q)
		if label != c.label {
			t.Errorf("%q label=%q want %q", c.q, label, c.label)
		}
		if strings.TrimSpace(rest) != c.rest {
			t.Errorf("%q rest=%q want %q", c.q, strings.TrimSpace(rest), c.rest)
		}
	}
}

func TestSearchGroups(t *testing.T) {
	now := time.Now()
	st := newStore(&Config{MaxPerCategory: 5})
	st.byCat = map[string][]Item{
		"Dünya": {
			{Title: "Türkiye'de deprem meydana geldi", Link: "a", Source: "BBC", Published: now, HasDate: true, Category: "Dünya"},
			{Title: "Türkiye'de deprem: artçılar sürüyor", Link: "b", Source: "AP", Published: now.Add(-15 * time.Minute), HasDate: true, Category: "Dünya"},
			{Title: "Hava durumu bugün güneşli", Link: "c", Source: "X", Published: now, HasDate: true, Category: "Dünya"},
		},
	}
	b, _ := st.searchJSON("deprem türkiye son 24 saat")
	var r outSearch
	_ = json.Unmarshal(b, &r)
	if r.WindowLabel != "son 24 saat" {
		t.Errorf("window=%q", r.WindowLabel)
	}
	if r.Total != 1 {
		t.Fatalf("expected 1 event, got %d (%s)", r.Total, string(b))
	}
	if r.Items[0].SourceCount != 2 {
		t.Errorf("merged sources=%d want 2", r.Items[0].SourceCount)
	}
}

func TestEventFile(t *testing.T) {
	now := time.Now()
	items := []Item{
		{Title: "Türkiye'de büyük deprem meydana geldi", Link: "a", Source: "BBC", Published: now, HasDate: true},
		{Title: "Türkiye'de büyük deprem: tsunami uyarısı", Link: "b", Source: "AP", Published: now.Add(-20 * time.Minute), HasDate: true},
	}
	st := newStore(&Config{})
	cl := clusterItems(items)
	rankStories(cl)
	st.storiesByCat = map[string][]*Story{"X": cl}
	b, _ := st.storyJSON(cl[0].Link)
	var e outEvent
	_ = json.Unmarshal(b, &e)
	if !e.Found || len(e.Timeline) != 2 {
		t.Fatalf("event file wrong: %s", string(b))
	}
	// timeline ascending: oldest first
	if e.Timeline[0].TS > e.Timeline[1].TS {
		t.Errorf("timeline not ascending")
	}
}
