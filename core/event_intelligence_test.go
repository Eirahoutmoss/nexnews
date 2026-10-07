package core

import (
	"testing"
	"time"
)

func TestEventIntelligenceIndependentSources(t *testing.T) {
	now := time.Now()
	c := &Story{Title: "Test deprem", Category: "Türkiye", Published: now, HasDate: true}
	c.items = []Item{
		{Title: "Test deprem", Link: "https://example-a.com/a", Source: "A", Published: now.Add(-20 * time.Minute), HasDate: true},
		{Title: "Test deprem", Link: "https://example-b.com/b", Source: "B", Published: now.Add(-10 * time.Minute), HasDate: true},
		{Title: "Test deprem", Link: "https://example-b.com/c", Source: "B", Published: now, HasDate: true},
	}
	c.Count = len(c.items)
	finalizeEvent(c)
	if c.IndependentSources != 2 {
		t.Fatalf("independent=%d", c.IndependentSources)
	}
	if c.Confidence < 55 {
		t.Fatalf("confidence too low: %d", c.Confidence)
	}
	if c.EventID == "" {
		t.Fatal("missing event id")
	}
}

func TestEventLifecycleDeveloping(t *testing.T) {
	now := time.Now()
	c := &Story{Title: "Breaking deprem", Category: "Türkiye", Published: now, HasDate: true, Breaking: true}
	c.items = []Item{
		{Title: "a", Link: "https://a.example/a", Published: now.Add(-30 * time.Minute), HasDate: true},
		{Title: "b", Link: "https://b.example/b", Published: now.Add(-15 * time.Minute), HasDate: true},
		{Title: "c", Link: "https://c.example/c", Published: now, HasDate: true},
	}
	c.Count = len(c.items)
	finalizeEvent(c)
	if c.Lifecycle != eventDeveloping {
		t.Fatalf("lifecycle=%q", c.Lifecycle)
	}
}
