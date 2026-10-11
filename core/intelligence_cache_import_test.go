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


func TestIntelligenceSnapshotFromFeedCacheDeduplicatesSameArticleAcrossCategories(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	itemWorld := Item{
		ID: "world-copy",
		Title: "Regional officials announced a coordinated response",
		Link: "https://news.example.org/story/42?utm_source=world",
		Canonical: "https://news.example.org/story/42",
		Source: "Example News",
		Category: "World",
		Published: now.Add(-time.Hour),
		HasDate: true,
		Summary: "Regional officials announced a coordinated response after Tuesday's meeting.",
	}
	itemLocal := itemWorld
	itemLocal.ID = "local-copy"
	itemLocal.Link = "https://news.example.org/story/42?utm_medium=local"
	itemLocal.Category = "Local"

	cache := persistentCache{
		Version: cacheSchema,
		SavedAt: now,
		Feeds: []FeedCacheEntry{
			{URL: "https://news.example.org/world.xml", Category: "World", Name: "Example News", FetchedAt: now, Items: []Item{itemWorld}},
			{URL: "https://news.example.org/local.xml", Category: "Local", Name: "Example News", FetchedAt: now, Items: []Item{itemLocal}},
		},
	}
	payload, err := json.Marshal(cache)
	if err != nil { t.Fatal(err) }
	path := filepath.Join(t.TempDir(), "feed-cache.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil { t.Fatal(err) }

	snapshot, err := IntelligenceSnapshotFromFeedCacheFile(path)
	if err != nil { t.Fatalf("snapshot from cache: %v", err) }
	if len(snapshot.Articles) != 1 {
		t.Fatalf("same canonical URL should yield one article, got %d", len(snapshot.Articles))
	}
	if len(snapshot.Events) != 1 {
		t.Fatalf("same article split across feed categories should yield one candidate, got %d: %+v", len(snapshot.Events), snapshot.Events)
	}
	if len(snapshot.Events[0].ArticleURLs) != 1 || snapshot.Events[0].ArticleURLs[0] != "https://news.example.org/story/42" {
		t.Fatalf("unexpected event article membership: %+v", snapshot.Events[0].ArticleURLs)
	}
}
