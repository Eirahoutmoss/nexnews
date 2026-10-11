package intelligence

import (
	"sort"
	"strings"
	"time"
	"unicode"
)

// StatementOccurrence is one source-attributed occurrence of a literal statement.
// Publisher domains are counted for display, but never assumed to be independent
// origins: wire-copy and syndication lineage may still be unknown.
type StatementOccurrence struct {
	ClaimID         string      `json:"claimId"`
	ArticleID       string      `json:"articleId"`
	PublisherName   string      `json:"publisherName"`
	PublisherDomain string      `json:"publisherDomain"`
	ArticleTitle    string      `json:"articleTitle"`
	CanonicalURL    string      `json:"canonicalUrl"`
	Quote           string      `json:"quote"`
	ExcerptField    string      `json:"excerptField"`
	ClaimStatus     ClaimStatus `json:"claimStatus"`
	PublishedAt     *time.Time  `json:"publishedAt,omitempty"`
}

// StatementComparisonGroup groups only text that is identical after a narrow
// punctuation/spacing/case normalization. It is not a semantic or paraphrase
// match, and it is not a claim-verification verdict.
type StatementComparisonGroup struct {
	ID                   string                `json:"id"`
	Statement            string                `json:"statement"`
	ArticleCount         int                   `json:"articleCount"`
	PublisherDomainCount int                   `json:"publisherDomainCount"`
	Occurrences          []StatementOccurrence `json:"occurrences"`
	Assessment           string                `json:"assessment"`
	Limitations          []string              `json:"limitations"`
}

type SourceComparisonReport struct {
	GeneratedAt              time.Time                  `json:"generatedAt"`
	ArticlesReviewed         int                        `json:"articlesReviewed"`
	SourceStatementsReviewed int                        `json:"sourceStatementsReviewed"`
	UniqueStatementCount     int                        `json:"uniqueStatementCount"`
	RepeatedWordingGroups    int                        `json:"repeatedWordingGroups"`
	RepeatedWordingArticles  int                        `json:"repeatedWordingArticles"`
	Groups                   []StatementComparisonGroup `json:"groups"`
	Limitations              []string                   `json:"limitations"`
}

type comparisonAccumulator struct {
	key         string
	statement   string
	occurrences []StatementOccurrence
	articles    map[string]bool
	domains     map[string]bool
}

// comparisonKey intentionally does only surface normalization. It preserves
// word order and all words; no synonym, entity co-occurrence, or semantic
// similarity is used to put different statements into the same group.
func comparisonKey(text string) string {
	var b strings.Builder
	space := true
	for _, r := range text {
		folded := foldRune(r)
		for _, fr := range folded {
			if unicode.IsLetter(fr) || unicode.IsDigit(fr) {
				b.WriteRune(fr)
				space = false
			} else if !space {
				b.WriteByte(' ')
				space = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// BuildSourceComparisonReport compares literal, source-attributed statements
// already stored in the Intelligence database. Exact repeated wording is a
// review signal only. It does not infer that sources are independent, that the
// statement is true, or that differing wording is contradictory.
func BuildSourceComparisonReport(db Database) SourceComparisonReport {
	out := SourceComparisonReport{
		GeneratedAt: time.Now().UTC(),
		ArticlesReviewed: len(db.Articles),
		Groups: []StatementComparisonGroup{},
		Limitations: []string{
			"Only exact wording matches after case, spacing, and punctuation normalization are grouped; paraphrases are not compared.",
			"Different publisher domains do not prove independent reporting. Shared wire copy and syndication origin must be assessed separately.",
			"Repeated wording is neither proof that a claim is true nor proof that it is false.",
			"Conflicting or negated claims are not automatically labeled contradictory by this pass.",
		},
	}

	byKey := map[string]*comparisonAccumulator{}
	seenStatementArticle := map[string]bool{}
	for _, claim := range db.Claims {
		if claim.Kind != ClaimSourceStatement || strings.TrimSpace(claim.ClaimText) == "" {
			continue
		}
		foundEvidence := false
		for _, evidenceID := range claim.EvidenceIDs {
			evidence, ok := db.Evidence[evidenceID]
			if !ok || evidence.ArticleID == "" {
				continue
			}
			article, ok := db.Articles[evidence.ArticleID]
			if !ok {
				continue
			}
			key := comparisonKey(evidence.Quote)
			if key == "" {
				key = comparisonKey(claim.ClaimText)
			}
			if key == "" {
				continue
			}
			// One article may repeat the same sentence in both title and summary.
			// Count it once per normalized statement, regardless of excerpt field.
			occurrenceKey := key + "\x00" + article.ID
			if seenStatementArticle[occurrenceKey] {
				foundEvidence = true
				continue
			}
			seenStatementArticle[occurrenceKey] = true
			foundEvidence = true

			source := db.Sources[article.PublisherSourceID]
			domain := strings.ToLower(strings.TrimSpace(source.CanonicalDomain))
			if domain == "" {
				domain = strings.ToLower(strings.TrimSpace(article.PublisherSourceID))
			}
			name := strings.TrimSpace(source.Name)
			if name == "" {
				name = domain
			}
			occ := StatementOccurrence{
				ClaimID: claim.ID,
				ArticleID: article.ID,
				PublisherName: name,
				PublisherDomain: domain,
				ArticleTitle: article.Title,
				CanonicalURL: article.CanonicalURL,
				Quote: evidence.Quote,
				ExcerptField: evidence.ExcerptField,
				ClaimStatus: claim.Status,
				PublishedAt: article.PublishedAt,
			}
			group := byKey[key]
			if group == nil {
				group = &comparisonAccumulator{
					key: key,
					statement: evidence.Quote,
					occurrences: []StatementOccurrence{},
					articles: map[string]bool{},
					domains: map[string]bool{},
				}
				byKey[key] = group
			}
			group.occurrences = append(group.occurrences, occ)
			group.articles[article.ID] = true
			group.domains[domain] = true
		}
		if foundEvidence {
			out.SourceStatementsReviewed++
		}
	}

	out.UniqueStatementCount = len(byKey)
	articleInRepeatGroup := map[string]bool{}
	for _, group := range byKey {
		if len(group.articles) < 2 {
			continue
		}
		out.RepeatedWordingGroups++
		for articleID := range group.articles {
			articleInRepeatGroup[articleID] = true
		}
		occurrences := append([]StatementOccurrence(nil), group.occurrences...)
		sort.Slice(occurrences, func(i, j int) bool {
			if occurrences[i].PublisherDomain != occurrences[j].PublisherDomain {
				return occurrences[i].PublisherDomain < occurrences[j].PublisherDomain
			}
			if occurrences[i].CanonicalURL != occurrences[j].CanonicalURL {
				return occurrences[i].CanonicalURL < occurrences[j].CanonicalURL
			}
			return occurrences[i].ExcerptField < occurrences[j].ExcerptField
		})
		out.Groups = append(out.Groups, StatementComparisonGroup{
			ID: stableID("statement_comparison", group.key),
			Statement: group.statement,
			ArticleCount: len(group.articles),
			PublisherDomainCount: len(group.domains),
			Occurrences: occurrences,
			Assessment: "Birebir ifade tekrarı bulundu. Bu kayıtlar olası yeniden yayım veya ortak ajans metni olabilir; bağımsız doğrulama olarak sayılmamalıdır.",
			Limitations: []string{
				"Bu eşleşme yalnızca metin biçiminin aynı olduğunu gösterir; haberlerin aynı kaynaktan kopyalandığını kesinleştirmez.",
				"Yayıncı alan adı sayısı bağımsız kaynak sayısı değildir.",
				"İfadenin doğruluğu bu karşılaştırmayla belirlenmez.",
			},
		})
	}
	out.RepeatedWordingArticles = len(articleInRepeatGroup)
	sort.Slice(out.Groups, func(i, j int) bool {
		if out.Groups[i].ArticleCount != out.Groups[j].ArticleCount {
			return out.Groups[i].ArticleCount > out.Groups[j].ArticleCount
		}
		if out.Groups[i].PublisherDomainCount != out.Groups[j].PublisherDomainCount {
			return out.Groups[i].PublisherDomainCount > out.Groups[j].PublisherDomainCount
		}
		return out.Groups[i].Statement < out.Groups[j].Statement
	})
	return out
}
