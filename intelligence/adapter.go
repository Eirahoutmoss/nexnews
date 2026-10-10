package intelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"

	"nexnews/core"
)

const ImporterVersion = "nexnews-adapter/1"

// ImportSnapshot is a normalized read-only import batch. Articles and events
// are imported as observations/candidates; claims, evidence and semantic
// relations are intentionally not synthesized by this adapter.
type ImportSnapshot struct {
	SchemaVersion string    `json:"schemaVersion"`
	CapturedAt    time.Time `json:"capturedAt"`
	Sources       []Source  `json:"sources"`
	Articles      []Article `json:"articles"`
	Events        []Event   `json:"events"`
}

type SyncStats struct {
	SourcesSeen  int `json:"sourcesSeen"`
	ArticlesSeen int `json:"articlesSeen"`
	EventsSeen   int `json:"eventsSeen"`
	SourcesSaved int `json:"sourcesSaved"`
	ArticlesSaved int `json:"articlesSaved"`
	EventsSaved  int `json:"eventsSaved"`
}

// stableID creates deterministic identifiers for imported records.
func stableID(prefix, raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return prefix + "_" + hex.EncodeToString(sum[:16])
}

func normalizedHTTPURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" ||
		(!strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https")) {
		return "", errors.New("URL must be absolute http or https")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	return u.String(), nil
}

func domainOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	return host
}

func eventStatusFromLifecycle(lifecycle string) EventStatus {
	switch strings.ToLower(strings.TrimSpace(lifecycle)) {
	case "developing":
		return EventDeveloping
	case "stabilizing":
		return EventStabilizing
	case "archived":
		return EventArchived
	case "detected", "emerging":
		return EventEmerging
	default:
		// "confirmed" is NexNews' event-lifecycle label, not independent
		// Intelligence verification. Keep it under review rather than asserting
		// truth or confirmation at the product boundary.
		return EventNeedsReview
	}
}

func addLimitation(in []string, want string) []string {
	for _, item := range in {
		if item == want {
			return in
		}
	}
	return append(in, want)
}

// FromNexNews maps the core's snapshot into Intelligence records. It performs
// no network calls, never mutates the source Store, does not infer claims, and
// does not create entity relations from co-occurrence or title similarity.
func FromNexNews(input core.IntelligenceSnapshot) (ImportSnapshot, error) {
	captured := input.CapturedAt.UTC()
	if captured.IsZero() {
		captured = time.Now().UTC()
	}
	out := ImportSnapshot{
		SchemaVersion: SchemaVersion,
		CapturedAt: captured,
		Sources: []Source{},
		Articles: []Article{},
		Events: []Event{},
	}
	sourceByDomain := map[string]Source{}
	articleIDByURL := map[string]string{}
	articlesByURL := map[string]Article{}

	for _, row := range input.Articles {
		canonicalURL, err := normalizedHTTPURL(row.CanonicalURL)
		if err != nil {
			return ImportSnapshot{}, err
		}
		title := strings.TrimSpace(row.Title)
		if title == "" {
			continue
		}
		domain := domainOf(canonicalURL)
		if domain == "" {
			return ImportSnapshot{}, errors.New("article has no canonical domain")
		}
		sourceID := stableID("source", domain)
		sourceName := strings.TrimSpace(row.SourceName)
		if sourceName == "" {
			sourceName = domain
		}
		if existing, ok := sourceByDomain[domain]; ok {
			if existing.Name == domain && sourceName != "" {
				existing.Name = sourceName
				sourceByDomain[domain] = existing
			}
		} else {
			sourceByDomain[domain] = Source{
				ID: sourceID, Name: sourceName, CanonicalDomain: domain,
				Type: SourcePublisher, IdentityBasis: "canonical article URL domain",
				CreatedAt: captured,
			}
		}

		availability := ContentTitleOnly
		if strings.TrimSpace(row.Summary) != "" {
			availability = ContentTitleSummary
		}
		retrievedAt := row.RetrievedAt.UTC()
		if retrievedAt.IsZero() {
			retrievedAt = captured
		}
		articleID := stableID("article", canonicalURL)
		article := Article{
			ID: articleID, CanonicalURL: canonicalURL, PublisherSourceID: sourceID,
			Title: title, Summary: strings.TrimSpace(row.Summary),
			PublishedAt: row.PublishedAt, RetrievedAt: retrievedAt,
			FirstObservedAt: captured, LastObservedAt: captured,
			Language: strings.TrimSpace(row.Language),
			ContentAvailability: availability,
		}
		if existing, ok := articlesByURL[canonicalURL]; ok {
			// Multiple feed/category paths may expose the same article. Merge
			// metadata, but keep one canonical record.
			if existing.Summary == "" && article.Summary != "" {
				existing.Summary = article.Summary
				existing.ContentAvailability = article.ContentAvailability
			}
			if existing.PublishedAt == nil && article.PublishedAt != nil {
				existing.PublishedAt = article.PublishedAt
			}
			if existing.Language == "" {
				existing.Language = article.Language
			}
			if article.RetrievedAt.After(existing.RetrievedAt) {
				existing.RetrievedAt = article.RetrievedAt
			}
			articlesByURL[canonicalURL] = existing
			continue
		}
		articlesByURL[canonicalURL] = article
		articleIDByURL[canonicalURL] = articleID
	}

	for _, src := range sourceByDomain {
		out.Sources = append(out.Sources, src)
	}
	sort.Slice(out.Sources, func(i, j int) bool { return out.Sources[i].ID < out.Sources[j].ID })

	for _, article := range articlesByURL {
		if err := ValidateArticle(article); err != nil {
			return ImportSnapshot{}, err
		}
		out.Articles = append(out.Articles, article)
	}
	sort.Slice(out.Articles, func(i, j int) bool { return out.Articles[i].ID < out.Articles[j].ID })

	eventSeen := map[string]bool{}
	for _, row := range input.Events {
		candidateID := strings.TrimSpace(row.CandidateID)
		if candidateID == "" {
			continue
		}
		eventID := stableID("event_candidate", candidateID)
		if eventSeen[eventID] {
			continue
		}
		eventSeen[eventID] = true
		articleIDs := make([]string, 0, len(row.ArticleURLs))
		seenIDs := map[string]bool{}
		for _, rawURL := range row.ArticleURLs {
			u, err := normalizedHTTPURL(rawURL)
			if err != nil {
				continue // source event may reference a link absent from the snapshot
			}
			id, ok := articleIDByURL[u]
			if !ok || seenIDs[id] {
				continue
			}
			seenIDs[id] = true
			articleIDs = append(articleIDs, id)
		}
		sort.Strings(articleIDs)
		firstObserved := captured
		lastObserved := captured
		coverage := []string{
			"This is a NexNews event-cluster candidate, not an independently verified Intelligence finding.",
			"Relations, claims, and evidence are not inferred by the import adapter.",
		}
		if len(articleIDs) == 0 {
			coverage = addLimitation(coverage, "No article in this import snapshot could be resolved to the event candidate.")
		}
		if row.IndependentSources <= 1 {
			coverage = addLimitation(coverage, "Independent source support is limited or not independently reassessed by Intelligence.")
		}
		event := Event{
			ID: eventID, Title: strings.TrimSpace(row.Title), Scope: strings.TrimSpace(row.Category),
			Status: eventStatusFromLifecycle(row.Lifecycle),
			FirstObservedAt: firstObserved, LastObservedAt: lastObserved,
			ArticleIDs: articleIDs, ClaimIDs: []string{}, TimelineEntryIDs: []string{},
			CoverageLimitations: coverage, RuleVersion: ImporterVersion, UpdatedAt: captured,
		}
		if event.Title == "" {
			event.Title = candidateID
		}
		out.Events = append(out.Events, event)
	}
	sort.Slice(out.Events, func(i, j int) bool { return out.Events[i].ID < out.Events[j].ID })
	return out, nil
}
