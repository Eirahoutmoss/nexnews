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

const EntityCandidateExtractorVersion = "entity-lexicon/1"

type EntityExtractionStats struct {
	ArticlesScanned int      `json:"articlesScanned"`
	MentionsFound   int      `json:"mentionsFound"`
	EntitiesCreated int      `json:"entitiesCreated"`
	Warnings        []string `json:"warnings,omitempty"`
	RunID           string   `json:"runId"`
}

type entityLexiconEntry struct {
	name    string
	kind    EntityType
	aliases []string
}

// EntityCandidate is intentionally a curated lexicon, not open-ended NER.
// It produces candidate mentions only when the text contains an exact known
// name or alias. Unknown strings are not guessed to be people/organizations.
var entityCandidateLexicon = []entityLexiconEntry{
	{name: "Birleşmiş Milletler", kind: EntityOrganization, aliases: []string{"BM", "United Nations", "UN"}},
	{name: "Avrupa Birliği", kind: EntityOrganization, aliases: []string{"AB", "European Union", "EU"}},
	{name: "NATO", kind: EntityOrganization, aliases: []string{"North Atlantic Treaty Organization"}},
	{name: "Anadolu Ajansı", kind: EntityOrganization, aliases: []string{"AA", "Anadolu Agency"}},
	{name: "Reuters", kind: EntityOrganization, aliases: []string{"Reuters News Agency"}},
	{name: "Associated Press", kind: EntityOrganization, aliases: []string{"AP News", "AP"}},
	{name: "ABD", kind: EntityPlace, aliases: []string{"Amerika Birleşik Devletleri", "United States", "USA", "US"}},
	{name: "Birleşik Krallık", kind: EntityPlace, aliases: []string{"United Kingdom", "UK", "Britain"}},
	{name: "Türkiye", kind: EntityPlace, aliases: []string{"Turkey", "Turkiye"}},
	{name: "İran", kind: EntityPlace, aliases: []string{"Iran"}},
	{name: "İsrail", kind: EntityPlace, aliases: []string{"Israel"}},
	{name: "Ukrayna", kind: EntityPlace, aliases: []string{"Ukraine"}},
	{name: "Rusya", kind: EntityPlace, aliases: []string{"Russia"}},
	{name: "Çin", kind: EntityPlace, aliases: []string{"China", "PRC"}},
	{name: "Suriye", kind: EntityPlace, aliases: []string{"Syria"}},
	{name: "Gazze", kind: EntityPlace, aliases: []string{"Gaza", "Gaza Strip"}},
	{name: "Avrupa", kind: EntityPlace, aliases: []string{"Europe"}},
	{name: "Orta Doğu", kind: EntityPlace, aliases: []string{"Middle East"}},
	{name: "Kızıldeniz", kind: EntityPlace, aliases: []string{"Red Sea"}},
	{name: "Karadeniz", kind: EntityPlace, aliases: []string{"Black Sea"}},
}

type EntityMention struct {
	EntityID    string    `json:"entityId"`
	Canonical  string    `json:"canonicalName"`
	EntityType  EntityType `json:"entityType"`
	AliasMatched string   `json:"aliasMatched"`
	ArticleID   string    `json:"articleId"`
	Field       string    `json:"field"`
	Quote       string    `json:"quote"`
	StartOffset int       `json:"startOffset"`
	EndOffset   int       `json:"endOffset"`
	ObservedAt  time.Time `json:"observedAt"`
	Status      IdentityStatus `json:"identityStatus"`
}

type entityCandidateResult struct {
	Entities []Entity        `json:"entities"`
	Mentions []EntityMention `json:"mentions"`
	Warnings []string        `json:"warnings,omitempty"`
}

type lexiconMatch struct {
	start, end int
	entry entityLexiconEntry
	alias string
}

func foldForMatch(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case 'ı':
			b.WriteRune('i')
		case 'İ':
			b.WriteRune('i')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func entityStableID(name string, kind EntityType) string {
	sum := sha256.Sum256([]byte(string(kind)+"\x00"+foldForMatch(name)))
	return "entity_"+hex.EncodeToString(sum[:16])
}

func hasWordBoundary(s string, start, end int) bool {
	isWord := func(r rune) bool { return r == '_' || r == '-' || r == '\'' || r == '’' || isLetterOrDigit(r) }
	if start > 0 {
		r, _ := runeBefore(s, start)
		if isWord(r) { return false }
	}
	if end < len(s) {
		r, _ := runeAt(s, end)
		if isWord(r) { return false }
	}
	return true
}

func isLetterOrDigit(r rune) bool {
	return r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
		r >= 'A' && r <= 'Z' || r >= 0x00C0 && r <= 0x02AF || r >= 0x0370 && r <= 0x052F
}

func runeBefore(s string, at int) (rune, int) {
	if at <= 0 || at > len(s) { return 0, 0 }
	for i := at-1; i >= 0; i-- {
		if s[i]&0xC0 != 0x80 {
			r, size := runeAt(s, i)
			return r, size
		}
	}
	return 0, 0
}
func runeAt(s string, at int) (rune, int) {
	if at < 0 || at >= len(s) { return 0, 0 }
	for _, r := range s[at:] { return r, len(string(r)) }
	return 0, 0
}

func findLexiconMatches(text string) []lexiconMatch {
	folded := foldForMatch(text)
	matches := []lexiconMatch{}
	for _, entry := range entityCandidateLexicon {
		aliases := append([]string{entry.name}, entry.aliases...)
		for _, alias := range aliases {
			needle := foldForMatch(alias)
			if needle == "" { continue }
			for from := 0; from < len(folded); {
				rel := strings.Index(folded[from:], needle)
				if rel < 0 { break }
				start := from + rel
				end := start + len(needle)
				from = start + 1
				if !hasWordBoundary(folded, start, end) { continue }
				matches = append(matches, lexiconMatch{start:start, end:end, entry:entry, alias:alias})
			}
		}
	}
	// Prefer the longest alias when aliases overlap, then stable positions/names.
	sort.Slice(matches, func(i,j int) bool {
		li, lj := matches[i].end-matches[i].start, matches[j].end-matches[j].start
		if li != lj { return li > lj }
		if matches[i].start != matches[j].start { return matches[i].start < matches[j].start }
		return matches[i].entry.name < matches[j].entry.name
	})
	out := []lexiconMatch{}
	for _, m := range matches {
		overlap := false
		for _, prior := range out {
			if m.start < prior.end && prior.start < m.end { overlap = true; break }
		}
		if !overlap { out = append(out,m) }
	}
	sort.Slice(out, func(i,j int) bool {
		if out[i].start != out[j].start { return out[i].start < out[j].start }
		return out[i].entry.name < out[j].entry.name
	})
	return out
}

// ExtractEntityCandidates finds only exact matches from a small, versioned
// alias list. Every mention keeps an exact quote and byte offsets into the
// stored title/summary. Identity remains "candidate": no entity resolution,
// co-reference, relation, or real-world fact is inferred.
func ExtractEntityCandidates(article Article, observedAt time.Time) (entityCandidateResult, error) {
	if err := ValidateArticle(article); err != nil { return entityCandidateResult{}, err }
	if observedAt.IsZero() { observedAt = time.Now().UTC() } else { observedAt = observedAt.UTC() }
	out := entityCandidateResult{Entities: []Entity{}, Mentions: []EntityMention{}, Warnings: []string{
		"Entity matches come from a limited deterministic alias lexicon; absence does not mean the article contains no entities.",
		"All matches are identity candidates. No disambiguation or semantic relationship is inferred.",
	}}
	entityMap := map[string]Entity{}
	fields := []TextExcerpt{{Field:"title", Text:article.Title}}
	if strings.TrimSpace(article.Summary) != "" && article.ContentAvailability != ContentTitleOnly {
		fields = append(fields, TextExcerpt{Field:"summary", Text:article.Summary})
	}
	seenMentions := map[string]bool{}
	for _, field := range fields {
		matches := findLexiconMatches(field.Text)
		for _, m := range matches {
			quote := field.Text[m.start:m.end]
			entityID := entityStableID(m.entry.name, m.entry.kind)
			if _, ok := entityMap[entityID]; !ok {
				aliases := append([]string(nil), m.entry.aliases...)
				entityMap[entityID] = Entity{
					ID: entityID, CanonicalName: m.entry.name, Type: m.entry.kind,
					Aliases: aliases, IdentityStatus: IdentityCandidate,
				}
			}
			key := article.ID+"\x00"+field.Field+"\x00"+fmt.Sprint(m.start)+"\x00"+fmt.Sprint(m.end)+"\x00"+entityID
			if seenMentions[key] { continue }
			seenMentions[key] = true
			out.Mentions = append(out.Mentions, EntityMention{
				EntityID:entityID, Canonical:m.entry.name, EntityType:m.entry.kind,
				AliasMatched:m.alias, ArticleID:article.ID, Field:field.Field,
				Quote:quote, StartOffset:m.start, EndOffset:m.end,
				ObservedAt:observedAt, Status:IdentityCandidate,
			})
		}
	}
	for _, e := range entityMap { out.Entities = append(out.Entities,e) }
	sort.Slice(out.Entities, func(i,j int) bool { return out.Entities[i].ID < out.Entities[j].ID })
	sort.Slice(out.Mentions, func(i,j int) bool {
		if out.Mentions[i].ArticleID != out.Mentions[j].ArticleID { return out.Mentions[i].ArticleID < out.Mentions[j].ArticleID }
		if out.Mentions[i].Field != out.Mentions[j].Field { return out.Mentions[i].Field < out.Mentions[j].Field }
		return out.Mentions[i].StartOffset < out.Mentions[j].StartOffset
	})
	return out,nil
}

func (r *JSONRepository) ExtractStoredEntityCandidates() (EntityExtractionStats, error) {
	started := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	db, err := r.loadLocked()
	if err != nil { return EntityExtractionStats{}, err }
	stats := EntityExtractionStats{Warnings: []{}}
	articleIDs := make([]string,0,len(db.Articles))
	for id := range db.Articles { articleIDs=append(articleIDs,id) }
	sort.Strings(articleIDs)
	stats.ArticlesScanned=len(articleIDs)
	newEntities:=map[string]Entity{}
	for _, articleID := range articleIDs {
		article:=db.Articles[articleID]
		result, err:=ExtractEntityCandidates(article,started)
		if err != nil { return EntityExtractionStats{}, fmt.Errorf("extract entities for %q: %w",articleID,err) }
		stats.MentionsFound += len(result.Mentions)
		for _,e:=range result.Entities {
			if prior,ok:=db.Entities[e.ID]; ok {
				if prior.IdentityStatus==IdentityResolved || prior.IdentityStatus==IdentityAmbiguous {
					e.IdentityStatus=prior.IdentityStatus
				}
				e.ResolutionEvidenceIDs=append([]string(nil),prior.ResolutionEvidenceIDs...)
			}
			newEntities[e.ID]=e
		}
	}
	for id,e:=range newEntities {
		if _,ok:=db.Entities[id]; !ok { stats.EntitiesCreated++ }
		db.Entities[id]=e
	}
	stats.RunID=stableID("analysis_run",EntityCandidateExtractorVersion+"\x00"+started.Format(time.RFC3339Nano))
	completed:=time.Now().UTC()
	db.Runs[stats.RunID]=AnalysisRun{
		ID:stats.RunID,InputIDs:articleIDs,RuleVersion:EntityCandidateExtractorVersion,
		StartedAt:started,CompletedAt:&completed,Status:"completed",Warnings:stats.Warnings,
	}
	db.SavedAt=completed
	if err:=r.writeLocked(db); err!=nil { return EntityExtractionStats{},err }
	return stats,nil
}

var _ = errors.New
