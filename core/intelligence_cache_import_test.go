package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIntelligenceSnapshotFromFeedCacheFile(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	cache := persistentCache{
		Version: cacheSchema,
		SavedAt: now,
		Feeds: []FeedCacheEntry{
			{
				URL: "https://news.example.org/rss.xml",
				Category: "World",
				Name: "Example News",
				FetchedAt: now,
				Items: []Item{
					{
						ID: "fixture-1",
						Title: "Türkiye ve İran temsilcileri görüşmeleri sürdürdü",
						Link: "https://news.example.org/turkiye-iran?utm_source=rss",
						Canonical: "https://news.example.org/turkiye-iran",
						Source: "Example News",
						Category: "World",
						Published: now.Add(-time.Hour),
						HasDate: true,
						Summary: "Türkiye ve İran temsilcileri bugün bölgesel gelişmeleri değerlendirmek üzere bir araya geldi.",
					},
				},
			},
		},
	}
	payload, err := json.Marshal(cache)
	if err != nil { t.Fatal(err) }
	path := filepath.Join(t.TempDir(), "feed-cache.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil { t.Fatal(err) }

	snapshot, err := IntelligenceSnapshotFromFeedCacheFile(path)
	if err != nil { t.Fatalf("snapshot from cache: %v", err) }
	if snapshot.FeedCount != 1 || len(snapshot.Articles) != 1 {
		t.Fatalf("unexpected snapshot counts: feeds=%d articles=%d", snapshot.FeedCount, len(snapshot.Articles))
	}
	if snapshot.Articles[0].CanonicalURL != "https://news.example.org/turkiye-iran" {
		t.Fatalf("unexpected canonical URL: %q", snapshot.Articles[0].CanonicalURL)
	}
	if snapshot.Articles[0].Summary == "" || snapshot.Articles[0].SourceName != "Example News" {
		t.Fatalf("feed metadata was not preserved: %+v", snapshot.Articles[0])
	}
	if snapshot.CapturedAt.IsZero() {
		t.Fatal("expected cache save time to be used as snapshot capture time")
	}
	if len(snapshot.Events) == 0 {
		t.Fatal("expected event candidates to be clustered from cached feed items")
	}
}

func TestIntelligenceSnapshotFromFeedCacheRejectsUnknownSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feed-cache.json")
	if err := os.WriteFile(path, []byte(`{"version":99,"feeds":[],"health":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := IntelligenceSnapshotFromFeedCacheFile(path); err == nil {
		t.Fatal("expected unsupported feed-cache schema to be rejected")
	}
}
