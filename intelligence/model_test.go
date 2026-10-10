package intelligence

import (
	"strings"
	"testing"
	"time"
)

func testEvidence(stance EvidenceStance) Evidence {
	return Evidence{
		ID:                "evidence-1",
		ArticleID:         "article-1",
		Stance:            stance,
		ExcerptField:      "summary",
		Quote:             "The ministry announced the new policy on Tuesday.",
		SourceURL:         "https://example.org/article",
		CapturedAt:        time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC),
		ContentLevel:      ContentTitleSummary,
		ExtractionMethod:  "test-fixture",
		ExtractionVersion: "1",
	}
}

func testRelation() Relation {
	return Relation{
		ID:          "relation-1",
		Subject:     NodeRef{ID: "event-policy", Type: NodeEvent},
		Predicate:   RelationAnnouncedBy,
		Object:      NodeRef{ID: "entity-ministry", Type: NodeEntity},
		Status:      RelationSupported,
		EvidenceIDs: []string{"evidence-1"},
		Rationale:   "The quoted source explicitly attributes the announcement to the ministry.",
		CreatedBy:   "deterministic-test",
		RuleVersion: "1",
	}
}

func TestValidateArticleRejectsUnsafeURL(t *testing.T) {
	a := Article{
		ID: "article-1", CanonicalURL: "javascript:alert(1)",
		PublisherSourceID: "source-1", Title: "Example",
		ContentAvailability: ContentTitleOnly,
	}
	if err := ValidateArticle(a); err == nil {
		t.Fatal("expected non-http(s) article URL to be rejected")
	}
}

func TestValidateArticleRequiresSummaryForSummaryLevel(t *testing.T) {
	a := Article{
		ID: "article-1", CanonicalURL: "https://example.org/a",
		PublisherSourceID: "source-1", Title: "Example",
		ContentAvailability: ContentTitleSummary,
	}
	if err := ValidateArticle(a); err == nil {
		t.Fatal("expected title_summary article without summary to be rejected")
	}
}

func TestValidateRelationRequiresEvidence(t *testing.T) {
	r := testRelation()
	r.EvidenceIDs = nil
	err := ValidateRelation(r, map[string]Evidence{})
	if err == nil || !strings.Contains(err.Error(), "evidence") {
		t.Fatalf("expected missing evidence error, got %v", err)
	}
}

func TestValidateRelationRejectsUnapprovedCoOccurrenceEdge(t *testing.T) {
	r := testRelation()
	r.Predicate = RelationPredicate("related_to")
	err := ValidateRelation(r, map[string]Evidence{"evidence-1": testEvidence(EvidenceSupports)})
	if err == nil {
		t.Fatal("expected arbitrary related_to predicate to be rejected")
	}
}

func TestSupportedRelationNeedsSupportingEvidence(t *testing.T) {
	r := testRelation()
	err := ValidateRelation(r, map[string]Evidence{"evidence-1": testEvidence(EvidenceMentions)})
	if err == nil || !strings.Contains(err.Error(), "supporting") {
		t.Fatalf("expected supporting evidence error, got %v", err)
	}
}

func TestContestedRelationNeedsBothSides(t *testing.T) {
	r := testRelation()
	r.Status = RelationContested
	err := ValidateRelation(r, map[string]Evidence{"evidence-1": testEvidence(EvidenceSupports)})
	if err == nil || !strings.Contains(err.Error(), "both") {
		t.Fatalf("expected both-sided evidence error, got %v", err)
	}
}

func TestCanDisplayInRelationMapOnlyForSupportedAndGroundedRelation(t *testing.T) {
	r := testRelation()
	evidence := map[string]Evidence{"evidence-1": testEvidence(EvidenceSupports)}
	if !CanDisplayInRelationMap(r, evidence) {
		t.Fatal("expected valid supported relation to be displayable")
	}
	r.Status = RelationCandidate
	if CanDisplayInRelationMap(r, evidence) {
		t.Fatal("candidate relation must not be shown as an established graph edge")
	}
	r.Status = RelationSupported
	delete(evidence, "evidence-1")
	if CanDisplayInRelationMap(r, evidence) {
		t.Fatal("relation without retrievable evidence must not be displayed")
	}
}

func TestContestedRelationAcceptsSupportingAndContradictingEvidence(t *testing.T) {
	r := testRelation()
	r.Status = RelationContested
	r.EvidenceIDs = []string{"evidence-1", "evidence-2"}
	evidence := map[string]Evidence{
		"evidence-1": testEvidence(EvidenceSupports),
		"evidence-2": {
			ID: "evidence-2", ArticleID: "article-2", Stance: EvidenceContradicts,
			ExcerptField: "summary",
			Quote: "The ministry denied announcing the policy.",
			SourceURL: "https://example.net/statement", CapturedAt: time.Now(),
			ContentLevel: ContentFullText, ExtractionMethod: "test-fixture", ExtractionVersion: "1",
		},
	}
	if err := ValidateRelation(r, evidence); err != nil {
		t.Fatalf("expected contested relation with both sides to validate, got %v", err)
	}
	if CanDisplayInRelationMap(r, evidence) {
		t.Fatal("contested relation must not appear as an established graph edge")
	}
}

func TestRelationRejectsSelfEdge(t *testing.T) {
	r := testRelation()
	r.Object = r.Subject
	if err := ValidateRelation(r, map[string]Evidence{"evidence-1": testEvidence(EvidenceSupports)}); err == nil {
		t.Fatal("expected self-edge to be rejected")
	}
}

func TestRelationRejectsIncompatibleEndpointTypes(t *testing.T) {
	r := testRelation()
	r.Object = NodeRef{ID: "event-other", Type: NodeEvent}
	err := ValidateRelation(r, map[string]Evidence{"evidence-1": testEvidence(EvidenceSupports)})
	if err == nil || !strings.Contains(err.Error(), "cannot connect") {
		t.Fatalf("expected invalid endpoint types to be rejected, got %v", err)
	}
}
