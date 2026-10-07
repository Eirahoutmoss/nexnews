package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCanonicalURL(t *testing.T) {
	got := canonicalURL("HTTPS://Example.COM:443/news/?utm_source=rss&b=2&fbclid=x&a=1#top")
	want := "https://example.com/news?a=1&b=2"
	if got != want {
		t.Fatalf("canonicalURL() = %q, want %q", got, want)
	}
}

func TestArticleIDStable(t *testing.T) {
	a := articleID("A headline", "https://example.com/news")
	b := articleID("A headline", "https://example.com/news")
	c := articleID("Different", "https://example.com/news")
	if a == "" || a != b || a == c {
		t.Fatalf("unexpected IDs: %q %q %q", a, b, c)
	}
}

func TestDiskRepositoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	r := &diskRepository{path: filepath.Join(dir, "cache.json"), feeds: map[string]FeedCacheEntry{}, health: map[string]SourceHealth{}}
	now := time.Now().UTC().Truncate(time.Second)
	r.putFeed(FeedCacheEntry{URL: "https://example.com/feed", Category: "Test", Name: "Example", FetchedAt: now, Items: []Item{{ID: "1", Title: "Hello", Link: "https://example.com/1"}}})
	r.putHealth(SourceHealth{URL: "https://example.com/feed", Name: "Example", Status: "healthy", LastSuccess: now, ItemCount: 1})
	if err := r.persist(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.path); err != nil {
		t.Fatal(err)
	}

	r2 := &diskRepository{path: r.path, feeds: map[string]FeedCacheEntry{}, health: map[string]SourceHealth{}}
	r2.load()
	feeds, health := r2.snapshot()
	if len(feeds) != 1 || len(health) != 1 {
		t.Fatalf("round trip lost data: feeds=%d health=%d", len(feeds), len(health))
	}
	if feeds["https://example.com/feed"].Items[0].Title != "Hello" {
		t.Fatal("cached item missing")
	}

	b, err := json.Marshal(feeds["https://example.com/feed"])
	if err != nil || len(b) == 0 {
		t.Fatal("cache entry is not serializable")
	}
}
