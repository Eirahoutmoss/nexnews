package intelligence

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeResearchCaseShowsMixedEvidenceAndSourceLimitations(t *testing.T) {
	repo, err := NewJSONRepository(filepath.Join(t.TempDir(), "intelligence.json"))
	if err != nil { t.Fatal(err) }
	item, err := repo.CreateResearchCase(CreateResearchCaseInput{
		Title: "Target pattern", Question: "Does ownership correlate with target selection?",
	})
	if err != nil { t.Fatal(err) }
	item, err = repo.AddResearchHypothesis(item.ID, "Facilities with foreign partners may be selected less often.")
	if err != nil { t.Fatal(err) }
	hid := item.Hypotheses[0].ID

	inputs := []AddResearchEvidenceInput{
		{
			Kind: ResearchEvidenceClaim, Title: "Claimed strike", SourceURL: "https://news-a.example/story",
			Publisher: "News A", Excerpt: "An actor claimed the facility was targeted.",
			ContentLevel: ResearchContentTitleSummary, SourceKind: ResearchSourceInternational,
			SourceOrigin: ResearchOriginSyndicated, SyndicationGroup: "agency-wire-1",
			HypothesisAssessments: []HypothesisEvidenceAssessment{{HypothesisID:hid,Direction:HypothesisEvidenceSupports,Rationale:"Reported target list contains one facility in the study set."}},
		},
		{
			Kind: ResearchEvidenceDataPoint, Title: "Independent incident dataset", SourceURL: "https://data.example.net/incidents",
			Publisher: "Dataset Provider", Excerpt: "Dataset description and coverage notes.",
			ContentLevel: ResearchContentStructured, SourceKind: ResearchSourceDataset,
			SourceOrigin: ResearchOriginOriginal,
			HypothesisAssessments: []HypothesisEvidenceAssessment{{HypothesisID:hid,Direction:HypothesisEvidenceContradicts,Rationale:"Comparable incident records include a facility with foreign partners."}},
		},
		{
			Kind: ResearchEvidenceClaim, Title: "Republished wire copy", SourceURL: "https://news-b.example/story",
			Publisher: "News B", Excerpt: "The same wire service also reports the claimed strike.",
			ContentLevel: ResearchContentTitleSummary, SourceKind: ResearchSourceInternational,
			SourceOrigin: ResearchOriginSyndicated, SyndicationGroup: "agency-wire-1",
			HypothesisAssessments: []HypothesisEvidenceAssessment{{HypothesisID:hid,Direction:HypothesisEvidenceSupports,Rationale:"This is the same syndicated report, retained for provenance but not a second independent confirmation."}},
		},
	}
	for _, in := range inputs {
		item, err = repo.AddResearchEvidence(item.ID, in)
		if err != nil { t.Fatal(err) }
	}
	item, err = repo.AddResearchQuestion(item.ID, "Is the facility comparison set complete for the same period?")
	if err != nil { t.Fatal(err) }
	report := AnalyzeResearchCase(item)
	if report.EvidenceCount != 3 || report.PublisherDomainCount != 3 || report.KnownSyndicationGroupCount != 1 {
		t.Fatalf("unexpected coverage counts: %+v", report)
	}
	if report.FullTextOrStructuredCount != 1 || report.AttributedClaimCount != 2 || report.OpenQuestionCount != 1 {
		t.Fatalf("unexpected data scope counts: %+v", report)
	}
	if len(report.Hypotheses) != 1 { t.Fatalf("expected one hypothesis coverage: %+v", report.Hypotheses) }
	coverage := report.Hypotheses[0]
	if coverage.SupportingEvidenceCount != 2 || coverage.ContradictingEvidenceCount != 1 || coverage.CoverageLabel != "mixed_evidence" {
		t.Fatalf("expected mixed evidence: %+v", coverage)
	}
	if len(coverage.SharedSyndicationGroups) != 1 || coverage.SharedSyndicationGroups[0] != "agency-wire-1" {
		t.Fatalf("expected shared source lineage to be surfaced: %+v", coverage.SharedSyndicationGroups)
	}
	joined := strings.Join(report.Limitations, " ")
	if !strings.Contains(joined, "bağımsız kaynak") || !strings.Contains(joined, "Yerel/saha haberciliği") {
		t.Fatalf("expected independence and local reporting limitations, got %q", joined)
	}
	notes := strings.Join(coverage.Limitations, " ")
	if !strings.Contains(notes, "karşı kanıt") || !strings.Contains(notes, "ortak") {
		t.Fatalf("expected counterevidence and syndication limitations, got %q", notes)
	}
}

func TestAnalyzeResearchCaseDoesNotTreatNoEvidenceAsSupport(t *testing.T) {
	report := AnalyzeResearchCase(ResearchCase{
		ID: "r1", Hypotheses: []ResearchHypothesis{{ID:"h1",Statement:"An untested hypothesis",Status:HypothesisOpen}},
	})
	if len(report.Hypotheses) != 1 { t.Fatalf("unexpected hypotheses: %+v", report.Hypotheses) }
	if report.Hypotheses[0].CoverageLabel != "no_linked_evidence" {
		t.Fatalf("no evidence must remain open, got %+v", report.Hypotheses[0])
	}
	if len(report.Limitations) == 0 { t.Fatal("expected report limitations for an empty case") }
}
