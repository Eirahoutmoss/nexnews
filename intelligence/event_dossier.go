package intelligence

import (
	"errors"
	"sort"
	"strings"
	"time"
)

// EventDossierArticle is a source-linked article in one event candidate.
// The order is by publication time when known; missing times remain explicit.
type EventDossierArticle struct {
	ID                 string              `json:"id"`
	Title              string              `json:"title"`
	Summary            string              `json:"summary,omitempty"`
	CanonicalURL       string              `json:"canonicalUrl"`
	PublisherName      string              `json:"publisherName"`
	PublisherDomain    string              `json:"publisherDomain"`
	PublishedAt        *time.Time          `json:"publishedAt,omitempty"`
	RetrievedAt        time.Time           `json:"retrievedAt"`
	ContentAvailability ContentAvailability `json:"contentAvailability"`
}

// EventDossierSource counts articles by publisher domain. This is coverage,
// not a count of independent reporting origins.
type EventDossierSource struct {
	PublisherName   string `json:"publisherName"`
	PublisherDomain string `json:"publisherDomain"`
	ArticleCount    int    `json:"articleCount"`
}

// EventDossierStatement is a literal excerpt from a stored source statement.
type EventDossierStatement struct {
	ClaimID        string      `json:"claimId"`
	ArticleID      string      `json:"articleId"`
	Quote          string      `json:"quote"`
	ExcerptField   string      `json:"excerptField"`
	SourceURL      string      `json:"sourceUrl"`
	PublisherName  string      `json:"publisherName"`
	PublisherDomain string     `json:"publisherDomain"`
	ClaimStatus    ClaimStatus `json:"claimStatus"`
	PublishedAt    *time.Time  `json:"publishedAt,omitempty"`
}

// EventDossierEntity collects exact, source-span entity mentions within the
// event's article set. It does not mean the names are correctly disambiguated.
type EventDossierEntity struct {
	EntityID       string                `json:"entityId"`
	CanonicalName  string                `json:"canonicalName"`
	EntityType     EntityType            `json:"entityType"`
	IdentityStatus IdentityStatus        `json:"identityStatus"`
	ArticleCount   int                   `json:"articleCount"`
	MentionCount   int                   `json:"mentionCount"`
	Mentions       []EventEntityMention  `json:"mentions"`
}

type EventEntityMention struct {
	ArticleID       string `json:"articleId"`
	ArticleTitle    string `json:"articleTitle"`
	CanonicalURL    string `json:"canonicalUrl"`
	PublisherDomain string `json:"publisherDomain"`
	Field           string `json:"field"`
	Quote           string `json:"quote"`
	StartOffset     int    `json:"startOffset"`
	EndOffset       int    `json:"endOffset"`
}

// EventDossier turns a candidate cluster into an inspectable evidence file.
// Its headline and counts describe stored coverage; it is not a generated
// factual summary and does not verify the event itself.
type EventDossier struct {
	GeneratedAt              time.Time                     `json:"generatedAt"`
	Event                    Event                         `json:"event"`
	ArticleCount             int                           `json:"articleCount"`
	PublisherDomainCount     int                           `json:"publisherDomainCount"`
	ArticlesWithoutDate      int                           `json:"articlesWithoutDate"`
	TitleOnlyArticles        int                           `json:"titleOnlyArticles"`
	ArticlesWithSummary      int                           `json:"articlesWithSummary"`
	FirstPublishedAt         *time.Time                    `json:"firstPublishedAt,omitempty"`
	LastPublishedAt          *time.Time                    `json:"lastPublishedAt,omitempty"`
	Sources                  []EventDossierSource          `json:"sources"`
	Articles                 []EventDossierArticle         `json:"articles"`
	StatementCount           int                           `json:"statementCount"`
	Statements               []EventDossierStatement       `json:"statements"`
	RepeatedWordingGroups    []StatementComparisonGroup    `json:"repeatedWordingGroups"`
	ConflictCandidates       []ExplicitOppositionCase      `json:"conflictCandidates"`
	Entities                 []EventDossierEntity          `json:"entities"`
	OpenChecks               []string                      `json:"openChecks"`
	Limitations              []string                      `json:"limitations"`
}

func dossierArticleLess(a, b EventDossierArticle) bool {
	aKnown := a.PublishedAt != nil && !a.PublishedAt.IsZero()
	bKnown := b.PublishedAt != nil && !b.PublishedAt.IsZero()
	if aKnown != bKnown {
		return aKnown
	}
	if aKnown && !a.PublishedAt.Equal(*b.PublishedAt) {
		return a.PublishedAt.Before(*b.PublishedAt)
	}
	if !a.RetrievedAt.Equal(b.RetrievedAt) {
		return a.RetrievedAt.Before(b.RetrievedAt)
	}
	return a.ID < b.ID
}

// BuildEventDossier produces one on-demand dossier from the existing database.
// It deliberately uses only articles assigned to the selected candidate and
// literal stored evidence. It does not make network calls or create relations.
func BuildEventDossier(db Database, eventID string) (EventDossier, error) {
	event, ok := db.Events[eventID]
	if !ok {
		return EventDossier{}, errors.New("event candidate not found")
	}
	out := EventDossier{
		GeneratedAt: time.Now().UTC(),
		Event: event,
		Sources: []EventDossierSource{},
		Articles: []EventDossierArticle{},
		Statements: []EventDossierStatement{},
		RepeatedWordingGroups: []StatementComparisonGroup{},
		ConflictCandidates: []ExplicitOppositionCase{},
		Entities: []EventDossierEntity{},
		OpenChecks: []string{},
		Limitations: []string{
			"Bu dosya NexNews olay grubu adayıdır; olayın doğruluğu veya haberlerin aynı olaya ait olduğu bağımsız olarak doğrulanmış değildir.",
			"Yayıncı alan adı sayısı bağımsız kaynak sayısı değildir; ajans kökeni ve ortak metin ayrıca incelenmelidir.",
			"Zaman çizelgesi haberlerin yayın zamanına göre sıralanır; olayın gerçek gerçekleşme zamanı olduğu varsayılmaz.",
			"Kaynak ifadeleri, kaynağın ne söylediğini gösterir; ifadenin doğru olduğunu tek başına kanıtlamaz.",
		},
	}

	articleIDs := map[string]bool{}
	sourceCounts := map[string]int{}
	sourceNames := map[string]string{}
	sub := emptyDatabase()
	sub.Schema, sub.SavedAt, sub.LastSyncAt = db.Schema, db.SavedAt, db.LastSyncAt
	sub.Events[event.ID] = event
	for _, id := range event.ArticleIDs {
		if articleIDs[id] {
			continue
		}
		article, exists := db.Articles[id]
		if !exists {
			out.OpenChecks = append(out.OpenChecks, "Olay adayının listesinde kayıtlı olmayan haber kimliği var: "+id)
			continue
		}
		articleIDs[id] = true
		sub.Articles[id] = article
		source, sourceExists := db.Sources[article.PublisherSourceID]
		name, domain := article.PublisherSourceID, article.PublisherSourceID
		if sourceExists {
			if strings.TrimSpace(source.Name) != "" {
				name = source.Name
			}
			if strings.TrimSpace(source.CanonicalDomain) != "" {
				domain = strings.ToLower(strings.TrimSpace(source.CanonicalDomain))
			}
			sub.Sources[source.ID] = source
		}
		sourceCounts[domain]++
		if _, exists := sourceNames[domain]; !exists || sourceNames[domain] == domain {
			sourceNames[domain] = name
		}
		out.Articles = append(out.Articles, EventDossierArticle{
			ID: article.ID, Title: article.Title, Summary: article.Summary,
			CanonicalURL: article.CanonicalURL, PublisherName: name, PublisherDomain: domain,
			PublishedAt: article.PublishedAt, RetrievedAt: article.RetrievedAt,
			ContentAvailability: article.ContentAvailability,
		})
		if article.PublishedAt == nil || article.PublishedAt.IsZero() {
			out.ArticlesWithoutDate++
		} else {
			t := article.PublishedAt.UTC()
			if out.FirstPublishedAt == nil || t.Before(*out.FirstPublishedAt) {
				copy := t
				out.FirstPublishedAt = &copy
			}
			if out.LastPublishedAt == nil || t.After(*out.LastPublishedAt) {
				copy := t
				out.LastPublishedAt = &copy
			}
		}
		if article.ContentAvailability == ContentTitleOnly {
			out.TitleOnlyArticles++
		}
		if strings.TrimSpace(article.Summary) != "" {
			out.ArticlesWithSummary++
		}
	}
	sort.Slice(out.Articles, func(i, j int) bool { return dossierArticleLess(out.Articles[i], out.Articles[j]) })
	out.ArticleCount = len(out.Articles)
	out.PublisherDomainCount = len(sourceCounts)
	for domain, count := range sourceCounts {
		out.Sources = append(out.Sources, EventDossierSource{
			PublisherName: sourceNames[domain], PublisherDomain: domain, ArticleCount: count,
		})
	}
	sort.Slice(out.Sources, func(i, j int) bool {
		if out.Sources[i].ArticleCount != out.Sources[j].ArticleCount {
			return out.Sources[i].ArticleCount > out.Sources[j].ArticleCount
		}
		return out.Sources[i].PublisherDomain < out.Sources[j].PublisherDomain
	})

	// Claims and evidence are included only when the associated evidence belongs
	// to an article in this dossier, preventing nearby/unrelated claims leaking in.
	for claimID, claim := range db.Claims {
		if claim.Kind != ClaimSourceStatement {
			continue
		}
		selectedEvidenceIDs := []string{}
		for _, evidenceID := range claim.EvidenceIDs {
			ev, exists := db.Evidence[evidenceID]
			if !exists || !articleIDs[ev.ArticleID] {
				continue
			}
			selectedEvidenceIDs = append(selectedEvidenceIDs, evidenceID)
			sub.Evidence[evidenceID] = ev
			article := db.Articles[ev.ArticleID]
			source := db.Sources[article.PublisherSourceID]
			domain, name := strings.ToLower(strings.TrimSpace(source.CanonicalDomain)), strings.TrimSpace(source.Name)
			if domain == "" { domain = article.PublisherSourceID }
			if name == "" { name = domain }
			out.Statements = append(out.Statements, EventDossierStatement{
				ClaimID: claim.ID, ArticleID: article.ID, Quote: ev.Quote,
				ExcerptField: ev.ExcerptField, SourceURL: ev.SourceURL,
				PublisherName: name, PublisherDomain: domain,
				ClaimStatus: claim.Status, PublishedAt: article.PublishedAt,
			})
		}
		if len(selectedEvidenceIDs) > 0 {
			claim.EvidenceIDs = selectedEvidenceIDs
			sub.Claims[claimID] = claim
		}
	}
	seenStatements := map[string]bool{}
	statements := out.Statements[:0]
	for _, statement := range out.Statements {
		key := statement.ArticleID + "\x00" + comparisonKey(statement.Quote)
		if seenStatements[key] { continue }
		seenStatements[key] = true
		statements = append(statements, statement)
	}
	out.Statements = statements
	out.StatementCount = len(out.Statements)
	sort.Slice(out.Statements, func(i, j int) bool {
		a, b := out.Statements[i], out.Statements[j]
		if a.PublishedAt != nil && b.PublishedAt != nil && !a.PublishedAt.Equal(*b.PublishedAt) {
			return a.PublishedAt.Before(*b.PublishedAt)
		}
		if a.PublishedAt != nil && b.PublishedAt == nil { return true }
		if a.PublishedAt == nil && b.PublishedAt != nil { return false }
		if a.PublisherDomain != b.PublisherDomain { return a.PublisherDomain < b.PublisherDomain }
		return a.Quote < b.Quote
	})

	// Reuse the existing narrow comparison rules on this dossier's subset only.
	comparison := BuildSourceComparisonReport(sub)
	out.RepeatedWordingGroups = comparison.Groups
	assessment := BuildClaimAssessmentReport(sub)
	out.ConflictCandidates = assessment.Cases

	mentionGroups := map[string]*EventDossierEntity{}
	mentionArticleSets := map[string]map[string]bool{}
	for id, mention := range db.EntityMentions {
		if !articleIDs[mention.ArticleID] { continue }
		article, exists := db.Articles[mention.ArticleID]
		if !exists { continue }
		entity := db.Entities[mention.EntityID]
		group := mentionGroups[mention.EntityID]
		if group == nil {
			group = &EventDossierEntity{
				EntityID: mention.EntityID, CanonicalName: entity.CanonicalName,
				EntityType: entity.Type, IdentityStatus: entity.IdentityStatus,
				Mentions: []EventEntityMention{},
			}
			if group.CanonicalName == "" { group.CanonicalName = mention.Canonical }
			if group.EntityType == "" { group.EntityType = mention.EntityType }
			if group.IdentityStatus == "" { group.IdentityStatus = IdentityCandidate }
			mentionGroups[mention.EntityID] = group
			mentionArticleSets[mention.EntityID] = map[string]bool{}
		}
		source := db.Sources[article.PublisherSourceID]
		domain := strings.ToLower(strings.TrimSpace(source.CanonicalDomain))
		if domain == "" { domain = article.PublisherSourceID }
		group.MentionCount++
		mentionArticleSets[mention.EntityID][article.ID] = true
		group.Mentions = append(group.Mentions, EventEntityMention{
			ArticleID: article.ID, ArticleTitle: article.Title, CanonicalURL: article.CanonicalURL,
			PublisherDomain: domain, Field: mention.Field, Quote: mention.Quote,
			StartOffset: mention.StartOffset, EndOffset: mention.EndOffset,
		})
		_ = id
	}
	for id, group := range mentionGroups {
		group.ArticleCount = len(mentionArticleSets[id])
		sort.Slice(group.Mentions, func(i, j int) bool {
			a, b := group.Mentions[i], group.Mentions[j]
			if a.ArticleID != b.ArticleID { return a.ArticleID < b.ArticleID }
			if a.Field != b.Field { return a.Field < b.Field }
			return a.StartOffset < b.StartOffset
		})
		if len(group.Mentions) > 10 { group.Mentions = group.Mentions[:10] }
		out.Entities = append(out.Entities, *group)
	}
	sort.Slice(out.Entities, func(i, j int) bool {
		if out.Entities[i].ArticleCount != out.Entities[j].ArticleCount { return out.Entities[i].ArticleCount > out.Entities[j].ArticleCount }
		if out.Entities[i].MentionCount != out.Entities[j].MentionCount { return out.Entities[i].MentionCount > out.Entities[j].MentionCount }
		return out.Entities[i].CanonicalName < out.Entities[j].CanonicalName
	})
	if len(out.Entities) > 80 { out.Entities = out.Entities[:80] }

	if out.ArticleCount == 0 {
		out.OpenChecks = append(out.OpenChecks, "Bu aday için kullanılabilir haber kaydı yok; kaynak grubunun yeniden incelenmesi gerekiyor.")
	}
	if out.ArticlesWithoutDate > 0 {
		out.OpenChecks = append(out.OpenChecks, "Yayın zamanı bulunmayan "+itoa(out.ArticlesWithoutDate)+" haber var; zaman çizelgesi bu kayıtlar için kesin sıralama veremez.")
	}
	if out.TitleOnlyArticles > 0 {
		out.OpenChecks = append(out.OpenChecks, "Yalnızca başlık bulunan "+itoa(out.TitleOnlyArticles)+" haber var; özet/tam metin kontrolü kapsamı genişletebilir.")
	}
	if out.StatementCount == 0 {
		out.OpenChecks = append(out.OpenChecks, "Bu dosyada kaynak ifadesi kaydı yok; yerel analiz hattı çalıştırılmalı veya kaynak metni kapsamı incelenmeli.")
	}
	if len(out.ConflictCandidates) == 0 {
		out.OpenChecks = append(out.OpenChecks, "Bu dosyada mevcut dar karşıt ifade kurallarıyla inceleme adayı bulunmadı; bu, çelişki olmadığı anlamına gelmez.")
	}
	if len(out.RepeatedWordingGroups) == 0 {
		out.OpenChecks = append(out.OpenChecks, "Aynı ifade tekrarı bulunmadı; farklı anlatımlar bu sürümde semantik olarak eşdeğer sayılmaz.")
	}
	sort.Strings(out.OpenChecks)
	return out, nil
}

func itoa(n int) string {
	if n == 0 { return "0" }
	negative := n < 0
	if negative { n = -n }
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}
