package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexnews/intelligence"
)

func requestJSON(t *testing.T, handler http.Handler, method, target string, input any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if input == nil { body = bytes.NewReader(nil) } else {
		data, err := json.Marshal(input)
		if err != nil { t.Fatal(err) }
		body = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, target, body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestResearchWorkspacePageIsServedWithLocalSecurityHeaders(t *testing.T) {
	h := newHandler(testRepository(t))
	req := httptest.NewRequest(http.MethodGet, "/research-workspace", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
	for _, phrase := range []string{"Araştırma Masası", "Kurumsal hafıza", "Kaynaklı kanıt ekle", "Açık sorular", "hipotez"} {
		if !strings.Contains(rec.Body.String(), phrase) { t.Errorf("workspace page missing %q", phrase) }
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" { t.Fatal("expected frame protection header") }
	if rec.Header().Get("Cache-Control") != "no-store" { t.Fatal("research workspace must not be cached") }
}

func TestResearchWorkspaceAPICreatesCaseHypothesisAndEvidence(t *testing.T) {
	h := newHandler(testRepository(t))
	created := requestJSON(t, h, http.MethodPost, "/api/research-cases", intelligence.CreateResearchCaseInput{
		Title: "Tesis hedef seçimi", Question: "Hedef seçimi ile sahiplik arasında ilişki var mı?",
	})
	if created.Code != http.StatusCreated { t.Fatalf("create case status=%d: %s", created.Code, created.Body.String()) }
	var item intelligence.ResearchCase
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil { t.Fatal(err) }
	if item.ID == "" || item.Status != intelligence.ResearchCaseDraft { t.Fatalf("unexpected created case: %+v", item) }

	hyp := requestJSON(t, h, http.MethodPost, "/api/research-hypothesis?caseId="+item.ID, map[string]string{
		"statement": "Yabancı ortaklığı olan tesisler daha seyrek hedef alınıyor olabilir.",
	})
	if hyp.Code != http.StatusCreated { t.Fatalf("create hypothesis status=%d: %s", hyp.Code, hyp.Body.String()) }
	if err := json.Unmarshal(hyp.Body.Bytes(), &item); err != nil { t.Fatal(err) }
	if len(item.Hypotheses) != 1 { t.Fatalf("expected one hypothesis: %+v", item.Hypotheses) }
	hypothesisID := item.Hypotheses[0].ID

	evidence := requestJSON(t, h, http.MethodPost, "/api/research-evidence?caseId="+item.ID, intelligence.AddResearchEvidenceInput{
		Kind: intelligence.ResearchEvidenceClaim,
		Title: "Birincil açıklama",
		SourceURL: "https://www.example.org/statement",
		Publisher: "Example Org",
		Excerpt: "The source claimed the facility was targeted.",
		ContentLevel: intelligence.ResearchContentTitleSummary,
		SourceKind: intelligence.ResearchSourcePrimary,
		SourceOrigin: intelligence.ResearchOriginOfficial,
		HypothesisAssessments: []intelligence.HypothesisEvidenceAssessment{{
			HypothesisID: hypothesisID, Direction: intelligence.HypothesisEvidenceContext,
			Rationale: "Bu açıklama hedef alındığı iddiasını bildirir ancak tesis sahipliğine ilişkin örüntüyü tek başına sınamaz.",
		}},
	})
	if evidence.Code != http.StatusCreated { t.Fatalf("add evidence status=%d: %s", evidence.Code, evidence.Body.String()) }
	if err := json.Unmarshal(evidence.Body.Bytes(), &item); err != nil { t.Fatal(err) }
	if len(item.Evidence) != 1 || item.Evidence[0].PublisherDomain != "example.org" { t.Fatalf("unexpected evidence record: %+v", item.Evidence) }

	question := requestJSON(t, h, http.MethodPost, "/api/research-question?caseId="+item.ID, map[string]string{
		"question": "Karşılaştırma evrenindeki diğer tesisler hangileri?",
	})
	if question.Code != http.StatusCreated { t.Fatalf("add question status=%d: %s", question.Code, question.Body.String()) }

	read := httptest.NewRecorder()
	h.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/research-case?caseId="+item.ID, nil))
	if read.Code != http.StatusOK { t.Fatalf("read case status=%d: %s", read.Code, read.Body.String()) }
	if err := json.Unmarshal(read.Body.Bytes(), &item); err != nil { t.Fatal(err) }
	if len(item.Evidence) != 1 || len(item.Hypotheses) != 1 || len(item.OpenQuestions) != 1 {
		t.Fatalf("research case was not persisted: evidence=%d hypotheses=%d questions=%d", len(item.Evidence), len(item.Hypotheses), len(item.OpenQuestions))
	}
}

func TestResearchImportAPIImportsLinkedNexNewsEvent(t *testing.T) {
	h := newHandler(testRepository(t))
	created := requestJSON(t, h, http.MethodPost, "/api/research-cases", intelligence.CreateResearchCaseInput{
		Title:"Import test", Question:"What was reported?", RelatedEventID:"e1",
	})
	if created.Code != http.StatusCreated { t.Fatalf("create case status=%d body=%s",created.Code,created.Body.String()) }
	var item intelligence.ResearchCase
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil { t.Fatal(err) }
	first := requestJSON(t,h,http.MethodPost,"/api/research-import?caseId="+item.ID,nil)
	if first.Code != http.StatusOK { t.Fatalf("import status=%d body=%s",first.Code,first.Body.String()) }
	var result struct {
		Case intelligence.ResearchCase `json:"case"`
		Stats intelligence.ResearchImportStats `json:"stats"`
	}
	if err := json.Unmarshal(first.Body.Bytes(),&result); err != nil { t.Fatal(err) }
	if len(result.Case.Evidence)!=1 || result.Stats.EvidenceImported!=1 {
		t.Fatalf("expected linked article to be imported, got stats=%+v evidence=%+v",result.Stats,result.Case.Evidence)
	}
	second := requestJSON(t,h,http.MethodPost,"/api/research-import?caseId="+item.ID,nil)
	if second.Code != http.StatusOK { t.Fatalf("repeat import status=%d body=%s",second.Code,second.Body.String()) }
	if err := json.Unmarshal(second.Body.Bytes(),&result); err != nil { t.Fatal(err) }
	if result.Stats.UnchangedSkipped!=1 || len(result.Case.Evidence)!=1 {
		t.Fatalf("repeat import was not idempotent: stats=%+v evidence=%d",result.Stats,len(result.Case.Evidence))
	}
}

func TestResearchAnalysisAPIReportsEvidenceCoverage(t *testing.T) {
	h := newHandler(testRepository(t))
	created := requestJSON(t, h, http.MethodPost, "/api/research-cases", map[string]string{
		"title":"Evidence test", "question":"What does the record support?",
	})
	if created.Code != http.StatusCreated { t.Fatalf("create case status=%d body=%s",created.Code,created.Body.String()) }
	var item intelligence.ResearchCase
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil { t.Fatal(err) }
	item, err := func() (intelligence.ResearchCase,error) {
		rec := requestJSON(t,h,http.MethodPost,"/api/research-hypothesis?caseId="+item.ID,map[string]string{"statement":"One testable hypothesis"})
		if rec.Code != http.StatusCreated { t.Fatalf("add hypothesis status=%d body=%s",rec.Code,rec.Body.String()) }
		var v intelligence.ResearchCase
		e := json.Unmarshal(rec.Body.Bytes(),&v)
		return v,e
	}()
	if err != nil { t.Fatal(err) }
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec,httptest.NewRequest(http.MethodGet,"/api/research-analysis?caseId="+item.ID,nil))
	if rec.Code != http.StatusOK { t.Fatalf("analysis status=%d body=%s",rec.Code,rec.Body.String()) }
	var report intelligence.ResearchAnalysisReport
	if err := json.Unmarshal(rec.Body.Bytes(),&report); err != nil { t.Fatal(err) }
	if report.CaseID != item.ID || len(report.Hypotheses) != 1 || report.Hypotheses[0].CoverageLabel != "no_linked_evidence" {
		t.Fatalf("unexpected coverage report: %+v",report)
	}
	if rec.Header().Get("Cache-Control") != "no-store" { t.Fatal("analysis response must not be cached") }
}

func TestResearchQuestionStatusAPIRequiresResolutionNote(t *testing.T) {
	h:=newHandler(testRepository(t))
	created:=requestJSON(t,h,http.MethodPost,"/api/research-cases",intelligence.CreateResearchCaseInput{Title:"Question status",Question:"Does this endpoint preserve resolution context?"})
	if created.Code!=http.StatusCreated { t.Fatalf("create status=%d body=%s",created.Code,created.Body.String()) }
	var item intelligence.ResearchCase
	if err:=json.Unmarshal(created.Body.Bytes(),&item);err!=nil { t.Fatal(err) }
	added:=requestJSON(t,h,http.MethodPost,"/api/research-question?caseId="+item.ID,map[string]string{"question":"What is missing from the record?"})
	if added.Code!=http.StatusCreated { t.Fatalf("add question status=%d body=%s",added.Code,added.Body.String()) }
	if err:=json.Unmarshal(added.Body.Bytes(),&item);err!=nil { t.Fatal(err) }
	qid:=item.OpenQuestions[0].ID
	bad:=requestJSON(t,h,http.MethodPost,"/api/research-question-status?caseId="+item.ID+"&questionId="+qid,map[string]string{"status":"resolved","note":""})
	if bad.Code!=http.StatusBadRequest { t.Fatalf("expected resolution note validation, status=%d body=%s",bad.Code,bad.Body.String()) }
	good:=requestJSON(t,h,http.MethodPost,"/api/research-question-status?caseId="+item.ID+"&questionId="+qid,map[string]string{"status":"resolved","note":"The source dataset covers the relevant period; see linked evidence evidence-1."})
	if good.Code!=http.StatusOK { t.Fatalf("resolve question status=%d body=%s",good.Code,good.Body.String()) }
	if err:=json.Unmarshal(good.Body.Bytes(),&item);err!=nil { t.Fatal(err) }
	if item.OpenQuestions[0].Status!="resolved" || item.OpenQuestions[0].ResolutionNote=="" {
		t.Fatalf("resolution note not persisted: %+v",item.OpenQuestions[0])
	}
}

func TestResearchAPIRejectsBadMethodsBodiesAndUnknownCases(t *testing.T) {
	h := newHandler(testRepository(t))
	cases := []struct{ method, path, body string; want int }{
		{http.MethodDelete, "/api/research-cases", "", http.StatusMethodNotAllowed},
		{http.MethodPost, "/api/research-cases", "{broken", http.StatusBadRequest},
		{http.MethodGet, "/api/research-case", "", http.StatusBadRequest},
		{http.MethodGet, "/api/research-case?caseId=unknown", "", http.StatusNotFound},
		{http.MethodPost, "/api/research-hypothesis?caseId=unknown", `{"statement":"hypothesis"}`, http.StatusNotFound},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want { t.Errorf("%s %s got %d want %d: %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String()) }
	}
}
