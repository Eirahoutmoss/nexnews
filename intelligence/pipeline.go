package intelligence

import (
	"fmt"
	"time"
)

// PipelineStats summarizes one explicit local analysis pass. Stages persist
// independently, so Partial is true when an earlier stage completed before a
// later stage returned an error. Retrying is safe because record IDs are stable.
type PipelineStats struct {
	StartedAt       time.Time                `json:"startedAt"`
	CompletedAt     *time.Time               `json:"completedAt,omitempty"`
	Partial         bool                     `json:"partial"`
	Statements      StatementExtractionStats `json:"statements"`
	Entities        EntityExtractionStats    `json:"entities"`
	Warnings        []string                 `json:"warnings,omitempty"`
	FailedStage     string                   `json:"failedStage,omitempty"`
}

// AnalyzeStoredContent runs the two conservative, local extraction stages over
// content already in the Intelligence repository: literal source statements,
// then exact lexicon-based entity mentions. It does not fetch articles, infer
// truth, perform entity disambiguation, or create semantic relations.
//
// The stages are persisted independently. If stage two fails after stage one
// succeeds, Partial and FailedStage make that visible; a retry is idempotent.
func (r *JSONRepository) AnalyzeStoredContent() (PipelineStats, error) {
	started := time.Now().UTC()
	out := PipelineStats{
		StartedAt: started,
		Warnings: []string{},
	}

	statements, err := r.ExtractStoredSourceStatements()
	out.Statements = statements
	if err != nil {
		out.FailedStage = "source_statements"
		return out, fmt.Errorf("source-statement extraction: %w", err)
	}
	out.Warnings = append(out.Warnings, statements.Warnings...)

	entities, err := r.ExtractStoredEntityCandidates()
	out.Entities = entities
	if err != nil {
		out.Partial = true
		out.FailedStage = "entity_candidates"
		out.Warnings = append(out.Warnings, "Source statements were persisted, but entity candidate extraction failed. Retry is safe.")
		return out, fmt.Errorf("entity-candidate extraction after successful source statements: %w", err)
	}
	out.Warnings = append(out.Warnings, entities.Warnings...)

	completed := time.Now().UTC()
	out.CompletedAt = &completed
	return out, nil
}
