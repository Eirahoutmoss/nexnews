package intelligence

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type TargetOwnershipCategory string

const (
	TargetOwnershipForeignParticipation    TargetOwnershipCategory = "foreign_participation"
	TargetOwnershipNoForeignParticipation TargetOwnershipCategory = "no_foreign_participation"
	TargetOwnershipUnknown                TargetOwnershipCategory = "unknown"
)

type TargetIncidentStatus string

const (
	// ClaimOnly means an actor or source claimed the target, but the record does
	// not establish that an attempt occurred.
	TargetIncidentClaimOnly TargetIncidentStatus = "claim_only"
	TargetIncidentAttemptReported TargetIncidentStatus = "attempt_reported"
	TargetIncidentIntercepted TargetIncidentStatus = "intercepted"
	TargetIncidentImpactReported TargetIncidentStatus = "impact_reported"
	TargetIncidentDamageReported TargetIncidentStatus = "damage_reported"
)

type TargetOwnershipPeriod struct {
	ValidFrom  time.Time              `json:"validFrom"`
	ValidTo    *time.Time             `json:"validTo,omitempty"`
	Category   TargetOwnershipCategory `json:"category"`
	Details    string                 `json:"details"`
	EvidenceIDs []string               `json:"evidenceIds"`
}

type TargetAsset struct {
	ID                     string                 `json:"id"`
	Name                   string                 `json:"name"`
	AssetType              string                 `json:"assetType"`
	InclusionRationale     string                 `json:"inclusionRationale"`
	EligibilityEvidenceIDs []string               `json:"eligibilityEvidenceIds"`
	OwnershipHistory       []TargetOwnershipPeriod `json:"ownershipHistory"`
}

type TargetIncident struct {
	ID          string               `json:"id"`
	AssetID     string               `json:"assetId"`
	Actor       string               `json:"actor"`
	OccurredAt  time.Time            `json:"occurredAt"`
	Status      TargetIncidentStatus `json:"status"`
	Description string               `json:"description"`
	EvidenceIDs []string             `json:"evidenceIds"`
}

type TargetSelectionStudy struct {
	Title                string           `json:"title"`
	Question             string           `json:"question"`
	PeriodStart          time.Time        `json:"periodStart"`
	PeriodEnd            time.Time        `json:"periodEnd"`
	UniverseDefinition   string           `json:"universeDefinition"`
	UniverseComplete     bool             `json:"universeComplete"`
	Assets               []TargetAsset    `json:"assets"`
	Incidents            []TargetIncident `json:"incidents"`
}

type TargetSelectionGroup struct {
	Category                     TargetOwnershipCategory `json:"category"`
	EligibleAssetCount           int                    `json:"eligibleAssetCount"`
	AssetsWithRecordedTargeting  int                    `json:"assetsWithRecordedTargeting"`
	AssetsWithClaimOnly          int                    `json:"assetsWithClaimOnly"`
	RecordedTargetingIncidents   int                    `json:"recordedTargetingIncidents"`
	ClaimOnlyRecords             int                    `json:"claimOnlyRecords"`
	TargetedAssetRatePercent     float64                `json:"targetedAssetRatePercent"`
	RateAvailable                bool                   `json:"rateAvailable"`
	SmallDenominator             bool                   `json:"smallDenominator"`
}

type TargetSelectionReport struct {
	Title                         string                 `json:"title"`
	Question                      string                 `json:"question"`
	PeriodStart                   time.Time              `json:"periodStart"`
	PeriodEnd                     time.Time              `json:"periodEnd"`
	UniverseDefinition            string                 `json:"universeDefinition"`
	UniverseDeclaredComplete      bool                   `json:"universeDeclaredComplete"`
	UniverseAssetCount            int                    `json:"universeAssetCount"`
	ClassifiedAssetCount           int                    `json:"classifiedAssetCount"`
	UnclassifiedAssetCount         int                    `json:"unclassifiedAssetCount"`
	IncidentRecordCount            int                    `json:"incidentRecordCount"`
	ClaimOnlyRecordCount           int                    `json:"claimOnlyRecordCount"`
	ComparisonStatus               string                 `json:"comparisonStatus"`
	DescriptiveDifferencePoints    *float64               `json:"descriptiveDifferencePercentagePoints,omitempty"`
	Groups                         []TargetSelectionGroup `json:"groups"`
	Warnings                       []string               `json:"warnings"`
	InterpretationLimitations      []string               `json:"interpretationLimitations"`
}

var targetOwnershipCategories = map[TargetOwnershipCategory]bool{
	TargetOwnershipForeignParticipation: true,
	TargetOwnershipNoForeignParticipation: true,
	TargetOwnershipUnknown: true,
}

func validTargetIncidentStatus(v TargetIncidentStatus) bool {
	switch v {
	case TargetIncidentClaimOnly, TargetIncidentAttemptReported, TargetIncidentIntercepted,
		TargetIncidentImpactReported, TargetIncidentDamageReported:
		return true
	default:
		return false
	}
}

func targetEvidenceExists(id string, evidenceByID map[string]ResearchEvidence) bool {
	_, ok := evidenceByID[strings.TrimSpace(id)]
	return strings.TrimSpace(id) != "" && ok
}

func validateTargetEvidenceIDs(ids []string, evidenceByID map[string]ResearchEvidence, field string) error {
	if len(ids) == 0 {
		return fmt.Errorf("%s must reference at least one saved source/evidence record", field)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return fmt.Errorf("%s contains an empty or duplicate evidence id", field)
		}
		if !targetEvidenceExists(id, evidenceByID) {
			return fmt.Errorf("%s references missing research evidence %q", field, id)
		}
	}
	return nil
}

func validateTargetSelectionStudy(study TargetSelectionStudy, evidence []ResearchEvidence) error {
	switch {
	case strings.TrimSpace(study.Title) == "":
		return errors.New("target-selection study title is required")
	case strings.TrimSpace(study.Question) == "":
		return errors.New("target-selection research question is required")
	case study.PeriodStart.IsZero() || study.PeriodEnd.IsZero() || study.PeriodEnd.Before(study.PeriodStart):
		return errors.New("target-selection study period is invalid")
	case strings.TrimSpace(study.UniverseDefinition) == "":
		return errors.New("comparison universe definition is required")
	case len(study.Assets) == 0:
		return errors.New("at least one eligible asset is required")
	}

	evidenceByID := make(map[string]ResearchEvidence, len(evidence))
	for _, item := range evidence {
		evidenceByID[item.ID] = item
	}
	assetIDs := make(map[string]bool, len(study.Assets))
	for _, asset := range study.Assets {
		asset.ID = strings.TrimSpace(asset.ID)
		if asset.ID == "" || assetIDs[asset.ID] {
			return errors.New("asset ids must be non-empty and unique")
		}
		assetIDs[asset.ID] = true
		if strings.TrimSpace(asset.Name) == "" || strings.TrimSpace(asset.AssetType) == "" {
			return fmt.Errorf("asset %q requires a name and asset type", asset.ID)
		}
		if strings.TrimSpace(asset.InclusionRationale) == "" {
			return fmt.Errorf("asset %q requires a comparison-universe inclusion rationale", asset.ID)
		}
		if err := validateTargetEvidenceIDs(asset.EligibilityEvidenceIDs, evidenceByID, "asset "+asset.ID+" eligibility"); err != nil {
			return err
		}
		if len(asset.OwnershipHistory) == 0 {
			return fmt.Errorf("asset %q requires time-valid ownership information or an explicit unknown record", asset.ID)
		}
		periods := append([]TargetOwnershipPeriod(nil), asset.OwnershipHistory...)
		sort.Slice(periods, func(i, j int) bool { return periods[i].ValidFrom.Before(periods[j].ValidFrom) })
		for i, period := range periods {
			if period.ValidFrom.IsZero() || (period.ValidTo != nil && period.ValidTo.Before(period.ValidFrom)) {
				return fmt.Errorf("asset %q has an invalid ownership period", asset.ID)
			}
			if !targetOwnershipCategories[period.Category] {
				return fmt.Errorf("asset %q has an invalid foreign-participation category", asset.ID)
			}
			if strings.TrimSpace(period.Details) == "" {
				return fmt.Errorf("asset %q ownership period requires a rationale/details field", asset.ID)
			}
			if err := validateTargetEvidenceIDs(period.EvidenceIDs, evidenceByID, "asset "+asset.ID+" ownership"); err != nil {
				return err
			}
			if i > 0 {
				previous := periods[i-1]
				if previous.ValidTo == nil || previous.ValidTo.After(period.ValidFrom) {
					return fmt.Errorf("asset %q has overlapping ownership periods", asset.ID)
				}
			}
		}
	}

	incidentIDs := map[string]bool{}
	for _, incident := range study.Incidents {
		if strings.TrimSpace(incident.ID) == "" || incidentIDs[incident.ID] {
			return errors.New("incident ids must be non-empty and unique")
		}
		incidentIDs[incident.ID] = true
		if !assetIDs[strings.TrimSpace(incident.AssetID)] {
			return fmt.Errorf("incident %q references an asset outside the comparison universe", incident.ID)
		}
		if strings.TrimSpace(incident.Actor) == "" || strings.TrimSpace(incident.Description) == "" {
			return fmt.Errorf("incident %q requires an actor and a description", incident.ID)
		}
		if incident.OccurredAt.IsZero() || incident.OccurredAt.Before(study.PeriodStart) || incident.OccurredAt.After(study.PeriodEnd) {
			return fmt.Errorf("incident %q is outside the study period or has no event time", incident.ID)
		}
		if !validTargetIncidentStatus(incident.Status) {
			return fmt.Errorf("incident %q has an invalid evidence status", incident.ID)
		}
		if err := validateTargetEvidenceIDs(incident.EvidenceIDs, evidenceByID, "incident "+incident.ID); err != nil {
			return err
		}
	}
	return nil
}

func ownershipCategoryForWholePeriod(asset TargetAsset, study TargetSelectionStudy) (TargetOwnershipCategory, bool) {
	var categories []TargetOwnershipCategory
	for _, period := range asset.OwnershipHistory {
		coversStart := !period.ValidFrom.After(study.PeriodStart)
		coversEnd := period.ValidTo == nil || !period.ValidTo.Before(study.PeriodEnd)
		if coversStart && coversEnd {
			categories = append(categories, period.Category)
		}
	}
	if len(categories) != 1 || categories[0] == TargetOwnershipUnknown {
		return TargetOwnershipUnknown, false
	}
	return categories[0], true
}

// AnalyzeTargetSelection compares the fraction of included facilities with at
// least one recorded targeting observation across evidence-backed ownership
// categories. It intentionally emits descriptive coverage only: this is not a
// test of attacker intent, causality, or statistical significance.
func AnalyzeTargetSelection(study TargetSelectionStudy, evidence []ResearchEvidence) (TargetSelectionReport, error) {
	if err := validateTargetSelectionStudy(study, evidence); err != nil {
		return TargetSelectionReport{}, err
	}
	evidenceByID := make(map[string]ResearchEvidence, len(evidence))
	for _, item := range evidence {
		evidenceByID[item.ID] = item
	}
	report := TargetSelectionReport{
		Title: study.Title, Question: study.Question,
		PeriodStart: study.PeriodStart, PeriodEnd: study.PeriodEnd,
		UniverseDefinition: study.UniverseDefinition,
		UniverseDeclaredComplete: study.UniverseComplete,
		UniverseAssetCount: len(study.Assets),
		ComparisonStatus: "insufficient_comparison_groups",
		Groups: []TargetSelectionGroup{},
		Warnings: []string{},
		InterpretationLimitations: []string{
			"Bu rapor kaydedilmiş gözlemlerin betimsel karşılaştırmasıdır; saldırganın niyetini, nedenselliği veya istatistiksel anlamlılığı kanıtlamaz.",
			"Bir kaynakta hedef gösterilmesi, fiilî saldırı veya hasar meydana geldiğini tek başına doğrulamaz.",
			"Sonuçlar yalnızca bu dosyaya alınmış tesisler, dönem ve kaynaklar kadar kapsayıcıdır.",
			"Yayıncı sayısı bağımsız doğrulama sayısı değildir; kaynak kökeni ve kanıt kalitesi ayrıca değerlendirilmelidir.",
		},
	}

	groupIndex := map[TargetOwnershipCategory]int{}
	for _, category := range []TargetOwnershipCategory{
		TargetOwnershipForeignParticipation,
		TargetOwnershipNoForeignParticipation,
		TargetOwnershipUnknown,
	} {
		groupIndex[category] = len(report.Groups)
		report.Groups = append(report.Groups, TargetSelectionGroup{Category: category})
	}

	assetGroup := make(map[string]TargetOwnershipCategory, len(study.Assets))
	targetedAssets := map[string]bool{}
	claimOnlyAssets := map[string]bool{}
	for _, asset := range study.Assets {
		category, classified := ownershipCategoryForWholePeriod(asset, study)
		if !classified {
			report.UnclassifiedAssetCount++
			category = TargetOwnershipUnknown
		} else {
			report.ClassifiedAssetCount++
		}
		assetGroup[asset.ID] = category
		report.Groups[groupIndex[category]].EligibleAssetCount++
	}

	targetingPerGroup := make([]int, len(report.Groups))
	claimPerGroup := make([]int, len(report.Groups))
	for _, incident := range study.Incidents {
		if incident.Status == TargetIncidentClaimOnly {
			report.ClaimOnlyRecordCount++
			claimPerGroup[groupIndex[assetGroup[incident.AssetID]]]++
			claimOnlyAssets[incident.AssetID] = true
			continue
		}
		report.IncidentRecordCount++
		targetingPerGroup[groupIndex[assetGroup[incident.AssetID]]]++
		targetedAssets[incident.AssetID] = true
	}
	for assetID := range targetedAssets {
		report.Groups[groupIndex[assetGroup[assetID]]].AssetsWithRecordedTargeting++
	}
	for assetID := range claimOnlyAssets {
		if !targetedAssets[assetID] {
			report.Groups[groupIndex[assetGroup[assetID]]].AssetsWithClaimOnly++
		}
	}
	for i := range report.Groups {
		group := &report.Groups[i]
		group.RecordedTargetingIncidents = targetingPerGroup[i]
		group.ClaimOnlyRecords = claimPerGroup[i]
		if group.EligibleAssetCount > 0 {
			group.RateAvailable = true
			group.TargetedAssetRatePercent = float64(group.AssetsWithRecordedTargeting) * 100 / float64(group.EligibleAssetCount)
		}
		group.SmallDenominator = group.EligibleAssetCount > 0 && group.EligibleAssetCount < 5
	}

	foreign := report.Groups[groupIndex[TargetOwnershipForeignParticipation]]
	noForeign := report.Groups[groupIndex[TargetOwnershipNoForeignParticipation]]
	switch {
	case !study.UniverseComplete:
		report.ComparisonStatus = "incomplete_universe"
		report.Warnings = append(report.Warnings, "Karşılaştırma evreni eksiksiz olarak işaretlenmedi. Bu oranlardan hedef seçimi genellemesi yapılamaz.")
	case report.UnclassifiedAssetCount > 0:
		report.ComparisonStatus = "incomplete_ownership_coverage"
		report.Warnings = append(report.Warnings, "Bazı tesislerin sahiplik kategorisi dönemin tamamı için kanıtlanamadı; bu tesisler iki ana karşılaştırma grubuna dahil edilmedi.")
	case foreign.EligibleAssetCount == 0 || noForeign.EligibleAssetCount == 0:
		report.ComparisonStatus = "insufficient_comparison_groups"
		report.Warnings = append(report.Warnings, "Yabancı katılımı bulunan ve bulunmayan tesislerin her iki grubu da gerekli; karşılaştırma için grup verisi eksik.")
	default:
		report.ComparisonStatus = "descriptive_comparison_only"
		difference := foreign.TargetedAssetRatePercent - noForeign.TargetedAssetRatePercent
		report.DescriptiveDifferencePoints = &difference
		report.Warnings = append(report.Warnings, "Gruplar arasındaki fark betimseldir; tesislerin stratejik önemi, konumu, savunması, maruziyet süresi ve raporlama olasılığı kontrol edilmemiştir.")
	}
	for _, group := range report.Groups {
		if group.SmallDenominator {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s grubunda yalnızca %d tesis var; tek bir kayıt oranı belirgin biçimde değiştirebilir.", group.Category, group.EligibleAssetCount))
		}
	}
	if report.IncidentRecordCount == 0 {
		report.Warnings = append(report.Warnings, "Kaydedilmiş hedefleme gözlemi yok. Yalnızca hedef iddiaları varsa fiilî olay varmış gibi sayılmadı.")
	}
	unknownOriginCount := 0
	for _, incident := range study.Incidents {
		for _, id := range incident.EvidenceIDs {
			if evidenceByID[id].SourceOrigin == ResearchOriginUnknown {
				unknownOriginCount++
			}
		}
	}
	if unknownOriginCount > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d olay-kanıt bağlantısında kaynağın özgün, resmî veya yeniden yayımlanmış kökeni bilinmiyor.", unknownOriginCount))
	}
	return report, nil
}

// ValidateTargetSelectionStudy exposes structural/provenance validation to
// importers and API handlers without running the comparison.
func ValidateTargetSelectionStudy(study TargetSelectionStudy, evidence []ResearchEvidence) error {
	return validateTargetSelectionStudy(study, evidence)
}

// validResearchEvidenceURL is retained here for import adapters that need a
// non-network URL check; the engine never fetches the source itself.
func validResearchEvidenceURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && u.Hostname() != "" && (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https"))
}
