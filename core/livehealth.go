package core

// Canlı TV kaynak denetimi (RC83).
//
// Kanallar zamanla adres değiştirir, oturum anahtarlı adresler bayatlar ve bir
// kısmı yalnızca kendi ülkesinden izlenebilir. Oynatıcıyı tahmine bırakmak
// yerine her aday adres kısaca denenir: oynatma listesi gelmeli, ana listeyse
// ilk alt liste açılmalı ve ilk video parçasının ilk baytları inmelidir.
// Sonuç (çalışan adres ya da okunur hata nedeni) livehealth.json'a yazılır;
// oynatıcı bir dahaki sefere doğrudan çalışan adresle başlar.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type liveCandidate struct {
	U  string `json:"u"`
	R  string `json:"r,omitempty"`
	UA string `json:"ua,omitempty"`
}

type liveHealth struct {
	OK     bool   `json:"ok"`
	URL    string `json:"url,omitempty"` // çalışan aday
	Index  int    `json:"index"`
	Reason string `json:"reason,omitempty"`
	Code   int    `json:"code,omitempty"`
	MS     int64  `json:"ms,omitempty"`
	TS     int64  `json:"ts"`
}

type liveHealthStore struct {
	mu   sync.Mutex
	m    map[string]liveHealth
	path string
}

var liveHealthDB = &liveHealthStore{m: map[string]liveHealth{}}

func (h *liveHealthStore) load() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.path = filepath.Join(filepath.Dir(configPath()), "livehealth.json")
	if b, err := os.ReadFile(h.path); err == nil {
		_ = json.Unmarshal(b, &h.m)
	}
	if h.m == nil {
		h.m = map[string]liveHealth{}
	}
}

func (h *liveHealthStore) put(id string, r liveHealth) {
	h.mu.Lock()
	h.m[id] = r
	b, _ := json.Marshal(h.m)
	p := h.path
	h.mu.Unlock()
	if p != "" {
		_ = atomicWriteFile(p, b)
	}
}

func (h *liveHealthStore) json() []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	b, _ := json.Marshal(h.m)
	return b
}

// probeLive tries the candidates in order and returns the first that plays.
func probeLive(ctx context.Context, cands []liveCandidate) liveHealth {
	res := liveHealth{TS: time.Now().Unix(), Index: -1}
	worst := ""
	for i, c := range cands {
		if c.U == "" {
			continue
		}
		start := time.Now()
		code, reason := probeOne(ctx, c)
		if reason == "" {
			res.OK, res.URL, res.Index, res.MS, res.Reason, res.Code = true, c.U, i, time.Since(start).Milliseconds(), "", 0
			return res
		}
		// En açıklayıcı hatayı tut (bölge kısıtı > kaldırılmış > zaman aşımı).
		if worst == "" || code == 403 || code == 451 {
			worst, res.Code = reason, code
		}
		if ctx.Err() != nil {
			break
		}
	}
	if worst == "" {
		worst = "Yayın adresi yok"
	}
	res.Reason = worst
	return res
}

func probeOne(parent context.Context, c liveCandidate) (int, string) {
	ctx, cancel := context.WithTimeout(parent, 9*time.Second)
	defer cancel()
	u, err := url.Parse(c.U)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return 0, "Geçersiz adres"
	}
	if err := publicHost(ctx, u.Hostname()); err != nil {
		switch err {
		case errDNSFail:
			return 0, "Alan adı çözülemedi (" + u.Hostname() + ") — adres artık yok ya da DNS sunucunuz yanıt vermiyor"
		case errDNSSinkhole:
			return 0, "DNS sunucunuz bu yayın adresini engelliyor (" + u.Hostname() + " → 0.0.0.0). Reklam/filtre DNS'i (AdGuard, Pi-hole, NextDNS, kurum DNS'i) kullanıyorsanız bu alan adına izin verin"
		}
		return 0, "Adres yerel ağa çözülüyor (" + u.Hostname() + "), güvenlik için engellendi"
	}
	hdr := liveHeaders{ref: c.R, ua: c.UA}
	cur := c.U
	for depth := 0; depth < 3; depth++ {
		resp, err := liveFetch(ctx, cur, hdr, "")
		if err != nil {
			if ctx.Err() != nil {
				return 0, "Zaman aşımı (sunucu yanıt vermiyor)"
			}
			if errors.Is(err, errDNSSinkhole) {
				return 0, "DNS sunucunuz bu yayın adresini engelliyor (0.0.0.0)"
			}
			if errors.Is(err, errDNSFail) {
				return 0, "Alan adı çözülemedi (DNS)"
			}
			return 0, "Sunucuya ulaşılamadı: " + shortErr(err)
		}
		code := resp.StatusCode
		if code < 200 || code >= 300 {
			resp.Body.Close()
			return code, httpReason(code)
		}
		br := bufio.NewReader(io.LimitReader(resp.Body, 2*1024*1024))
		body, _ := io.ReadAll(br)
		resp.Body.Close()
		base := cur
		if resp.Request != nil && resp.Request.URL != nil {
			base = resp.Request.URL.String()
		}
		txt := strings.TrimLeft(string(body), "\ufeff \r\n\t")
		if !strings.HasPrefix(txt, "#EXTM3U") {
			return 0, "Yanıt bir yayın listesi değil"
		}
		next, isMaster := probePlaylistURI(txt)
		if next == "" {
			return 0, "Yayın listesi boş"
		}
		bu, _ := url.Parse(base)
		abs := resolveLiveURL(bu, next)
		if isMaster {
			cur = abs
			continue
		}
		// Medya listesi: ilk parçanın ilk baytları inmeli.
		sr, err := liveFetch(ctx, abs, hdr, "bytes=0-4095")
		if err != nil {
			return 0, "Video parçası indirilemedi"
		}
		defer sr.Body.Close()
		if sr.StatusCode != 200 && sr.StatusCode != 206 {
			return sr.StatusCode, "Video parçası: " + httpReason(sr.StatusCode)
		}
		n, _ := io.Copy(io.Discard, io.LimitReader(sr.Body, 4096))
		if n == 0 {
			return 0, "Video parçası boş"
		}
		return 0, ""
	}
	return 0, "Çok katmanlı liste çözülemedi"
}

// probePlaylistURI picks the URI to probe: the first variant of a master
// playlist, or the NEWEST segment of a media playlist. (Uzun DVR pencereli
// canlı listelerde — ör. TRT EBA — ilk parçalar sunucudan silinmiş olabiliyor;
// eskiden ilk parça denendiği için çalışan yayın "404" sanılıyordu.)
func probePlaylistURI(txt string) (string, bool) {
	first, master := firstPlaylistURI(txt)
	if master || first == "" {
		return first, master
	}
	last := first
	for _, l := range strings.Split(txt, "\n") {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#") {
			last = l
		}
	}
	return last, false
}

// firstPlaylistURI returns the first URI line; isMaster when the playlist lists
// variants (#EXT-X-STREAM-INF) rather than media segments.
func firstPlaylistURI(txt string) (string, bool) {
	master := strings.Contains(txt, "#EXT-X-STREAM-INF")
	sc := bufio.NewScanner(strings.NewReader(txt))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l != "" && !strings.HasPrefix(l, "#") {
			return l, master
		}
	}
	return "", master
}

func httpReason(code int) string {
	switch {
	case code == 401 || code == 403 || code == 451:
		return fmt.Sprintf("Erişim reddedildi (%d) — yayın büyük olasılıkla yalnızca kendi ülkesinden izlenebiliyor", code)
	case code == 404 || code == 410:
		return fmt.Sprintf("Yayın adresi kaldırılmış (%d)", code)
	case code == 429:
		return "Sunucu çok fazla istek nedeniyle geçici olarak reddetti (429)"
	case code >= 500:
		return fmt.Sprintf("Yayın sunucusu hata verdi (%d)", code)
	}
	return fmt.Sprintf("Beklenmeyen yanıt (%d)", code)
}

// ---- otomatik kaynak onarımı ---------------------------------------------------
//
// Bir kanalın bilinen bütün adresleri çalışmazsa, açık kaynaklı iptv-org
// dizininin ilgili ülke listesinden aynı kanal kimliğine (tvg-id) sahip güncel
// adresler aranır. Telif sorunu doğurmamak için yalnızca yayıncıya ya da
// lisanslı ücretsiz platformlara (Samsung TV Plus, Pluto, Xumo, Plex vb.) ait
// alan adları kabul edilir; IP adresli, kişisel ya da korsan "yeniden yayın"
// sunucuları elenir.

type m3uCacheEntry struct {
	at    time.Time
	items []m3uEntry
}

type m3uEntry struct {
	id, name, url, ref, ua string
	geo                    bool
}

var (
	m3uMu    sync.Mutex
	m3uCache = map[string]m3uCacheEntry{}
)

// Tanınmış korsan / yeniden yayın alan adları ve kalıpları.
var liveRepairBlock = []string{
	"ayakkabiparti", "mangeinde4", "siteyaptim", "cinerama.uz", "smartplaytv", "freeott",
	"pishow", "uzunmuhalefet", "onrender.com", "raw.githubusercontent.com", "thinkkast",
	"mcquack", "aynaott", "corestream", "bozztv", "megatv.fun", "dyndns", "ddns", "sason.top",
	"iptvperu", "githubusercontent", "playhdnew", "ottrast", "tvsen", "rezofoot", ".lat/",
	".xyz", ".top/", ".lol/",
}

func repairAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	if strings.Trim(h, "0123456789.") == "" || strings.Contains(h, ":") { // IP adresi
		return false
	}
	l := strings.ToLower(raw)
	for _, b := range liveRepairBlock {
		if strings.Contains(l, b) {
			return false
		}
	}
	return true
}

func fetchM3U(ctx context.Context, file string) ([]m3uEntry, error) {
	m3uMu.Lock()
	if e, ok := m3uCache[file]; ok && time.Since(e.at) < 12*time.Hour {
		m3uMu.Unlock()
		return e.items, nil
	}
	m3uMu.Unlock()
	for _, r := range file {
		if !(r >= 'a' && r <= 'z' || r == '_') {
			return nil, fmt.Errorf("geçersiz liste adı")
		}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://raw.githubusercontent.com/iptv-org/iptv/master/streams/"+file+".m3u", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("liste %d", resp.StatusCode)
	}
	var items []m3uEntry
	var cur *m3uEntry
	sc := bufio.NewScanner(io.LimitReader(resp.Body, 4*1024*1024))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(l, "#EXTINF"):
			e := m3uEntry{id: attr(l, "tvg-id"), ref: attr(l, "http-referrer"), ua: attr(l, "http-user-agent")}
			if i := strings.LastIndex(l, ","); i >= 0 {
				e.name = strings.TrimSpace(l[i+1:])
			}
			e.geo = strings.Contains(e.name, "[Geo-blocked]")
			cur = &e
		case strings.HasPrefix(l, "#EXTVLCOPT:") && cur != nil:
			if v, ok := strings.CutPrefix(l, "#EXTVLCOPT:http-referrer="); ok {
				cur.ref = v
			}
			if v, ok := strings.CutPrefix(l, "#EXTVLCOPT:http-user-agent="); ok {
				cur.ua = v
			}
		case cur != nil && (strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")):
			cur.url = l
			items = append(items, *cur)
			cur = nil
		}
	}
	m3uMu.Lock()
	m3uCache[file] = m3uCacheEntry{at: time.Now(), items: items}
	m3uMu.Unlock()
	return items, nil
}

func attr(line, key string) string {
	k := key + `="`
	i := strings.Index(line, k)
	if i < 0 {
		return ""
	}
	r := line[i+len(k):]
	if j := strings.IndexByte(r, '"'); j >= 0 {
		return r[:j]
	}
	return ""
}

// liveRepair returns fresh candidates for the channel id (tvg-id without the
// "@feed" part) from the listed iptv-org files; geo-blocked ones go last.
func liveRepair(ctx context.Context, tvg string, files []string, known map[string]bool) []liveCandidate {
	tvg = strings.ToLower(strings.TrimSpace(tvg))
	if tvg == "" {
		return nil
	}
	var free, geo []liveCandidate
	for _, f := range files {
		items, err := fetchM3U(ctx, strings.ToLower(strings.TrimSpace(f)))
		if err != nil {
			continue
		}
		for _, e := range items {
			id := strings.ToLower(e.id)
			if at := strings.IndexByte(id, '@'); at >= 0 {
				id = id[:at]
			}
			if id != tvg || known[e.url] || !repairAllowed(e.url) {
				continue
			}
			known[e.url] = true
			c := liveCandidate{U: e.url, R: e.ref, UA: e.ua}
			if e.geo {
				geo = append(geo, c)
			} else {
				free = append(free, c)
			}
		}
	}
	return append(free, geo...)
}

func shortErr(err error) string {
	m := err.Error()
	if i := strings.LastIndex(m, ": "); i >= 0 && i+2 < len(m) {
		m = m[i+2:]
	}
	if len(m) > 90 {
		m = m[:90]
	}
	return m
}

// probeLiveJSON runs a channel check (with optional iptv-org repair) and
// records the result. Shared by the desktop and Android bindings.
func probeLiveJSON(id, candsJSON, tvg, files string) ([]byte, error) {
	var cands []liveCandidate
	if err := json.Unmarshal([]byte(candsJSON), &cands); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	res := probeLive(ctx, cands)
	var extra []liveCandidate
	if !res.OK && tvg != "" && files != "" {
		known := map[string]bool{}
		for _, c := range cands {
			known[c.U] = true
		}
		extra = liveRepair(ctx, tvg, strings.Split(files, ","), known)
		if len(extra) > 0 {
			if r2 := probeLive(ctx, extra); r2.OK {
				r2.Index = -2 // onarımdan gelen adres
				res = r2
			}
		}
	}
	type out struct {
		liveHealth
		Repair *liveCandidate `json:"repair,omitempty"`
	}
	o := out{liveHealth: res}
	if res.OK && res.Index == -2 {
		for i := range extra {
			if extra[i].U == res.URL {
				o.Repair = &extra[i]
			}
		}
	}
	if id != "" {
		liveHealthDB.put(id, res)
	}
	return json.Marshal(o)
}

// ---- canlı TV kullanıcı durumu (favori, son kanal, gizlenen, elle eklenen) ----

type liveState struct {
	Favs   []string          `json:"favs"`
	Last   string            `json:"last"`
	Hidden []string          `json:"hidden"`
	Custom []json.RawMessage `json:"custom"`
	Check  string            `json:"check"`
}

func liveStateJSON() string {
	cfg.mu.RLock()
	st := liveState{Favs: cfg.LiveFavs, Last: cfg.LiveLast, Hidden: cfg.LiveHidden, Custom: cfg.LiveCustom, Check: cfg.LiveCheck}
	cfg.mu.RUnlock()
	if st.Check == "" {
		st.Check = "ask"
	}
	b, _ := json.Marshal(st)
	return string(b)
}

func setLiveState(raw string) error {
	var st liveState
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return err
	}
	clip := func(a []string, n int) []string {
		if len(a) > n {
			return a[:n]
		}
		return a
	}
	if len(st.Custom) > 300 {
		st.Custom = st.Custom[:300]
	}
	// Elle eklenen kanalların adresleri yalnızca http(s) olabilir.
	kept := st.Custom[:0]
	for _, c := range st.Custom {
		var ch struct {
			URLs []json.RawMessage `json:"urls"`
		}
		if json.Unmarshal(c, &ch) == nil && len(c) < 8192 {
			kept = append(kept, c)
		}
	}
	switch st.Check {
	case "ask", "auto", "never":
	default:
		st.Check = "ask"
	}
	cfg.mu.Lock()
	cfg.LiveFavs, cfg.LiveLast, cfg.LiveHidden, cfg.LiveCustom, cfg.LiveCheck = clip(st.Favs, 300), st.Last, clip(st.Hidden, 1000), kept, st.Check
	cfg.mu.Unlock()
	return cfg.save()
}
