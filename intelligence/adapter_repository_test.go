package intelligence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nexnews/core"
)

func fixedTime(day int) time.Time {
	return time.Date(2026, 10, day, 12, 0, 0, 0, time.UTC)
}

func sampleCoreSnapshot(day int) core.IntelligenceSnapshot {
	captured := fixedTime(day)
	published := fixedTime(day - 1)
	return core.IntelligenceSnapshot{
		CapturedAt: captured,
		FeedCount: 2,
		Articles: []core.IntelligenceArticleSnapshot{
			{
				ItemID: "feed-item-1",
				CanonicalURL: "https://news.example.org/story/42",
				Title: "The ministry announced a flood response",
				Summary: "The ministry announced a flood response on Tuesday.",
				SourceName: "Example News",
				FeedURL: "https://news.example.org/rss.xml",
				PublishedAt: &published,
				RetrievedAt: fixedTime(day - 1),
				Category: "World",
			},
			{
				ItemID: "same-url-other-category",
				CanonicalURL: "https://news.example.org/story/42",
				Title: "The ministry announced a flood response",
				SourceName: "Example News",
				FeedURL: "https://news.example.org/world.xml",
				RetrievedAt: fixedTime(day),
				Category: "Local",
			},
		},
		Events: []core.IntelligenceEventSnapshot{
			{
				CandidateID: "candidate-from-nexnews",
				Title: "The ministry announced a flood response",
				Category: "World",
				Lifecycle: "confirmed",
				ArticleURLs: []string{"https://news.example.org/story/42"},
				SourceNames: []string{"Example News"},
				IndependentSources: 1,
				ImportanceScore: 72,
				FirstPublishedAt: &published,
				LastPublishedAt: &published,
			},
		},
	}
}

func TestFromNexNewsDeduplicatesArticlesAndDoesNotInventAnalysis(t *testing.T) {
	batch, err := FromNexNews(sampleCoreSnapshot(10))
	if err != nil {
		t.Fatalf("FromNexNews returned error: %v", err)
	}
	if len(batch.Articles) != 1 {
		t.Fatalf("expected one canonical article, got %d", len(batch.Articles))
	}
	if len(batch.Sources) != 1 {
		t.Fatalf("expected one publisher domain, got %d", len(batch.Sources))
	}
	if len(batch.Events) != 1 {
		t.Fatalf("expected one event candidate, got %d", len(batch.Events))
	}
	if batch.Events[0].Status != EventNeedsReview {
		t.Fatalf("NexNews lifecycle 'confirmed' must not be treated as Intelligence verification; got %q", batch.Events[0].Status)
	}
	if len(batch.Events[0].ArticleIDs) != 1 || batch.Events[0].ArticleIDs[0] != batch.Articles[0].ID {
		t.Fatalf("event/article mapping mismatch: event=%#v article=%#v", batch.Events[0].ArticleIDs, batch.Articles[0].ID)
	}
	if stringsContain(batch.Events[0].CoverageLimitations, "not an independently verified") == false {
		t.Fatal("expected explicit coverage limitation for imported event candidate")
	}
	if batch.Articles[0].ContentAvailability != ContentTitleSummary {
		t.Fatalf("summary coverage should be preserved, got %q", batch.Articles[0].ContentAvailability)
	}
}

func stringsContain(items []string, part string) bool {
	for _, item := range items {
		if len(item) >= len(part) && contains(item, part) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestJSONRepositoryPersistsAndPreservesFirstObservation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence", "intelligence.json")
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	first := sampleCoreSnapshot(10)
	stats, err := repo.SyncFromNexNews(first)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}
	if stats.SourcesSaved != 1 || stats.ArticlesSaved != 1 || stats.EventsSaved != 1 {
		t.Fatalf("unexpected first sync stats: %+v", stats)
	}
	db1, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	articleID := stableID("article", "https://news.example.org/story/42")
	eventID := stableID("event_candidate", "candidate-from-nexnews")
	firstArticleObserved := db1.Articles[articleID].FirstObservedAt
	firstEventObserved := db1.Events[eventID].FirstObservedAt

	second := sampleCoreSnapshot(11)
	if _, err := repo.SyncFromNexNews(second); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}
	db2, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !db2.Articles[articleID].FirstObservedAt.Equal(firstArticleObserved) {
		t.Fatal("article firstObservedAt changed during update")
	}
	if !db2.Articles[articleID].LastObservedAt.Equal(fixedTime(11)) {
		t.Fatalf("article lastObservedAt not advanced: %v", db2.Articles[articleID].LastObservedAt)
	}
	if !db2.Events[eventID].FirstObservedAt.Equal(firstEventObserved) {
		t.Fatal("event firstObservedAt changed during update")
	}
	if !db2.Events[eventID].LastObservedAt.Equal(fixedTime(11)) {
		t.Fatalf("event lastObservedAt not advanced: %v", db2.Events[eventID].LastObservedAt)
	}
	if len(db2.Claims) != 0 || len(db2.Evidence) != 0 || len(db2.Relations) != 0 {
		t.Fatal("import adapter must not invent claims, evidence or relations")
	}
	if db2.LastSyncAt.IsZero() {
		t.Fatal("lastSyncAt must be persisted")
	}
}

func TestRepositoryRejectsUnsupportedArticleURLWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence.json")
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	input := sampleCoreSnapshot(10)
	input.Articles[0].CanonicalURL = "javascript:alert(1)"
	if _, err := repo.SyncFromNexNews(input); err == nil {
		t.Fatal("expected unsafe URL to be rejected")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("database must not be written after failed adaptation; stat err=%v", err)
	}
}

func TestRepositoryRejectsCorruptDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Load(); err == nil {
		t.Fatal("expected corrupt database to return an error")
	}
}

func TestSyncSnapshotJSONAcceptsBridgePayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence.json")
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(sampleCoreSnapshot(10))
	if err != nil {
		t.Fatal(err)
	}
	stats, err := repo.SyncSnapshotJSON(payload)
	if err != nil {
		t.Fatalf("valid snapshot payload failed: %v", err)
	}
	if stats.ArticlesSaved != 1 || stats.EventsSaved != 1 {
		t.Fatalf("unexpected sync stats: %+v", stats)
	}
}

func TestSyncSnapshotJSONRejectsBridgeErrorEnvelope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence.json")
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.SyncSnapshotJSON([]byte(`{"error":"çekirdek hazır değil","articles":[],"events":[]}`))
	if err == nil {
		t.Fatal("expected bridge error envelope to be rejected")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("database must not be written after bridge error; stat err=%v", statErr)
	}
}
