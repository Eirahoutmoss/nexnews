package core

import (
	"encoding/json"
	"testing"
	"time"
)

// A feed listed in two categories must not produce the same event twice in
// the top sections.
func TestOverviewDedupesAcrossCategories(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	c := &Config{MaxPerCategory: 5, Categories: []Category{
		{Name: "A", Sources: []Source{{URL: "https://a.example/rss"}, {URL: "https://b.example/rss"}}},
		{Name: "B", Sources: []Source{{URL: "https://b.example/rss"}}},
	}}
	s := newStore(c)
	now := time.Now()
	a := Item{Title: "Leaders agree migration package at summit", Link: "https://a.example/1", Source: "A", Published: now.Add(-10 * time.Minute), HasDate: true}
	b := Item{Title: "Leaders agree new migration package after summit", Link: "https://b.example/1", Source: "B", Published: now.Add(-20 * time.Minute), HasDate: true}
	aB := b
	aA, bB := a, b
	aA.Category, aB.Category, bB.Category = "A", "A", "B"
	s.rebuildStories(map[string][]Item{"A": {aA, aB}, "B": {bB}}, []string{"A", "B"})
	raw, err := s.overviewJSON()
	if err != nil {
		t.Fatal(err)
	}
	var ov overview
	if err := json.Unmarshal(raw, &ov); err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, it := range ov.Important {
		if it.Link == a.Link || it.Link == b.Link {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("same event listed %d times in important", seen)
	}
}
