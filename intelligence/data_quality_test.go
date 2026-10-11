package intelligence

import (
	"strings"
	"testing"
)

func TestBuildDataQualityReportFindsBrokenReferencesAndOffsets(t *testing.T) {
	db := Database{
		Sources: map[string]Source{
			"s1": {ID: "s1", Name: "Example", CanonicalDomain: "example.org"},
		},
		Articles: map[string]Article{
			"a1": {ID: "a1", CanonicalURL: "https://example.org/news", PublisherSourceID: "s1", Title: "Başlık: Türkiye görüşme yaptı", ContentAvailability: ContentTitleOnly},
			"a2": {ID: "a2", CanonicalURL: "not-a-url", PublisherSourceID: "missing-source", Title: "", ContentAvailability: ContentTitleOnly},
		},
		Events: map[string]Event{
			"event1": {ID: "event1", Title: "Candidate", Status: EventNeedsReview, ArticleIDs: []string{"missing-article"}},
		},
		Claims: map[string]Claim{
			"claim1": {ID: "claim1", Kind: ClaimSourceStatement, ClaimText: "Reported text", EventID: "missing-event", EvidenceIDs: []string{"missing-evidence"}},
			"claim2": {ID: "claim2", Kind: ClaimSourceStatement, ClaimText: "No evidence", EvidenceIDs: []string{}},
		},
		Evidence: map[string]Evidence{
			"evidence1": {ID: "evidence1", ArticleID: "a1", ClaimID: "claim1", Quote: "Türkiye", ExcerptField: "title", StartOffset: intPtr(0), EndOffset: intPtr(5), SourceURL: "https://example.org/news"},
			"evidence2": {ID: "evidence2", ArticleID: "gone", Quote: "", SourceURL: "bad-url", StartOffset: intPtr(1)},
		},
		Entities: map[string]Entity{},
		EntityMentions: map[string]EntityMention{
			"mention1": {ID: "mention1", EntityID: "missing-entity", ArticleID: "gone", Field: "title", Quote: "x", StartOffset: 0, EndOffset: 1},
		},
		Relations: map[string]Relation{
			"relation1": {ID: "relation1", Subject: NodeRef{ID: "missing-event", Type: NodeEvent}, Object: NodeRef{ID: "missing-entity", Type: NodeEntity}, Status: RelationSupported},
		},
		Timeline: map[string]TimelineEntry{
			"timeline1": {ID: "timeline1", EventID: "gone", ArticleID: "gone", EvidenceIDs: []string{"gone"}},
		},
	}

	report := BuildDataQualityReport(db)
	if report.ErrorCount < 10 {
		t.Fatalf("expected multiple structural integrity errors, got %d: %+v", report.ErrorCount, report.Issues)
	}
	if report.WarningCount < 1 {
		t.Fatalf("expected warnings for records lacking evidence, got %+v", report)
	}
	if report.ArticlesWithoutPublishedAt != 2 || report.TitleOnlyArticles != 2 || report.EventsNeedingReview != 1 {
		t.Fatalf("unexpected coverage metrics: %+v", report)
	}
	codes := map[string]bool{}
	for _, issue := range report.Issues {
		codes[issue.Code] = true
	}
	for _, code := range []string{
		"article_source_unresolved", "article_url_invalid", "event_article_unresolved",
		"claim_evidence_unresolved", "evidence_offset_mismatch", "evidence_article_unresolved",
		"mention_entity_unresolved", "relation_without_evidence", "relation_node_unresolved",
		"timeline_event_unresolved", "timeline_evidence_unresolved",
	} {
		if !codes[code] {
			t.Errorf("expected quality finding %q; got codes: %+v", code, codes)
		}
	}
}

func TestBuildDataQualityReportIsStableAndDoesNotChangeDatabase(t *testing.T) {
	db := Database{
		Sources: map[string]Source{"s1": {ID: "s1", Name: "Example"}},
		Articles: map[string]Article{
			"a1": {ID: "a1", CanonicalURL: "https://example.org/news", PublisherSourceID: "s1", Title: "A", ContentAvailability: ContentTitleOnly},
		},
		Events: map[string]Event{},
		Claims: map[string]Claim{},
		Evidence: map[string]Evidence{},
		Entities: map[string]Entity{},
		EntityMentions: map[string]EntityMention{},
		Relations: map[string]Relation{},
		Timeline: map[string]TimelineEntry{},
	}
	before := len(db.Articles) + len(db.Sources) + len(db.Events)
	a := BuildDataQualityReport(db)
	b := BuildDataQualityReport(db)
	if len(a.Issues) == 0 || len(a.Issues) != len(b.Issues) {
		t.Fatalf("expected stable findings, first=%+v second=%+v", a, b)
	}
	for i := range a.Issues {
		if a.Issues[i].ID != b.Issues[i].ID || a.Issues[i].Code != b.Issues[i].Code {
			t.Fatal("issue IDs and ordering must be deterministic")
		}
	}
	if before != len(db.Articles)+len(db.Sources)+len(db.Events) {
		t.Fatal("report must not mutate database records")
	}
}

func intPtr(v int) *int { return &v }

func TestDataQualityIssueMessagesRemainNonJudgmental(t *testing.T) {
	db := Database{Sources: map[string]Source{}, Articles: map[string]Article{}, Events: map[string]Event{}, Claims: map[string]Claim{}, Evidence: map[string]Evidence{}, Entities: map[string]Entity{}, EntityMentions: map[string]EntityMention{}, Relations: map[string]Relation{}, Timeline: map[string]TimelineEntry{}}
	report := BuildDataQualityReport(db)
	for _, issue := range report.Issues {
		if strings.Contains(strings.ToLower(issue.Message), "true") || strings.Contains(strings.ToLower(issue.Message), "false") {
			t.Fatalf("integrity report should not issue truth verdicts: %+v", issue)
		}
	}
}
