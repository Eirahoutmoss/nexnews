// Package intelligence provides the persistent, evidence-first research workspace.
// A research case stores the question, hypotheses, source-linked observations and
// unresolved questions separately from the transient NexNews feed cache.
package intelligence

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

type ResearchCaseStatus string

const (
	ResearchCaseDraft      ResearchCaseStatus = "draft"
	ResearchCaseActive     ResearchCaseStatus = "active"
	ResearchCaseMonitoring ResearchCaseStatus = "monitoring"
	ResearchCaseClosed     ResearchCaseStatus = "closed"
)

type ResearchHypothesisStatus string

const (
	HypothesisOpen         ResearchHypothesisStatus = "open"
	HypothesisUnderReview  ResearchHypothesisStatus = "under_review"
	HypothesisSupported    ResearchHypothesisStatus = "supported"
	HypothesisWeakened     ResearchHypothesisStatus = "weakened"
	HypothesisRejected     ResearchHypothesisStatus = "rejected"
	HypothesisInconclusive ResearchHypothesisStatus = "inconclusive"
)

type ResearchSourceKind string

const (
	ResearchSourcePrimary       ResearchSourceKind = "primary_source"
	ResearchSourceLocalReport   ResearchSourceKind = "local_reporting"
	ResearchSourceInternational ResearchSourceKind = "international_media"
	ResearchSourceOfficialDoc   ResearchSourceKind = "official_document"
	ResearchSourceDataset       ResearchSourceKind = "dataset"
	ResearchSourceAcademic      ResearchSourceKind = "academic_research"
	ResearchSourceOther         ResearchSourceKind = "other"
)

type ResearchContentLevel string

const (
	ResearchContentTitleOnly    ResearchContentLevel = "title_only"
	ResearchContentTitleSummary ResearchContentLevel = "title_summary"
	ResearchContentFullText     ResearchContentLevel = "full_text"
	ResearchContentStructured   ResearchContentLevel = "structured_data"
)

type ResearchSourceOrigin string

const (
	ResearchOriginOriginal   ResearchSourceOrigin = "original_reporting"
	ResearchOriginOfficial   ResearchSourceOrigin = "official_statement"
	ResearchOriginSyndicated ResearchSourceOrigin = "syndicated_or_republished"
	ResearchOriginUnknown    ResearchSourceOrigin = "unknown"
)

type ResearchEvidenceKind string

const (
	ResearchEvidenceArticle     ResearchEvidenceKind = "source_article"
	ResearchEvidenceObservation ResearchEvidenceKind = "observation"
	ResearchEvidenceClaim       ResearchEvidenceKind = "attributed_claim"
	ResearchEvidenceDataPoint   ResearchEvidenceKind = "data_point"
	ResearchEvidenceBackground  ResearchEvidenceKind = "background"
	ResearchEvidenceCorrection  ResearchEvidenceKind = "correction_or_update"
)

type HypothesisEvidenceDirection string

const (
	HypothesisEvidenceSupports     HypothesisEvidenceDirection = "supports"
	HypothesisEvidenceContradicts  HypothesisEvidenceDirection = "contradicts"
	HypothesisEvidenceContext      HypothesisEvidenceDirection = "context"
)

type ResearchCase struct {
	ID                     string                  `json:"id"`
	Title                  string                  `json:"title"`
	Question               string                  `json:"question"`
	Context                string                  `json:"context,omitempty"`
	Status                 ResearchCaseStatus      `json:"status"`
	RelatedEventID         string                  `json:"relatedEventId,omitempty"`
	CreatedAt              time.Time               `json:"createdAt"`
	UpdatedAt              time.Time               `json:"updatedAt"`
	Hypotheses             []ResearchHypothesis    `json:"hypotheses"`
	Evidence               []ResearchEvidence      `json:"evidence"`
	OpenQuestions          []ResearchQuestion      `json:"openQuestions"`
	TargetSelectionStudies []TargetSelectionStudy  `json:"targetSelectionStudies,omitempty"`
}

type ResearchHypothesis struct {
	ID             string                  `json:"id"`
	Statement      string                  `json:"statement"`
	Status         ResearchHypothesisStatus `json:"status"`
	AssessmentNote string                  `json:"assessmentNote,omitempty"`
	CreatedAt      time.Time               `json:"createdAt"`
	UpdatedAt      time.Time               `json:"updatedAt"`
}

type ResearchEvidence struct {
	ID                    string                       `json:"id"`
	NexNewsArticleID      string                       `json:"nexNewsArticleId,omitempty"`
	ContentFingerprint    string                       `json:"contentFingerprint,omitempty"`
	Kind                  ResearchEvidenceKind         `json:"kind"`
	Title                 string                       `json:"title"`
	SourceURL             string                       `json:"sourceUrl"`
	Publisher             string                       `json:"publisher"`
	PublisherDomain       string                       `json:"publisherDomain"`
	Author                string                       `json:"author,omitempty"`
	PublishedAt           *time.Time                   `json:"publishedAt,omitempty"`
	RetrievedAt           time.Time                    `json:"retrievedAt"`
	Excerpt               string                       `json:"excerpt"`
	ContentLevel          ResearchContentLevel         `json:"contentLevel"`
	SourceKind            ResearchSourceKind           `json:"sourceKind"`
	SourceOrigin          ResearchSourceOrigin         `json:"sourceOrigin"`
	SyndicationGroup      string                       `json:"syndicationGroup,omitempty"`
	MethodNote            string                       `json:"methodNote,omitempty"`
	HypothesisAssessments []HypothesisEvidenceAssessment `json:"hypothesisAssessments,omitempty"`
}

type HypothesisEvidenceAssessment struct {
	HypothesisID string                       `json:"hypothesisId"`
	Direction    HypothesisEvidenceDirection  `json:"direction"`
	Rationale    string                       `json:"rationale"`
}

type ResearchQuestion struct {
	ID             string    `json:"id"`
	Text           string    `json:"text"`
	Status         string    `json:"status"`
	ResolutionNote string    `json:"resolutionNote,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ResearchCaseSummary struct {
	Case                       ResearchCase `json:"case"`
	EvidenceCount              int          `json:"evidenceCount"`
	HypothesisCount            int          `json:"hypothesisCount"`
	OpenQuestionCount          int          `json:"openQuestionCount"`
	PublisherDomainCount       int          `json:"publisherDomainCount"`
	KnownSyndicationGroupCount int          `json:"knownSyndicationGroupCount"`
	OriginUnknownCount         int          `json:"originUnknownCount"`
	FullTextEvidenceCount      int          `json:"fullTextEvidenceCount"`
	Limitations                []string     `json:"limitations"`
}

type CreateResearchCaseInput struct {
	Title          string `json:"title"`
	Question       string `json:"question"`
	Context        string `json:"context"`
	RelatedEventID string `json:"relatedEventId,omitempty"`
}

type AddResearchEvidenceInput struct {
	Kind                  ResearchEvidenceKind          `json:"kind"`
	Title                 string                        `json:"title"`
	SourceURL             string                        `json:"sourceUrl"`
	Publisher             string                        `json:"publisher"`
	PublisherDomain       string                        `json:"publisherDomain,omitempty"`
	Author                string                        `json:"author,omitempty"`
	PublishedAt           *time.Time                    `json:"publishedAt,omitempty"`
	Excerpt               string                        `json:"excerpt"`
	ContentLevel          ResearchContentLevel          `json:"contentLevel"`
	SourceKind            ResearchSourceKind            `json:"sourceKind"`
	SourceOrigin          ResearchSourceOrigin          `json:"sourceOrigin"`
	SyndicationGroup      string                        `json:"syndicationGroup,omitempty"`
	MethodNote            string                        `json:"methodNote,omitempty"`
	HypothesisAssessments []HypothesisEvidenceAssessment `json:"hypothesisAssessments,omitempty"`
}

func newResearchID(prefix string) (string, error) {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate %s id: %w", prefix, err)
	}
	return prefix + "_" + hex.EncodeToString(raw[:]), nil
}

func normalizedPublisherDomain(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Hostname() != "" {
		raw = parsed.Hostname()
	}
	return strings.TrimPrefix(strings.TrimSuffix(strings.ToLower(raw), "."), "www.")
}

func validResearchSourceURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && u.Hostname() != "" && (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https"))
}

func validateResearchCaseInput(in CreateResearchCaseInput) error {
	switch {
	case strings.TrimSpace(in.Title) == "":
		return errors.New("research title is required")
	case len([]rune(strings.TrimSpace(in.Title))) > 200:
		return errors.New("research title must be 200 characters or fewer")
	case strings.TrimSpace(in.Question) == "":
		return errors.New("research question is required")
	case len([]rune(strings.TrimSpace(in.Question))) > 2000:
		return errors.New("research question must be 2000 characters or fewer")
	case len([]rune(strings.TrimSpace(in.Context))) > 10000:
		return errors.New("research context must be 10000 characters or fewer")
	}
	return nil
}

func validResearchCaseStatus(status ResearchCaseStatus) bool {
	switch status {
	case ResearchCaseDraft, ResearchCaseActive, ResearchCaseMonitoring, ResearchCaseClosed:
		return true
	default:
		return false
	}
}

func validHypothesisStatus(status ResearchHypothesisStatus) bool {
	switch status {
	case HypothesisOpen, HypothesisUnderReview, HypothesisSupported, HypothesisWeakened, HypothesisRejected, HypothesisInconclusive:
		return true
	default:
		return false
	}
}

func validResearchEvidenceKind(kind ResearchEvidenceKind) bool {
	switch kind {
	case ResearchEvidenceArticle, ResearchEvidenceObservation, ResearchEvidenceClaim, ResearchEvidenceDataPoint, ResearchEvidenceBackground, ResearchEvidenceCorrection:
		return true
	default:
		return false
	}
}

func validResearchSourceKind(kind ResearchSourceKind) bool {
	switch kind {
	case ResearchSourcePrimary, ResearchSourceLocalReport, ResearchSourceInternational, ResearchSourceOfficialDoc, ResearchSourceDataset, ResearchSourceAcademic, ResearchSourceOther:
		return true
	default:
		return false
	}
}

func validResearchContentLevel(level ResearchContentLevel) bool {
	switch level {
	case ResearchContentTitleOnly, ResearchContentTitleSummary, ResearchContentFullText, ResearchContentStructured:
		return true
	default:
		return false
	}
}

func validResearchOrigin(origin ResearchSourceOrigin) bool {
	switch origin {
	case ResearchOriginOriginal, ResearchOriginOfficial, ResearchOriginSyndicated, ResearchOriginUnknown:
		return true
	default:
		return false
	}
}

func validHypothesisDirection(direction HypothesisEvidenceDirection) bool {
	return direction == HypothesisEvidenceSupports || direction == HypothesisEvidenceContradicts || direction == HypothesisEvidenceContext
}

func validateResearchEvidence(in AddResearchEvidenceInput) (string, error) {
	switch {
	case !validResearchEvidenceKind(in.Kind):
		return "", errors.New("evidence kind is invalid")
	case strings.TrimSpace(in.Title) == "":
		return "", errors.New("source title is required")
	case !validResearchSourceURL(in.SourceURL):
		return "", errors.New("source URL must use http or https")
	case strings.TrimSpace(in.Publisher) == "":
		return "", errors.New("publisher name is required")
	case strings.TrimSpace(in.Excerpt) == "":
		return "", errors.New("source excerpt or data description is required")
	case !validResearchContentLevel(in.ContentLevel):
		return "", errors.New("content level is invalid")
	case !validResearchSourceKind(in.SourceKind):
		return "", errors.New("source kind is invalid")
	case !validResearchOrigin(in.SourceOrigin):
		return "", errors.New("source origin is invalid")
	case len([]rune(in.Excerpt)) > 12000:
		return "", errors.New("excerpt must be 12000 characters or fewer")
	}
	u, _ := url.Parse(strings.TrimSpace(in.SourceURL))
	domain := normalizedPublisherDomain(in.PublisherDomain)
	if domain == "" {
		domain = normalizedPublisherDomain(u.Hostname())
	}
	if domain == "" {
		return "", errors.New("publisher domain could not be determined")
	}
	seen := map[string]bool{}
	for _, assessment := range in.HypothesisAssessments {
		if strings.TrimSpace(assessment.HypothesisID) == "" {
			return "", errors.New("hypothesis assessment id is required")
		}
		if !validHypothesisDirection(assessment.Direction) {
			return "", errors.New("hypothesis assessment direction is invalid")
		}
		if strings.TrimSpace(assessment.Rationale) == "" {
			return "", errors.New("hypothesis assessment rationale is required")
		}
		if seen[assessment.HypothesisID] {
			return "", errors.New("an evidence item may assess a hypothesis only once")
		}
		seen[assessment.HypothesisID] = true
	}
	return domain, nil
}

func markResearchCaseActive(item *ResearchCase, at time.Time) {
	if item != nil && item.Status == ResearchCaseDraft {
		item.Status = ResearchCaseActive
		item.UpdatedAt = at
	}
}

func (r *JSONRepository) CreateResearchCase(in CreateResearchCaseInput) (ResearchCase, error) {
	if err := validateResearchCaseInput(in); err != nil {
		return ResearchCase{}, err
	}
	now := time.Now().UTC()
	id, err := newResearchID("research")
	if err != nil {
		return ResearchCase{}, err
	}
	item := ResearchCase{
		ID: id, Title: strings.TrimSpace(in.Title), Question: strings.TrimSpace(in.Question),
		Context: strings.TrimSpace(in.Context), Status: ResearchCaseDraft,
		RelatedEventID: strings.TrimSpace(in.RelatedEventID), CreatedAt: now, UpdatedAt: now,
		Hypotheses: []ResearchHypothesis{}, Evidence: []ResearchEvidence{}, OpenQuestions: []ResearchQuestion{},
		TargetSelectionStudies: []TargetSelectionStudy{},
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	db, err := r.loadLocked()
	if err != nil {
		return ResearchCase{}, err
	}
	if item.RelatedEventID != "" {
		if _, ok := db.Events[item.RelatedEventID]; !ok {
			return ResearchCase{}, errors.New("related NexNews event was not found")
		}
	}
	db.ResearchCases[item.ID] = item
	db.SavedAt = now
	if err := r.writeLocked(db); err != nil {
		return ResearchCase{}, err
	}
	return item, nil
}

func (r *JSONRepository) ListResearchCases() ([]ResearchCaseSummary, error) {
	db, err := r.Load()
	if err != nil {
		return nil, err
	}
	out := make([]ResearchCaseSummary, 0, len(db.ResearchCases))
	for _, item := range db.ResearchCases {
		out = append(out, summarizeResearchCase(item))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Case.UpdatedAt.Equal(out[j].Case.UpdatedAt) {
			return out[i].Case.UpdatedAt.After(out[j].Case.UpdatedAt)
		}
		return out[i].Case.ID < out[j].Case.ID
	})
	return out, nil
}

func (r *JSONRepository) GetResearchCase(id string) (ResearchCase, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ResearchCase{}, errors.New("research case id is required")
	}
	db, err := r.Load()
	if err != nil {
		return ResearchCase{}, err
	}
	item, ok := db.ResearchCases[id]
	if !ok {
		return ResearchCase{}, errors.New("research case not found")
	}
	return item, nil
}

func (r *JSONRepository) AddResearchHypothesis(caseID, statement string) (ResearchCase, error) {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return ResearchCase{}, errors.New("hypothesis statement is required")
	}
	if len([]rune(statement)) > 3000 {
		return ResearchCase{}, errors.New("hypothesis statement must be 3000 characters or fewer")
	}
	id, err := newResearchID("hypothesis")
	if err != nil {
		return ResearchCase{}, err
	}
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	db, err := r.loadLocked()
	if err != nil {
		return ResearchCase{}, err
	}
	item, ok := db.ResearchCases[strings.TrimSpace(caseID)]
	if !ok {
		return ResearchCase{}, errors.New("research case not found")
	}
	item.Hypotheses = append(item.Hypotheses, ResearchHypothesis{
		ID: id, Statement: statement, Status: HypothesisOpen, CreatedAt: now, UpdatedAt: now,
	})
	markResearchCaseActive(&item, now)
	item.UpdatedAt = now
	db.ResearchCases[item.ID] = item
	db.SavedAt = now
	if err := r.writeLocked(db); err != nil {
		return ResearchCase{}, err
	}
	return item, nil
}

func (r *JSONRepository) AddResearchEvidence(caseID string, in AddResearchEvidenceInput) (ResearchCase, error) {
	domain, err := validateResearchEvidence(in)
	if err != nil {
		return ResearchCase{}, err
	}
	id, err := newResearchID("evidence")
	if err != nil {
		return ResearchCase{}, err
	}
	now := time.Now().UTC()
	sourceURL := strings.TrimSpace(in.SourceURL)
	if parsed, parseErr := url.Parse(sourceURL); parseErr == nil {
		parsed.Fragment = ""
		sourceURL = parsed.String()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	db, err := r.loadLocked()
	if err != nil {
		return ResearchCase{}, err
	}
	item, ok := db.ResearchCases[strings.TrimSpace(caseID)]
	if !ok {
		return ResearchCase{}, errors.New("research case not found")
	}
	hypotheses := map[string]bool{}
	for _, h := range item.Hypotheses {
		hypotheses[h.ID] = true
	}
	for _, assessment := range in.HypothesisAssessments {
		if !hypotheses[assessment.HypothesisID] {
			return ResearchCase{}, fmt.Errorf("hypothesis %q does not belong to this research case", assessment.HypothesisID)
		}
	}
	evidence := ResearchEvidence{
		ID: id, Kind: in.Kind, Title: strings.TrimSpace(in.Title), SourceURL: sourceURL,
		Publisher: strings.TrimSpace(in.Publisher), PublisherDomain: domain,
		Author: strings.TrimSpace(in.Author), PublishedAt: in.PublishedAt, RetrievedAt: now,
		Excerpt: strings.TrimSpace(in.Excerpt), ContentLevel: in.ContentLevel,
		SourceKind: in.SourceKind, SourceOrigin: in.SourceOrigin,
		SyndicationGroup: strings.TrimSpace(in.SyndicationGroup), MethodNote: strings.TrimSpace(in.MethodNote),
		HypothesisAssessments: append([]HypothesisEvidenceAssessment(nil), in.HypothesisAssessments...),
	}
	item.Evidence = append(item.Evidence, evidence)
	markResearchCaseActive(&item, now)
	item.UpdatedAt = now
	db.ResearchCases[item.ID] = item
	db.SavedAt = now
	if err := r.writeLocked(db); err != nil {
		return ResearchCase{}, err
	}
	return item, nil
}


func (r *JSONRepository) AddResearchQuestion(caseID, question string) (ResearchCase, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return ResearchCase{}, errors.New("research question is required")
	}
	if len([]rune(question)) > 2000 {
		return ResearchCase{}, errors.New("research question must be 2000 characters or fewer")
	}
	id, err := newResearchID("question")
	if err != nil {
		return ResearchCase{}, err
	}
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	db, err := r.loadLocked()
	if err != nil {
		return ResearchCase{}, err
	}
	item, ok := db.ResearchCases[strings.TrimSpace(caseID)]
	if !ok {
		return ResearchCase{}, errors.New("research case not found")
	}
	item.OpenQuestions = append(item.OpenQuestions, ResearchQuestion{ID: id, Text: question, Status: "open", CreatedAt: now, UpdatedAt: now})
	markResearchCaseActive(&item, now)
	item.UpdatedAt = now
	db.ResearchCases[item.ID] = item
	db.SavedAt = now
	if err := r.writeLocked(db); err != nil {
		return ResearchCase{}, err
	}
	return item, nil
}

func (r *JSONRepository) SetResearchQuestionStatus(caseID, questionID, status, note string) (ResearchCase, error) {
	status = strings.TrimSpace(status)
	note = strings.TrimSpace(note)
	if status != "open" && status != "resolved" {
		return ResearchCase{}, errors.New("research question status must be open or resolved")
	}
	if status == "resolved" && note == "" {
		return ResearchCase{}, errors.New("a resolution note is required before resolving a research question")
	}
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	db, err := r.loadLocked()
	if err != nil { return ResearchCase{}, err }
	item, ok := db.ResearchCases[strings.TrimSpace(caseID)]
	if !ok { return ResearchCase{}, errors.New("research case not found") }
	found := false
	for i := range item.OpenQuestions {
		question := &item.OpenQuestions[i]
		if question.ID != strings.TrimSpace(questionID) { continue }
		found = true
		question.Status = status
		question.ResolutionNote = note
		if status == "open" { question.ResolutionNote = "" }
		question.UpdatedAt = now
		break
	}
	if !found { return ResearchCase{}, errors.New("research question not found") }
	markResearchCaseActive(&item, now)
	item.UpdatedAt = now
	db.ResearchCases[item.ID] = item
	db.SavedAt = now
	if err := r.writeLocked(db); err != nil { return ResearchCase{}, err }
	return item, nil
}

func (r *JSONRepository) SetResearchHypothesisStatus(caseID, hypothesisID string, status ResearchHypothesisStatus, note string) (ResearchCase, error) {
	if !validHypothesisStatus(status) {
		return ResearchCase{}, errors.New("hypothesis status is invalid")
	}
	note = strings.TrimSpace(note)
	if status != HypothesisOpen && status != HypothesisUnderReview && note == "" {
		return ResearchCase{}, errors.New("an assessment note is required for a concluded hypothesis status")
	}
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	db, err := r.loadLocked()
	if err != nil {
		return ResearchCase{}, err
	}
	item, ok := db.ResearchCases[strings.TrimSpace(caseID)]
	if !ok {
		return ResearchCase{}, errors.New("research case not found")
	}
	found := false
	for i := range item.Hypotheses {
		h := &item.Hypotheses[i]
		if h.ID != strings.TrimSpace(hypothesisID) {
			continue
		}
		found = true
		if status == HypothesisSupported || status == HypothesisWeakened || status == HypothesisRejected || status == HypothesisInconclusive {
			hasSupport, hasContradiction, hasAny := false, false, false
			for _, evidence := range item.Evidence {
				for _, assessment := range evidence.HypothesisAssessments {
					if assessment.HypothesisID != h.ID {
						continue
					}
					hasAny = true
					if assessment.Direction == HypothesisEvidenceSupports {
						hasSupport = true
					}
					if assessment.Direction == HypothesisEvidenceContradicts {
						hasContradiction = true
					}
				}
			}
			switch status {
			case HypothesisSupported:
				if !hasSupport { return ResearchCase{}, errors.New("supported status requires source-linked supporting evidence") }
			case HypothesisWeakened, HypothesisRejected:
				if !hasContradiction { return ResearchCase{}, errors.New("weakened or rejected status requires source-linked contradictory evidence") }
			case HypothesisInconclusive:
				if !hasAny { return ResearchCase{}, errors.New("inconclusive status requires at least one linked evidence item") }
			}
		}
		h.Status = status
		h.AssessmentNote = note
		h.UpdatedAt = now
		break
	}
	if !found {
		return ResearchCase{}, errors.New("research hypothesis not found")
	}
	markResearchCaseActive(&item, now)
	item.UpdatedAt = now
	db.ResearchCases[item.ID] = item
	db.SavedAt = now
	if err := r.writeLocked(db); err != nil {
		return ResearchCase{}, err
	}
	return item, nil
}

func summarizeResearchCase(item ResearchCase) ResearchCaseSummary {
	domains := map[string]bool{}
	syndication := map[string]bool{}
	unknownOrigins, fullText := 0, 0
	for _, evidence := range item.Evidence {
		if domain := normalizedPublisherDomain(evidence.PublisherDomain); domain != "" {
			domains[domain] = true
		}
		if group := strings.TrimSpace(evidence.SyndicationGroup); group != "" {
			syndication[group] = true
		}
		if evidence.SourceOrigin == ResearchOriginUnknown {
			unknownOrigins++
		}
		if evidence.ContentLevel == ResearchContentFullText || evidence.ContentLevel == ResearchContentStructured {
			fullText++
		}
	}
	openQuestions := 0
	for _, question := range item.OpenQuestions {
		if question.Status != "resolved" {
			openQuestions++
		}
	}
	limitations := []string{}
	if len(item.Evidence) == 0 {
		limitations = append(limitations, "Henüz kaynaklı kanıt kaydedilmedi; hipotezlerin durumu değerlendirilmemeli.")
	}
	if unknownOrigins > 0 {
		limitations = append(limitations, "Bazı kaynakların özgün haber mi yoksa yeniden yayımlanmış içerik mi olduğu bilinmiyor.")
	}
	if len(item.Evidence) > 0 && fullText == 0 {
		limitations = append(limitations, "Kaydedilen kaynaklar arasında tam metin veya yapılandırılmış veri incelemesi bulunmuyor.")
	}
	if len(domains) > 1 {
		limitations = append(limitations, "Farklı yayıncı alan adları tek başına bağımsız doğrulama anlamına gelmez.")
	}
	if len(syndication) > 0 {
		limitations = append(limitations, "Bilinen ortak ajans veya yeniden yayımlama grupları ayrıca değerlendirilmelidir; grup sayısı bağımsız kaynak sayısı değildir.")
	}
	return ResearchCaseSummary{
		Case: item, EvidenceCount: len(item.Evidence), HypothesisCount: len(item.Hypotheses),
		OpenQuestionCount: openQuestions, PublisherDomainCount: len(domains),
		KnownSyndicationGroupCount: len(syndication), OriginUnknownCount: unknownOrigins,
		FullTextEvidenceCount: fullText, Limitations: limitations,
	}
}
