package intelligence

import (
	"path/filepath"
	"testing"
)

func TestAnalyzeStoredContentRunsOnlyConservativeLocalStages(t *testing.T) {
	repo, err := NewJSONRepository(filepath.Join(t.TempDir(), "intelligence.json"))
	if err != nil {
		t.Fatal(err)
	}
	input := sampleCoreSnapshot(10)
	title := "Türkiye ve İran görüşmelerin sürdüğünü açıkladı"
	summary := "Türkiye ve İran temsilcileri bölgesel gelişmeleri değerlendirmek üzere bugün bir araya geldi."
	input.Articles[0].Title = title
	input.Articles[0].Summary = summary
	input.Articles[1].Title = title
	input.Articles[1].Summary = summary
	input.Events[0].Title = title

	if _, err := repo.SyncFromNexNews(input); err != nil {
		t.Fatalf("import NexNews snapshot: %v", err)
	}
	stats, err := repo.AnalyzeStoredContent()
	if err != nil {
		t.Fatalf("AnalyzeStoredContent: %v", err)
	}
	if stats.CompletedAt == nil || stats.Partial {
		t.Fatalf("expected complete pipeline result, got %+v", stats)
	}
	if stats.Statements.ArticlesScanned != 1 || stats.Entities.ArticlesScanned != 1 {
		t.Fatalf("canonical article dedup should leave one article per stage: %+v", stats)
	}
	if stats.Statements.ClaimsCreated == 0 || stats.Statements.EvidenceCreated == 0 {
		t.Fatalf("expected literal source statements and evidence: %+v", stats.Statements)
	}
	if stats.Entities.MentionsCreated < 2 {
		t.Fatalf("expected source-grounded Türkiye/İran candidate mentions: %+v", stats.Entities)
	}

	db, err := repo.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Relations) != 0 {
		t.Fatalf("co-occurring entities must not create relations, got %d", len(db.Relations))
	}
	for _, claim := range db.Claims {
		if claim.Status != ClaimReported {
			t.Fatalf("extractor must not upgrade source statements to verified claims: %+v", claim)
		}
		for _, evidenceID := range claim.EvidenceIDs {
			evidence, ok := db.Evidence[evidenceID]
			if !ok {
				t.Fatalf("claim has missing evidence %q", evidenceID)
			}
			if evidence.Stance != EvidenceReports && evidence.Stance != EvidenceSourceText {
				t.Fatalf("literal extraction must not present source wording as truth-support: %+v", evidence)
			}
		}
	}
}
