package intelligence

import (
	"strings"
	"testing"
	"time"
)

func TestClaimAssessmentFindsExplicitOppositeAssertions(t *testing.T) {
	t1 := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	t2 := t1.Add(90 * time.Minute)
	db := Database{
		Sources: map[string]Source{
			"s1": {ID:"s1",Name:"Kaynak A",CanonicalDomain:"a.example"},
			"s2": {ID:"s2",Name:"Kaynak B",CanonicalDomain:"b.example"},
			"s3": {ID:"s3",Name:"Kaynak C",CanonicalDomain:"c.example"},
		},
		Articles: map[string]Article{
			"a1": {ID:"a1",CanonicalURL:"https://a.example/1",PublisherSourceID:"s1",Title:"Toplantı anlaşması kabul etti",PublishedAt:&t1},
			"a2": {ID:"a2",CanonicalURL:"https://b.example/2",PublisherSourceID:"s2",Title:"Toplantı anlaşması kabul etmedi",PublishedAt:&t2},
			"a3": {ID:"a3",CanonicalURL:"https://c.example/3",PublisherSourceID:"s3",Title:"Toplantı anlaşması kabul etti",PublishedAt:&t2},
		},
		Claims: map[string]Claim{
			"c1": {ID:"c1",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etti.",Status:ClaimReported,EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etmedi.",Status:ClaimReported,EvidenceIDs:[]string{"e2"}},
			"c3": {ID:"c3",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etti!",Status:ClaimReported,EvidenceIDs:[]string{"e3"}},
		},
		Evidence: map[string]Evidence{
			"e1": {ID:"e1",ClaimID:"c1",ArticleID:"a1",Stance:EvidenceReports,ExcerptField:"title",Quote:"Kabine anlaşmayı kabul etti.",SourceURL:"https://a.example/1"},
			"e2": {ID:"e2",ClaimID:"c2",ArticleID:"a2",Stance:EvidenceReports,ExcerptField:"title",Quote:"Kabine anlaşmayı kabul etmedi.",SourceURL:"https://b.example/2"},
			"e3": {ID:"e3",ClaimID:"c3",ArticleID:"a3",Stance:EvidenceReports,ExcerptField:"title",Quote:"Kabine anlaşmayı kabul etti!",SourceURL:"https://c.example/3"},
		},
	}
	report := BuildClaimAssessmentReport(db)
	if report.ExplicitOppositionCases != 1 || len(report.Cases) != 1 {
		t.Fatalf("expected one explicit opposition case, got %+v", report)
	}
	c := report.Cases[0]
	if len(c.PositiveSources) != 2 || len(c.NegativeSources) != 1 {
		t.Fatalf("expected two positive and one negative source statement, got +%d -%d",len(c.PositiveSources),len(c.NegativeSources))
	}
	if c.ArticleCount != 3 || c.DomainCount != 3 {
		t.Fatalf("unexpected source aggregation: %+v", c)
	}
	if c.Assessment == "" || len(c.Limitations) == 0 {
		t.Fatal("possible conflict must include review assessment and limitations")
	}
}

func TestClaimAssessmentDoesNotMergeDifferentContextsOrNegationWordsInMiddle(t *testing.T) {
	db := Database{
		Sources: map[string]Source{"s1": {ID:"s1",CanonicalDomain:"a.example"}, "s2": {ID:"s2",CanonicalDomain:"b.example"}},
		Articles: map[string]Article{
			"a1": {ID:"a1",CanonicalURL:"https://a.example/1",PublisherSourceID:"s1",Title:"Toplantı yapıldı"},
			"a2": {ID:"a2",CanonicalURL:"https://b.example/2",PublisherSourceID:"s2",Title:"Başka toplantı yapılmadı"},
			"a3": {ID:"a3",CanonicalURL:"https://b.example/3",PublisherSourceID:"s2",Title:"Yetkililer toplantının gerçekleşmediğini söyledi"},
		},
		Claims: map[string]Claim{
			"c1": {ID:"c1",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etti.",Status:ClaimReported,EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2",Kind:ClaimSourceStatement,ClaimText:"Başka anlaşma kabul etmedi.",Status:ClaimReported,EvidenceIDs:[]string{"e2"}},
			"c3": {ID:"c3",Kind:ClaimSourceStatement,ClaimText:"Yetkililer gerçekleşmediğini söyledi.",Status:ClaimReported,EvidenceIDs:[]string{"e3"}},
		},
		Evidence: map[string]Evidence{
			"e1": {ID:"e1",ClaimID:"c1",ArticleID:"a1",Stance:EvidenceReports,ExcerptField:"summary",Quote:"Kabine anlaşmayı kabul etti.",SourceURL:"https://a.example/1"},
			"e2": {ID:"e2",ClaimID:"c2",ArticleID:"a2",Stance:EvidenceReports,ExcerptField:"summary",Quote:"Başka anlaşma kabul etmedi.",SourceURL:"https://b.example/2"},
			"e3": {ID:"e3",ClaimID:"c3",ArticleID:"a3",Stance:EvidenceReports,ExcerptField:"summary",Quote:"Yetkililer toplantının gerçekleşmediğini söyledi.",SourceURL:"https://b.example/3"},
		},
	}
	report := BuildClaimAssessmentReport(db)
	if len(report.Cases) != 0 {
		t.Fatalf("different contexts or negation in a non-terminal position must not be called conflict: %+v", report.Cases)
	}
}

func TestClaimAssessmentSeparatesAssertionsFarApartInTime(t *testing.T) {
	t1 := time.Date(2021, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	db := Database{
		Sources: map[string]Source{"s1": {ID:"s1",CanonicalDomain:"a.example"}, "s2": {ID:"s2",CanonicalDomain:"b.example"}},
		Articles: map[string]Article{
			"a1": {ID:"a1",CanonicalURL:"https://a.example/1",PublisherSourceID:"s1",PublishedAt:&t1},
			"a2": {ID:"a2",CanonicalURL:"https://b.example/2",PublisherSourceID:"s2",PublishedAt:&t2},
		},
		Claims: map[string]Claim{
			"c1": {ID:"c1",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etti.",Status:ClaimReported,EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etmedi.",Status:ClaimReported,EvidenceIDs:[]string{"e2"}},
		},
		Evidence: map[string]Evidence{
			"e1": {ID:"e1",ClaimID:"c1",ArticleID:"a1",Stance:EvidenceReports,ExcerptField:"summary",Quote:"Kabine anlaşmayı kabul etti.",SourceURL:"https://a.example/1"},
			"e2": {ID:"e2",ClaimID:"c2",ArticleID:"a2",Stance:EvidenceReports,ExcerptField:"summary",Quote:"Kabine anlaşmayı kabul etmedi.",SourceURL:"https://b.example/2"},
		},
	}
	report := BuildClaimAssessmentReport(db)
	if len(report.Cases) != 0 {
		t.Fatalf("assertions years apart must not be paired by this pass: %+v", report.Cases)
	}
}

func TestClaimAssessmentKeepsMissingDateCasesButMarksLimitation(t *testing.T) {
	db := Database{
		Sources: map[string]Source{"s1": {ID:"s1",CanonicalDomain:"a.example"}, "s2": {ID:"s2",CanonicalDomain:"b.example"}},
		Articles: map[string]Article{
			"a1": {ID:"a1",CanonicalURL:"https://a.example/1",PublisherSourceID:"s1"},
			"a2": {ID:"a2",CanonicalURL:"https://b.example/2",PublisherSourceID:"s2"},
		},
		Claims: map[string]Claim{
			"c1": {ID:"c1",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etti.",Status:ClaimReported,EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etmedi.",Status:ClaimReported,EvidenceIDs:[]string{"e2"}},
		},
		Evidence: map[string]Evidence{
			"e1": {ID:"e1",ClaimID:"c1",ArticleID:"a1",Stance:EvidenceReports,ExcerptField:"summary",Quote:"Kabine anlaşmayı kabul etti.",SourceURL:"https://a.example/1"},
			"e2": {ID:"e2",ClaimID:"c2",ArticleID:"a2",Stance:EvidenceReports,ExcerptField:"summary",Quote:"Kabine anlaşmayı kabul etmedi.",SourceURL:"https://b.example/2"},
		},
	}
	report := BuildClaimAssessmentReport(db)
	if len(report.Cases) != 1 {
		t.Fatalf("expected possible conflict with missing dates, got %+v",report)
	}
	found := false
	for _, limit := range report.Cases[0].Limitations {
		if strings.Contains(limit,"timestamp is missing") { found = true }
	}
	if !found { t.Fatal("missing publication time must be disclosed") }
}
