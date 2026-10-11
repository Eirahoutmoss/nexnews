package intelligence

import (
	"testing"
	"time"
)

func TestBuildSourceComparisonGroupsOnlyExactWordingAcrossArticles(t *testing.T) {
	published := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	db := Database{
		Sources: map[string]Source{
			"s1": {ID:"s1", Name:"Haber A", CanonicalDomain:"a.example"},
			"s2": {ID:"s2", Name:"Haber B", CanonicalDomain:"b.example"},
			"s3": {ID:"s3", Name:"Haber C", CanonicalDomain:"c.example"},
		},
		Articles: map[string]Article{
			"a1": {ID:"a1", CanonicalURL:"https://a.example/1", PublisherSourceID:"s1", Title:"Başlık A", PublishedAt:&published},
			"a2": {ID:"a2", CanonicalURL:"https://b.example/2", PublisherSourceID:"s2", Title:"Başlık B", PublishedAt:&published},
			"a3": {ID:"a3", CanonicalURL:"https://c.example/3", PublisherSourceID:"s3", Title:"Başlık C", PublishedAt:&published},
			"a4": {ID:"a4", CanonicalURL:"https://c.example/4", PublisherSourceID:"s3", Title:"Başka anlatım", PublishedAt:&published},
		},
		Claims: map[string]Claim{
			"c1": {ID:"c1", Kind:ClaimSourceStatement, ClaimText:"Bakanlık yeni programı bugün açıkladı.", Status:ClaimReported, EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2", Kind:ClaimSourceStatement, ClaimText:"BAKANLIK yeni programı bugün açıkladı", Status:ClaimReported, EvidenceIDs:[]string{"e2","e2-title"}},
			"c3": {ID:"c3", Kind:ClaimSourceStatement, ClaimText:"Bakanlık yeni programı bugün açıkladı!", Status:ClaimReported, EvidenceIDs:[]string{"e3"}},
			"c4": {ID:"c4", Kind:ClaimSourceStatement, ClaimText:"Bakanlık programın bugün yürürlüğe girdiğini belirtti.", Status:ClaimReported, EvidenceIDs:[]string{"e4"}},
		},
		Evidence: map[string]Evidence{
			"e1": {ID:"e1", ClaimID:"c1", ArticleID:"a1", Quote:"Bakanlık yeni programı bugün açıkladı.", ExcerptField:"summary", SourceURL:"https://a.example/1", Stance:EvidenceReports},
			"e2": {ID:"e2", ClaimID:"c2", ArticleID:"a2", Quote:"BAKANLIK yeni programı bugün açıkladı", ExcerptField:"title", SourceURL:"https://b.example/2", Stance:EvidenceReports},
			"e2-title": {ID:"e2-title", ClaimID:"c2", ArticleID:"a2", Quote:"BAKANLIK yeni programı bugün açıkladı!", ExcerptField:"summary", SourceURL:"https://b.example/2", Stance:EvidenceReports},
			"e3": {ID:"e3", ClaimID:"c3", ArticleID:"a3", Quote:"Bakanlık yeni programı bugün açıkladı!", ExcerptField:"summary", SourceURL:"https://c.example/3", Stance:EvidenceReports},
			"e4": {ID:"e4", ClaimID:"c4", ArticleID:"a4", Quote:"Bakanlık programın bugün yürürlüğe girdiğini belirtti.", ExcerptField:"summary", SourceURL:"https://c.example/4", Stance:EvidenceReports},
		},
	}
	report := BuildSourceComparisonReport(db)
	if report.SourceStatementsReviewed != 4 {
		t.Fatalf("expected four statements to be reviewed, got %d", report.SourceStatementsReviewed)
	}
	if report.UniqueStatementCount != 2 {
		t.Fatalf("expected exact sentence normalization to yield two groups, got %d", report.UniqueStatementCount)
	}
	if report.RepeatedWordingGroups != 1 || len(report.Groups) != 1 {
		t.Fatalf("expected one repeated wording group, got groups=%d len=%d", report.RepeatedWordingGroups, len(report.Groups))
	}
	group := report.Groups[0]
	if group.ArticleCount != 3 {
		t.Fatalf("same article must only count once even if title and summary repeat it; got %d articles", group.ArticleCount)
	}
	if group.PublisherDomainCount != 3 {
		t.Fatalf("expected 3 publisher domains, got %d", group.PublisherDomainCount)
	}
	if group.Assessment == "" || len(group.Limitations) == 0 {
		t.Fatal("comparison must carry assessment and explicit limitations")
	}
	if got := len(group.Occurrences); got != 3 {
		t.Fatalf("expected one occurrence per distinct article, got %d", got)
	}
}

func TestBuildSourceComparisonDoesNotMergeParaphrasesOrInferContradiction(t *testing.T) {
	db := Database{
		Sources: map[string]Source{
			"s1": {ID:"s1", Name:"Outlet A", CanonicalDomain:"a.example"},
			"s2": {ID:"s2", Name:"Outlet B", CanonicalDomain:"b.example"},
		},
		Articles: map[string]Article{
			"a1": {ID:"a1", CanonicalURL:"https://a.example/1", PublisherSourceID:"s1", Title:"Headline 1"},
			"a2": {ID:"a2", CanonicalURL:"https://b.example/2", PublisherSourceID:"s2", Title:"Headline 2"},
		},
		Claims: map[string]Claim{
			"c1": {ID:"c1", Kind:ClaimSourceStatement, ClaimText:"Bakanlık programı bugün açıkladı.", Status:ClaimReported, EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2", Kind:ClaimSourceStatement, ClaimText:"Yeni program bugün kamuoyuna duyuruldu.", Status:ClaimReported, EvidenceIDs:[]string{"e2"}},
		},
		Evidence: map[string]Evidence{
			"e1": {ID:"e1", ClaimID:"c1", ArticleID:"a1", Quote:"Bakanlık programı bugün açıkladı.", ExcerptField:"summary", SourceURL:"https://a.example/1", Stance:EvidenceReports},
			"e2": {ID:"e2", ClaimID:"c2", ArticleID:"a2", Quote:"Yeni program bugün kamuoyuna duyuruldu.", ExcerptField:"summary", SourceURL:"https://b.example/2", Stance:EvidenceReports},
		},
	}
	report := BuildSourceComparisonReport(db)
	if report.RepeatedWordingGroups != 0 || len(report.Groups) != 0 {
		t.Fatalf("paraphrases must not be grouped without semantic analysis: %+v", report.Groups)
	}
	if len(report.Limitations) == 0 {
		t.Fatal("report must clearly state limitations")
	}
}

func TestComparisonKeyNormalizesTurkishCaseAndPunctuationOnly(t *testing.T) {
	a := comparisonKey("İran, yeni programı AÇIKLADI!")
	b := comparisonKey("İRAN yeni PROGRAMI açıkladı")
	if a != b {
		t.Fatalf("expected punctuation and Turkish case normalization, got %q vs %q", a, b)
	}
	if comparisonKey("İran programı reddetti") == comparisonKey("İran programı kabul etti") {
		t.Fatal("different wordings must remain separate")
	}
}
