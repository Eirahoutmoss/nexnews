package intelligence

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResearchWorkspacePersistsHypothesesEvidenceAndOpenQuestions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence.json")
	repo, err := NewJSONRepository(path)
	if err != nil { t.Fatal(err) }

	item, err := repo.CreateResearchCase(CreateResearchCaseInput{
		Title: "Enerji tesisi hedef seçimi",
		Question: "Hedef seçimi ile tesis sahipliği arasında gözlemlenebilir bir örüntü var mı?",
		Context: "Açık hipotezler araştırılacak; niyet yalnızca örüntüden çıkarılmayacak.",
	})
	if err != nil { t.Fatal(err) }
	if item.Status != ResearchCaseDraft || item.ID == "" { t.Fatalf("unexpected case: %+v", item) }

	item, err = repo.AddResearchHypothesis(item.ID, "Yabancı ortakların bulunduğu tesisler daha seyrek hedef alınmış olabilir.")
	if err != nil { t.Fatal(err) }
	if len(item.Hypotheses) != 1 || item.Hypotheses[0].Status != HypothesisOpen { t.Fatalf("unexpected hypotheses: %+v", item.Hypotheses) }
	hypothesisID := item.Hypotheses[0].ID

	published := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	item, err = repo.AddResearchEvidence(item.ID, AddResearchEvidenceInput{
		Kind: ResearchEvidenceClaim,
		Title: "Örnek birincil açıklama",
		SourceURL: "https://www.example.org/report#section",
		Publisher: "Example Org",
		Author: "Research desk",
		PublishedAt: &published,
		Excerpt: "The statement claims that a facility was targeted.",
		ContentLevel: ResearchContentTitleSummary,
		SourceKind: ResearchSourcePrimary,
		SourceOrigin: ResearchOriginOfficial,
		HypothesisAssessments: []HypothesisEvidenceAssessment{{
			HypothesisID: hypothesisID,
			Direction: HypothesisEvidenceContext,
			Rationale: "Bu kayıt yalnızca saldırı iddiasını belgelemektedir; sahiplik örüntüsünü tek başına sınamaz.",
		}},
	})
	if err != nil { t.Fatal(err) }
	if len(item.Evidence) != 1 { t.Fatalf("expected one evidence item, got %d", len(item.Evidence)) }
	ev := item.Evidence[0]
	if ev.PublisherDomain != "example.org" || strings.Contains(ev.SourceURL, "#section") {
		t.Fatalf("expected normalized source domain and URL, got %+v", ev)
	}
	if ev.HypothesisAssessments[0].Direction != HypothesisEvidenceContext {
		t.Fatalf("unexpected assessment: %+v", ev.HypothesisAssessments[0])
	}

	if _, err := repo.SetResearchHypothesisStatus(item.ID, hypothesisID, HypothesisSupported, "Only a note, no supporting evidence."); err == nil || !strings.Contains(err.Error(), "supporting evidence") {
		t.Fatalf("expected supported status to require positive evidence, got %v", err)
	}
	item, err = repo.AddResearchEvidence(item.ID, AddResearchEvidenceInput{
		Kind: ResearchEvidenceDataPoint, Title: "Example supporting dataset",
		SourceURL: "https://data.example.net/facilities.csv", Publisher: "Example Dataset",
		Excerpt: "Dataset description: records for the selected time window.",
		ContentLevel: ResearchContentStructured, SourceKind: ResearchSourceDataset,
		SourceOrigin: ResearchOriginOriginal,
		HypothesisAssessments: []HypothesisEvidenceAssessment{{
			HypothesisID: hypothesisID, Direction: HypothesisEvidenceSupports,
			Rationale: "This structured dataset provides a denominator and comparable ownership fields for the tested period.",
		}},
	})
	if err != nil { t.Fatal(err) }
	item, err = repo.SetResearchHypothesisStatus(item.ID, hypothesisID, HypothesisSupported, "The registered structured comparison supports this hypothesis, subject to coverage limitations.")
	if err != nil { t.Fatal(err) }
	if item.Hypotheses[0].Status != HypothesisSupported { t.Fatalf("status not saved: %+v", item.Hypotheses[0]) }

	item, err = repo.AddResearchQuestion(item.ID, "Were attempted, intercepted and confirmed hits separated in the incident ledger?")
	if err != nil { t.Fatal(err) }
	if len(item.OpenQuestions) != 1 || item.OpenQuestions[0].Status != "open" { t.Fatalf("unexpected questions: %+v", item.OpenQuestions) }

	reloaded, err := NewJSONRepository(path)
	if err != nil { t.Fatal(err) }
	got, err := reloaded.GetResearchCase(item.ID)
	if err != nil { t.Fatal(err) }
	if len(got.Evidence) != 2 || len(got.Hypotheses) != 1 || len(got.OpenQuestions) != 1 {
		t.Fatalf("research workspace did not persist, got evidence=%d hypotheses=%d questions=%d", len(got.Evidence), len(got.Hypotheses), len(got.OpenQuestions))
	}
	summaries, err := reloaded.ListResearchCases()
	if err != nil { t.Fatal(err) }
	if len(summaries) != 1 || summaries[0].EvidenceCount != 2 || summaries[0].OpenQuestionCount != 1 || summaries[0].PublisherDomainCount != 2 {
		t.Fatalf("unexpected summary: %+v", summaries)
	}
}

func TestResearchWorkspaceRejectsInvalidSourceAndUnlinkedHypothesis(t *testing.T) {
	repo, err := NewJSONRepository(filepath.Join(t.TempDir(), "intelligence.json"))
	if err != nil { t.Fatal(err) }
	item, err := repo.CreateResearchCase(CreateResearchCaseInput{Title:"Case", Question:"What happened?"})
	if err != nil { t.Fatal(err) }
	item, err = repo.AddResearchHypothesis(item.ID, "A testable explanation.")
	if err != nil { t.Fatal(err) }

	_, err = repo.AddResearchEvidence(item.ID, AddResearchEvidenceInput{
		Kind: ResearchEvidenceClaim, Title:"Bad source", SourceURL:"file:///etc/passwd",
		Publisher:"Bad", Excerpt:"untrusted fixture", ContentLevel:ResearchContentFullText,
		SourceKind:ResearchSourceOther, SourceOrigin:ResearchOriginUnknown,
	})
	if err == nil || !strings.Contains(err.Error(), "http or https") { t.Fatalf("expected invalid URL rejection, got %v", err) }

	_, err = repo.AddResearchEvidence(item.ID, AddResearchEvidenceInput{
		Kind: ResearchEvidenceClaim, Title:"Foreign hypothesis", SourceURL:"https://source.example/story",
		Publisher:"Example", Excerpt:"an assertion", ContentLevel:ResearchContentTitleOnly,
		SourceKind:ResearchSourceInternational, SourceOrigin:ResearchOriginUnknown,
		HypothesisAssessments: []HypothesisEvidenceAssessment{{
			HypothesisID:"hypothesis_from_another_case", Direction:HypothesisEvidenceSupports, Rationale:"test",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "does not belong") { t.Fatalf("expected cross-case hypothesis rejection, got %v", err) }

	if _, err := repo.CreateResearchCase(CreateResearchCaseInput{Title:" ", Question:"x"}); err == nil {
		t.Fatal("expected empty research title to be rejected")
	}
}

func TestResearchQuestionMustHaveResolutionNoteAndCanReopen(t *testing.T) {
	repo, err := NewJSONRepository(filepath.Join(t.TempDir(), "intelligence.json"))
	if err != nil { t.Fatal(err) }
	item, err := repo.CreateResearchCase(CreateResearchCaseInput{Title:"Research process",Question:"Which question can be answered?"})
	if err != nil { t.Fatal(err) }
	item, err = repo.AddResearchQuestion(item.ID, "Does the record distinguish a claimed strike from confirmed damage?")
	if err != nil { t.Fatal(err) }
	if item.Status != ResearchCaseActive { t.Fatalf("first research action should activate a draft case: %s", item.Status) }
	qid := item.OpenQuestions[0].ID
	if _, err := repo.SetResearchQuestionStatus(item.ID,qid,"resolved",""); err == nil || !strings.Contains(err.Error(),"resolution note") {
		t.Fatalf("expected a resolution note requirement, got %v",err)
	}
	item, err = repo.SetResearchQuestionStatus(item.ID,qid,"resolved","The incident ledger records claims and confirmed damage separately; the report sources are linked.")
	if err != nil { t.Fatal(err) }
	if item.OpenQuestions[0].Status!="resolved" || item.OpenQuestions[0].ResolutionNote=="" {
		t.Fatalf("expected resolved question with rationale, got %+v",item.OpenQuestions[0])
	}
	item, err = repo.SetResearchQuestionStatus(item.ID,qid,"open","")
	if err != nil { t.Fatal(err) }
	if item.OpenQuestions[0].Status!="open" || item.OpenQuestions[0].ResolutionNote!="" {
		t.Fatalf("reopened question should clear stale resolution note, got %+v",item.OpenQuestions[0])
	}
	summary:=summarizeResearchCase(item)
	if summary.OpenQuestionCount!=1 { t.Fatalf("expected reopened question to count as open, got %+v",summary) }
}

func TestResearchCaseCannotConcludeHypothesisWithoutLinkedEvidence(t *testing.T) {
	repo, err := NewJSONRepository(filepath.Join(t.TempDir(), "intelligence.json"))
	if err != nil { t.Fatal(err) }
	item, err := repo.CreateResearchCase(CreateResearchCaseInput{Title:"Case", Question:"Question?"})
	if err != nil { t.Fatal(err) }
	item, err = repo.AddResearchHypothesis(item.ID, "A hypothesis")
	if err != nil { t.Fatal(err) }
	if _, err := repo.SetResearchHypothesisStatus(item.ID, item.Hypotheses[0].ID, HypothesisRejected, "Seems unlikely."); err == nil || !strings.Contains(err.Error(), "contradictory evidence") {
		t.Fatalf("expected rejection to require contradictory evidence, got %v", err)
	}
}
