package intelligence

import (
	"strings"
	"testing"
	"time"
)

func TestBuildEventDossierAssemblesScopedEvidence(t *testing.T) {
	t1 := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Hour)
	db := Database{
		Schema: SchemaVersion,
		Sources: map[string]Source{
			"s1": {ID:"s1", Name:"Kaynak A", CanonicalDomain:"a.example"},
			"s2": {ID:"s2", Name:"Kaynak B", CanonicalDomain:"b.example"},
			"s3": {ID:"s3", Name:"Alakasız Kaynak", CanonicalDomain:"elsewhere.example"},
		},
		Articles: map[string]Article{
			"a1": {ID:"a1", CanonicalURL:"https://a.example/1", PublisherSourceID:"s1", Title:"Kabine anlaşmayı kabul etti", Summary:"Türkiye toplantı yaptı.", PublishedAt:&t1, ContentAvailability:ContentTitleSummary},
			"a2": {ID:"a2", CanonicalURL:"https://b.example/2", PublisherSourceID:"s2", Title:"Kabine anlaşmayı kabul etmedi", PublishedAt:&t2, ContentAvailability:ContentTitleOnly},
			"a3": {ID:"a3", CanonicalURL:"https://elsewhere.example/3", PublisherSourceID:"s3", Title:"Başka bir konuda haber", ContentAvailability:ContentTitleOnly},
		},
		Events: map[string]Event{
			"event1": {ID:"event1",Title:"Kabine anlaşması",Status:EventNeedsReview,ArticleIDs:[]string{"a1","a2"}},
			"event2": {ID:"event2",Title:"Başka konu",Status:EventEmerging,ArticleIDs:[]string{"a3"}},
		},
		Claims: map[string]Claim{
			"c1": {ID:"c1",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etti.",Status:ClaimReported,EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2",Kind:ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etmedi.",Status:ClaimReported,EvidenceIDs:[]string{"e2"}},
			"c3": {ID:"c3",Kind:ClaimSourceStatement,ClaimText:"Başka konuda ifade.",Status:ClaimReported,EvidenceIDs:[]string{"e3"}},
		},
		Evidence: map[string]Evidence{
			"e1": {ID:"e1",ClaimID:"c1",ArticleID:"a1",Stance:EvidenceReports,ExcerptField:"title",Quote:"Kabine anlaşmayı kabul etti",SourceURL:"https://a.example/1"},
			"e2": {ID:"e2",ClaimID:"c2",ArticleID:"a2",Stance:EvidenceReports,ExcerptField:"title",Quote:"Kabine anlaşmayı kabul etmedi",SourceURL:"https://b.example/2"},
			"e3": {ID:"e3",ClaimID:"c3",ArticleID:"a3",Stance:EvidenceReports,ExcerptField:"title",Quote:"Başka konuda ifade",SourceURL:"https://elsewhere.example/3"},
		},
		Entities: map[string]Entity{
			"ent1": {ID:"ent1",CanonicalName:"Türkiye",Type:EntityPlace,IdentityStatus:IdentityCandidate},
		},
		EntityMentions: map[string]EntityMention{
			"m1": {ID:"m1",EntityID:"ent1",Canonical:"Türkiye",EntityType:EntityPlace,ArticleID:"a1",Field:"summary",Quote:"Türkiye",StartOffset:0,EndOffset:7},
			"m2": {ID:"m2",EntityID:"ent1",Canonical:"Türkiye",EntityType:EntityPlace,ArticleID:"a3",Field:"title",Quote:"Türkiye",StartOffset:0,EndOffset:7},
		},
		Relations: map[string]Relation{}, Timeline: map[string]TimelineEntry{}, Runs: map[string]AnalysisRun{},
	}
	dossier, err := BuildEventDossier(db, "event1")
	if err != nil { t.Fatal(err) }
	if dossier.ArticleCount != 2 || dossier.PublisherDomainCount != 2 {
		t.Fatalf("wrong coverage totals: %+v", dossier)
	}
	if dossier.TitleOnlyArticles != 1 || dossier.ArticlesWithSummary != 1 {
		t.Fatalf("wrong content coverage: %+v", dossier)
	}
	if len(dossier.Statements) != 2 { t.Fatalf("expected two scoped source statements, got %d",len(dossier.Statements)) }
	if len(dossier.ConflictCandidates) != 1 { t.Fatalf("expected explicit opposition candidate from dossier articles, got %+v", dossier.ConflictCandidates) }
	if len(dossier.Entities) != 1 || dossier.Entities[0].ArticleCount != 1 {
		t.Fatalf("unrelated entity mention must not leak into dossier: %+v", dossier.Entities)
	}
	if dossier.Articles[0].ID != "a1" || dossier.Articles[1].ID != "a2" {
		t.Fatalf("articles must be in publication order: %+v",dossier.Articles)
	}
	if len(dossier.Limitations) == 0 { t.Fatal("dossier must communicate limitations") }
}

func TestBuildEventDossierRejectsUnknownEvent(t *testing.T) {
	_, err := BuildEventDossier(emptyDatabase(), "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") { t.Fatalf("expected unknown candidate error, got %v",err) }
}


func TestEventDossierMarksExactWordingRepeatsAndListsLatestFirst(t *testing.T) {
	t1 := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	t2 := t1.Add(3 * time.Hour)
	t3 := t1.Add(5 * time.Hour)
	db := Database{
		Schema: SchemaVersion,
		Sources: map[string]Source{
			"s1": {ID:"s1",Name:"Kaynak A",CanonicalDomain:"a.example"},
			"s2": {ID:"s2",Name:"Kaynak B",CanonicalDomain:"b.example"},
		},
		Articles: map[string]Article{
			"a1": {ID:"a1",CanonicalURL:"https://a.example/1",PublisherSourceID:"s1",Title:"İlk haber",PublishedAt:&t1,ContentAvailability:ContentTitleOnly},
			"a2": {ID:"a2",CanonicalURL:"https://b.example/2",PublisherSourceID:"s2",Title:"Son haber",PublishedAt:&t3,ContentAvailability:ContentTitleOnly},
			"a3": {ID:"a3",CanonicalURL:"https://a.example/3",PublisherSourceID:"s1",Title:"Ara haber",PublishedAt:&t2,ContentAvailability:ContentTitleOnly},
		},
		Events: map[string]Event{
			"event1": {ID:"event1",Title:"Test olayı",Status:EventDeveloping,ArticleIDs:[]string{"a1","a2","a3"}},
		},
		Claims: map[string]Claim{
			"c1": {ID:"c1",Kind:ClaimSourceStatement,ClaimText:"Kurul toplantıyı erteledi",Status:ClaimReported,EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2",Kind:ClaimSourceStatement,ClaimText:"Kurul toplantıyı erteledi",Status:ClaimReported,EvidenceIDs:[]string{"e2"}},
			"c3": {ID:"c3",Kind:ClaimSourceStatement,ClaimText:"Yeni karar açıklandı",Status:ClaimReported,EvidenceIDs:[]string{"e3"}},
		},
		Evidence: map[string]Evidence{
			"e1": {ID:"e1",ClaimID:"c1",ArticleID:"a1",Stance:EvidenceReports,ExcerptField:"title",Quote:"Kurul toplantıyı erteledi.",SourceURL:"https://a.example/1"},
			"e2": {ID:"e2",ClaimID:"c2",ArticleID:"a2",Stance:EvidenceReports,ExcerptField:"title",Quote:"Kurul toplantıyı erteledi!",SourceURL:"https://b.example/2"},
			"e3": {ID:"e3",ClaimID:"c3",ArticleID:"a3",Stance:EvidenceReports,ExcerptField:"title",Quote:"Yeni karar açıklandı",SourceURL:"https://a.example/3"},
		},
		Entities: map[string]Entity{}, EntityMentions: map[string]EntityMention{},
		Relations: map[string]Relation{}, Timeline: map[string]TimelineEntry{}, Runs: map[string]AnalysisRun{},
	}
	dossier, err := BuildEventDossier(db,"event1")
	if err != nil { t.Fatal(err) }
	if len(dossier.LatestArticles) != 3 || dossier.LatestArticles[0].ID != "a2" || dossier.LatestArticles[1].ID != "a3" || dossier.LatestArticles[2].ID != "a1" {
		t.Fatalf("latest articles must be newest-first: %+v",dossier.LatestArticles)
	}
	if dossier.StatementCount != 3 || dossier.DistinctWordingCount != 2 || dossier.ExactRepeatOccurrences != 1 {
		t.Fatalf("unexpected wording counts: total=%d distinct=%d repeats=%d",dossier.StatementCount,dossier.DistinctWordingCount,dossier.ExactRepeatOccurrences)
	}
	var foundRepeat bool
	for _, statement := range dossier.Statements {
		if statement.WordingStatus == "repeated_verbatim" {
			foundRepeat = true
			if statement.FirstOccurrence == nil || statement.FirstOccurrence.ArticleID != "a1" {
				t.Fatalf("repeat must link to the earliest occurrence: %+v",statement)
			}
		}
	}
	if !foundRepeat { t.Fatal("expected exact repeat to be marked") }
}
