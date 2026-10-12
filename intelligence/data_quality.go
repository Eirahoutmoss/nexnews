package intelligence

import (
	"net/url"
	"sort"
	"strings"
	"time"
)

// DataQualityIssue is a deterministic, actionable finding about stored record
// integrity. It does not judge the truth of a news statement.
type DataQualityIssue struct {
	ID         string `json:"id"`
	Severity   string `json:"severity"`
	Code       string `json:"code"`
	RecordType string `json:"recordType"`
	RecordID   string `json:"recordId"`
	Message    string `json:"message"`
	SourceURL  string `json:"sourceUrl,omitempty"`
}

// DataQualityReport describes references and evidence coverage without
// changing the database. Missing publication dates and title-only articles
// are coverage metrics, not automatic integrity failures.
type DataQualityReport struct {
	GeneratedAt               time.Time          `json:"generatedAt"`
	RecordsChecked             int                `json:"recordsChecked"`
	ErrorCount                 int                `json:"errorCount"`
	WarningCount               int                `json:"warningCount"`
	InfoCount                  int                `json:"infoCount"`
	ArticlesWithoutPublishedAt int                `json:"articlesWithoutPublishedAt"`
	TitleOnlyArticles          int                `json:"titleOnlyArticles"`
	EventsNeedingReview        int                `json:"eventsNeedingReview"`
	Issues                     []DataQualityIssue `json:"issues"`
}

func validQualityURL(raw string) bool {
	if !validHTTPURL(raw) {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && u.Hostname() != ""
}

// BuildDataQualityReport inspects referential integrity, evidence linkage and
// source-span offsets. Findings are review aids, not truth assessments. The
// report is read-only and sorts output deterministically for reproducibility.
func BuildDataQualityReport(db Database) DataQualityReport {
	out := DataQualityReport{
		GeneratedAt: time.Now().UTC(),
		Issues:      []DataQualityIssue{},
	}
	add := func(severity, code, kind, id, message, sourceURL string) {
		out.Issues = append(out.Issues, DataQualityIssue{
			ID: stableID("quality_issue", severity+"\x00"+code+"\x00"+kind+"\x00"+id+"\x00"+message),
			Severity: severity, Code: code, RecordType: kind, RecordID: id,
			Message: message, SourceURL: sourceURL,
		})
	}

	out.RecordsChecked = len(db.Sources) + len(db.Articles) + len(db.Events) +
		len(db.Claims) + len(db.Evidence) + len(db.Entities) +
		len(db.EntityMentions) + len(db.Relations) + len(db.Timeline)

	for id, source := range db.Sources {
		if strings.TrimSpace(source.CanonicalDomain) == "" {
			add("error", "source_domain_missing", "source", id, "Kaynağın kanonik alan adı boş.", "")
		}
	}

	for id, article := range db.Articles {
		sourceURL := article.CanonicalURL
		if strings.TrimSpace(article.PublisherSourceID) == "" {
			add("error", "article_source_missing", "article", id, "Haberin yayıncı kaynak kimliği boş.", sourceURL)
		} else if _, ok := db.Sources[article.PublisherSourceID]; !ok {
			add("error", "article_source_unresolved", "article", id, "Haberin yayıncı kaydı veritabanında bulunamadı: "+article.PublisherSourceID, sourceURL)
		}
		if !validQualityURL(article.CanonicalURL) {
			add("error", "article_url_invalid", "article", id, "Haberin kanonik URL'si geçerli bir HTTP(S) adresi değil.", "")
		}
		if strings.TrimSpace(article.Title) == "" {
			add("error", "article_title_missing", "article", id, "Haber başlığı boş.", sourceURL)
		}
		if article.PublishedAt == nil || article.PublishedAt.IsZero() {
			out.ArticlesWithoutPublishedAt++
		}
		if article.ContentAvailability == ContentTitleOnly {
			out.TitleOnlyArticles++
		}
		for _, eventID := range article.EventCandidateIDs {
			if _, ok := db.Events[eventID]; !ok {
				add("warning", "article_event_unresolved", "article", id, "Haber, bulunmayan olay adayına işaret ediyor: "+eventID, sourceURL)
			}
		}
	}

	for id, event := range db.Events {
		if len(event.ArticleIDs) == 0 {
			add("warning", "event_without_articles", "event", id, "Olay adayı herhangi bir haber belgesine bağlı değil.", "")
		}
		if event.Status == EventNeedsReview {
			out.EventsNeedingReview++
		}
		for _, articleID := range event.ArticleIDs {
			if _, ok := db.Articles[articleID]; !ok {
				add("error", "event_article_unresolved", "event", id, "Olay adayı bulunmayan haber kaydına işaret ediyor: "+articleID, "")
			}
		}
		for _, claimID := range event.ClaimIDs {
			if _, ok := db.Claims[claimID]; !ok {
				add("error", "event_claim_unresolved", "event", id, "Olay adayı bulunmayan iddia kaydına işaret ediyor: "+claimID, "")
			}
		}
		for _, timelineID := range event.TimelineEntryIDs {
			if _, ok := db.Timeline[timelineID]; !ok {
				add("error", "event_timeline_unresolved", "event", id, "Olay adayı bulunmayan zaman çizelgesi kaydına işaret ediyor: "+timelineID, "")
			}
		}
	}

	for id, claim := range db.Claims {
		if strings.TrimSpace(claim.EventID) != "" {
			if _, ok := db.Events[claim.EventID]; !ok {
				add("error", "claim_event_unresolved", "claim", id, "İddia bulunmayan olay kaydına işaret ediyor: "+claim.EventID, "")
			}
		}
		if len(claim.EvidenceIDs) == 0 {
			add("warning", "claim_without_evidence", "claim", id, "İddia kaydına bağlı kanıt kimliği yok.", "")
		}
		for _, evidenceID := range claim.EvidenceIDs {
			evidence, ok := db.Evidence[evidenceID]
			if !ok {
				add("error", "claim_evidence_unresolved", "claim", id, "İddia bulunmayan kanıt kaydına işaret ediyor: "+evidenceID, "")
				continue
			}
			if evidence.ClaimID != "" && evidence.ClaimID != id {
				add("error", "claim_evidence_mismatch", "claim", id, "Bağlı kanıt başka bir iddia kaydını gösteriyor: "+evidenceID, evidence.SourceURL)
			}
		}
	}

	for id, evidence := range db.Evidence {
		if _, ok := db.Articles[evidence.ArticleID]; !ok || evidence.ArticleID == "" {
			add("error", "evidence_article_unresolved", "evidence", id, "Kanıt alıntısının bağlı olduğu haber kaydı bulunamadı.", evidence.SourceURL)
		}
		if evidence.ClaimID != "" {
			if _, ok := db.Claims[evidence.ClaimID]; !ok {
				add("error", "evidence_claim_unresolved", "evidence", id, "Kanıtın bağlı olduğu iddia kaydı bulunamadı: "+evidence.ClaimID, evidence.SourceURL)
			}
		}
		if strings.TrimSpace(evidence.Quote) == "" {
			add("error", "evidence_quote_missing", "evidence", id, "Kanıt alıntısı boş.", evidence.SourceURL)
		}
		if !validQualityURL(evidence.SourceURL) {
			add("error", "evidence_url_invalid", "evidence", id, "Kanıtın kaynak URL'si geçerli bir HTTP(S) adresi değil.", "")
		}
		if (evidence.StartOffset == nil) != (evidence.EndOffset == nil) {
			add("warning", "evidence_offset_incomplete", "evidence", id, "Kanıtın başlangıç/bitiş konumlarından yalnızca biri kayıtlı.", evidence.SourceURL)
		} else if evidence.StartOffset != nil && evidence.EndOffset != nil {
			if article, ok := db.Articles[evidence.ArticleID]; ok {
				var field string
				switch evidence.ExcerptField {
				case "title":
					field = article.Title
				case "summary":
					field = article.Summary
				}
				if field != "" {
					start, end := *evidence.StartOffset, *evidence.EndOffset
					if start < 0 || end < start || end > len(field) || field[start:end] != evidence.Quote {
						add("error", "evidence_offset_mismatch", "evidence", id, "Kanıt konumları, kayıtlı başlık/özet içinde aynı alıntıya karşılık gelmiyor.", evidence.SourceURL)
					}
				}
			}
		}
	}

	for id, mention := range db.EntityMentions {
		article, articleOK := db.Articles[mention.ArticleID]
		if !articleOK {
			add("error", "mention_article_unresolved", "mention", id, "Varlık anılmasının bağlı olduğu haber kaydı bulunamadı.", "")
		}
		if _, ok := db.Entities[mention.EntityID]; !ok {
			add("error", "mention_entity_unresolved", "mention", id, "Varlık anılması bulunmayan varlık kaydına işaret ediyor: "+mention.EntityID, "")
		}
		if articleOK {
			var field string
			switch mention.Field {
			case "title":
				field = article.Title
			case "summary":
				field = article.Summary
			}
			if field != "" {
				start, end := mention.StartOffset, mention.EndOffset
				if start < 0 || end <= start || end > len(field) || field[start:end] != mention.Quote {
					add("error", "mention_offset_mismatch", "mention", id, "Anılma konumları, kayıtlı başlık/özet içindeki alıntıyla uyuşmuyor.", article.CanonicalURL)
				}
			}
		}
	}

	for id, relation := range db.Relations {
		if len(relation.EvidenceIDs) == 0 {
			severity := "warning"
			if relation.Status == RelationSupported {
				severity = "error"
			}
			add(severity, "relation_without_evidence", "relation", id, "İlişki kaydında destekleyen kanıt kimliği yok.", "")
		}
		for _, evidenceID := range relation.EvidenceIDs {
			if _, ok := db.Evidence[evidenceID]; !ok {
				add("error", "relation_evidence_unresolved", "relation", id, "İlişki bulunmayan kanıt kaydına işaret ediyor: "+evidenceID, "")
			}
		}
		for _, node := range []NodeRef{relation.Subject, relation.Object} {
			exists := false
			switch node.Type {
			case NodeEntity:
				_, exists = db.Entities[node.ID]
			case NodeEvent:
				_, exists = db.Events[node.ID]
			case NodeClaim:
				_, exists = db.Claims[node.ID]
			case NodeArticle:
				_, exists = db.Articles[node.ID]
			}
			if !exists {
				add("error", "relation_node_unresolved", "relation", id, "İlişkinin düğümü bulunamadı: "+string(node.Type)+"/"+node.ID, "")
			}
		}
	}

	for id, entry := range db.Timeline {
		if _, ok := db.Events[entry.EventID]; !ok {
			add("error", "timeline_event_unresolved", "timeline", id, "Zaman çizelgesi kaydının bağlı olduğu olay bulunamadı.", "")
		}
		if article, ok := db.Articles[entry.ArticleID]; !ok {
			add("error", "timeline_article_unresolved", "timeline", id, "Zaman çizelgesi kaydının bağlı olduğu haber bulunamadı.", "")
		} else if strings.TrimSpace(article.CanonicalURL) != "" {
			// Keep the source link available to the local UI for any related repair.
			_ = article.CanonicalURL
		}
		for _, evidenceID := range entry.EvidenceIDs {
			if _, ok := db.Evidence[evidenceID]; !ok {
				add("error", "timeline_evidence_unresolved", "timeline", id, "Zaman çizelgesi kaydı bulunmayan kanıta işaret ediyor: "+evidenceID, "")
			}
		}
	}

	for _, issue := range out.Issues {
		switch issue.Severity {
		case "error":
			out.ErrorCount++
		case "warning":
			out.WarningCount++
		case "info":
			out.InfoCount++
		}
	}
	sort.Slice(out.Issues, func(i, j int) bool {
		order := map[string]int{"error": 0, "warning": 1, "info": 2}
		if order[out.Issues[i].Severity] != order[out.Issues[j].Severity] {
			return order[out.Issues[i].Severity] < order[out.Issues[j].Severity]
		}
		if out.Issues[i].Code != out.Issues[j].Code {
			return out.Issues[i].Code < out.Issues[j].Code
		}
		if out.Issues[i].RecordID != out.Issues[j].RecordID {
			return out.Issues[i].RecordID < out.Issues[j].RecordID
		}
		return out.Issues[i].ID < out.Issues[j].ID
	})
	return out
}
