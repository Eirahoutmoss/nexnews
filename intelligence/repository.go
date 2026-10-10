package intelligence

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"nexnews/core"
)

const databaseVersion = 1

type Database struct {
	Version   int                `json:"version"`
	Schema    string             `json:"schema"`
	SavedAt   time.Time          `json:"savedAt"`
	LastSyncAt time.Time         `json:"lastSyncAt,omitempty"`
	Sources   map[string]Source  `json:"sources"`
	Articles  map[string]Article `json:"articles"`
	Events    map[string]Event   `json:"events"`
	Claims    map[string]Claim   `json:"claims"`
	Evidence  map[string]Evidence `json:"evidence"`
	Relations map[string]Relation `json:"relations"`
	Timeline  map[string]TimelineEntry `json:"timeline"`
	Runs      map[string]AnalysisRun `json:"analysisRuns"`
}

// JSONRepository is a small dependency-free persistence implementation for the
// Intelligence database. It writes a file separate from NexNews feed-cache.json.
type JSONRepository struct {
	path string
	mu   sync.Mutex
}

func NewJSONRepository(path string) (*JSONRepository, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("Intelligence database path is required")
	}
	if strings.HasSuffix(path, string(os.PathSeparator)) {
		return nil, errors.New("Intelligence database path must name a file")
	}
	return &JSONRepository{path: path}, nil
}

func emptyDatabase() Database {
	return Database{
		Version: databaseVersion, Schema: SchemaVersion,
		Sources: map[string]Source{}, Articles: map[string]Article{}, Events: map[string]Event{},
		Claims: map[string]Claim{}, Evidence: map[string]Evidence{}, Relations: map[string]Relation{},
		Timeline: map[string]TimelineEntry{}, Runs: map[string]AnalysisRun{},
	}
}

func ensureDatabaseMaps(db *Database) {
	if db.Sources == nil { db.Sources = map[string]Source{} }
	if db.Articles == nil { db.Articles = map[string]Article{} }
	if db.Events == nil { db.Events = map[string]Event{} }
	if db.Claims == nil { db.Claims = map[string]Claim{} }
	if db.Evidence == nil { db.Evidence = map[string]Evidence{} }
	if db.Relations == nil { db.Relations = map[string]Relation{} }
	if db.Timeline == nil { db.Timeline = map[string]TimelineEntry{} }
	if db.Runs == nil { db.Runs = map[string]AnalysisRun{} }
}

func (r *JSONRepository) loadLocked() (Database, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyDatabase(), nil
	}
	if err != nil {
		return Database{}, err
	}
	var db Database
	if err := json.Unmarshal(data, &db); err != nil {
		return Database{}, fmt.Errorf("decode Intelligence database: %w", err)
	}
	if db.Version != databaseVersion || db.Schema != SchemaVersion {
		return Database{}, fmt.Errorf("unsupported Intelligence database version=%d schema=%q", db.Version, db.Schema)
	}
	ensureDatabaseMaps(&db)
	return db, nil
}

// Load returns the persisted database. It returns a new empty database if the
// configured file does not exist; malformed or incompatible files are errors.
func (r *JSONRepository) Load() (Database, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.loadLocked()
}

// SyncFromNexNews imports a read-only view of NexNews. It never changes NexNews
// feed data and never fabricates claims, evidence, or relations.
func (r *JSONRepository) SyncFromNexNews(input core.IntelligenceSnapshot) (SyncStats, error) {
	batch, err := FromNexNews(input)
	if err != nil {
		return SyncStats{}, fmt.Errorf("adapt NexNews snapshot: %w", err)
	}
	return r.Sync(batch)
}

// SyncCurrentNexNews takes a read-only snapshot from the active NexNews core
// singleton and persists it. Call explicitly from a user-triggered action or a
// controlled background job; constructing the repository does not start syncs.
func (r *JSONRepository) SyncCurrentNexNews() (SyncStats, error) {
	return r.SyncFromNexNews(core.CurrentIntelligenceSnapshot())
}

// SyncSnapshotJSON accepts the JSON contract exposed by the mobile bridge or
// another process. Error envelopes and malformed snapshots are rejected rather
// than interpreted as an empty successful sync.
func (r *JSONRepository) SyncSnapshotJSON(payload []byte) (SyncStats, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return SyncStats{}, fmt.Errorf("decode NexNews snapshot envelope: %w", err)
	}
	if rawError, ok := envelope["error"]; ok {
		var message string
		if json.Unmarshal(rawError, &message) == nil && strings.TrimSpace(message) != "" {
			return SyncStats{}, fmt.Errorf("NexNews snapshot unavailable: %s", message)
		}
	}
	var snapshot core.IntelligenceSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return SyncStats{}, fmt.Errorf("decode NexNews snapshot: %w", err)
	}
	if snapshot.CapturedAt.IsZero() {
		return SyncStats{}, errors.New("NexNews snapshot is missing capturedAt")
	}
	return r.SyncFromNexNews(snapshot)
}

// Sync persists a normalized import batch. First-observed timestamps are kept
// across updates; last-observed timestamps advance with each successful sync.
func (r *JSONRepository) Sync(batch ImportSnapshot) (SyncStats, error) {
	if batch.SchemaVersion != SchemaVersion {
		return SyncStats{}, fmt.Errorf("import schema %q does not match %q", batch.SchemaVersion, SchemaVersion)
	}
	captured := batch.CapturedAt.UTC()
	if captured.IsZero() {
		captured = time.Now().UTC()
	}

	stats := SyncStats{
		SourcesSeen: len(batch.Sources), ArticlesSeen: len(batch.Articles), EventsSeen: len(batch.Events),
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	db, err := r.loadLocked()
	if err != nil {
		return SyncStats{}, err
	}
	now := captured

	for _, src := range batch.Sources {
		if strings.TrimSpace(src.ID) == "" || strings.TrimSpace(src.CanonicalDomain) == "" {
			return SyncStats{}, errors.New("source id and canonical domain are required")
		}
		if prior, ok := db.Sources[src.ID]; ok {
			if !prior.CreatedAt.IsZero() {
				src.CreatedAt = prior.CreatedAt
			}
			if strings.TrimSpace(src.Name) == "" {
				src.Name = prior.Name
			}
			if src.IdentityBasis == "" {
				src.IdentityBasis = prior.IdentityBasis
			}
		}
		db.Sources[src.ID] = src
	}
	for _, a := range batch.Articles {
		if err := ValidateArticle(a); err != nil {
			return SyncStats{}, fmt.Errorf("article %q: %w", a.ID, err)
		}
		if _, ok := db.Sources[a.PublisherSourceID]; !ok {
			return SyncStats{}, fmt.Errorf("article %q references unknown source %q", a.ID, a.PublisherSourceID)
		}
		prior, exists := db.Articles[a.ID]
		if exists {
			if !prior.FirstObservedAt.IsZero() {
				a.FirstObservedAt = prior.FirstObservedAt
			}
			a.LastObservedAt = now
			if a.PublishedAt == nil {
				a.PublishedAt = prior.PublishedAt
			}
			if a.Summary == "" {
				a.Summary = prior.Summary
				a.ContentAvailability = prior.ContentAvailability
			}
			if a.ContentHash == "" {
				a.ContentHash = prior.ContentHash
			}
			if a.Language == "" {
				a.Language = prior.Language
			}
			if a.RetrievedAt.Before(prior.RetrievedAt) {
				a.RetrievedAt = prior.RetrievedAt
			}
		} else {
			if a.FirstObservedAt.IsZero() {
				a.FirstObservedAt = now
			}
			if a.LastObservedAt.IsZero() {
				a.LastObservedAt = now
			}
		}
		db.Articles[a.ID] = a
	}
	for _, e := range batch.Events {
		if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Title) == "" {
			return SyncStats{}, errors.New("event id and title are required")
		}
		for _, articleID := range e.ArticleIDs {
			if _, ok := db.Articles[articleID]; !ok {
				return SyncStats{}, fmt.Errorf("event %q references unknown article %q", e.ID, articleID)
			}
		}
		prior, exists := db.Events[e.ID]
		if exists {
			if !prior.FirstObservedAt.IsZero() {
				e.FirstObservedAt = prior.FirstObservedAt
			}
			e.LastObservedAt = now
			e.ArticleIDs = union(prior.ArticleIDs, e.ArticleIDs)
			e.ClaimIDs = union(prior.ClaimIDs, e.ClaimIDs)
			e.TimelineEntryIDs = union(prior.TimelineEntryIDs, e.TimelineEntryIDs)
			e.CoverageLimitations = union(prior.CoverageLimitations, e.CoverageLimitations)
		} else {
			if e.FirstObservedAt.IsZero() {
				e.FirstObservedAt = now
			}
			if e.LastObservedAt.IsZero() {
				e.LastObservedAt = now
			}
		}
		if e.UpdatedAt.IsZero() {
			e.UpdatedAt = now
		}
		db.Events[e.ID] = e
	}

	db.SavedAt = time.Now().UTC()
	db.LastSyncAt = captured
	if err := r.writeLocked(db); err != nil {
		return SyncStats{}, err
	}
	stats.SourcesSaved = len(db.Sources)
	stats.ArticlesSaved = len(db.Articles)
	stats.EventsSaved = len(db.Events)
	return stats, nil
}

type StatementExtractionStats struct {
	ArticlesScanned int `json:"articlesScanned"`
	ClaimsCreated   int `json:"claimsCreated"`
	EvidenceCreated int `json:"evidenceCreated"`
	SkippedSegments int `json:"skippedSegments"`
	Warnings        []string `json:"warnings,omitempty"`
	RunID           string `json:"runId"`
}

type extractionPending struct {
	claim Claim
	evidence []Evidence
}

// ExtractStoredSourceStatements runs only over text already stored in the
// Intelligence database (article titles and summaries). It makes no network
// calls, never infers factual truth, and persists every extract with exact
// source-field offsets. Full article text is never fabricated from a summary.
func (r *JSONRepository) ExtractStoredSourceStatements() (StatementExtractionStats, error) {
	started := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()

	db, err := r.loadLocked()
	if err != nil {
		return StatementExtractionStats{}, err
	}
	stats := StatementExtractionStats{Warnings: []string{}}
	articleIDs := make([]string, 0, len(db.Articles))
	for id := range db.Articles {
		articleIDs = append(articleIDs, id)
	}
	sort.Strings(articleIDs)
	stats.ArticlesScanned = len(articleIDs)

	articleEvents := map[string][]string{}
	for eventID, event := range db.Events {
		for _, articleID := range event.ArticleIDs {
			articleEvents[articleID] = union(articleEvents[articleID], []string{eventID})
		}
	}

	byClaim := map[string]*extractionPending{}
	for _, articleID := range articleIDs {
		article := db.Articles[articleID]
		sourceName := article.PublisherSourceID
		if source, ok := db.Sources[article.PublisherSourceID]; ok && strings.TrimSpace(source.Name) != "" {
			sourceName = source.Name
		}
		excerpts := []TextExcerpt{{Field: "title", Text: article.Title}}
		if strings.TrimSpace(article.Summary) != "" && article.ContentAvailability != ContentTitleOnly {
			excerpts = append(excerpts, TextExcerpt{Field: "summary", Text: article.Summary})
		}
		result, extractErr := ExtractSourceStatements(article, sourceName, excerpts, started)
		if extractErr != nil {
			return StatementExtractionStats{}, fmt.Errorf("extract article %q: %w", articleID, extractErr)
		}
		stats.SkippedSegments += result.Skipped
		stats.Warnings = append(stats.Warnings, result.Warnings...)
		evidenceForClaim := map[string][]Evidence{}
		for _, evidence := range result.Evidence {
			evidenceForClaim[evidence.ClaimID] = append(evidenceForClaim[evidence.ClaimID], evidence)
		}
		for _, claim := range result.Claims {
			p := byClaim[claim.ID]
			if p == nil {
				p = &extractionPending{claim: claim, evidence: evidenceForClaim[claim.ID]}
				byClaim[claim.ID] = p
			} else {
				p.claim.EvidenceIDs = union(p.claim.EvidenceIDs, claim.EvidenceIDs)
				p.evidence = append(p.evidence, evidenceForClaim[claim.ID]...)
			}
		}
	}

	evidenceByID := make(map[string]Evidence, len(db.Evidence)+len(byClaim))
	for id, evidence := range db.Evidence {
		evidenceByID[id] = evidence
	}
	newEvidence := 0
	claimIDs := make([]string, 0, len(byClaim))
	for id := range byClaim {
		claimIDs = append(claimIDs, id)
	}
	sort.Strings(claimIDs)

	for _, claimID := range claimIDs {
		p := byClaim[claimID]
		claim := p.claim
		if prior, ok := db.Claims[claimID]; ok {
			if prior.Kind != ClaimSourceStatement {
				return StatementExtractionStats{}, fmt.Errorf("refusing to overwrite non-source claim %q", claimID)
			}
			if !prior.FirstSeenAt.IsZero() {
				claim.FirstSeenAt = prior.FirstSeenAt
			}
			claim.EvidenceIDs = union(prior.EvidenceIDs, claim.EvidenceIDs)
			if prior.Status != "" {
				claim.Status = prior.Status
			}
			if prior.StatusReason != "" {
				claim.StatusReason = prior.StatusReason
			}
			if prior.EventID != "" {
				claim.EventID = prior.EventID
			}
		}
		for _, evidence := range p.evidence {
			if prior, ok := db.Evidence[evidence.ID]; ok {
				if prior.Quote != evidence.Quote || prior.ArticleID != evidence.ArticleID || prior.SourceURL != evidence.SourceURL {
					return StatementExtractionStats{}, fmt.Errorf("evidence identity collision for %q", evidence.ID)
				}
				if !prior.CapturedAt.IsZero() {
					evidence.CapturedAt = prior.CapturedAt
				}
			} else {
				newEvidence++
			}
			if err := ValidateEvidence(evidence); err != nil {
				return StatementExtractionStats{}, err
			}
			db.Evidence[evidence.ID] = evidence
			evidenceByID[evidence.ID] = evidence
		}
		if err := ValidateClaim(claim, evidenceByID); err != nil {
			return StatementExtractionStats{}, err
		}
		if _, existed := db.Claims[claimID]; !existed {
			stats.ClaimsCreated++
		}
		db.Claims[claimID] = claim
	}

	// Only attach a statement to an event when exactly one stored event candidate
	// contains its source article. Ambiguous many-event memberships stay unlinked.
	eventClaims := map[string][]string{}
	articleEventCount := map[string][]string{}
	for eventID, event := range db.Events {
		for _, articleID := range event.ArticleIDs {
			articleEventCount[articleID] = union(articleEventCount[articleID], []string{eventID})
		}
	}
	for _, claimID := range claimIDs {
		claim := db.Claims[claimID]
		for _, evidenceID := range claim.EvidenceIDs {
			evidence, ok := db.Evidence[evidenceID]
			if !ok {
				continue
			}
			matches := articleEventCount[evidence.ArticleID]
			if len(matches) == 1 {
				claim.EventID = matches[0]
				db.Claims[claimID] = claim
				eventClaims[matches[0]] = union(eventClaims[matches[0]], []string{claimID})
			} else if len(matches) > 1 {
				stats.Warnings = append(stats.Warnings, "Article "+evidence.ArticleID+" belongs to multiple event candidates; statement was not assigned to one event.")
			}
			break
		}
	}
	for eventID, claimIDs := range eventClaims {
		event := db.Events[eventID]
		event.ClaimIDs = union(event.ClaimIDs, claimIDs)
		event.UpdatedAt = started
		db.Events[eventID] = event
	}

	stats.EvidenceCreated = newEvidence
	stats.RunID = stableID("analysis_run", StatementExtractorVersion+"\x00"+started.Format(time.RFC3339Nano))
	completed := time.Now().UTC()
	run := AnalysisRun{
		ID: stats.RunID, InputIDs: articleIDs, RuleVersion: StatementExtractorVersion,
		StartedAt: started, CompletedAt: &completed, Status: "completed",
		Warnings: append([]string(nil), stats.Warnings...),
	}
	db.Runs[run.ID] = run
	db.SavedAt = completed
	if err := r.writeLocked(db); err != nil {
		return StatementExtractionStats{}, err
	}
	return stats, nil
}

func union(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item == "" || seen[item] { continue }
			seen[item] = true
			out = append(out, item)
		}
	}
	sort.Strings(out)
	return out
}

func (r *JSONRepository) writeLocked(db Database) error {
	ensureDatabaseMaps(&db)
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Intelligence database: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, r.path); err != nil {
		// Windows does not replace an existing destination using Rename. This
		// fallback mirrors NexNews' existing cache writer behavior.
		if removeErr := os.Remove(r.path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			_ = os.Remove(tmp)
			return err
		}
		if retryErr := os.Rename(tmp, r.path); retryErr != nil {
			_ = os.Remove(tmp)
			return retryErr
		}
	}
	return nil
}
