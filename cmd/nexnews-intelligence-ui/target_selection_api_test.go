package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexnews/intelligence"
)

func targetSelectionTestTimePtr(t time.Time) *time.Time { return &t }

func apiTestDate(value string) time.Time {
	t,err:=time.Parse(time.RFC3339,value)
	if err!=nil {panic(err)}
	return t
}

func TestTargetSelectionStudyAPIStoresAndReturnsDescriptiveComparison(t *testing.T) {
	h:=newHandler(testRepository(t))
	created:=requestJSON(t,h,http.MethodPost,"/api/research-cases",intelligence.CreateResearchCaseInput{
		Title:"Target pattern API test",Question:"Do included facilities show different recorded targeting rates?",
	})
	if created.Code!=http.StatusCreated {t.Fatalf("create case status=%d: %s",created.Code,created.Body.String())}
	var researchCase intelligence.ResearchCase
	if err:=json.Unmarshal(created.Body.Bytes(),&researchCase);err!=nil {t.Fatal(err)}

	addEvidence:=func(title string) string {
		t.Helper()
		rec:=requestJSON(t,h,http.MethodPost,"/api/research-evidence?caseId="+researchCase.ID,intelligence.AddResearchEvidenceInput{
			Kind:intelligence.ResearchEvidenceDataPoint,Title:title,
			SourceURL:"https://data.example.org/"+title,
			Publisher:"Example Data Registry",Excerpt:"Source record used in test.",
			ContentLevel:intelligence.ResearchContentStructured,
			SourceKind:intelligence.ResearchSourceDataset,
			SourceOrigin:intelligence.ResearchOriginOriginal,
		})
		if rec.Code!=http.StatusCreated {t.Fatalf("add evidence status=%d: %s",rec.Code,rec.Body.String())}
		var updated intelligence.ResearchCase
		if err:=json.Unmarshal(rec.Body.Bytes(),&updated);err!=nil {t.Fatal(err)}
		for _,e:=range updated.Evidence {if e.Title==title {return e.ID}}
		t.Fatalf("evidence %q not found in response",title)
		return ""
	}
	eligibilityID:=addEvidence("eligibility-record")
	ownershipID:=addEvidence("ownership-record")
	incidentID:=addEvidence("incident-record")
	study:=intelligence.TargetSelectionStudy{
		Title:"Foreign participation comparison",
		Question:"Do facilities with foreign participation show a different recorded targeting rate?",
		PeriodStart:apiTestDate("2024-01-01T00:00:00Z"),
		PeriodEnd:apiTestDate("2024-12-31T23:59:59Z"),
		UniverseDefinition:"All eligible refineries in the documented study geography and time window.",
		UniverseComplete:true,
		Assets:[]intelligence.TargetAsset{
			{ID:"foreign",Name:"Facility A",AssetType:"refinery",InclusionRationale:"Matches the predefined facility type and geography.",EligibilityEvidenceIDs:[]string{eligibilityID},OwnershipHistory:[]intelligence.TargetOwnershipPeriod{{ValidFrom:apiTestDate("2023-01-01T00:00:00Z"),ValidTo:targetSelectionTestTimePtr(apiTestDate("2025-01-01T00:00:00Z")),Category:intelligence.TargetOwnershipForeignParticipation,Details:"Ownership classification sourced in the case.",EvidenceIDs:[]string{ownershipID}}}},
			{ID:"domestic",Name:"Facility B",AssetType:"refinery",InclusionRationale:"Matches the predefined facility type and geography.",EligibilityEvidenceIDs:[]string{eligibilityID},OwnershipHistory:[]intelligence.TargetOwnershipPeriod{{ValidFrom:apiTestDate("2023-01-01T00:00:00Z"),ValidTo:datePtr(apiTestDate("2025-01-01T00:00:00Z")),Category:intelligence.TargetOwnershipNoForeignParticipation,Details:"Ownership classification sourced in the case.",EvidenceIDs:[]string{ownershipID}}}},
		},
		Incidents:[]intelligence.TargetIncident{{ID:"recorded-attempt",AssetID:"foreign",Actor:"Actor X",OccurredAt:apiTestDate("2024-07-01T00:00:00Z"),Status:intelligence.TargetIncidentAttemptReported,Description:"Reported attempt; not treated as independent confirmation by this analysis.",EvidenceIDs:[]string{incidentID}}},
	}
	rec:=requestJSON(t,h,http.MethodPost,"/api/target-selection-studies?caseId="+researchCase.ID,study)
	if rec.Code!=http.StatusCreated {t.Fatalf("create target study status=%d: %s",rec.Code,rec.Body.String())}
	var saved intelligence.TargetSelectionStudy
	if err:=json.Unmarshal(rec.Body.Bytes(),&saved);err!=nil {t.Fatal(err)}
	if saved.ID=="" {t.Fatal("study ID was not returned")}

	get:=httptest.NewRecorder()
	h.ServeHTTP(get,httptest.NewRequest(http.MethodGet,"/api/target-selection-studies?caseId="+researchCase.ID,nil))
	if get.Code!=http.StatusOK {t.Fatalf("list studies status=%d: %s",get.Code,get.Body.String())}
	var list struct { Items []intelligence.TargetSelectionStudy `json:"items"` }
	if err:=json.Unmarshal(get.Body.Bytes(),&list);err!=nil {t.Fatal(err)}
	if len(list.Items)!=1 || list.Items[0].ID!=saved.ID {t.Fatalf("unexpected saved studies: %+v",list)}

	analysis:=httptest.NewRecorder()
	url:="/api/target-selection-analysis?caseId="+researchCase.ID+"&studyId="+saved.ID
	h.ServeHTTP(analysis,httptest.NewRequest(http.MethodGet,url,nil))
	if analysis.Code!=http.StatusOK {t.Fatalf("analysis status=%d: %s",analysis.Code,analysis.Body.String())}
	var report intelligence.TargetSelectionReport
	if err:=json.Unmarshal(analysis.Body.Bytes(),&report);err!=nil {t.Fatal(err)}
	if report.ComparisonStatus!="descriptive_comparison_only" || report.IncidentRecordCount!=1 {
		t.Fatalf("unexpected target-selection report: %+v",report)
	}
	if analysis.Header().Get("Cache-Control")!="no-store" {t.Fatal("analysis result should not be cached")}
}

func TestTargetSelectionStudyAPIRejectsCrossCaseReferencesAndWrongMethods(t *testing.T) {
	h:=newHandler(testRepository(t))
	rec:=requestJSON(t,h,http.MethodGet,"/api/target-selection-analysis",nil)
	if rec.Code!=http.StatusBadRequest {t.Fatalf("missing case and study IDs should be rejected, got %d",rec.Code)}
	rec=requestJSON(t,h,http.MethodPost,"/api/target-selection-analysis?caseId=case&studyId=study",map[string]any{})
	if rec.Code!=http.StatusMethodNotAllowed {t.Fatalf("analysis endpoint should reject POST, got %d",rec.Code)}
	rec=requestJSON(t,h,http.MethodPost,"/api/target-selection-studies",map[string]any{})
	if rec.Code!=http.StatusBadRequest {t.Fatalf("missing case id should be rejected, got %d",rec.Code)}
}
