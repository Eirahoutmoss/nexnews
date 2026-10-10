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

func TestExtractStoredSourceStatementsPersistsEvidenceAndLinksUniqueEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence.json")
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SyncFromNexNews(sampleCoreSnapshot(10)); err != nil {
		t.Fatalf("initial NexNews sync failed: %v", err)
	}
	first, err := repo.ExtractStoredSourceStatements()
	if err != nil {
		t.Fatalf("source statement extraction failed: %v", err)
	}
	if first.ArticlesScanned != 1 || first.ClaimsCreated == 0 || first.EvidenceCreated == 0 {
		t.Fatalf("unexpected extraction stats: %+v", first)
	}
	db, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Claims) != first.ClaimsCreated {
		t.Fatalf("claims in database=%d but extraction reported=%d", len(db.Claims), first.ClaimsCreated)
	}
	if len(db.Evidence) != first.EvidenceCreated {
		t.Fatalf("evidence in database=%d but extraction reported=%d", len(db.Evidence), first.EvidenceCreated)
	}
	if len(db.Runs) != 1 {
		t.Fatalf("expected a persisted analysis run, got %d", len(db.Runs))
	}
	for _, claim := range db.Claims {
		if claim.Kind != ClaimSourceStatement || claim.Status != ClaimReported {
			t.Fatalf("source text should remain a reported statement candidate: %+v", claim)
		}
		if claim.EventID == "" {
			t.Fatalf("statement should attach to exactly one matching event candidate: %+v", claim)
		}
		if len(claim.EvidenceIDs) == 0 {
			t.Fatalf("claim lacks traceable evidence: %+v", claim)
		}
		for _, evidenceID := range claim.EvidenceIDs {
			evidence, ok := db.Evidence[evidenceID]
			if !ok || evidence.ClaimID != claim.ID {
				t.Fatalf("claim evidence link is broken: claim=%s evidence=%s", claim.ID, evidenceID)
			}
			if evidence.Stance != EvidenceReports {
				t.Fatalf("literal source statement must not be marked as factual support, got %q", evidence.Stance)
			}
		}
	}
	for _, event := range db.Events {
		if len(event.ClaimIDs) != len(db.Claims) {
			t.Fatalf("unique source article's event should link its statements: event claims=%d, stored claims=%d", len(event.ClaimIDs), len(db.Claims))
		}
	}

	second, err := repo.ExtractStoredSourceStatements()
	if err != nil {
		t.Fatalf("repeat extraction failed: %v", err)
	}
	if second.ClaimsCreated != 0 || second.EvidenceCreated != 0 {
		t.Fatalf("repeat extraction should be idempotent, got %+v", second)
	}
	db2, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(db2.Claims) != len(db.Claims) || len(db2.Evidence) != len(db.Evidence) || len(db2.Runs) != 2 {
		t.Fatalf("unexpected database growth after repeated extraction: claims %d->%d evidence %d->%d runs=%d", len(db.Claims), len(db2.Claims), len(db.Evidence), len(db2.Evidence), len(db2.Runs))
	}
	if len(db2.Relations) != 0 {
		t.Fatal("source statement extraction must not invent semantic relations")
	}
}

func TestExtractStoredSourceStatementsOnlyUsesPersistedArticleText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence.json")
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SyncFromNexNews(sampleCoreSnapshot(10)); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ExtractStoredSourceStatements(); err != nil {
		t.Fatal(err)
	}
	db, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, evidence := range db.Evidence {
		if evidence.ExcerptField != "title" && evidence.ExcerptField != "summary" {
			t.Fatalf("extractor invented or mislabeled a content field: %q", evidence.ExcerptField)
		}
		if evidence.ContentLevel == ContentFullText {
			t.Fatal("feed summary must not be represented as full article text")
		}
	}
}
