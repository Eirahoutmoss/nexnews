package core

import (
	"testing"
	"time"
)

func TestIntelligenceSnapshotIsReadOnlyAndDeduplicatesCanonicalURLs(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	published := now.Add(-time.Hour)
	itemA := Item{
		ID: "item-a", Title: "Flood response announced", Link: "https://news.example.org/story/42?utm_source=feed",
		Canonical: "https://news.example.org/story/42", Source: "Example News", Category: "World",
		Published: published, HasDate: true, Summary: "Officials announced flood response measures.",
	}
	itemB := Item{
		ID: "item-b", Title: "Flood response announced", Link: "https://news.example.org/story/42",
		Canonical: "https://news.example.org/story/42", Source: "Example News", Category: "Local",
		Published: published, HasDate: true, Summary: "Officials announced flood response measures.",
	}
	story := &Story{
		Title: "Flood response announced", Link: itemA.Link, Category: "World",
		EventID: "candidate-event-42", Lifecycle: "detected", Score: 70,
		IndependentSources: 1, PrimarySources: 0, Breaking: true,
		Sources: []string{"Example News"}, items: []Item{itemA, itemB},
	}
	s := &Store{
		cache: map[string]FeedCacheEntry{
			"https://news.example.org/rss.xml": {
				URL: "https://news.example.org/rss.xml", Name: "Example News",
				FetchedAt: now, Items: []Item{itemA, itemB},
			},
		},
		important: []*Story{story},
	}

	snapshot := s.IntelligenceSnapshot()
	if snapshot.FeedCount != 1 {
		t.Fatalf("expected one cached feed, got %d", snapshot.FeedCount)
	}
	if len(snapshot.Articles) != 1 {
		t.Fatalf("expected duplicate canonical URLs to collapse to one article, got %d", len(snapshot.Articles))
	}
	if snapshot.Articles[0].CanonicalURL != "https://news.example.org/story/42" {
		t.Fatalf("canonical URL mismatch: %q", snapshot.Articles[0].CanonicalURL)
	}
	if len(snapshot.Events) != 1 || snapshot.Events[0].CandidateID != "candidate-event-42" {
		t.Fatalf("event candidate export mismatch: %+v", snapshot.Events)
	}
	if len(snapshot.Events[0].ArticleURLs) != 1 {
		t.Fatalf("event should carry one unique article URL, got %d", len(snapshot.Events[0].ArticleURLs))
	}
	if s.cache["https://news.example.org/rss.xml"].Items[0].Link != itemA.Link {
		t.Fatal("export mutated the underlying NexNews cache")
	}
	if story.Title != "Flood response announced" || story.EventID != "candidate-event-42" {
		t.Fatal("export mutated the underlying event candidate")
	}
	if snapshot.Articles[0].RetrievedAt.IsZero() {
		t.Fatal("expected retrieved-at timestamp from cached feed")
	}
}
