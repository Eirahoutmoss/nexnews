package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexnews/intelligence"
)

func testRepository(t *testing.T) *intelligence.JSONRepository {
	t.Helper()
	path := filepath.Join(t.TempDir(), "intelligence.json")
	db := intelligence.Database{
		Version: 1, Schema: intelligence.SchemaVersion,
		SavedAt: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC),
		Sources: map[string]intelligence.Source{
			"s1": {ID:"s1",Name:"Example",CanonicalDomain:"example.org",Type:intelligence.SourcePublisher},
		},
		Articles: map[string]intelligence.Article{
			"a1": {ID:"a1",CanonicalURL:"https://example.org/news",PublisherSourceID:"s1",Title:"Example headline",ContentAvailability:intelligence.ContentTitleOnly},
		},
		Events: map[string]intelligence.Event{
			"e1": {ID:"e1",Title:"Example event candidate",Status:intelligence.EventNeedsReview,ArticleIDs:[]string{"a1"}},
		},
		Claims: map[string]intelligence.Claim{},
		Evidence: map[string]intelligence.Evidence{},
		Entities: map[string]intelligence.Entity{},
		EntityMentions: map[string]intelligence.EntityMention{},
		Relations: map[string]intelligence.Relation{},
		Timeline: map[string]intelligence.TimelineEntry{},
		Runs: map[string]intelligence.AnalysisRun{},
	}
	b,err:=json.Marshal(db)
	if err!=nil {t.Fatal(err)}
	if err:=os.WriteFile(path,b,0o600);err!=nil {t.Fatal(err)}
	repo,err:=intelligence.NewJSONRepository(path)
	if err!=nil {t.Fatal(err)}
	return repo
}

func TestIndexServesReadOnlyInterfaceAndSecurityHeaders(t *testing.T) {
	h:=newHandler(testRepository(t))
	r:=httptest.NewRequest(http.MethodGet,"/",nil)
	w:=httptest.NewRecorder()
	h.ServeHTTP(w,r)
	if w.Code!=http.StatusOK {t.Fatalf("GET / status=%d body=%s",w.Code,w.Body.String())}
	if !strings.Contains(w.Body.String(),"NexNews Intelligence") || !strings.Contains(w.Body.String(),"İlişkiler") || !strings.Contains(w.Body.String(),"Kaynak Karşılaştırma") || !strings.Contains(w.Body.String(),"İddia Değerlendirme") || !strings.Contains(w.Body.String(),"Veri Kalitesi") || !strings.Contains(w.Body.String(),"Olay dosyasını aç") || !strings.Contains(w.Body.String(),"Son gelişmeler · en yeni kayıtlar") || !strings.Contains(w.Body.String(),"Birebir tekrar") {
		t.Fatal("expected standalone Intelligence interface")
	}
	if w.Header().Get("X-Frame-Options")!="DENY" {t.Fatal("expected frame protection header")}
	if !strings.Contains(w.Header().Get("Content-Security-Policy"),"connect-src 'self'") {t.Fatal("expected local-only fetch policy")}
}

func TestStateAPIContainsDatabaseRecords(t *testing.T) {
	h:=newHandler(testRepository(t))
	r:=httptest.NewRequest(http.MethodGet,"/api/state",nil)
	w:=httptest.NewRecorder()
	h.ServeHTTP(w,r)
	if w.Code!=http.StatusOK {t.Fatalf("status=%d body=%s",w.Code,w.Body.String())}
	var state viewState
	if err:=json.Unmarshal(w.Body.Bytes(),&state);err!=nil {t.Fatal(err)}
	if state.Summary.Articles!=1 || state.Summary.Events!=1 || state.Summary.Relations!=0 || state.DataQuality.RecordsChecked == 0 {
		t.Fatalf("unexpected state summary: %+v",state.Summary)
	}
}

func TestStateAPIIsReadOnlyAndRejectsOtherMethods(t *testing.T) {
	h:=newHandler(testRepository(t))
	r:=httptest.NewRequest(http.MethodPost,"/api/state",strings.NewReader("{}"))
	w:=httptest.NewRecorder()
	h.ServeHTTP(w,r)
	if w.Code!=http.StatusMethodNotAllowed {t.Fatalf("expected 405, got %d",w.Code)}
	if w.Header().Get("Allow")!="GET" {t.Fatal("expected only GET to be allowed")}
}

func TestStateAPIHandlesCorruptDatabaseWithoutLeakingPath(t *testing.T) {
	path:=filepath.Join(t.TempDir(),"broken.json")
	if err:=os.WriteFile(path,[]byte("{broken"),0o600);err!=nil {t.Fatal(err)}
	repo,err:=intelligence.NewJSONRepository(path)
	if err!=nil {t.Fatal(err)}
	h:=newHandler(repo)
	r:=httptest.NewRequest(http.MethodGet,"/api/state",nil)
	w:=httptest.NewRecorder()
	h.ServeHTTP(w,r)
	if w.Code!=http.StatusInternalServerError {t.Fatalf("expected 500, got %d",w.Code)}
	if strings.Contains(w.Body.String(),path) {t.Fatal("server error must not leak the filesystem path")}
}

func TestUnknownPathReturns404(t *testing.T) {
	h:=newHandler(testRepository(t))
	r:=httptest.NewRequest(http.MethodGet,"/not-found",nil)
	w:=httptest.NewRecorder()
	h.ServeHTTP(w,r)
	if w.Code!=http.StatusNotFound {t.Fatalf("expected 404, got %d",w.Code)}
}
 
func TestStateAPIIncludesSourceComparisonReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comparison.json")
	published := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)
	db := intelligence.Database{
		Version: 1, Schema: intelligence.SchemaVersion,
		Sources: map[string]intelligence.Source{
			"s1": {ID:"s1",Name:"Kaynak A",CanonicalDomain:"a.example"},
			"s2": {ID:"s2",Name:"Kaynak B",CanonicalDomain:"b.example"},
		},
		Articles: map[string]intelligence.Article{
			"a1": {ID:"a1",CanonicalURL:"https://a.example/1",PublisherSourceID:"s1",Title:"Haber 1",PublishedAt:&published},
			"a2": {ID:"a2",CanonicalURL:"https://b.example/2",PublisherSourceID:"s2",Title:"Haber 2",PublishedAt:&published},
		},
		Events: map[string]intelligence.Event{},
		Claims: map[string]intelligence.Claim{
			"c1": {ID:"c1",Kind:intelligence.ClaimSourceStatement,ClaimText:"Bakanlık yeni programı açıkladı.",Status:intelligence.ClaimReported,EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2",Kind:intelligence.ClaimSourceStatement,ClaimText:"BAKANLIK yeni programı açıkladı",Status:intelligence.ClaimReported,EvidenceIDs:[]string{"e2"}},
		},
		Evidence: map[string]intelligence.Evidence{
			"e1": {ID:"e1",ClaimID:"c1",ArticleID:"a1",Stance:intelligence.EvidenceReports,ExcerptField:"summary",Quote:"Bakanlık yeni programı açıkladı.",SourceURL:"https://a.example/1",ExtractionMethod:"test",ExtractionVersion:"1"},
			"e2": {ID:"e2",ClaimID:"c2",ArticleID:"a2",Stance:intelligence.EvidenceReports,ExcerptField:"summary",Quote:"BAKANLIK yeni programı açıkladı",SourceURL:"https://b.example/2",ExtractionMethod:"test",ExtractionVersion:"1"},
		},
		Entities: map[string]intelligence.Entity{},
		EntityMentions: map[string]intelligence.EntityMention{},
		Relations: map[string]intelligence.Relation{},
		Timeline: map[string]intelligence.TimelineEntry{},
		Runs: map[string]intelligence.AnalysisRun{},
	}
	payload, err := json.Marshal(db)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, payload, 0o600); err != nil { t.Fatal(err) }
	repo, err := intelligence.NewJSONRepository(path)
	if err != nil { t.Fatal(err) }
	h := newHandler(repo)
	r := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK { t.Fatalf("status=%d body=%s", w.Code, w.Body.String()) }
	var state viewState
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil { t.Fatal(err) }
	if state.Summary.RepeatedWordingGroups != 1 || len(state.SourceComparisons.Groups) != 1 {
		t.Fatalf("expected one exact repeated wording group, summary=%+v report=%+v", state.Summary, state.SourceComparisons)
	}
	if state.SourceComparisons.Groups[0].PublisherDomainCount != 2 {
		t.Fatalf("expected two publisher domains, got %+v", state.SourceComparisons.Groups[0])
	}
}

func TestStateAPIIncludesExplicitClaimAssessment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "assessment.json")
	t1 := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)
	db := intelligence.Database{
		Version: 1, Schema: intelligence.SchemaVersion,
		Sources: map[string]intelligence.Source{
			"s1": {ID:"s1",Name:"Kaynak A",CanonicalDomain:"a.example"},
			"s2": {ID:"s2",Name:"Kaynak B",CanonicalDomain:"b.example"},
		},
		Articles: map[string]intelligence.Article{
			"a1": {ID:"a1",CanonicalURL:"https://a.example/1",PublisherSourceID:"s1",Title:"Haber 1",PublishedAt:&t1},
			"a2": {ID:"a2",CanonicalURL:"https://b.example/2",PublisherSourceID:"s2",Title:"Haber 2",PublishedAt:&t2},
		},
		Events: map[string]intelligence.Event{},
		Claims: map[string]intelligence.Claim{
			"c1": {ID:"c1",Kind:intelligence.ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etti.",Status:intelligence.ClaimReported,EvidenceIDs:[]string{"e1"}},
			"c2": {ID:"c2",Kind:intelligence.ClaimSourceStatement,ClaimText:"Kabine anlaşmayı kabul etmedi.",Status:intelligence.ClaimReported,EvidenceIDs:[]string{"e2"}},
		},
		Evidence: map[string]intelligence.Evidence{
			"e1": {ID:"e1",ClaimID:"c1",ArticleID:"a1",Stance:intelligence.EvidenceReports,ExcerptField:"title",Quote:"Kabine anlaşmayı kabul etti.",SourceURL:"https://a.example/1",ExtractionMethod:"test",ExtractionVersion:"1"},
			"e2": {ID:"e2",ClaimID:"c2",ArticleID:"a2",Stance:intelligence.EvidenceReports,ExcerptField:"title",Quote:"Kabine anlaşmayı kabul etmedi.",SourceURL:"https://b.example/2",ExtractionMethod:"test",ExtractionVersion:"1"},
		},
		Entities: map[string]intelligence.Entity{},
		EntityMentions: map[string]intelligence.EntityMention{},
		Relations: map[string]intelligence.Relation{},
		Timeline: map[string]intelligence.TimelineEntry{},
		Runs: map[string]intelligence.AnalysisRun{},
	}
	payload, err := json.Marshal(db)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, payload, 0o600); err != nil { t.Fatal(err) }
	repo, err := intelligence.NewJSONRepository(path)
	if err != nil { t.Fatal(err) }
	h := newHandler(repo)
	r := httptest.NewRequest(http.MethodGet, "/api/state", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK { t.Fatalf("status=%d body=%s", w.Code, w.Body.String()) }
	var state viewState
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil { t.Fatal(err) }
	if state.Summary.ExplicitOppositionCases != 1 || len(state.ClaimAssessment.Cases) != 1 {
		t.Fatalf("expected one explicit opposition review case, summary=%+v report=%+v",state.Summary,state.ClaimAssessment)
	}
}
