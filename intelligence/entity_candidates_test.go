package intelligence

import (
	"path/filepath"
	"testing"
	"time"

	"nexnews/core"
)

func TestEntityCandidatesKeepExactUnicodeSourceOffsets(t *testing.T) {
	published := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	a := Article{
		ID: "unicode-article",
		CanonicalURL: "https://news.example.org/turkiye-iran",
		PublisherSourceID: "source-example",
		Title: "Türkiye'nin İran ile görüşmesi bugün başladı",
		PublishedAt: &published,
		RetrievedAt: published,
		FirstObservedAt: published,
		LastObservedAt: published,
		ContentAvailability: ContentTitleOnly,
	}
	result, err := ExtractEntityCandidates(a, published)
	if err != nil {
		t.Fatalf("ExtractEntityCandidates failed: %v", err)
	}
	if len(result.Mentions) < 2 {
		t.Fatalf("expected Türkiye and İran mentions, got %+v", result.Mentions)
	}
	found := map[string]bool{}
	for _, mention := range result.Mentions {
		if mention.StartOffset < 0 || mention.EndOffset > len(a.Title) || mention.StartOffset >= mention.EndOffset {
			t.Fatalf("invalid UTF-8 byte offsets: %+v", mention)
		}
		if a.Title[mention.StartOffset:mention.EndOffset] != mention.Quote {
			t.Fatalf("mention offsets do not match original title: %+v title=%q", mention, a.Title)
		}
		found[mention.Canonical] = true
		if mention.Status != IdentityCandidate {
			t.Fatalf("lexicon hit must remain a candidate, got %q", mention.Status)
		}
	}
	if !found["Türkiye"] || !found["İran"] {
		t.Fatalf("expected both Türkiye and İran; found=%v", found)
	}
}

func TestEntityCandidateWordBoundariesAvoidSubstrings(t *testing.T) {
	published := time.Now().UTC()
	a := Article{
		ID: "boundary-article",
		CanonicalURL: "https://news.example.org/iranian",
		PublisherSourceID: "source-example",
		Title: "Iranian officials released the report on Tuesday",
		PublishedAt: &published,
		RetrievedAt: published,
		FirstObservedAt: published,
		LastObservedAt: published,
		ContentAvailability: ContentTitleOnly,
	}
	result, err := ExtractEntityCandidates(a, published)
	if err != nil {
		t.Fatal(err)
	}
	for _, mention := range result.Mentions {
		if mention.Canonical == "İran" {
			t.Fatalf("Iran alias must not match inside the word Iranian: %+v", mention)
		}
	}
}

func TestEntityExtractionPersistsMentionsWithoutCreatingRelations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence.json")
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	input := sampleCoreSnapshot(10)
	input.Articles[0].Title = "Türkiye'nin İran ile görüşmesi bugün başladı"
	input.Articles[0].Summary = "Türkiye'nin İran ile yaptığı görüşmede taraflar bölgesel gelişmeleri değerlendirdi."
	input.Articles[1].Title = input.Articles[0].Title
	input.Events[0].Title = input.Articles[0].Title
	if _, err := repo.SyncFromNexNews(input); err != nil {
		t.Fatalf("SyncFromNexNews failed: %v", err)
	}
	stats, err := repo.ExtractStoredEntityCandidates()
	if err != nil {
		t.Fatalf("ExtractStoredEntityCandidates failed: %v", err)
	}
	if stats.ArticlesScanned != 1 || stats.EntitiesCreated < 2 || stats.MentionsCreated < 2 {
		t.Fatalf("unexpected extraction stats: %+v", stats)
	}
	db, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(db.EntityMentions) < 2 {
		t.Fatalf("expected mention records to persist, got %d", len(db.EntityMentions))
	}
	if len(db.Relations) != 0 {
		t.Fatalf("entity co-occurrence must not create relations, got %d", len(db.Relations))
	}
	if len(db.Claims) != 0 || len(db.Evidence) != 0 {
		t.Fatal("entity extraction must not fabricate claims or claim evidence")
	}
	for _, mention := range db.EntityMentions {
		article, ok := db.Articles[mention.ArticleID]
		if !ok {
			t.Fatalf("mention references missing article %q", mention.ArticleID)
		}
		var sourceText string
		switch mention.Field {
		case "title":
			sourceText = article.Title
		case "summary":
			sourceText = article.Summary
		default:
			t.Fatalf("unexpected source field %q", mention.Field)
		}
		if sourceText[mention.StartOffset:mention.EndOffset] != mention.Quote {
			t.Fatalf("persisted mention is not source-verbatim: %+v", mention)
		}
	}
}

// This compile-time reference documents that the adapter consumes only the
// NexNews snapshot contract, rather than reaching into private Store state.
var _ core.IntelligenceSnapshot
