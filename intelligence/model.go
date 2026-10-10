// Package intelligence defines the evidence-first data contract for NexNews
// Intelligence. It intentionally has no dependency on NexNews UI or feed code.
package intelligence

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const SchemaVersion = "1.0.0"

type SourceType string

const (
	SourcePublisher       SourceType = "publisher"
	SourcePrimary         SourceType = "primary"
	SourceOfficialDataset SourceType = "official_dataset"
	SourceOther           SourceType = "other"
)

type ContentAvailability string

const (
	ContentTitleOnly    ContentAvailability = "title_only"
	ContentTitleSummary ContentAvailability = "title_summary"
	ContentFullText     ContentAvailability = "full_text"
)

type EntityType string

const (
	EntityPerson       EntityType = "person"
	EntityOrganization EntityType = "organization"
	EntityCompany      EntityType = "company"
	EntityPlace        EntityType = "place"
	EntityLaw           EntityType = "law"
	EntityProduct       EntityType = "product"
	EntityOther         EntityType = "other"
)

type IdentityStatus string

const (
	IdentityCandidate IdentityStatus = "candidate"
	IdentityResolved  IdentityStatus = "resolved"
	IdentityAmbiguous IdentityStatus = "ambiguous"
)

type ClaimStatus string

const (
	ClaimReported     ClaimStatus = "reported"
	ClaimCorroborated ClaimStatus = "corroborated"
	ClaimContested    ClaimStatus = "contested"
	ClaimRetracted    ClaimStatus = "retracted"
	ClaimUnverified   ClaimStatus = "unverified"
	ClaimUnknown      ClaimStatus = "unknown"
)

type EvidenceStance string

const (
	EvidenceSupports     EvidenceStance = "supports"
	EvidenceContradicts  EvidenceStance = "contradicts"
	EvidenceContext      EvidenceStance = "context"
	EvidenceMentions     EvidenceStance = "mentions"
)

type RelationPredicate string

const (
	RelationAnnouncedBy  RelationPredicate = "announced_by"
	RelationSaidBy       RelationPredicate = "said_by"
	RelationIssuedBy     RelationPredicate = "issued_by"
	RelationParticipates  RelationPredicate = "participates_in"
	RelationMemberOf      RelationPredicate = "member_of"
	RelationLocatedIn     RelationPredicate = "located_in"
	RelationOwns          RelationPredicate = "owns"
	RelationOperates      RelationPredicate = "operates"
	RelationControls      RelationPredicate = "controls"
	RelationRespondsTo    RelationPredicate = "responds_to"
	RelationSupportsClaim RelationPredicate = "supports_claim"
	RelationContradictsClaim RelationPredicate = "contradicts_claim"
	RelationUpdatesEvent  RelationPredicate = "updates_event"
	RelationSameEventAs   RelationPredicate = "same_event_as"
	RelationCausedBy      RelationPredicate = "caused_by"
	RelationLeadsTo       RelationPredicate = "leads_to"
)

type RelationStatus string

const (
	RelationCandidate RelationStatus = "candidate"
	RelationSupported RelationStatus = "supported"
	RelationContested RelationStatus = "contested"
	RelationRejected  RelationStatus = "rejected"
	RelationExpired   RelationStatus = "expired"
)

type NodeType string

const (
	NodeEntity  NodeType = "entity"
	NodeEvent   NodeType = "event"
	NodeClaim   NodeType = "claim"
	NodeArticle NodeType = "article"
)

type Source struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	CanonicalDomain string   `json:"canonicalDomain"`
	Type           SourceType `json:"type"`
	IdentityBasis  string    `json:"identityBasis"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Article struct {
	ID                 string              `json:"id"`
	CanonicalURL       string              `json:"canonicalUrl"`
	PublisherSourceID  string              `json:"publisherSourceId"`
	Title              string              `json:"title"`
	Summary            string              `json:"summary,omitempty"`
	PublishedAt        *time.Time          `json:"publishedAt,omitempty"`
	RetrievedAt        time.Time           `json:"retrievedAt"`
	Language           string              `json:"language,omitempty"`
	ContentAvailability ContentAvailability `json:"contentAvailability"`
	ContentHash        string              `json:"contentHash,omitempty"`
	SyndicationOriginID string             `json:"syndicationOriginId,omitempty"`
	EventCandidateIDs  []string            `json:"eventCandidateIds,omitempty"`
}

type EventStatus string

const (
	EventEmerging     EventStatus = "emerging"
	EventDeveloping   EventStatus = "developing"
	EventStabilizing  EventStatus = "stabilizing"
	EventArchived     EventStatus = "archived"
	EventNeedsReview  EventStatus = "needs_review"
)

type Event struct {
	ID                  string      `json:"id"`
	Title               string      `json:"title"`
	Scope               string      `json:"scope,omitempty"`
	Status              EventStatus `json:"status"`
	FirstObservedAt     time.Time   `json:"firstObservedAt"`
	LastObservedAt      time.Time   `json:"lastObservedAt"`
	ArticleIDs          []string    `json:"articleIds,omitempty"`
	ClaimIDs            []string    `json:"claimIds,omitempty"`
	TimelineEntryIDs    []string    `json:"timelineEntryIds,omitempty"`
	CoverageLimitations []string    `json:"coverageLimitations,omitempty"`
	RuleVersion         string      `json:"ruleVersion"`
	UpdatedAt           time.Time   `json:"updatedAt"`
}

type Entity struct {
	ID                  string        `json:"id"`
	CanonicalName       string        `json:"canonicalName"`
	Type                EntityType    `json:"type"`
	Aliases             []string      `json:"aliases,omitempty"`
	IdentityStatus      IdentityStatus `json:"identityStatus"`
	ResolutionEvidenceIDs []string    `json:"resolutionEvidenceIds,omitempty"`
}

type Claim struct {
	ID               string      `json:"id"`
	SubjectEntityID  string      `json:"subjectEntityId,omitempty"`
	Predicate        string      `json:"predicate"`
	ObjectText       string      `json:"objectText,omitempty"`
	ObjectEntityID   string      `json:"objectEntityId,omitempty"`
	Polarity         string      `json:"polarity,omitempty"`
	ClaimText        string      `json:"claimText"`
	AttributedTo     string      `json:"attributedTo,omitempty"`
	FirstSeenAt      time.Time   `json:"firstSeenAt"`
	EventID          string      `json:"eventId,omitempty"`
	EvidenceIDs      []string    `json:"evidenceIds"`
	Status           ClaimStatus `json:"status"`
	StatusReason     string      `json:"statusReason,omitempty"`
}

type Evidence struct {
	ID                string        `json:"id"`
	ArticleID         string        `json:"articleId"`
	ClaimID           string        `json:"claimId,omitempty"`
	Stance            EvidenceStance `json:"stance"`
	Quote             string        `json:"quote"`
	StartOffset       *int          `json:"startOffset,omitempty"`
	EndOffset         *int          `json:"endOffset,omitempty"`
	SourceURL         string        `json:"sourceUrl"`
	CapturedAt        time.Time     `json:"capturedAt"`
	ContentLevel      ContentAvailability `json:"contentLevel"`
	ExtractionMethod  string        `json:"extractionMethod"`
	ExtractionVersion string        `json:"extractionVersion"`
}

type NodeRef struct {
	ID   string   `json:"id"`
	Type NodeType `json:"type"`
}

type Relation struct {
	ID          string           `json:"id"`
	Subject     NodeRef          `json:"subject"`
	Predicate   RelationPredicate `json:"predicate"`
	Object      NodeRef          `json:"object"`
	Status      RelationStatus   `json:"status"`
	EvidenceIDs []string         `json:"evidenceIds"`
	Rationale   string           `json:"rationale"`
	ValidFrom   *time.Time       `json:"validFrom,omitempty"`
	ValidTo     *time.Time       `json:"validTo,omitempty"`
	CreatedBy   string           `json:"createdBy"`
	RuleVersion string           `json:"ruleVersion"`
	ReviewStatus string          `json:"reviewStatus,omitempty"`
}

type TimelineEntry struct {
	ID                 string    `json:"id"`
	EventID            string    `json:"eventId"`
	ArticleID          string    `json:"articleId"`
	Title              string    `json:"title"`
	Description        string    `json:"description,omitempty"`
	ObservedAt         time.Time `json:"observedAt"`
	EventTime          *time.Time `json:"eventTime,omitempty"`
	EventTimePrecision string    `json:"eventTimePrecision,omitempty"`
	EvidenceIDs        []string  `json:"evidenceIds"`
}

type AnalysisRun struct {
	ID             string    `json:"id"`
	InputIDs       []string  `json:"inputIds"`
	RuleVersion    string    `json:"ruleVersion"`
	ModelVersion   string    `json:"modelVersion,omitempty"`
	StartedAt      time.Time `json:"startedAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	Status         string    `json:"status"`
	Warnings       []string  `json:"warnings,omitempty"`
}

func validHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" {
		return false
	}
	return strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")
}

func validContentAvailability(v ContentAvailability) bool {
	return v == ContentTitleOnly || v == ContentTitleSummary || v == ContentFullText
}

func validEvidenceStance(v EvidenceStance) bool {
	switch v {
	case EvidenceSupports, EvidenceContradicts, EvidenceContext, EvidenceMentions:
		return true
	default:
		return false
	}
}

func validNodeType(v NodeType) bool {
	return v == NodeEntity || v == NodeEvent || v == NodeClaim || v == NodeArticle
}

func validRelationPredicate(v RelationPredicate) bool {
	switch v {
	case RelationAnnouncedBy, RelationSaidBy, RelationIssuedBy,
		RelationParticipates, RelationMemberOf, RelationLocatedIn,
		RelationOwns, RelationOperates, RelationControls, RelationRespondsTo,
		RelationSupportsClaim, RelationContradictsClaim, RelationUpdatesEvent,
		RelationSameEventAs, RelationCausedBy, RelationLeadsTo:
		return true
	default:
		return false
	}
}

func validRelationStatus(v RelationStatus) bool {
	switch v {
	case RelationCandidate, RelationSupported, RelationContested, RelationRejected, RelationExpired:
		return true
	default:
		return false
	}
}

// ValidateArticle prevents records with unsupported URLs or ambiguous content
// coverage from entering the Intelligence store.
func ValidateArticle(a Article) error {
	switch {
	case strings.TrimSpace(a.ID) == "":
		return errors.New("article id is required")
	case !validHTTPURL(a.CanonicalURL):
		return errors.New("article canonical URL must be http or https")
	case strings.TrimSpace(a.PublisherSourceID) == "":
		return errors.New("article publisher source id is required")
	case strings.TrimSpace(a.Title) == "":
		return errors.New("article title is required")
	case !validContentAvailability(a.ContentAvailability):
		return errors.New("article content availability is invalid")
	case a.ContentAvailability == ContentTitleSummary && strings.TrimSpace(a.Summary) == "":
		return errors.New("title_summary content level requires a summary")
	}
	return nil
}

// ValidateEvidence validates provenance and the actual captured text segment.
// It cannot determine whether the quotation semantically proves a claim; that
// requires a separate review/extraction step and must never be silently assumed.
func ValidateEvidence(e Evidence) error {
	switch {
	case strings.TrimSpace(e.ID) == "":
		return errors.New("evidence id is required")
	case strings.TrimSpace(e.ArticleID) == "":
		return errors.New("evidence article id is required")
	case !validHTTPURL(e.SourceURL):
		return errors.New("evidence source URL must be http or https")
	case strings.TrimSpace(e.Quote) == "":
		return errors.New("evidence quote is required")
	case !validEvidenceStance(e.Stance):
		return errors.New("evidence stance is invalid")
	case !validContentAvailability(e.ContentLevel):
		return errors.New("evidence content level is invalid")
	case strings.TrimSpace(e.ExtractionMethod) == "":
		return errors.New("evidence extraction method is required")
	case strings.TrimSpace(e.ExtractionVersion) == "":
		return errors.New("evidence extraction version is required")
	case (e.StartOffset == nil) != (e.EndOffset == nil):
		return errors.New("evidence offsets must both be set or both be omitted")
	case e.StartOffset != nil && (*e.StartOffset < 0 || *e.EndOffset < *e.StartOffset):
		return errors.New("evidence offsets are invalid")
	}
	return nil
}

// ValidateRelation requires provenance for every relation status, including
// candidates. Co-occurrence and free-form "related_to" are intentionally not
// part of the allowlist. A relation cannot be accepted just because an LLM
// proposed it or two names appear in the same document.
func ValidateRelation(r Relation, evidenceByID map[string]Evidence) error {
	switch {
	case strings.TrimSpace(r.ID) == "":
		return errors.New("relation id is required")
	case strings.TrimSpace(r.Subject.ID) == "" || !validNodeType(r.Subject.Type):
		return errors.New("relation subject must have a valid id and node type")
	case strings.TrimSpace(r.Object.ID) == "" || !validNodeType(r.Object.Type):
		return errors.New("relation object must have a valid id and node type")
	case r.Subject.ID == r.Object.ID && r.Subject.Type == r.Object.Type:
		return errors.New("relation subject and object cannot be the same node")
	case !validRelationPredicate(r.Predicate):
		return fmt.Errorf("relation predicate %q is not allowed", r.Predicate)
	case !validRelationStatus(r.Status):
		return errors.New("relation status is invalid")
	case strings.TrimSpace(r.Rationale) == "":
		return errors.New("relation rationale is required")
	case strings.TrimSpace(r.RuleVersion) == "":
		return errors.New("relation rule version is required")
	case len(r.EvidenceIDs) == 0:
		return errors.New("relation requires at least one evidence reference")
	}

	hasSupport, hasContradiction := false, false
	seen := map[string]bool{}
	for _, id := range r.EvidenceIDs {
		if strings.TrimSpace(id) == "" || seen[id] {
			return errors.New("relation evidence ids must be non-empty and unique")
		}
		seen[id] = true
		e, ok := evidenceByID[id]
		if !ok {
			return fmt.Errorf("relation evidence %q was not found", id)
		}
		if err := ValidateEvidence(e); err != nil {
			return fmt.Errorf("relation evidence %q is invalid: %w", id, err)
		}
		switch e.Stance {
		case EvidenceSupports:
			hasSupport = true
		case EvidenceContradicts:
			hasContradiction = true
		}
	}

	if r.Status == RelationSupported && !hasSupport {
		return errors.New("supported relation requires supporting evidence")
	}
	if r.Status == RelationContested && (!hasSupport || !hasContradiction) {
		return errors.New("contested relation requires both supporting and contradicting evidence")
	}
	if r.Status == RelationRejected && !hasContradiction {
		return errors.New("rejected relation requires contradicting evidence")
	}
	return nil
}

// CanDisplayInRelationMap exposes only supported relations with valid, traceable
// evidence. Candidate/contested/rejected relations belong in review panels.
func CanDisplayInRelationMap(r Relation, evidenceByID map[string]Evidence) bool {
	return r.Status == RelationSupported && ValidateRelation(r, evidenceByID) == nil
}
