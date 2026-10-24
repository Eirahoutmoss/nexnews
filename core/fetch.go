package core

// Store holds the aggregated headlines in memory and refreshes them in the
// background. Binding callbacks (which run on the WebView2 UI thread) only ever
// read from this cache, so the UI never blocks on the network.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store struct {
	cfg *Config

	mu           sync.RWMutex
	byCat        map[string][]Item   // full, date-sorted list per category
	storiesByCat map[string][]*Story // deduped/ranked stories per category
	important    []*Story            // cross-category, top-ranked stories
	order        []string            // category order as configured
	lastFetch    time.Time
	fetching     bool

	client       *http.Client
	repo         *diskRepository
	sourceHealth map[string]SourceHealth
	cache        map[string]FeedCacheEntry

	tr       *translator
	trOn     bool
	trTarget string

	interests []string // lowercased
	hide      []string // lowercased

	notified   map[string]bool // stories already toasted
	firstAlert bool            // seed on first refresh (no toast flood)
}

// alerts returns newly-appeared important stories to notify about (breaking, or
// score >= minScore), skipping anything already notified and the hide filter.
// The first call after startup only seeds the set so the user isn't flooded.
func (s *Store) alerts(minScore int) []*Story {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Story
	for _, st := range s.important {
		if st.Link == "" || s.notified[st.Link] || s.isHidden(st) {
			continue
		}
		if st.Breaking || st.Score >= minScore {
			s.notified[st.Link] = true
			if !s.firstAlert && len(out) < 2 {
				out = append(out, st)
			}
		}
	}
	s.firstAlert = false
	return out
}

func (s *Store) setPersonal(interests, hide []string) {
	lc := func(in []string) []string {
		out := make([]string, 0, len(in))
		for _, w := range in {
			w = strings.ToLower(strings.TrimSpace(w))
			if w != "" {
				out = append(out, w)
			}
		}
		return out
	}
	s.mu.Lock()
	s.interests = lc(interests)
	s.hide = lc(hide)
	s.mu.Unlock()
}

// isHidden applies contextual suppression rather than blindly matching a word.
// A hidden topic is suppressed when it is the subject of the story, but a
// security/crime/public-safety story that merely mentions the topic remains
// visible. Unknown terms keep the legacy literal-match behaviour.
func (s *Store) isHidden(st *Story) bool {
	if st == nil || len(s.hide) == 0 {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(st.Title + " " + st.Summary))
	for _, w := range s.hide {
		w = strings.ToLower(strings.TrimSpace(w))
		if w == "" || !strings.Contains(text, w) {
			continue
		}
		if contextualTopicSuppressed(w, text, st.Kind) {
			return true
		}
	}
	return false
}

func contextualTopicSuppressed(term, text, kind string) bool {
	// Crypto is the canonical contextual example: market/investment stories
	// are suppressible, while crime, fraud, laundering, cyberattack and legal
	// enforcement stories remain visible.
	crypto := term == "kripto" || term == "kripto para" || term == "bitcoin" || term == "ethereum" || term == "btc" || term == "eth"
	if crypto {
		exception := []string{
			"dolandırıc", "dolandiric", "kara para", "aklama", "suç", "suc", "operasyon", "soruştur", "sorustur", "gözalt", "gozalti", "tutukla", "dava", "mahkeme", "savc", "polis", "jandarma", "hack", "siber", "saldırı", "saldiri", "fidye", "çalın", "calin", "çalıntı", "calinti", "hırsız", "hirsiz", "hırsızlık", "hirsizlik", "el koy", "el kon", "elkon", "müsadere", "musadere", "haciz", "zapt", "kaçak", "kacak", "sahte", "terör", "teror", "yaptırım", "yaptirim", "ceza", "soruşturma",
		}
		for _, x := range exception {
			if strings.Contains(text, x) {
				return false
			}
		}
		// A crypto term in a market/pricing/investment context is treated as
		// the main subject and can therefore be hidden.
		market := []string{
			"fiyat", "yüksel", "yuksel", "düş", "dus", "borsa", "piyasa", "yatırım", "yatirim",
			"altcoin", "token", "nft", "defi", "cüzdan", "cuzdan", "exchange", "fon",
			"coin", "bitcoin", "ethereum", "kripto",
		}
		for _, x := range market {
			if strings.Contains(text, x) {
				return true
			}
		}
		return true
	}

	// For ordinary user-defined terms, preserve the familiar literal behaviour.
	return strings.Contains(text, term)
}

func (s *Store) matchesInterest(st *Story) bool {
	if len(s.interests) == 0 {
		return false
	}
	low := strings.ToLower(st.Title)
	for _, w := range s.interests {
		if strings.Contains(low, w) {
			return true
		}
	}
	return false
}

func newStore(cfg *Config) *Store {
	s := &Store{
		cfg:          cfg,
		byCat:        map[string][]Item{},
		storiesByCat: map[string][]*Story{},
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		tr:           newTranslator(),
		notified:     map[string]bool{},
		repo:         newDiskRepository(),
		sourceHealth: map[string]SourceHealth{},
		cache:        map[string]FeedCacheEntry{},
		firstAlert:   true,
	}
	s.tr.attachDisk(filepath.Join(filepath.Dir(configPath()), "translations.json"))
	s.cache, s.sourceHealth = s.repo.snapshot()
	s.hydrateCache()
	return s
}

func (s *Store) hydrateCache() {
	s.mu.Lock()
	agg := map[string][]Item{}
	orderSet := map[string]bool{}
	// A feed can sit in several categories; restore it into each of them.
	catsOf := map[string][]string{}
	for _, cat := range s.cfg.Categories {
		for _, src := range cat.Sources {
			if u := canonicalURL(src.URL); u != "" && !containsString(catsOf[u], cat.Name) {
				catsOf[u] = append(catsOf[u], cat.Name)
			}
		}
	}
	var newest time.Time
	for url, entry := range s.cache {
		if len(catsOf[url]) == 0 || len(entry.Items) == 0 {
			continue
		}
		for _, cat := range catsOf[url] {
			for _, it := range entry.Items {
				it.Category = cat
				agg[cat] = append(agg[cat], it)
			}
			orderSet[cat] = true
		}
		if entry.FetchedAt.After(newest) {
			newest = entry.FetchedAt
		}
	}
	if !newest.IsZero() {
		s.lastFetch = newest
	}
	var order []string
	for _, cat := range s.cfg.Categories {
		if orderSet[cat.Name] && !containsString(order, cat.Name) {
			order = append(order, cat.Name)
		}
	}
	s.mu.Unlock()
	for name := range agg {
		agg[name] = dedupeItems(agg[name])
		sortItemsNewest(agg[name])
	}
	// Same clustering/ranking path as a live refresh (incl. cross-category
	// de-duplication), so a cold start looks exactly like a fresh fetch.
	s.rebuildStories(agg, order)
}

func dedupeItems(items []Item) []Item {
	seen := make(map[string]bool, len(items))
	out := make([]Item, 0, len(items))
	for _, it := range items {
		key := it.ID
		if key == "" {
			key = canonicalURL(it.Link) + "\x00" + strings.ToLower(strings.TrimSpace(it.Title))
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, it)
	}
	return out
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// setTranslation configures whether/what language titles are shown in.
func (s *Store) setTranslation(on bool, target string) {
	s.mu.Lock()
	s.trOn = on
	s.trTarget = target
	s.mu.Unlock()
	if on && target != "" {
		tr := s.tr
		clusterTitleFn.Store(func(title string) string {
			if v, ok := tr.cached(target, title); ok {
				return v
			}
			return ""
		})
	} else {
		clusterTitleFn.Store(func(string) string { return "" })
	}
}

// translateTitles fills the translation cache for all current titles into the
// target language (best-effort, concurrency-limited). Call from a background
// goroutine, then pushData so the UI re-reads with translations applied.
func (s *Store) translateTitles(ctx context.Context) { s.translateTitlesN(ctx, 0) }

// translateTitlesN translates up to max titles (0 = all), most important
// stories first, so a time- or data-limited pass (Android background work)
// spends its budget on what the user will actually see.
func (s *Store) translateTitlesN(ctx context.Context, max int) {
	s.mu.RLock()
	on, target := s.trOn, s.trTarget
	titles := []string{}
	seen := map[string]bool{}
	add := func(t string) {
		if t != "" && !seen[t] {
			seen[t] = true
			titles = append(titles, t)
		}
	}
	for _, st := range s.important {
		for _, it := range st.items {
			add(it.Title)
		}
	}
	for _, name := range s.order {
		for _, it := range s.byCat[name] {
			add(it.Title)
		}
	}
	s.mu.RUnlock()
	if !on || target == "" {
		return
	}

	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	n := 0
	for _, title := range titles {
		if ctx.Err() != nil || (max > 0 && n >= max) {
			break
		}
		if _, ok := s.tr.cached(target, title); ok {
			continue
		}
		n++
		wg.Add(1)
		sem <- struct{}{}
		go func(tt string) {
			defer wg.Done()
			defer func() { <-sem }()
			_ = s.tr.translate(ctx, tt, target)
		}(title)
	}
	wg.Wait()
	_ = s.tr.saveDisk()
}

// --- output shapes for the UI --------------------------------------------

// outItem is a Story as the UI consumes it (a merged, ranked headline).
type outItem struct {
	Title              string        `json:"title"`
	Link               string        `json:"link"`
	Source             string        `json:"source"`      // primary source (Sources[0])
	Sources            []string      `json:"sources"`     // all corroborating sources
	SourceCount        int           `json:"sourceCount"` // len(Sources)
	Count              int           `json:"count"`       // number of merged headlines
	Category           string        `json:"category,omitempty"`
	TS                 int64         `json:"ts"` // newest, unix seconds, 0 if unknown
	Breaking           bool          `json:"breaking,omitempty"`
	Score              int           `json:"score"`             // 0–100 importance
	Kind               string        `json:"kind"`              // acil/teknoloji/bilim/…
	Summary            string        `json:"summary,omitempty"` // feed summary (reader fallback)
	EventID            string        `json:"eventId,omitempty"`
	IndependentSources int           `json:"independentSources"`
	PrimarySources     int           `json:"primarySources"`
	Confidence         int           `json:"confidence"`
	Velocity           float64       `json:"velocity"`
	Lifecycle          string        `json:"lifecycle"`
	Reason             string        `json:"reason,omitempty"`
	Evidence           EventEvidence `json:"evidence,omitempty"`
	Followed           bool          `json:"followed"`
	Image              string        `json:"image,omitempty"`
	Syndicated         int           `json:"syndicated,omitempty"`
	Read               bool          `json:"read,omitempty"`
	Sens               int           `json:"sens,omitempty"`     // RC97: başlığın duygu yükü (0–100)
	CalmFrom           string        `json:"calmFrom,omitempty"` // RC97: sakin başlık hangi kaynaktan alındı
	OrigTitle          string        `json:"origTitle,omitempty"`
	FactCheck          int           `json:"fc,omitempty"`  // RC97: eşleşen doğrulama yazısı sayısı
	Upd                int           `json:"upd,omitempty"` // RC97: kullanıcı son baktığından beri gelen başlık
}

type outCat struct {
	Name   string    `json:"name"`
	Total  int       `json:"total"`
	Unread int       `json:"unread"`
	Items  []outItem `json:"items"`
}

type overview struct {
	CountryCode string    `json:"countryCode"`
	CountryName string    `json:"countryName"`
	Categories  []outCat  `json:"categories"`
	Breaking    []outItem `json:"breaking"`  // acil/son dakika stories
	Important   []outItem `json:"important"` // "ŞU ANDA ÖNEMLİ" — top ranked
	ForYou      []outItem `json:"forYou"`    // "SENİN İÇİN" — matches interests
	UpdatedTS   int64     `json:"updatedTs"`
	Loading     bool      `json:"loading"`
	Pulse       int       `json:"pulse"`
	PulseBand   string    `json:"pulseBand"`
	// "FARKLI BİR PENCERE": one strong story from the region/topic the user
	// has read least lately — a small counterweight to the interest filter.
	Discover       *outItem    `json:"discover,omitempty"`
	DiscoverReason string      `json:"discoverReason,omitempty"`
	TotalUnread    int         `json:"totalUnread"`
	Crisis         *crisisInfo `json:"crisis,omitempty"` // RC97: kriz modu kartı
	Today          []agendaOut `json:"today,omitempty"`  // RC98: bugünün bayram / anma / dünya günleri
}

// Refresh fetches every source concurrently and rebuilds the cache. Cached feeds
// are used immediately when a source returns 304 or is temporarily unavailable.
func (s *Store) Refresh(ctx context.Context) {
	s.mu.Lock()
	if s.fetching {
		s.mu.Unlock()
		return
	}
	s.fetching = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.fetching = false
		s.lastFetch = time.Now()
		s.mu.Unlock()
		_ = s.repo.persist()
	}()

	cats, _ := s.cfg.snapshot()

	// One fetch per feed URL even when the same feed is listed in several
	// categories; the items are then fanned out to every category.
	type job struct {
		src  Source
		cats []string
	}
	var jobs []*job
	byURL := map[string]*job{}
	for _, cat := range cats {
		for _, src := range cat.Sources {
			key := canonicalURL(src.URL)
			if strings.TrimSpace(src.URL) == "" || key == "" {
				continue
			}
			if j, ok := byURL[key]; ok {
				if !containsString(j.cats, cat.Name) {
					j.cats = append(j.cats, cat.Name)
				}
				continue
			}
			j := &job{src: src, cats: []string{cat.Name}}
			byURL[key] = j
			jobs = append(jobs, j)
		}
	}

	type result struct {
		cats  []string
		items []Item
	}
	results := make(chan result, len(jobs)+1)
	var wg sync.WaitGroup
	// Politeness: at most fetchConcurrency requests in flight overall and
	// perHostConcurrency per host (BBC alone hosts a dozen default feeds).
	global := make(chan struct{}, fetchConcurrency)
	hostSem := map[string]chan struct{}{}
	for _, j := range jobs {
		h := feedHost(j.src.URL)
		if hostSem[h] == nil {
			hostSem[h] = make(chan struct{}, perHostConcurrency)
		}
		wg.Add(1)
		go func(j *job, hs chan struct{}) {
			defer wg.Done()
			hs <- struct{}{}
			global <- struct{}{}
			items := s.fetchOne(ctx, j.cats[0], j.src)
			<-global
			<-hs
			results <- result{cats: j.cats, items: items}
		}(j, hostSem[h])
	}
	go func() { wg.Wait(); close(results) }()

	agg := map[string][]Item{}
	for r := range results {
		for i, cat := range r.cats {
			if i == 0 {
				agg[cat] = append(agg[cat], r.items...)
				continue
			}
			for _, it := range r.items {
				it.Category = cat
				agg[cat] = append(agg[cat], it)
			}
		}
	}
	for name := range agg {
		agg[name] = dedupeItems(agg[name])
	}

	order := make([]string, 0, len(cats))
	for _, cat := range cats {
		order = append(order, cat.Name)
		sortItemsNewest(agg[cat.Name])
	}
	s.rebuildStories(agg, order)
}

// rebuildStories clusters and ranks the per-category items and swaps the
// result in. It is used after a fetch and again after a translation pass, when
// translated titles let cross-language duplicates merge.
func (s *Store) rebuildStories(agg map[string][]Item, order []string) {
	stories := map[string][]*Story{}
	var important []*Story
	for _, name := range order {
		cl := clusterItems(agg[name])
		rankStories(cl)
		stories[name] = cl
		important = append(important, cl...)
	}
	rankStories(important)
	important = dedupeAcrossCategories(important)

	s.mu.Lock()
	s.byCat, s.storiesByCat, s.important, s.order = agg, stories, important, order
	s.mu.Unlock()
}

// dedupeAcrossCategories keeps, for stories that share member articles (the
// same feed listed in two categories), only the highest-ranked one. Input must
// already be ranked.
func dedupeAcrossCategories(ranked []*Story) []*Story {
	seen := map[string]bool{}
	out := ranked[:0:0]
	for _, st := range ranked {
		dup := false
		for _, it := range st.items {
			if seen[it.Link] {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		for _, it := range st.items {
			seen[it.Link] = true
		}
		out = append(out, st)
	}
	return out
}

// recluster re-runs clustering on the cached items (no network).
func (s *Store) recluster() {
	s.mu.RLock()
	agg := make(map[string][]Item, len(s.byCat))
	for k, v := range s.byCat {
		agg[k] = v
	}
	order := append([]string(nil), s.order...)
	s.mu.RUnlock()
	s.rebuildStories(agg, order)
}

const (
	fetchConcurrency   = 16
	perHostConcurrency = 2
)

func feedHost(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return raw
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

func (s *Store) fetchOne(ctx context.Context, catName string, src Source) []Item {
	started := time.Now()
	canonical := canonicalURL(src.URL)
	if canonical == "" {
		return nil
	}
	s.mu.RLock()
	cached, haveCache := s.cache[canonical]
	s.mu.RUnlock()

	h := SourceHealth{URL: canonical, Name: src.Name, Status: "error", LastAttempt: started}
	s.mu.RLock()
	old, ok := s.sourceHealth[canonical]
	s.mu.RUnlock()
	if ok {
		h = old
		h.URL = canonical
		h.Name = src.Name
		h.LastAttempt = started
	}
	recordHealth := func(status string, errText string, code int, count int) {
		h.Status, h.HTTPStatus, h.LatencyMS, h.ItemCount, h.LastError = status, code, time.Since(started).Milliseconds(), count, errText
		if status == "healthy" {
			h.LastSuccess = time.Now()
			h.LastChange = h.LastSuccess
			h.ConsecutiveFail = 0
		} else {
			h.ConsecutiveFail++
		}
		s.mu.Lock()
		s.sourceHealth[canonical] = h
		s.mu.Unlock()
		s.repo.putHealth(h)
	}

	// Use the user-configured URL for the network request. The canonical URL is
	// intentionally used as the cache/identity key, but normalizing a live
	// endpoint must never change how a local/test/proxied server is addressed.
	requestURL := strings.TrimSpace(src.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		recordHealth("error", err.Error(), 0, len(cached.Items))
		return cached.Items
	}
	req.Header.Set("User-Agent", "NexNews/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.8")
	if haveCache {
		if cached.ETag != "" {
			req.Header.Set("If-None-Match", cached.ETag)
		}
		if cached.LastModified != "" {
			req.Header.Set("If-Modified-Since", cached.LastModified)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		recordHealth("stale", err.Error(), 0, len(cached.Items))
		return cached.Items
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified && haveCache {
		recordHealth("healthy", "", resp.StatusCode, len(cached.Items))
		return cached.Items
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		recordHealth("stale", fmt.Sprintf("HTTP %d", resp.StatusCode), resp.StatusCode, len(cached.Items))
		return cached.Items
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		recordHealth("stale", err.Error(), resp.StatusCode, len(cached.Items))
		return cached.Items
	}
	items, feedTitle, err := parseFeed(data)
	if err != nil {
		h.ParseErrors++
		recordHealth("stale", err.Error(), resp.StatusCode, len(cached.Items))
		return cached.Items
	}
	name := src.Name
	if name == "" {
		name = feedTitle
	}
	if name == "" {
		name = canonical
	}
	for i := range items {
		items[i].Category = catName
		items[i].Source = name
		items[i].Breaking = classifyBreaking(items[i].Title, items[i].Published, items[i].HasDate)
	}
	if len(items) > 120 {
		sortItemsNewest(items)
		items = items[:120]
	}
	entry := FeedCacheEntry{URL: canonical, Category: catName, Name: name, ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified"), FetchedAt: time.Now(), Items: items}
	s.mu.Lock()
	s.cache[canonical] = entry
	s.mu.Unlock()
	s.repo.putFeed(entry)
	recordHealth("healthy", "", resp.StatusCode, len(items))
	return items
}

// sourceHealthJSON exposes source diagnostics without exposing transport errors
// to the main feed UI.
func (s *Store) sourceHealthJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SourceHealth, 0, len(s.sourceHealth))
	for _, h := range s.sourceHealth {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return json.Marshal(out)
}

type sourceHealthSummary struct {
	Updated       string         `json:"updated"`
	Configured    int            `json:"configured"`
	Checked       int            `json:"checked"`
	Healthy       int            `json:"healthy"`
	Stale         int            `json:"stale"`
	Errors        int            `json:"errors"`
	Unchecked     int            `json:"unchecked"`
	HealthyPct    int            `json:"healthyPct"`
	CoveragePct   int            `json:"coveragePct"`
	UniqueDomains int            `json:"uniqueDomains"`
	Failures      []SourceHealth `json:"failures"`
}

func (s *Store) sourceHealthSummaryJSON() ([]byte, error) {
	cats, _ := s.cfg.snapshot()
	configured := map[string]bool{}
	for _, c := range cats {
		for _, src := range c.Sources {
			if u := canonicalURL(src.URL); u != "" {
				configured[u] = true
			}
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sum sourceHealthSummary
	sum.Updated = time.Now().UTC().Format(time.RFC3339)
	sum.Configured = len(configured)
	domains := map[string]bool{}
	for u := range configured {
		if pu, err := url.Parse(u); err == nil && pu.Hostname() != "" {
			domains[strings.ToLower(pu.Hostname())] = true
		}
		h, ok := s.sourceHealth[u]
		if !ok {
			sum.Unchecked++
			continue
		}
		sum.Checked++
		switch h.Status {
		case "healthy":
			sum.Healthy++
		case "stale":
			sum.Stale++
		default:
			sum.Errors++
			sum.Failures = append(sum.Failures, h)
		}
	}
	sum.UniqueDomains = len(domains)
	if sum.Configured > 0 {
		sum.CoveragePct = int(float64(sum.Checked) * 100 / float64(sum.Configured))
	}
	if sum.Checked > 0 {
		sum.HealthyPct = int(float64(sum.Healthy) * 100 / float64(sum.Checked))
	}
	sort.Slice(sum.Failures, func(i, j int) bool {
		if sum.Failures[i].ConsecutiveFail != sum.Failures[j].ConsecutiveFail {
			return sum.Failures[i].ConsecutiveFail > sum.Failures[j].ConsecutiveFail
		}
		return sum.Failures[i].Name < sum.Failures[j].Name
	})
	if len(sum.Failures) > 6 {
		sum.Failures = sum.Failures[:6]
	}
	return json.Marshal(sum)
}

// --- cache readers (called from UI thread; must be fast) ------------------

func (s *Store) countryFilterEnabled() (string, string) {
	s.cfg.mu.RLock()
	code, scope := s.cfg.NewsCountry, s.cfg.CountryScope
	s.cfg.mu.RUnlock()
	if code != "" {
		code = strings.ToUpper(strings.TrimSpace(code))
	}
	if scope != "strict" {
		scope = "related"
	}
	return code, scope
}

func (s *Store) countryAllowed(st *Story) bool {
	code, scope := s.countryFilterEnabled()
	if code == "" {
		return true
	}
	r := countryRelevance(st, code)
	if scope == "strict" {
		return r >= 0.70
	}
	return r >= 0.42
}

func (s *Store) overviewJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	max := s.cfg.MaxPerCategory
	if max <= 0 {
		max = 5
	}

	code, _ := s.countryFilterEnabled()
	p, _ := countryProfile(code)
	ov := overview{
		CountryCode: code,
		CountryName: p.Name,
		UpdatedTS:   s.lastFetch.Unix(),
		Loading:     s.fetching,
	}
	if s.lastFetch.IsZero() {
		ov.UpdatedTS = 0
	}
	read := map[string]bool{}
	readByCat := map[string]int{}
	if favs != nil {
		read, readByCat = favs.readStats(14 * 24 * time.Hour)
	}
	out := func(st *Story) outItem {
		o := s.storyOut(st)
		o.Read = read[st.Link]
		return o
	}
	seenBreak := map[string]bool{}
	seenFor := map[string]bool{}
	for _, name := range s.order {
		cl := s.storiesByCat[name]
		shown := 0
		oc := outCat{Name: name}
		for _, st := range cl {
			if !s.countryAllowed(st) || s.isHidden(st) { // negatif filtre
				continue
			}
			oc.Total++
			if !read[st.Link] {
				oc.Unread++
			}
			if shown < max {
				oc.Items = append(oc.Items, out(st))
				shown++
			}
			if st.Breaking && !seenBreak[st.Link] {
				seenBreak[st.Link] = true
				ov.Breaking = append(ov.Breaking, out(st))
			}
		}
		ov.TotalUnread += oc.Unread
		ov.Categories = append(ov.Categories, oc)
	}
	sort.SliceStable(ov.Breaking, func(i, j int) bool { return ov.Breaking[i].TS > ov.Breaking[j].TS })
	if len(ov.Breaking) > 15 {
		ov.Breaking = ov.Breaking[:15]
	}

	// Each story appears in at most one of the three top sections, in this
	// priority: SON DAKİKA → SENİN İÇİN → ŞU ANDA ÖNEMLİ. A feed listed in two
	// categories yields the same event twice, so overlap is detected on the
	// member articles, not only on the representative link.
	shownTop := map[string]bool{}
	shownArticles := map[string]bool{}
	overlaps := func(st *Story) bool {
		for _, it := range st.items {
			if shownArticles[it.Link] {
				return true
			}
		}
		return false
	}
	mark := func(st *Story) {
		shownTop[st.Link] = true
		for _, it := range st.items {
			shownArticles[it.Link] = true
		}
	}
	byLink := map[string]*Story{}
	for _, st := range s.important {
		byLink[st.Link] = st
	}
	for _, b := range ov.Breaking {
		shownTop[b.Link] = true
		if st := byLink[b.Link]; st != nil {
			mark(st)
		}
	}
	for _, st := range s.important {
		if !s.countryAllowed(st) || s.isHidden(st) || shownTop[st.Link] || seenFor[st.Link] || !s.matchesInterest(st) || overlaps(st) {
			continue
		}
		seenFor[st.Link] = true
		mark(st)
		ov.ForYou = append(ov.ForYou, out(st))
	}
	sort.SliceStable(ov.ForYou, func(i, j int) bool { return ov.ForYou[i].Score > ov.ForYou[j].Score })
	if len(ov.ForYou) > 6 {
		ov.ForYou = ov.ForYou[:6]
	}

	// "ŞU ANDA ÖNEMLİ": top-scored stories not already shown above, above a
	// relevance floor.
	for _, st := range s.important {
		if !s.countryAllowed(st) || st.Breaking || shownTop[st.Link] || st.Score < 40 || s.isHidden(st) || overlaps(st) {
			continue
		}
		mark(st)
		ov.Important = append(ov.Important, out(st))
		if len(ov.Important) >= 8 {
			break
		}
	}

	// Diversity window: the least-read category (over the last two weeks)
	// that still has a reasonably strong, unshown story.
	type cand struct {
		cat   string
		reads int
		st    *Story
	}
	var best *cand
	for _, name := range s.order {
		if strings.HasPrefix(name, "⚡") {
			continue
		}
		for _, st := range s.storiesByCat[name] {
			if shownTop[st.Link] || overlaps(st) || read[st.Link] || st.Score < 25 || !s.countryAllowed(st) || s.isHidden(st) {
				continue
			}
			c := &cand{cat: name, reads: readByCat[name], st: st}
			if best == nil || c.reads < best.reads || (c.reads == best.reads && st.Score > best.st.Score) {
				best = c
			}
			break // top story per category is enough
		}
	}
	if best != nil && len(readByCat) > 0 {
		o := out(best.st)
		ov.Discover = &o
		if best.reads == 0 {
			ov.DiscoverReason = "Son iki haftada " + best.cat + " başlığından hiç haber okumadın."
		} else {
			ov.DiscoverReason = "Son iki haftada en az okuduğun başlık: " + best.cat + "."
		}
	}
	ov.Pulse = pulseScore(s.important)
	ov.PulseBand = pulseBand(ov.Pulse)
	ov.Crisis = s.crisisFor()
	ov.Today = todayDays()
	return json.Marshal(ov)
}

func (s *Store) categoryJSON(name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cl := s.storiesByCat[name]
	oc := outCat{Name: name}
	read := map[string]bool{}
	if favs != nil {
		read, _ = favs.readStats(0)
	}
	for _, st := range cl {
		if !s.countryAllowed(st) || s.isHidden(st) {
			continue
		}
		oc.Total++
		o := s.storyOut(st)
		o.Read = read[st.Link]
		if !o.Read {
			oc.Unread++
		}
		oc.Items = append(oc.Items, o)
	}
	return json.Marshal(oc)
}

// storyOut builds the UI item from a story, substituting a cached translation of
// the representative title when translation is enabled.
func (s *Store) storyOut(st *Story) outItem {
	var ts int64
	if st.HasDate {
		ts = st.Published.Unix()
	}
	title := st.Title
	if s.trOn && s.trTarget != "" {
		if tr, ok := s.tr.cached(s.trTarget, st.Title); ok && tr != "" {
			title = tr
		}
	}
	src := ""
	if len(st.Sources) > 0 {
		src = st.Sources[0]
	}
	return outItem{
		Title:              title,
		Link:               st.Link,
		Source:             src,
		Sources:            st.Sources,
		SourceCount:        len(st.Sources),
		Count:              st.Count,
		Category:           st.Category,
		TS:                 ts,
		Breaking:           st.Breaking,
		Score:              st.Score,
		Kind:               st.Kind,
		Summary:            st.Summary,
		EventID:            st.EventID,
		IndependentSources: st.IndependentSources,
		PrimarySources:     st.PrimarySources,
		Confidence:         st.Confidence,
		Velocity:           st.Velocity,
		Lifecycle:          st.Lifecycle,
		Reason:             eventReason(st),
		Evidence:           evidenceForStory(st),
		Followed:           follows.isStory(st),
		Image:              st.Image,
		Syndicated:         func() int { n, _ := syndicatedCount(st); return n }(),
	}.withRC97(s, st)
}

// withRC97: duygu yükü, (ayar açıksa) sakin başlık ve doğrulama sayısı.
func (o outItem) withRC97(s *Store, st *Story) outItem {
	calm, src, sc := s.calmestTitle(st)
	o.Sens = sc
	if calm != "" && s.cfg != nil && s.cfg.CalmHeadlines {
		o.OrigTitle, o.Title, o.CalmFrom = o.Title, calm, src
		o.Sens = sensationScore(calm)
	}
	o.FactCheck = factCheckCount(s, st)
	if v := visits.lastSeen(visitKey(st)); v > 0 {
		for _, it := range st.items {
			if it.HasDate && it.Published.Unix() > v {
				o.Upd++
			}
		}
	}
	return o
}
