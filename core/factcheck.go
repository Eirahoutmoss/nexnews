package core

// Doğrulama rozeti (RC97).
//
// Bağımsız doğrulama kuruluşlarının (Teyit, Doğruluk Payı, Full Fact,
// PolitiFact, CORRECTIV…) herkese açık akışları birkaç saatte bir okunur;
// yazıları akıştaki olaylarla aynı kelime örtüşmesi yöntemiyle eşleştirilir.
// Eşleşen olayda "doğrulama yazısı var" rozeti ve bağlantısı görünür. NexNews
// kendisi doğru/yanlış hükmü vermez; yalnızca doğrulama kuruluşunun yazısını
// gösterir ve başlıktaki hükmü (yanlış / yanıltıcı / doğru) etiketler.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	factRefreshEvery = 3 * time.Hour
	factKeepFor      = 30 * 24 * time.Hour
	factPerSource    = 40
	factMinShared    = 3    // eşleşme için en az ortak anlamlı kelime
	factMinSim       = 0.45 // ve örtüşme katsayısı
	factMatchWindow  = 21 * 24 * time.Hour
)

type factItem struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	Link    string `json:"link"`
	TS      int64  `json:"ts"`
	Verdict string `json:"verdict,omitempty"` // yanlis / yaniltici / dogru / ""
	Lang    string `json:"lang,omitempty"`
}

type factMatch struct {
	factItem
	Sim float64 `json:"sim"`
}

type factFile struct {
	Updated int64             `json:"updated"`
	Feeds   map[string]string `json:"feeds"` // site → bulunan akış adresi
	Items   []factItem        `json:"items"`
}

var (
	factMu     sync.RWMutex
	factData   *factFile
	factTokens []map[string]bool // factData.Items ile aynı sırada (kök kelimeler)
	factBusy   bool
)

func factPath() string { return filepath.Join(appDataDir(), "factcheck.json") }

func factLoadLocked() {
	if factData != nil {
		return
	}
	factData = &factFile{Feeds: map[string]string{}}
	if b, err := os.ReadFile(factPath()); err == nil {
		_ = json.Unmarshal(b, factData)
		if factData.Feeds == nil {
			factData.Feeds = map[string]string{}
		}
	}
	factIndexLocked()
}

func factIndexLocked() {
	factTokens = make([]map[string]bool, len(factData.Items))
	for i, it := range factData.Items {
		factTokens[i] = clusterTokens(it.Title)
	}
}

// factVerdict: doğrulama başlığındaki hüküm sözcüğü. Önce "yanıltıcı/eksik",
// sonra "yanlış", en son "doğru" aranır ("doğru değil" yanlış sayılır).
func factVerdict(title string) string {
	t := " " + strings.ToLower(title) + " "
	has := func(xs ...string) bool {
		for _, x := range xs {
			if strings.Contains(t, x) {
				return true
			}
		}
		return false
	}
	switch {
	case has("yanıltıcı", "çarpıtıl", "bağlamından kopar", "eksik bilgi", "misleading", "missing context", "out of context",
		"half true", "half-true", "irreführend", "fehlender kontext", "trompeur", "engañoso", "манипуляц", "مضلل"):
		return "yaniltici"
	case has("yanlış", "asılsız", "gerçeği yansıtmıyor", "doğru değil", "uydurma", "sahte", " false", "fake", "pants on fire",
		"hoax", "fabricated", "falsch", "fälschung", " faux", "infox", " falso", " bulo", "фейк", "ложь", "неправда", "كاذب", "زائف"):
		return "yanlis"
	case has(" doğru:", "doğru mu? evet", " true:", "mostly true", "richtig:", " vrai", "verdadero", "правда:", "صحيح"):
		return "dogru"
	}
	return ""
}

// factSourcesFor: arayüz dilindeki, kullanıcının ülkesindeki ve İngilizce kaynaklar.
func factSourcesFor() []factSource {
	lang, country := uiLangCode(), ""
	if cfg != nil {
		country = localNewsCountry()
	}
	var out []factSource
	for _, s := range extras().FactCheck {
		if s.Lang == lang || s.Lang == "en" || (s.Country != "" && s.Country == country) {
			out = append(out, s)
		}
	}
	return out
}

// refreshFactChecks: en çok 3 saatte bir; sınırlı veri modunda ya da kapalıysa atlar.
func refreshFactChecks(ctx context.Context) {
	if cfg != nil {
		cfg.mu.RLock()
		off := cfg.FactCheckOff
		cfg.mu.RUnlock()
		if off {
			return
		}
	}
	if dataLimited() {
		return
	}
	factMu.Lock()
	factLoadLocked()
	if factBusy || time.Since(time.Unix(factData.Updated, 0)) < factRefreshEvery {
		factMu.Unlock()
		return
	}
	factBusy = true
	feeds := map[string]string{}
	for k, v := range factData.Feeds {
		feeds[k] = v
	}
	factMu.Unlock()
	defer func() { factMu.Lock(); factBusy = false; factMu.Unlock() }()

	client := &http.Client{Timeout: 15 * time.Second}
	type res struct {
		src   factSource
		feed  string
		items []factItem
	}
	srcs := factSourcesFor()
	ch := make(chan res, len(srcs))
	var wg sync.WaitGroup
	for _, src := range srcs {
		wg.Add(1)
		go func(src factSource) {
			defer wg.Done()
			feed := feeds[src.Site]
			if feed == "" {
				feed = src.Feed
			}
			items, ok := fetchFactFeed(ctx, client, feed, src)
			if !ok && src.Site != "" { // akış adresi değişmiş olabilir: siteden bul
				if f := discoverFactFeed(ctx, client, src.Site); f != "" && f != feed {
					feed = f
					items, ok = fetchFactFeed(ctx, client, feed, src)
				}
			}
			if !ok {
				feed = ""
			}
			ch <- res{src: src, feed: feed, items: items}
		}(src)
	}
	go func() { wg.Wait(); close(ch) }()

	fresh := map[string]factItem{}
	got := map[string]bool{}
	newFeeds := map[string]string{}
	for r := range ch {
		if r.feed != "" {
			newFeeds[r.src.Site] = r.feed
			got[r.src.Name] = true
		}
		for _, it := range r.items {
			fresh[it.Link] = it
		}
	}

	factMu.Lock()
	defer factMu.Unlock()
	cut := time.Now().Add(-factKeepFor).Unix()
	merged := map[string]factItem{}
	for _, it := range factData.Items {
		if it.TS >= cut {
			merged[it.Link] = it
		}
	}
	for k, it := range fresh {
		if it.TS >= cut || it.TS == 0 {
			merged[k] = it
		}
	}
	list := make([]factItem, 0, len(merged))
	for _, it := range merged {
		list = append(list, it)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].TS > list[j].TS })
	factData.Items = list
	for k, v := range newFeeds {
		factData.Feeds[k] = v
	}
	factData.Updated = time.Now().Unix()
	factIndexLocked()
	if b, err := json.Marshal(factData); err == nil {
		_ = atomicWriteFile(factPath(), b)
	}
}

func fetchFactFeed(ctx context.Context, client *http.Client, feed string, src factSource) ([]factItem, bool) {
	if feed == "" {
		return nil, false
	}
	data, err := httpGet(ctx, client, feed)
	if err != nil {
		return nil, false
	}
	items, _, err := parseFeed(data)
	if err != nil || len(items) == 0 {
		return nil, false
	}
	out := make([]factItem, 0, min(len(items), factPerSource))
	for _, it := range items {
		if len(out) >= factPerSource {
			break
		}
		t := strings.TrimSpace(it.Title)
		if t == "" || it.Link == "" {
			continue
		}
		fi := factItem{Source: src.Name, Title: t, Link: it.Link, Lang: src.Lang}
		if it.HasDate {
			fi.TS = it.Published.Unix()
		}
		fi.Verdict = factVerdict(t + " " + firstRunes(it.Summary, 160))
		out = append(out, fi)
	}
	return out, true
}

func discoverFactFeed(ctx context.Context, client *http.Client, site string) string {
	u, err := url.Parse(site)
	if err != nil || u.Host == "" {
		return ""
	}
	page, err := httpGet(ctx, client, site)
	if err != nil {
		return ""
	}
	for f := range feedLinksFromHTML(page, u) {
		if _, ok := validateFeed(ctx, client, f); ok {
			return f
		}
	}
	for _, p := range []string{"/feed", "/feed/", "/rss", "/rss.xml"} {
		f := strings.TrimRight(site, "/") + p
		if _, ok := validateFeed(ctx, client, f); ok {
			return f
		}
	}
	return ""
}

func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// factChecksFor: olayın kelimeleriyle örtüşen doğrulama yazıları (en iyi 3).
func factChecksFor(base map[string]bool, st *Story) []factMatch {
	if len(base) < factMinShared {
		return nil
	}
	factMu.Lock()
	factLoadLocked()
	factMu.Unlock()
	factMu.RLock()
	defer factMu.RUnlock()
	ref := time.Now()
	if st != nil && st.HasDate {
		ref = st.Published
	}
	var out []factMatch
	for i, it := range factData.Items {
		if it.TS > 0 {
			d := ref.Sub(time.Unix(it.TS, 0))
			if d < 0 {
				d = -d
			}
			if d > factMatchWindow {
				continue
			}
		}
		sim, inter := overlapCoef(base, factTokens[i])
		if inter >= factMinShared && sim >= factMinSim {
			out = append(out, factMatch{factItem: it, Sim: sim})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sim > out[j].Sim })
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

// factCheckCount: akış öğesi için hızlı sayım (yalnızca özgün başlık kelimeleri).
// Çağıran s.mu kilidini tutuyor olmalı.
func factCheckCount(s *Store, st *Story) int {
	factMu.RLock()
	empty := factData == nil || len(factData.Items) == 0
	factMu.RUnlock()
	if empty || st == nil {
		return 0
	}
	return len(factChecksFor(s.storyTokens(st), st))
}
