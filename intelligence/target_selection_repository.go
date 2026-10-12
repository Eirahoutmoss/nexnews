package intelligence

import (
	"errors"
	"strings"
	"time"
)

// CreateTargetSelectionStudy persists a structured target-selection comparison
// within the research case. Every inventory row, ownership period and incident
// must reference source records already held by that case.
func (r *JSONRepository) CreateTargetSelectionStudy(caseID string, study TargetSelectionStudy) (ResearchCase, error) {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		return ResearchCase{}, errors.New("research case id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	db, err := r.loadLocked()
	if err != nil {
		return ResearchCase{}, err
	}
	item, ok := db.ResearchCases[caseID]
	if !ok {
		return ResearchCase{}, errors.New("research case not found")
	}
	if err := ValidateTargetSelectionStudy(study, item.Evidence); err != nil {
		return ResearchCase{}, err
	}
	for _, prior := range item.TargetSelectionStudies {
		if strings.TrimSpace(prior.Title) == strings.TrimSpace(study.Title) &&
			prior.PeriodStart.Equal(study.PeriodStart) && prior.PeriodEnd.Equal(study.PeriodEnd) {
			return ResearchCase{}, errors.New("a target-selection study with the same title and period already exists")
		}
	}
	id, err := newResearchID("targetstudy")
	if err != nil {
		return ResearchCase{}, err
	}
	now := time.Now().UTC()
	study.ID = id
	study.Title = strings.TrimSpace(study.Title)
	study.Question = strings.TrimSpace(study.Question)
	study.UniverseDefinition = strings.TrimSpace(study.UniverseDefinition)
	study.CreatedAt = now
	study.UpdatedAt = now
	if study.Assets == nil { study.Assets = []TargetAsset{} }
	if study.Incidents == nil { study.Incidents = []TargetIncident{} }
	item.TargetSelectionStudies = append(item.TargetSelectionStudies, study)
	markResearchCaseActive(&item, now)
	item.UpdatedAt = now
	db.ResearchCases[item.ID] = item
	db.SavedAt = now
	if err := r.writeLocked(db); err != nil {
		return ResearchCase{}, err
	}
	return item, nil
}

// AnalyzeTargetSelectionStudy loads the saved structured comparison from one
// research case and evaluates it only against evidence belonging to that case.
func (r *JSONRepository) AnalyzeTargetSelectionStudy(caseID, studyID string) (TargetSelectionReport, error) {
	item, err := r.GetResearchCase(caseID)
	if err != nil { return TargetSelectionReport{}, err }
	studyID = strings.TrimSpace(studyID)
	if studyID == "" { return TargetSelectionReport{}, errors.New("target-selection study id is required") }
	for _, study := range item.TargetSelectionStudies {
		if study.ID != studyID { continue }
		return AnalyzeTargetSelection(study, item.Evidence)
	}
	return TargetSelectionReport{}, errors.New("target-selection study not found")
}
