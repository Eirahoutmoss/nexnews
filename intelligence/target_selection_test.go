package intelligence

import (
	"testing"
	"time"
)

func targetTestTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil { panic(err) }
	return t
}

func targetTestEvidence(id string) ResearchEvidence {
	return ResearchEvidence{
		ID:id, Kind:ResearchEvidenceBackground, Title:"Evidence record "+id,
		SourceURL:"https://example.org/"+id, Publisher:"Example Research Source",
		PublisherDomain:"example.org", RetrievedAt:targetTestTime("2025-01-01T00:00:00Z"),
		Excerpt:"A source-linked fixture record for a deterministic unit test.",
		ContentLevel:ResearchContentStructured, SourceKind:ResearchSourceDataset,
		SourceOrigin:ResearchOriginOriginal,
	}
}

func targetTestStudy() (TargetSelectionStudy, []ResearchEvidence) {
	start:=targetTestTime("2024-01-01T00:00:00Z")
	end:=targetTestTime("2024-12-31T23:59:59Z")
	periodStart:=targetTestTime("2023-01-01T00:00:00Z")
	periodEnd:=targetTestTime("2025-01-01T00:00:00Z")
	ownership:=func(category TargetOwnershipCategory) []TargetOwnershipPeriod {
		return []TargetOwnershipPeriod{{ValidFrom:periodStart,ValidTo:&periodEnd,Category:category,Details:"Documented ownership category for test fixture.",EvidenceIDs:[]string{"ownership"}}}
	}
	asset:=func(id string,category TargetOwnershipCategory) TargetAsset {
		return TargetAsset{ID:id,Name:"Facility "+id,AssetType:"refinery",InclusionRationale:"Included by a stated and consistent inclusion rule.",EligibilityEvidenceIDs:[]string{"eligibility"},OwnershipHistory:ownership(category)}
	}
	study:=TargetSelectionStudy{
		Title:"Test target selection study",
		Question:"Does recorded targeting vary across ownership categories?",
		PeriodStart:start,PeriodEnd:end,
		UniverseDefinition:"All facilities of the defined asset type in the defined geography and period.",
		UniverseComplete:true,
		Assets:[]TargetAsset{
			asset("f1",TargetOwnershipForeignParticipation),
			asset("f2",TargetOwnershipForeignParticipation),
			asset("n1",TargetOwnershipNoForeignParticipation),
			asset("n2",TargetOwnershipNoForeignParticipation),
		},
		Incidents:[]TargetIncident{
			{ID:"i1",AssetID:"f1",Actor:"Actor X",OccurredAt:targetTestTime("2024-03-01T00:00:00Z"),Status:TargetIncidentAttemptReported,Description:"A reported attempt.",EvidenceIDs:[]string{"incident"}},
			{ID:"i2",AssetID:"n1",Actor:"Actor X",OccurredAt:targetTestTime("2024-04-01T00:00:00Z"),Status:TargetIncidentClaimOnly,Description:"An attributed targeting claim only.",EvidenceIDs:[]string{"claim"}},
		},
	}
	return study,[]ResearchEvidence{targetTestEvidence("ownership"),targetTestEvidence("eligibility"),targetTestEvidence("incident"),targetTestEvidence("claim")}
}

func TestAnalyzeTargetSelectionSeparatesClaimOnlyFromTargetingObservations(t *testing.T) {
	study,evidence:=targetTestStudy()
	report,err:=AnalyzeTargetSelection(study,evidence)
	if err!=nil {t.Fatal(err)}
	if report.ComparisonStatus!="descriptive_comparison_only" {t.Fatalf("unexpected status: %s",report.ComparisonStatus)}
	if report.IncidentRecordCount!=1 || report.ClaimOnlyRecordCount!=1 {t.Fatalf("claims and incidents mixed: %+v",report)}
	if report.DescriptiveDifferencePoints==nil || *report.DescriptiveDifferencePoints!=50 {t.Fatalf("unexpected descriptive difference: %+v",report.DescriptiveDifferencePoints)}
	foreign,noForeign:=report.Groups[0],report.Groups[1]
	if foreign.AssetsWithRecordedTargeting!=1 || noForeign.AssetsWithRecordedTargeting!=0 {t.Fatalf("unexpected observed asset counts: %+v %+v",foreign,noForeign)}
	if noForeign.AssetsWithClaimOnly!=1 {t.Fatalf("claim-only asset should be reported separately: %+v",noForeign)}
}

func TestAnalyzeTargetSelectionRefusesInferenceFromIncompleteUniverse(t *testing.T) {
	study,evidence:=targetTestStudy()
	study.UniverseComplete=false
	report,err:=AnalyzeTargetSelection(study,evidence)
	if err!=nil {t.Fatal(err)}
	if report.ComparisonStatus!="incomplete_universe" {t.Fatalf("unexpected status: %s",report.ComparisonStatus)}
	if report.DescriptiveDifferencePoints!=nil {t.Fatalf("incomplete universe should not publish group difference: %v",*report.DescriptiveDifferencePoints)}
}

func TestAnalyzeTargetSelectionExcludesIncompleteOwnershipCoverage(t *testing.T) {
	study,evidence:=targetTestStudy()
	periodStart:=targetTestTime("2024-02-01T00:00:00Z")
	periodEnd:=targetTestTime("2024-11-01T00:00:00Z")
	study.Assets[0].OwnershipHistory=[]TargetOwnershipPeriod{{
		ValidFrom:periodStart,ValidTo:&periodEnd,Category:TargetOwnershipForeignParticipation,
		Details:"Ownership record covers only part of the study period.",EvidenceIDs:[]string{"ownership"},
	}}
	report,err:=AnalyzeTargetSelection(study,evidence)
	if err!=nil {t.Fatal(err)}
	if report.UnclassifiedAssetCount!=1 {t.Fatalf("expected one unclassified facility: %+v",report)}
	if report.ComparisonStatus!="incomplete_ownership_coverage" {t.Fatalf("unexpected status: %s",report.ComparisonStatus)}
	if report.DescriptiveDifferencePoints!=nil {t.Fatal("partial-period ownership must not produce a main-group rate difference")}
}

func TestValidateTargetSelectionRequiresLinkedEvidence(t *testing.T) {
	study,evidence:=targetTestStudy()
	study.Assets[0].EligibilityEvidenceIDs=[]string{"missing"}
	if err:=ValidateTargetSelectionStudy(study,evidence);err==nil {t.Fatal("expected missing evidence reference to fail")}
}

func TestValidateTargetSelectionRejectsOverlappingOwnershipPeriods(t *testing.T) {
	study,evidence:=targetTestStudy()
	period:=TargetOwnershipPeriod{
		ValidFrom:targetTestTime("2024-01-01T00:00:00Z"),Category:TargetOwnershipNoForeignParticipation,
		Details:"Conflicting ownership period.",EvidenceIDs:[]string{"ownership"},
	}
	study.Assets[0].OwnershipHistory=append(study.Assets[0].OwnershipHistory,period)
	if err:=ValidateTargetSelectionStudy(study,evidence);err==nil {t.Fatal("expected overlapping ownership records to fail")}
}

func TestValidateTargetSelectionRejectsIncidentOutsidePeriod(t *testing.T) {
	study,evidence:=targetTestStudy()
	study.Incidents[0].OccurredAt=targetTestTime("2025-02-01T00:00:00Z")
	if err:=ValidateTargetSelectionStudy(study,evidence);err==nil {t.Fatal("expected out-of-period incident to fail")}
}
