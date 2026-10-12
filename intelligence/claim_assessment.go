package intelligence

import (
	"sort"
	"strings"
	"time"
)

// ExplicitOppositionOccurrence is a literal source statement mapped by a narrow
// terminal-phrase rule. It is not a semantic parse and does not determine truth.
type ExplicitOppositionOccurrence struct {
	StatementOccurrence
	Polarity string `json:"polarity"`
}

// ExplicitOppositionCase is a review candidate where sources use the same
// normalized context and one of the allow-listed, directly opposing terminal
// phrases. "Candidate" is deliberate: the pass cannot fully interpret context,
// sarcasm, quotation, legal wording, or whether both texts refer to the same fact.
type ExplicitOppositionCase struct {
	ID              string                         `json:"id"`
	Context         string                         `json:"context"`
	AssertionType   string                         `json:"assertionType"`
	PositiveSources []ExplicitOppositionOccurrence `json:"positiveSources"`
	NegativeSources []ExplicitOppositionOccurrence `json:"negativeSources"`
	ArticleCount    int                            `json:"articleCount"`
	DomainCount     int                            `json:"domainCount"`
	Assessment      string                         `json:"assessment"`
	Limitations     []string                       `json:"limitations"`
}

type ClaimAssessmentReport struct {
	GeneratedAt              time.Time                `json:"generatedAt"`
	ArticlesReviewed         int                      `json:"articlesReviewed"`
	SourceStatementsReviewed int                      `json:"sourceStatementsReviewed"`
	ExplicitOppositionCases  int                      `json:"explicitOppositionCases"`
	Cases                    []ExplicitOppositionCase `json:"cases"`
	Limitations              []string                 `json:"limitations"`
}

type polarityPair struct {
	name     string
	positive string
	negative string
}

// This is intentionally a small allowlist of terminal forms that can be
// compared by exact shared context. It is not open-ended negation detection.
var explicitPolarityPairs = []polarityPair{
	{name: "Turkish: gerçekleşti / gerçekleşmedi", positive: "gerçekleşti", negative: "gerçekleşmedi"},
	{name: "Turkish: oldu / olmadı", positive: "oldu", negative: "olmadı"},
	{name: "Turkish: başladı / başlamadı", positive: "başladı", negative: "başlamadı"},
	{name: "Turkish: bitti / bitmedi", positive: "bitti", negative: "bitmedi"},
	{name: "Turkish: kabul etti / kabul etmedi", positive: "kabul etti", negative: "kabul etmedi"},
	{name: "Turkish: imzaladı / imzalamadı", positive: "imzaladı", negative: "imzalamadı"},
	{name: "Turkish: açıkladı / açıklamadı", positive: "açıkladı", negative: "açıklamadı"},
	{name: "Turkish: katıldı / katılmadı", positive: "katıldı", negative: "katılmadı"},
	{name: "Turkish: doğruladı / doğrulamadı", positive: "doğruladı", negative: "doğrulamadı"},
	{name: "Turkish: üretildi / üretilmedi", positive: "üretildi", negative: "üretilmedi"},
	{name: "English: happened / did not happen", positive: "happened", negative: "did not happen"},
	{name: "English: occurred / did not occur", positive: "occurred", negative: "did not occur"},
	{name: "English: started / did not start", positive: "started", negative: "did not start"},
	{name: "English: signed / did not sign", positive: "signed", negative: "did not sign"},
	{name: "English: approved / did not approve", positive: "approved", negative: "did not approve"},
	{name: "English: confirmed / did not confirm", positive: "confirmed", negative: "did not confirm"},
}

type explicitAssertion struct {
	key      string
	context  string
	pairName string
	polarity string
}

func parseExplicitAssertion(text string) (explicitAssertion, bool) {
	normalized := comparisonKey(text)
	if normalized == "" {
		return explicitAssertion{}, false
	}
	for _, pair := range explicitPolarityPairs {
		positive := comparisonKey(pair.positive)
		negative := comparisonKey(pair.negative)
		for _, candidate := range []struct {
			suffix   string
			polarity string
		}{
			{suffix: positive, polarity: "positive"},
			{suffix: negative, polarity: "negative"},
		} {
			if normalized == candidate.suffix || !strings.HasSuffix(normalized, " "+candidate.suffix) {
				continue
			}
			context := strings.TrimSpace(strings.TrimSuffix(normalized, candidate.suffix))
			// Require a meaningful shared context, not a bare verb or a one-word
			// fragment that could generate broad accidental pairings.
			if len(strings.Fields(context)) < 2 {
				continue
			}
			return explicitAssertion{
				key: context + "\x00" + pair.name,
				context: context,
				pairName: pair.name,
				polarity: candidate.polarity,
			}, true
		}
	}
	return explicitAssertion{}, false
}

type assertionBucket struct {
	context string
	pairName string
	positive []ExplicitOppositionOccurrence
	negative []ExplicitOppositionOccurrence
}

func knownPublishedTimeDistance(a, b *time.Time) (time.Duration, bool) {
	if a == nil || b == nil || a.IsZero() || b.IsZero() {
		return 0, false
	}
	d := a.Sub(*b)
	if d < 0 {
		d = -d
	}
	return d, true
}

const maxOppositionTimeDistance = 7 * 24 * time.Hour

func occurrenceLess(a, b ExplicitOppositionOccurrence) bool {
	if a.PublishedAt != nil && b.PublishedAt != nil && !a.PublishedAt.Equal(*b.PublishedAt) {
		return a.PublishedAt.Before(*b.PublishedAt)
	}
	if a.PublisherDomain != b.PublisherDomain {
		return a.PublisherDomain < b.PublisherDomain
	}
	return a.CanonicalURL < b.CanonicalURL
}

// BuildClaimAssessmentReport identifies a deliberately narrow class of possible
// conflicts: the same normalized statement context with an allow-listed opposing
// terminal phrase, supported by different stored article records. It does not
// infer paraphrases, general semantic contradiction, truth, or source independence.
// Records with known publication timestamps more than seven days apart are kept
// apart to reduce accidental matches across historical events.
func BuildClaimAssessmentReport(db Database) ClaimAssessmentReport {
	out := ClaimAssessmentReport{
		GeneratedAt: time.Now().UTC(),
		ArticlesReviewed: len(db.Articles),
		Cases: []ExplicitOppositionCase{},
		Limitations: []string{
			"Only a narrow allowlist of explicit terminal phrase pairs is compared; paraphrases and most contradictions are not detected.",
			"Each case is a review candidate, not a verdict that either statement is true or false.",
			"Different publisher domains do not prove independent reporting or independent evidence.",
			"Known publication times more than seven days apart are not paired; missing timestamps weaken temporal assessment.",
			"Shared words and entity co-occurrence are never sufficient to create a comparison case.",
		},
	}

	// Sort keys to make output deterministic despite Go map iteration order.
	claimIDs := make([]string, 0, len(db.Claims))
	for id := range db.Claims {
		claimIDs = append(claimIDs, id)
	}
	sort.Strings(claimIDs)

	buckets := map[string]*assertionBucket{}
	seenArticleAssertion := map[string]bool{}
	for _, claimID := range claimIDs {
		claim := db.Claims[claimID]
		if claim.Kind != ClaimSourceStatement || strings.TrimSpace(claim.ClaimText) == "" {
			continue
		}
		evidenceIDs := append([]string(nil), claim.EvidenceIDs...)
		sort.Strings(evidenceIDs)
		foundStatement := false
		for _, evidenceID := range evidenceIDs {
			evidence, ok := db.Evidence[evidenceID]
			if !ok || evidence.ArticleID == "" {
				continue
			}
			article, ok := db.Articles[evidence.ArticleID]
			if !ok {
				continue
			}
			parsed, ok := parseExplicitAssertion(evidence.Quote)
			if !ok {
				continue
			}
			dedupKey := parsed.key + "\x00" + article.ID + "\x00" + parsed.polarity
			if seenArticleAssertion[dedupKey] {
				foundStatement = true
				continue
			}
			seenArticleAssertion[dedupKey] = true
			foundStatement = true

			source := db.Sources[article.PublisherSourceID]
			domain := strings.ToLower(strings.TrimSpace(source.CanonicalDomain))
			if domain == "" {
				domain = strings.ToLower(strings.TrimSpace(article.PublisherSourceID))
			}
			name := strings.TrimSpace(source.Name)
			if name == "" {
				name = domain
			}
			occurrence := ExplicitOppositionOccurrence{
				StatementOccurrence: StatementOccurrence{
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
				},
				Polarity: parsed.polarity,
			}
			bucket := buckets[parsed.key]
			if bucket == nil {
				bucket = &assertionBucket{context: parsed.context, pairName: parsed.pairName}
				buckets[parsed.key] = bucket
			}
			if parsed.polarity == "positive" {
				bucket.positive = append(bucket.positive, occurrence)
			} else {
				bucket.negative = append(bucket.negative, occurrence)
			}
		}
		if foundStatement {
			out.SourceStatementsReviewed++
		}
	}

	type indexedOccurrence struct {
		value ExplicitOppositionOccurrence
		polarity string
	}
	for bucketKey, bucket := range buckets {
		if len(bucket.positive) == 0 || len(bucket.negative) == 0 {
			continue
		}
		all := make([]indexedOccurrence, 0, len(bucket.positive)+len(bucket.negative))
		for _, v := range bucket.positive {
			all = append(all, indexedOccurrence{value: v, polarity: "positive"})
		}
		for _, v := range bucket.negative {
			all = append(all, indexedOccurrence{value: v, polarity: "negative"})
		}
		sort.Slice(all, func(i, j int) bool {
			if all[i].value.ArticleID != all[j].value.ArticleID {
				return all[i].value.ArticleID < all[j].value.ArticleID
			}
			return all[i].polarity < all[j].polarity
		})

		// Build bipartite components: an occurrence joins a case only if it
		// can pair with an explicitly opposite occurrence in the same time window.
		adj := make([][]int, len(all))
		for i := range all {
			for j := i + 1; j < len(all); j++ {
				if all[i].polarity == all[j].polarity || all[i].value.ArticleID == all[j].value.ArticleID {
					continue
				}
				distance, known := knownPublishedTimeDistance(all[i].value.PublishedAt, all[j].value.PublishedAt)
				if known && distance > maxOppositionTimeDistance {
					continue
				}
				adj[i] = append(adj[i], j)
				adj[j] = append(adj[j], i)
			}
		}
		visited := make([]bool, len(all))
		componentNo := 0
		for i := range all {
			if visited[i] || len(adj[i]) == 0 {
				continue
			}
			componentNo++
			queue := []int{i}
			visited[i] = true
			component := []int{}
			for len(queue) > 0 {
				current := queue[0]
				queue = queue[1:]
				component = append(component, current)
				for _, next := range adj[current] {
					if !visited[next] {
						visited[next] = true
						queue = append(queue, next)
					}
				}
			}

			pos, neg := []ExplicitOppositionOccurrence{}, []ExplicitOppositionOccurrence{}
			articles, domains := map[string]bool{}, map[string]bool{}
			hasMissingTimestamp := false
			for _, index := range component {
				item := all[index]
				articles[item.value.ArticleID] = true
				domains[item.value.PublisherDomain] = true
				if item.value.PublishedAt == nil || item.value.PublishedAt.IsZero() {
					hasMissingTimestamp = true
				}
				if item.polarity == "positive" {
					pos = append(pos, item.value)
				} else {
					neg = append(neg, item.value)
				}
			}
			if len(pos) == 0 || len(neg) == 0 {
				continue
			}
			sort.Slice(pos, func(i, j int) bool { return occurrenceLess(pos[i], pos[j]) })
			sort.Slice(neg, func(i, j int) bool { return occurrenceLess(neg[i], neg[j]) })
			limitations := []string{
				"The comparison matched only an allow-listed opposite terminal phrase and identical normalized preceding text.",
				"This is a possible contradiction signal for human review, not a factual verdict.",
				"Publisher-domain count is not an independent-source count.",
			}
			if hasMissingTimestamp {
				limitations = append(limitations, "At least one publication timestamp is missing; temporal alignment is uncertain.")
			}
			out.Cases = append(out.Cases, ExplicitOppositionCase{
				ID: stableID("claim_assessment", bucketKey+"\x00"+strings.Join(sortedOccurrenceIDs(pos, neg), "\x00")),
				Context: bucket.context,
				AssertionType: bucket.pairName,
				PositiveSources: pos,
				NegativeSources: neg,
				ArticleCount: len(articles),
				DomainCount: len(domains),
				Assessment: "Aynı bağlam için açıkça karşıt iki ifade bulundu. Her iki tarafın kaynaklarını inceleyin; sistem hangi ifadenin doğru olduğunu belirlemedi.",
				Limitations: limitations,
			})
		}
	}

	sort.Slice(out.Cases, func(i, j int) bool {
		if out.Cases[i].ArticleCount != out.Cases[j].ArticleCount {
			return out.Cases[i].ArticleCount > out.Cases[j].ArticleCount
		}
		if out.Cases[i].Context != out.Cases[j].Context {
			return out.Cases[i].Context < out.Cases[j].Context
		}
		return out.Cases[i].ID < out.Cases[j].ID
	})
	out.ExplicitOppositionCases = len(out.Cases)
	return out
}

func sortedOccurrenceIDs(pos, neg []ExplicitOppositionOccurrence) []string {
	ids := make([]string, 0, len(pos)+len(neg))
	for _, item := range pos {
		ids = append(ids, "p:"+item.ArticleID+":"+item.Quote)
	}
	for _, item := range neg {
		ids = append(ids, "n:"+item.ArticleID+":"+item.Quote)
	}
	sort.Strings(ids)
	return ids
}
