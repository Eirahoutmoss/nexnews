package core

// Phase 2 — event files (a story's timeline) and advanced search.
//
// An "event file" is just a Story shown in full: its member headlines ordered in
// time. Search matches stored items by terms (+ an optional time window parsed
// from natural phrases like "son 24 saat" / "last 7 days"), then clusters the
// matches into events using the same deduplicator, so a query returns events,
// not a flat list of near-duplicate headlines.

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// --- event detail ---------------------------------------------------------

type tlItem struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Link   string `json:"link"`
	TS     int64  `json:"ts"`
	New    bool   `json:"new,omitempty"` // RC97: kullanıcının önceki ziyaretinden sonra geldi
}

type outEvent struct {
	Found              bool           `json:"found"`
	Coverage           *coverage      `json:"coverage,omitempty"`
	Title              string         `json:"title"`
	Category           string         `json:"category"`
	Kind               string         `json:"kind"`
	Score              int            `json:"score"`
	Sources            []string       `json:"sources"`
	SourceCount        int            `json:"sourceCount"`
	Count              int            `json:"count"`
	FirstTS            int64          `json:"firstTs"`
	LastTS             int64          `json:"lastTs"`
	Summary            string         `json:"summary,omitempty"`
	Latest             string         `json:"latest,omitempty"`
	Timeline           []tlItem       `json:"timeline"`
	RelLevel           string         `json:"relLevel"` // yüksek/orta/düşük
	RelText            string         `json:"relText"`
	EventID            string         `json:"eventId,omitempty"`
	IndependentSources int            `json:"independentSources"`
	PrimarySources     int            `json:"primarySources"`
	Confidence         int            `json:"confidence"`
	Velocity           float64        `json:"velocity"`
	Lifecycle          string         `json:"lifecycle"`
	Reason             string         `json:"reason,omitempty"`
	Evidence           EventEvidence  `json:"evidence,omitempty"`
	Followed           bool           `json:"followed"`
	Headlines          []headlineView `json:"headlines,omitempty"` // "Kim nasıl başlık attı?"
	Sens               int            `json:"sens,omitempty"`      // RC97: temsilci başlığın duygu yükü
	CalmTitle          string         `json:"calmTitle,omitempty"` // RC97: olayın en sakin başlığı
	CalmSource         string         `json:"calmSource,omitempty"`
	LastVisit          int64          `json:"lastVisit,omitempty"` // RC97: önceki ziyaret (ne değişti?)
	NewSince           int            `json:"newSince,omitempty"`  // önceki ziyaretten bu yana gelen başlık
	FactChecks         []factMatch    `json:"factChecks,omitempty"`
}

// headlineView is one outlet's headline for the framing comparison.
type headlineView struct {
	Source     string   `json:"source"`
	Title      string   `json:"title"`                // as published
	Translated string   `json:"translated,omitempty"` // reading-language version when different
	Link       string   `json:"link"`
	TS         int64    `json:"ts"`
	Via        string   `json:"via,omitempty"`      // wire agency copy
	Distinct   []string `json:"distinct,omitempty"` // words only this outlet used
	Sens       int      `json:"sens,omitempty"`     // RC97: başlığın duygu yükü (0–100)
}

// framingHeadlines returns one headline per outlet with the words that set
// it apart from the others (what this outlet chose to emphasise).
func (s *Store) framingHeadlines(st *Story) []headlineView {
	type cand struct {
		it    Item
		words []string
		stems []string
	}
	var cs []cand
	seen := map[string]bool{}
	for _, it := range st.items {
		key := sourceIdentity(it)
		if key == "" {
			key = it.Source
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		c := cand{it: it}
		for _, w := range strings.FieldsFunc(s.trTitle(it.Title), func(r rune) bool {
			return !(unicode.IsLetter(r) || unicode.IsDigit(r))
		}) {
			lw := strings.ToLower(w)
			if len([]rune(lw)) < 3 || stopwords[lw] {
				continue
			}
			c.words = append(c.words, w)
			c.stems = append(c.stems, firstKey(stemTokens(map[string]bool{lw: true})))
		}
		cs = append(cs, c)
	}
	if len(cs) < 2 {
		return nil
	}
	df := map[string]int{}
	for _, c := range cs {
		u := map[string]bool{}
		for _, st := range c.stems {
			if !u[st] {
				u[st] = true
				df[st]++
			}
		}
	}
	out := make([]headlineView, 0, len(cs))
	for _, c := range cs {
		hv := headlineView{Source: c.it.Source, Title: c.it.Title, Link: c.it.Link}
		if tr := s.trTitle(c.it.Title); tr != c.it.Title {
			hv.Translated = tr
		}
		if c.it.HasDate {
			hv.TS = c.it.Published.Unix()
		}
		_, hv.Via = wireOf(c.it)
		hv.Sens = sensationScore(s.trTitle(c.it.Title))
		dup := map[string]bool{}
		for i, w := range c.words {
			if df[c.stems[i]] == 1 && !dup[c.stems[i]] {
				dup[c.stems[i]] = true
				hv.Distinct = append(hv.Distinct, w)
			}
		}
		out = append(out, hv)
	}
	return out
}

func firstKey(m map[string]bool) string {
	for k := range m {
		return k
	}
	return ""
}

// findStory locates a story by its representative link across all categories.
func (s *Store) findStory(link string) *Story {
	for _, cl := range s.storiesByCat {
		for _, st := range cl {
			if st.Link == link {
				return st
			}
		}
	}
	return nil
}

func (s *Store) trTitle(title string) string {
	if s.trOn && s.trTarget != "" {
		if tr, ok := s.tr.cached(s.trTarget, title); ok && tr != "" {
			return tr
		}
	}
	return title
}

// storyJSON returns an event file (timeline ascending by time) for a story.
func (s *Store) storyJSON(link string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := s.findStory(link)
	if st == nil {
		return json.Marshal(outEvent{Found: false})
	}
	e := outEvent{
		Found:       true,
		Title:       s.trTitle(st.Title),
		Category:    st.Category,
		Kind:        st.Kind,
		Score:       st.Score,
		Sources:     st.Sources,
		SourceCount: len(st.Sources),
		Count:       st.Count,
		Summary:     st.Summary,
	}
	if st.HasDate {
		e.FirstTS = st.FirstSeen.Unix()
		e.LastTS = st.Published.Unix()
	}
	e.Latest = s.trTitle(st.Title)
	e.RelLevel, e.RelText = reliability(st.IndependentSources, st.PrimarySources, st.Confidence)
	e.EventID = st.EventID
	e.IndependentSources = st.IndependentSources
	e.PrimarySources = st.PrimarySources
	e.Confidence = st.Confidence
	e.Velocity = st.Velocity
	e.Lifecycle = st.Lifecycle
	e.Reason = eventReason(st)
	e.Evidence = evidenceForStory(st)
	e.Followed = follows.isStory(st)
	e.Headlines = s.framingHeadlines(st)
	e.Coverage = s.coverageFor(st)
	e.CalmTitle, e.CalmSource, e.Sens = s.calmestTitle(st)
	e.FactChecks = factChecksFor(s.storyTokens(st), st)
	prevVisit := visits.touch(visitKey(st))
	e.LastVisit = prevVisit

	// Timeline ascending (oldest → newest), like an event log.
	items := make([]Item, len(st.items))
	copy(items, st.items)
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	for _, it := range items {
		var ts int64
		if it.HasDate {
			ts = it.Published.Unix()
		}
		tl := tlItem{
			Title:  s.trTitle(it.Title),
			Source: it.Source,
			Link:   it.Link,
			TS:     ts,
		}
		// Yeni: önceki ziyaretten sonra yayımlanmış (tarihsizse ilk görülme sayılmaz).
		if prevVisit > 0 && ts > prevVisit {
			tl.New = true
			e.NewSince++
		}
		e.Timeline = append(e.Timeline, tl)
	}
	return json.Marshal(e)
}

// reliability turns the number of corroborating sources into a trust signal.
// It shows the source structure, not a truth verdict.
func reliability(independent int, extras ...int) (string, string) {
	primary, confidence := 0, 0
	if len(extras) > 0 {
		primary = extras[0]
	}
	if len(extras) > 1 {
		confidence = extras[1]
	}

	// Preserve the public/legacy reliability contract: the number of
	// genuinely independent sources is the fallback signal. Confidence can
	// refine it when explicitly supplied by Event Intelligence, but must not
	// make the legacy 2/4-source cases less trustworthy than their contract.
	level := "düşük"
	switch {
	case independent >= 4:
		level = "yüksek"
	case independent >= 2:
		level = "orta"
	}
	if confidence > 0 {
		if confidence >= 80 && independent >= 4 {
			level = "yüksek"
		} else if confidence >= 55 && independent >= 2 && level == "düşük" {
			level = "orta"
		}
	}

	tail := "henüz sınırlı bağımsız teyit"
	if level == "orta" {
		tail = "çoklu kaynak desteği var"
	} else if level == "yüksek" {
		tail = "yüksek düzeyde kaynak mutabakatı"
	}
	return level, fmtReliability(independent, primary, tail)
}

func fmtReliability(independent, primary int, tail string) string {
	base := strconv.Itoa(independent) + " bağımsız kaynak"
	if primary > 0 {
		base += ", " + strconv.Itoa(primary) + " birincil/resmî kaynak"
	}
	return base + " — " + tail + "."
}

// --- advanced search ------------------------------------------------------

type outSearch struct {
	Query       string    `json:"query"`
	WindowLabel string    `json:"windowLabel"`
	Total       int       `json:"total"`
	SourceCount int       `json:"sourceCount"`
	Items       []outItem `json:"items"`
}

var (
	reSonN   = regexp.MustCompile(`(?:son|last)\s+(\d+)\s*(saat|gün|gun|day|days|hour|hours|hafta|week|weeks)`)
	reNUnit  = regexp.MustCompile(`(\d+)\s*(saat|gün|gun|day|days|hour|hours|hafta|week|weeks)`)
	reSimple = regexp.MustCompile(`\b(bugün|bugun|today|dün|dun|yesterday|son\s+hafta|geçen\s+hafta|gecen\s+hafta|last\s+week|bu\s+hafta|this\s+week)\b`)
)

// parseWindow extracts a time window from the query, returning the duration (0 =
// no limit), a human label, and the remaining search terms.
func parseWindow(q string) (time.Duration, string, string) {
	low := " " + strings.ToLower(strings.TrimSpace(q)) + " "

	unitDur := func(n int, unit string) (time.Duration, string) {
		switch {
		case strings.HasPrefix(unit, "saat"), strings.HasPrefix(unit, "hour"):
			return time.Duration(n) * time.Hour, "son " + strconv.Itoa(n) + " saat"
		case strings.HasPrefix(unit, "hafta"), strings.HasPrefix(unit, "week"):
			return time.Duration(n) * 7 * 24 * time.Hour, "son " + strconv.Itoa(n) + " hafta"
		default: // gün/day
			return time.Duration(n) * 24 * time.Hour, "son " + strconv.Itoa(n) + " gün"
		}
	}

	if m := reSonN.FindStringSubmatch(low); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n > 0 {
			d, label := unitDur(n, m[2])
			return d, label, strings.TrimSpace(reSonN.ReplaceAllString(low, " "))
		}
	}
	if m := reSimple.FindStringSubmatch(low); m != nil {
		w := strings.TrimSpace(m[1])
		var d time.Duration
		var label string
		switch {
		case strings.Contains(w, "hafta") || strings.Contains(w, "week"):
			d, label = 7*24*time.Hour, "son 7 gün"
		case strings.Contains(w, "dün") || strings.Contains(w, "dun") || strings.Contains(w, "yesterday"):
			d, label = 48*time.Hour, "son 2 gün"
		default: // bugün / today
			d, label = 24*time.Hour, "bugün"
		}
		return d, label, strings.TrimSpace(reSimple.ReplaceAllString(low, " "))
	}
	if m := reNUnit.FindStringSubmatch(low); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n > 0 {
			d, label := unitDur(n, m[2])
			return d, label, strings.TrimSpace(reNUnit.ReplaceAllString(low, " "))
		}
	}
	return 0, "", strings.TrimSpace(low)
}

// searchJSON runs an advanced search: term match (+ optional window) → cluster
// into events → rank.
func (s *Store) searchJSON(query string) ([]byte, error) {
	dur, label, rest := parseWindow(query)

	terms := []string{}
	for _, t := range strings.Fields(rest) {
		if len([]rune(t)) >= 2 {
			terms = append(terms, t)
		}
	}

	res := outSearch{Query: strings.TrimSpace(query), WindowLabel: label}

	s.mu.RLock()
	cutoff := time.Time{}
	if dur > 0 {
		cutoff = time.Now().Add(-dur)
	}
	var matched []Item
	for _, list := range s.byCat {
		for _, it := range list {
			if dur > 0 {
				if !it.HasDate || it.Published.Before(cutoff) {
					continue
				}
			}
			if matchesTerms(it, terms) {
				matched = append(matched, it)
			}
		}
	}
	s.mu.RUnlock()

	// sort matched newest-first so clustering picks good representatives
	sortItemsNewest(matched)
	clusters := clusterItems(matched)
	rankStories(clusters)

	srcSet := map[string]bool{}
	s.mu.RLock()
	for _, st := range clusters {
		res.Items = append(res.Items, s.storyOut(st))
		for _, src := range st.Sources {
			srcSet[src] = true
		}
		if len(res.Items) >= 50 {
			break
		}
	}
	s.mu.RUnlock()
	res.Total = len(clusters)
	res.SourceCount = len(srcSet)

	return json.Marshal(res)
}

// matchesTerms requires every term to appear (substring) in the title; an empty
// term list matches everything (useful for a pure time-window query).
func matchesTerms(it Item, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	low := strings.ToLower(it.Title)
	for _, t := range terms {
		if !strings.Contains(low, t) {
			return false
		}
	}
	return true
}

// storyForFollow finds the story to follow by its link, falling back to the id.
func (s *Store) storyForFollow(link, id string) *Story {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if link != "" {
		if st := s.findStory(link); st != nil {
			return st
		}
	}
	if id != "" {
		for _, cl := range s.storiesByCat {
			for _, st := range cl {
				if st.EventID == id {
					return st
				}
			}
		}
	}
	return nil
}
