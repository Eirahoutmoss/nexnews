package intelligence

import (
	"path/filepath"
	"testing"
	"time"
)

func TestSyncResearchCaseFromNexNewsImportsSourceAndPreservesRevisions(t *testing.T) {
	repo, err := NewJSONRepository(filepath.Join(t.TempDir(), "intelligence.json"))
	if err != nil { t.Fatal(err) }
	published := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	now := published.Add(time.Hour)
	batch := ImportSnapshot{
		SchemaVersion: SchemaVersion, CapturedAt: now,
		Sources: []Source{{ID:"source-1",Name:"Local Reporter",CanonicalDomain:"local.example",Type:SourcePublisher,IdentityBasis:"test fixture",CreatedAt:now}},
		Articles: []Article{{
			ID:"article-1",CanonicalURL:"https://local.example/story",PublisherSourceID:"source-1",
			Title:"Breaking: facility targeted",Summary:"The group claimed that the facility was targeted.",
			PublishedAt:&published,RetrievedAt:now,FirstObservedAt:now,LastObservedAt:now,
			ContentAvailability:ContentTitleSummary,SyndicationOriginID:"agency-copy-7",
		}},
		Events: []Event{{ID:"event-1",Title:"Energy facility event",Status:EventNeedsReview,FirstObservedAt:now,LastObservedAt:now,ArticleIDs:[]string{"article-1"}}},
	}
	if _, err := repo.Sync(batch); err != nil { t.Fatal(err) }
	caseItem, err := repo.CreateResearchCase(CreateResearchCaseInput{
		Title:"Target selection review",Question:"What evidence supports each target-selection hypothesis?",RelatedEventID:"event-1",
	})
	if err != nil { t.Fatal(err) }

	first, stats, err := repo.SyncResearchCaseFromNexNews(caseItem.ID)
	if err != nil { t.Fatal(err) }
	if stats.EvidenceImported != 1 || stats.ArticlesSeen != 1 || len(first.Evidence) != 1 {
		t.Fatalf("unexpected initial import: stats=%+v evidence=%+v", stats, first.Evidence)
	}
	ev := first.Evidence[0]
	if ev.NexNewsArticleID != "article-1" || ev.ContentLevel != ResearchContentTitleSummary || ev.Excerpt != "The group claimed that the facility was targeted." {
		t.Fatalf("import lost source scope or metadata: %+v", ev)
	}
	if ev.SourceOrigin != ResearchOriginSyndicated || ev.SyndicationGroup != "agency-copy-7" {
		t.Fatalf("expected known syndication origin to survive import, got %+v", ev)
	}
	if ev.Kind != ResearchEvidenceArticle || ev.SourceKind != ResearchSourceOther {
		t.Fatalf("news report must be retained as a source article without pretending it is a verified fact: %+v", ev)
	}

	repeated, secondStats, err := repo.SyncResearchCaseFromNexNews(caseItem.ID)
	if err != nil { t.Fatal(err) }
	if secondStats.UnchangedSkipped != 1 || secondStats.EvidenceImported != 0 || len(repeated.Evidence) != 1 {
		t.Fatalf("identical import should be idempotent: stats=%+v evidence=%d", secondStats, len(repeated.Evidence))
	}

	batch.CapturedAt = now.Add(time.Hour)
	batch.Articles[0].Summary = "Updated report: the group repeated its claim, with new context."
	batch.Articles[0].LastObservedAt = batch.CapturedAt
	if _, err := repo.Sync(batch); err != nil { t.Fatal(err) }
	revised, revisionStats, err := repo.SyncResearchCaseFromNexNews(caseItem.ID)
	if err != nil { t.Fatal(err) }
	if revisionStats.RevisionsAdded != 1 || len(revised.Evidence) != 2 {
		t.Fatalf("a changed source report must be added as a new revision: stats=%+v evidence=%d", revisionStats, len(revised.Evidence))
	}
	if revised.Evidence[0].Excerpt == revised.Evidence[1].Excerpt {
		t.Fatal("expected both original and changed source text to be retained")
	}
	if revised.Evidence[0].ContentFingerprint == revised.Evidence[1].ContentFingerprint {
		t.Fatal("expected changed revision to have a different content fingerprint")
	}
}

func TestSyncResearchCaseFromNexNewsRequiresLinkedEvent(t *testing.T) {
	repo, err := NewJSONRepository(filepath.Join(t.TempDir(), "intelligence.json"))
	if err != nil { t.Fatal(err) }
	item, err := repo.CreateResearchCase(CreateResearchCaseInput{Title:"Standalone",Question:"A standalone research question"})
	if err != nil { t.Fatal(err) }
	if _, _, err := repo.SyncResearchCaseFromNexNews(item.ID); err == nil {
		t.Fatal("expected importing NexNews sources to require a linked event")
	}
}
