package core

// Bağlam kartı: "Bu konuya yeni misin?" (RC97).
//
// Haberler çoğu zaman okurun arka planı bildiğini varsayar. Olaydaki kişi,
// kurum ve yerler için Vikipedi'nin herkese açık özetinden 2–3 cümle ve
// NexNews'in kendi günlüğünden "bu konudaki önceki olaylar" gösterilir.
// Yalnızca kullanıcı istediğinde (karta dokununca) indirilir ve önbelleğe alınır.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

type wikiSummary struct {
	Query   string `json:"query"`
	Title   string `json:"title"`
	Desc    string `json:"desc,omitempty"`
	Extract string `json:"extract"`
	URL     string `json:"url"`
	Thumb   string `json:"thumb,omitempty"`
	Miss    bool   `json:"miss,omitempty"` // önbellek: bulunamadı
	At      int64  `json:"at"`
}

type relatedEvent struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Category string `json:"category"`
	First    int64  `json:"first"`
	Sources  int    `json:"sources"`
}

type contextBoard struct {
	Lang     string         `json:"lang"`
	Entities []wikiSummary  `json:"entities"`
	Related  []relatedEvent `json:"related"`
}

var (
	wikiMu    sync.Mutex
	wikiCache map[string]*wikiSummary
)

const (
	wikiTTL     = 7 * 24 * time.Hour
	wikiMissTTL = 2 * 24 * time.Hour
	wikiCap     = 400
)

func wikiPath() string { return filepath.Join(appDataDir(), "wikicache.json") }

func wikiLoadLocked() {
	if wikiCache != nil {
		return
	}
	wikiCache = map[string]*wikiSummary{}
	if b, err := os.ReadFile(wikiPath()); err == nil {
		_ = json.Unmarshal(b, &wikiCache)
		if wikiCache == nil {
			wikiCache = map[string]*wikiSummary{}
		}
	}
}

func wikiSaveLocked() {
	if len(wikiCache) > wikiCap { // en eskileri at
		type kv struct {
			k  string
			at int64
		}
		all := make([]kv, 0, len(wikiCache))
		for k, v := range wikiCache {
			all = append(all, kv{k, v.At})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].at < all[j].at })
		for _, x := range all[:len(all)-wikiCap*9/10] {
			delete(wikiCache, x.k)
		}
	}
	if b, err := json.Marshal(wikiCache); err == nil {
		_ = atomicWriteFile(wikiPath(), b)
	}
}

var contextStop = map[string]bool{
	// Başlıklarda büyük harfle başlayıp varlık adı olmayan yaygın sözcükler.
	"son": true, "dakika": true, "flaş": true, "canlı": true, "video": true, "galeri": true, "açıklama": true, "bakan": true,
	"cumhurbaşkanı": true, "başkan": true, "breaking": true, "live": true, "update": true, "news": true, "the": true, "watch": true,
	"exclusive": true, "analysis": true, "opinion": true, "eilmeldung": true, "direct": true, "urgent": true, "última": true, "hora": true,
}

// properPhrases: başlıklardaki özel ad öbekleri (büyük harfle başlayan ardışık
// sözcükler), Türkçe ekleri (Erdoğan'ın → Erdoğan) atılmış hâlde.
func properPhrases(title string) []string {
	words := strings.Fields(title)
	var out []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			p := strings.Join(cur, " ")
			if len([]rune(p)) >= 3 {
				out = append(out, p)
			}
			cur = nil
		}
	}
	for i, w := range words {
		suffixed := false
		if k := strings.IndexAny(w, "'’"); k > 0 { // Türkçe ek: öbek burada biter
			w, suffixed = w[:k], true
		}
		w = strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
		r := []rune(w)
		if len(r) < 2 || !unicode.IsUpper(r[0]) || contextStop[strings.ToLower(w)] {
			flush()
			continue
		}
		// Cümle başındaki tek sözcük çoğu zaman yalnızca büyük harfle başlar;
		// ardından başka bir özel ad gelmiyorsa sayma.
		if i == 0 && !suffixed && (len(words) < 2 || !startsUpper(words[1])) {
			continue
		}
		cur = append(cur, w)
		if suffixed || len(cur) == 4 {
			flush()
		}
	}
	flush()
	return out
}

func startsUpper(w string) bool {
	for _, r := range w {
		return unicode.IsUpper(r)
	}
	return false
}

// contextCandidates: olayın başlıklarında en sık geçen özel adlar (en çok 4).
func contextCandidates(titles []string) []string {
	cnt := map[string]int{}
	order := []string{}
	for _, t := range titles {
		seen := map[string]bool{}
		for _, ph := range properPhrases(t) {
			ws := strings.Fields(ph)
			// Öbeğin tüm ardışık alt dizileri de sayılır ("Dışişleri Bakanı Hakan Fidan" → "Hakan Fidan").
			for i := 0; i < len(ws); i++ {
				for j := i + 1; j <= len(ws); j++ {
					p := strings.Join(ws[i:j], " ")
					if seen[p] || len([]rune(p)) < 3 {
						continue
					}
					seen[p] = true
					if cnt[p] == 0 {
						order = append(order, p)
					}
					cnt[p]++
				}
			}
		}
	}
	sort.SliceStable(order, func(i, j int) bool {
		if cnt[order[i]] != cnt[order[j]] {
			return cnt[order[i]] > cnt[order[j]]
		}
		return len(order[i]) > len(order[j])
	})
	// Bir öbek, daha uzun ve en az o kadar sık bir öbeğin parçasıysa atla.
	var out []string
	for _, p := range order {
		skip := false
		for _, q := range out {
			if strings.Contains(q, p) || strings.Contains(p, q) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if len(titles) >= 3 && cnt[p] < 2 {
			continue
		}
		out = append(out, p)
		if len(out) == 4 {
			break
		}
	}
	return out
}

func wikiFetch(ctx context.Context, lang, q string) *wikiSummary {
	key := lang + ":" + strings.ToLower(q)
	wikiMu.Lock()
	wikiLoadLocked()
	if c := wikiCache[key]; c != nil {
		ttl := wikiTTL
		if c.Miss {
			ttl = wikiMissTTL
		}
		if time.Since(time.Unix(c.At, 0)) < ttl {
			wikiMu.Unlock()
			return c
		}
	}
	wikiMu.Unlock()

	res := &wikiSummary{Query: q, At: time.Now().Unix(), Miss: true}
	u := "https://" + lang + ".wikipedia.org/api/rest_v1/page/summary/" + url.PathEscape(strings.ReplaceAll(q, " ", "_")) + "?redirect=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err == nil {
		req.Header.Set("User-Agent", "NexNews/1.0 (https://github.com/Eirahoutmoss/nexnews)")
		req.Header.Set("Accept", "application/json")
		if resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req); err == nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			if resp.StatusCode == 200 {
				var w struct {
					Type        string `json:"type"`
					Title       string `json:"title"`
					Description string `json:"description"`
					Extract     string `json:"extract"`
					Thumbnail   struct {
						Source string `json:"source"`
					} `json:"thumbnail"`
					ContentURLs struct {
						Desktop struct {
							Page string `json:"page"`
						} `json:"desktop"`
						Mobile struct {
							Page string `json:"page"`
						} `json:"mobile"`
					} `json:"content_urls"`
				}
				if json.Unmarshal(body, &w) == nil && w.Type == "standard" && strings.TrimSpace(w.Extract) != "" {
					res.Miss = false
					res.Title, res.Desc, res.Thumb = w.Title, w.Description, w.Thumbnail.Source
					res.Extract = trimSentences(w.Extract, 3, 480)
					res.URL = w.ContentURLs.Desktop.Page
					if res.URL == "" {
						res.URL = w.ContentURLs.Mobile.Page
					}
				}
			} else if resp.StatusCode >= 500 || resp.StatusCode == 429 {
				return nil // geçici hata: önbelleğe alma
			}
		} else {
			return nil
		}
	}
	wikiMu.Lock()
	wikiCache[key] = res
	wikiSaveLocked()
	wikiMu.Unlock()
	return res
}

// trimSentences: en çok n cümle ve max karakter.
func trimSentences(s string, n, max int) string {
	s = strings.TrimSpace(s)
	out, count := []rune{}, 0
	rs := []rune(s)
	for i, r := range rs {
		out = append(out, r)
		if (r == '.' || r == '!' || r == '?' || r == '。') && (i+1 == len(rs) || rs[i+1] == ' ' || rs[i+1] == '\n') {
			count++
			if count >= n {
				break
			}
		}
		if len(out) >= max {
			return strings.TrimSpace(string(out)) + "…"
		}
	}
	return strings.TrimSpace(string(out))
}

func wikiLang() string {
	l := uiLangCode()
	switch l {
	case "tr", "en", "ar", "de", "fr", "es", "ru":
		return l
	}
	return "en"
}

// contextJSON: olayın bağlam kartı. Ağ işi kilit dışında yapılır.
func (s *Store) contextJSON(ctx context.Context, link string) ([]byte, error) {
	b := contextBoard{Lang: wikiLang(), Entities: []wikiSummary{}, Related: []relatedEvent{}}
	s.mu.RLock()
	st := s.findStory(link)
	var titles []string
	var base map[string]bool
	key := ""
	if st != nil {
		for _, it := range st.items {
			titles = append(titles, s.trTitle(it.Title))
		}
		base = s.storyTokens(st)
		key = visitKey(st)
	}
	s.mu.RUnlock()
	if st == nil {
		return json.Marshal(b)
	}

	// Önceki olaylar: haftalık günlükte kelimeleri örtüşen başka olaylar.
	weekMu.Lock()
	w := weekLoad()
	type rel struct {
		e     weekEntry
		inter int
	}
	var rels []rel
	for k, e := range w.Entries {
		if k == key || e.Link == link {
			continue
		}
		_, inter := overlapCoef(base, clusterTokens(e.Title))
		if inter >= 2 {
			rels = append(rels, rel{*e, inter})
		}
	}
	weekMu.Unlock()
	sort.Slice(rels, func(i, j int) bool {
		if rels[i].inter != rels[j].inter {
			return rels[i].inter > rels[j].inter
		}
		return rels[i].e.First > rels[j].e.First
	})
	for _, r := range rels {
		if len(b.Related) >= 5 {
			break
		}
		b.Related = append(b.Related, relatedEvent{Title: r.e.Title, Link: r.e.Link, Category: r.e.Category, First: r.e.First, Sources: r.e.Sources})
	}

	// Vikipedi özetleri (paralel, en çok 4 öbek).
	cands := contextCandidates(titles)
	res := make([]*wikiSummary, len(cands))
	var wg sync.WaitGroup
	for i, q := range cands {
		wg.Add(1)
		go func(i int, q string) {
			defer wg.Done()
			res[i] = wikiFetch(ctx, b.Lang, q)
		}(i, q)
	}
	wg.Wait()
	seen := map[string]bool{}
	for _, r := range res {
		if r != nil && !r.Miss && !seen[r.Title] {
			seen[r.Title] = true
			b.Entities = append(b.Entities, *r)
		}
	}
	return json.Marshal(b)
}
