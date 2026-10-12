package intelligence

import (
	"path/filepath"
	"testing"
)

func addTargetStudyEvidence(t *testing.T, repo *JSONRepository, caseID, title string) string {
	t.Helper()
	item, err := repo.AddResearchEvidence(caseID, AddResearchEvidenceInput{
		Kind:ResearchEvidenceDataPoint,
		Title:title,
		SourceURL:"https://data.example.org/"+titleToSlug(title),
		Publisher:"Example Data Registry",
		Excerpt:"A structured source record used by this test.",
		ContentLevel:ResearchContentStructured,
		SourceKind:ResearchSourceDataset,
		SourceOrigin:ResearchOriginOriginal,
	})
	if err != nil { t.Fatal(err) }
	for _, evidence := range item.Evidence {
		if evidence.Title == title { return evidence.ID }
	}
	t.Fatalf("evidence %q was not persisted",title)
	return ""
}

func titleToSlug(value string) string {
	out:=""
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' {
			out += string(ch)
		} else if ch == ' ' || ch == '-' || ch == '_' {
			out += "-"
		}
	}
	return out
}

func TestTargetSelectionStudyPersistsAndAnalyzesOnlyCaseEvidence(t *testing.T) {
	repo,err:=NewJSONRepository(filepath.Join(t.TempDir(),"intelligence.json"))
	if err!=nil {t.Fatal(err)}
	item,err:=repo.CreateResearchCase(CreateResearchCaseInput{Title:"Target selection",Question:"Is there a recorded target-selection pattern?"})
	if err!=nil {t.Fatal(err)}
	eligibilityID:=addTargetStudyEvidence(t,repo,item.ID,"Eligibility source")
	ownershipID:=addTargetStudyEvidence(t,repo,item.ID,"Ownership source")
	incidentID:=addTargetStudyEvidence(t,repo,item.ID,"Incident source")

	start:=targetTestTime("2024-01-01T00:00:00Z")
	end:=targetTestTime("2024-12-31T23:59:59Z")
	periodStart:=targetTestTime("2023-01-01T00:00:00Z")
	periodEnd:=targetTestTime("2025-01-01T00:00:00Z")
	period:=func(category TargetOwnershipCategory) []TargetOwnershipPeriod {
		return []TargetOwnershipPeriod{{ValidFrom:periodStart,ValidTo:&periodEnd,Category:category,Details:"Source-linked ownership classification.",EvidenceIDs:[]string{ownershipID}}}
	}
	asset:=func(id string,category TargetOwnershipCategory) TargetAsset {
		return TargetAsset{ID:id,Name:"Facility "+id,AssetType:"refinery",InclusionRationale:"Included according to the defined asset universe.",EligibilityEvidenceIDs:[]string{eligibilityID},OwnershipHistory:period(category)}
	}
	study:=TargetSelectionStudy{
		Title:"Ownership and recorded targeting",
		Question:"Does the recorded targeting rate vary between the two categories?",
		PeriodStart:start,PeriodEnd:end,UniverseDefinition:"All eligible refinery facilities in the stated geography and period.",UniverseComplete:true,
		Assets:[]TargetAsset{asset("foreign-1",TargetOwnershipForeignParticipation),asset("local-1",TargetOwnershipNoForeignParticipation)},
		Incidents:[]TargetIncident{{ID:"incident-1",AssetID:"foreign-1",Actor:"Actor X",OccurredAt:targetTestTime("2024-06-01T00:00:00Z"),Status:TargetIncidentAttemptReported,Description:"Reported attempt, sourced but not independently confirmed by this engine.",EvidenceIDs:[]string{incidentID}}},
	}
	updated,err:=repo.CreateTargetSelectionStudy(item.ID,study)
	if err!=nil {t.Fatal(err)}
	if len(updated.TargetSelectionStudies)!=1 || updated.TargetSelectionStudies[0].ID=="" {t.Fatalf("study was not persisted: %+v",updated.TargetSelectionStudies)}
	if updated.Status!=ResearchCaseActive {t.Fatalf("expected research case to be active, got %s",updated.Status)}

	report,err:=repo.AnalyzeTargetSelectionStudy(item.ID,updated.TargetSelectionStudies[0].ID)
	if err!=nil {t.Fatal(err)}
	if report.ComparisonStatus!="descriptive_comparison_only" {t.Fatalf("unexpected report: %+v",report)}
	if report.Groups[0].AssetsWithRecordedTargeting!=1 || report.Groups[1].AssetsWithRecordedTargeting!=0 {t.Fatalf("targeting counts incorrect: %+v",report.Groups)}

	reloaded,err:=repo.GetResearchCase(item.ID)
	if err!=nil {t.Fatal(err)}
	if len(reloaded.TargetSelectionStudies)!=1 || reloaded.TargetSelectionStudies[0].UpdatedAt.IsZero() {t.Fatalf("persisted study did not survive reload: %+v",reloaded.TargetSelectionStudies)}
}

func TestCreateTargetSelectionStudyCannotUseEvidenceFromAnotherCase(t *testing.T) {
	repo,err:=NewJSONRepository(filepath.Join(t.TempDir(),"intelligence.json"))
	if err!=nil {t.Fatal(err)}
	one,err:=repo.CreateResearchCase(CreateResearchCaseInput{Title:"Case one",Question:"First question"})
	if err!=nil {t.Fatal(err)}
	two,err:=repo.CreateResearchCase(CreateResearchCaseInput{Title:"Case two",Question:"Second question"})
	if err!=nil {t.Fatal(err)}
	evidenceID:=addTargetStudyEvidence(t,repo,one.ID,"Only in case one")
	study,_:=targetTestStudy()
	for i:=range study.Assets {
		study.Assets[i].EligibilityEvidenceIDs=[]string{evidenceID}
		for j:=range study.Assets[i].OwnershipHistory { study.Assets[i].OwnershipHistory[j].EvidenceIDs=[]string{evidenceID} }
	}
	for i:=range study.Incidents { study.Incidents[i].EvidenceIDs=[]string{evidenceID} }
	if _,err:=repo.CreateTargetSelectionStudy(two.ID,study);err==nil {t.Fatal("cross-case evidence reference should be rejected")}
}

func TestCreateTargetSelectionStudyRejectsDuplicateTitleAndPeriod(t *testing.T) {
	repo,err:=NewJSONRepository(filepath.Join(t.TempDir(),"intelligence.json"))
	if err!=nil {t.Fatal(err)}
	item,err:=repo.CreateResearchCase(CreateResearchCaseInput{Title:"Target study",Question:"Question"})
	if err!=nil {t.Fatal(err)}
	elig:=addTargetStudyEvidence(t,repo,item.ID,"Eligible source")
	owner:=addTargetStudyEvidence(t,repo,item.ID,"Owner source")
	study,_:=targetTestStudy()
	for i:=range study.Assets {
		study.Assets[i].EligibilityEvidenceIDs=[]string{elig}
		for j:=range study.Assets[i].OwnershipHistory { study.Assets[i].OwnershipHistory[j].EvidenceIDs=[]string{owner} }
	}
	for i:=range study.Incidents { study.Incidents[i].EvidenceIDs=[]string{owner} }
	created,err:=repo.CreateTargetSelectionStudy(item.ID,study)
	if err!=nil {t.Fatal(err)}
	if _,err:=repo.CreateTargetSelectionStudy(item.ID,study);err==nil {t.Fatal("duplicate study should be rejected")}
	if len(created.TargetSelectionStudies)!=1 {t.Fatalf("unexpected duplicate persistence")}
}
