package intelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ResearchImportStats struct {
	EventID          string `json:"eventId"`
	ArticlesSeen     int    `json:"articlesSeen"`
	EvidenceImported int    `json:"evidenceImported"`
	UnchangedSkipped int    `json:"unchangedSkipped"`
	RevisionsAdded   int    `json:"revisionsAdded"`
	ArticlesSkipped  int    `json:"articlesSkipped"`
	Warnings         []string `json:"warnings,omitempty"`
}

// SyncResearchCaseFromNexNews imports the linked NexNews event's article
// metadata into the research case. It reads only the Intelligence store's
// imported snapshot and does not claim that source reports are true. When a
// linked article changes, a new evidence revision is appended instead of
// overwriting the old record.
func (r *JSONRepository) SyncResearchCaseFromNexNews(caseID string) (ResearchCase, ResearchImportStats, error) {
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()

	db, err := r.loadLocked()
	if err != nil { return ResearchCase{}, ResearchImportStats{}, err }
	item, ok := db.ResearchCases[strings.TrimSpace(caseID)]
	if !ok { return ResearchCase{}, ResearchImportStats{}, errors.New("research case not found") }
	if strings.TrimSpace(item.RelatedEventID) == "" {
		return ResearchCase{}, ResearchImportStats{}, errors.New("research case is not linked to a NexNews event")
	}
	event, ok := db.Events[item.RelatedEventID]
	if !ok { return ResearchCase{}, ResearchImportStats{}, errors.New("related NexNews event was not found") }
	stats := ResearchImportStats{EventID:event.ID, Warnings:[]string{}}
	articleIDs := append([]string(nil), event.ArticleIDs...)
	sort.Strings(articleIDs)
	stats.ArticlesSeen = len(articleIDs)
	for _, articleID := range articleIDs {
		article, exists := db.Articles[articleID]
		if !exists {
			stats.ArticlesSkipped++
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("Event %s references missing article %s; it was not imported.", event.ID, articleID))
			continue
		}
		source, exists := db.Sources[article.PublisherSourceID]
		if !exists {
			stats.ArticlesSkipped++
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("Article %s has no publisher source record; it was not imported.", article.ID))
			continue
		}
		fingerprint := researchArticleFingerprint(article)
		alreadyImported, revised := false, false
		for _, previous := range item.Evidence {
			if previous.NexNewsArticleID != article.ID { continue }
			if previous.ContentFingerprint == fingerprint {
				alreadyImported = true
				break
			}
			revised = true
		}
		if alreadyImported {
			stats.UnchangedSkipped++
			continue
		}
		id, idErr := newResearchID("evidence")
		if idErr != nil { return ResearchCase{}, ResearchImportStats{}, idErr }
		contentLevel := ResearchContentTitleOnly
		excerpt := strings.TrimSpace(article.Title)
		if strings.TrimSpace(article.Summary) != "" {
			contentLevel = ResearchContentTitleSummary
			excerpt = strings.TrimSpace(article.Summary)
		}
		sourceKind := ResearchSourceOther
		switch source.Type {
		case SourcePrimary:
			sourceKind = ResearchSourcePrimary
		case SourceOfficialDataset:
			sourceKind = ResearchSourceDataset
		}
		sourceOrigin := ResearchOriginUnknown
		syndicationGroup := ""
		if strings.TrimSpace(article.SyndicationOriginID) != "" {
			sourceOrigin = ResearchOriginSyndicated
			syndicationGroup = strings.TrimSpace(article.SyndicationOriginID)
		}
		publisherName := strings.TrimSpace(source.Name)
		if publisherName == "" { publisherName = strings.TrimSpace(source.CanonicalDomain) }
		evidence := ResearchEvidence{
			ID:id, NexNewsArticleID:article.ID, ContentFingerprint:fingerprint,
			Kind:ResearchEvidenceArticle, Title:strings.TrimSpace(article.Title),
			SourceURL:strings.TrimSpace(article.CanonicalURL), Publisher:publisherName,
			PublisherDomain:normalizedPublisherDomain(source.CanonicalDomain),
			PublishedAt:article.PublishedAt, RetrievedAt:now, Excerpt:excerpt,
			ContentLevel:contentLevel, SourceKind:sourceKind, SourceOrigin:sourceOrigin,
			SyndicationGroup:syndicationGroup,
			MethodNote:fmt.Sprintf("Imported from NexNews event candidate %q. Only the title and summary fields present in the NexNews snapshot were imported; full article text was not added by this operation. Source truth and source independence have not been verified.", event.Title),
		}
		if err := validateImportedResearchEvidence(evidence); err != nil {
			stats.ArticlesSkipped++
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("Article %s failed research import validation: %v", article.ID, err))
			continue
		}
		item.Evidence = append(item.Evidence, evidence)
		if revised { stats.RevisionsAdded++ } else { stats.EvidenceImported++ }
	}
	item.UpdatedAt = now
	db.ResearchCases[item.ID] = item
	db.SavedAt = now
	if err := r.writeLocked(db); err != nil { return ResearchCase{}, ResearchImportStats{}, err }
	return item, stats, nil
}

func researchArticleFingerprint(article Article) string {
	published := ""
	if article.PublishedAt != nil { published = article.PublishedAt.UTC().Format(time.RFC3339Nano) }
	payload := strings.Join([]string{
		strings.TrimSpace(article.CanonicalURL),
		strings.TrimSpace(article.Title),
		strings.TrimSpace(article.Summary),
		strings.TrimSpace(article.ContentHash),
		published,
		string(article.ContentAvailability),
	}, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func validateImportedResearchEvidence(evidence ResearchEvidence) error {
	if strings.TrimSpace(evidence.NexNewsArticleID) == "" {
		return errors.New("NexNews article id is required")
	}
	if strings.TrimSpace(evidence.ContentFingerprint) == "" {
		return errors.New("article content fingerprint is required")
	}
	_, err := validateResearchEvidence(AddResearchEvidenceInput{
		Kind:evidence.Kind, Title:evidence.Title, SourceURL:evidence.SourceURL,
		Publisher:evidence.Publisher, PublisherDomain:evidence.PublisherDomain,
		PublishedAt:evidence.PublishedAt, Excerpt:evidence.Excerpt,
		ContentLevel:evidence.ContentLevel, SourceKind:evidence.SourceKind,
		SourceOrigin:evidence.SourceOrigin, SyndicationGroup:evidence.SyndicationGroup,
		MethodNote:evidence.MethodNote,
	})
	return err
}
