package core

// Mobile entry points. The Android app links the core through gomobile
// (see the nexnews/mobile package); every call here exchanges plain strings
// (JSON) so the binding stays trivial. Calls may arrive on any thread; the
// store and config are already mutex-protected.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var mobileMu sync.Mutex

// mobileSourcesPerCategory keeps the first Android start light on mobile
// data: the bundled list has ~300 feeds; the phone starts with a few per
// category and the user can restore the rest in Settings.
const mobileSourcesPerCategory = 4

// MobileInit prepares the core with the app's private data directory and the
// device language. Safe to call more than once (later calls are no-ops).
func MobileInit(dataDir, lang string) error {
	mobileMu.Lock()
	defer mobileMu.Unlock()
	if store != nil {
		return nil
	}
	if strings.TrimSpace(dataDir) == "" {
		return errors.New("dataDir boş")
	}
	dataDirOverride = dataDir
	if l := strings.ToLower(strings.TrimSpace(lang)); l != "" {
		if i := strings.IndexAny(l, "-_"); i > 0 {
			l = l[:i]
		}
		sysLangOverride = l
	}
	_, statErr := os.Stat(configPath())
	firstRun := os.IsNotExist(statErr)
	cfg = loadConfig()
	if firstRun {
		trimForMobile(cfg)
		_ = cfg.save()
	}
	store = newStore(cfg)
	favs = newFavStore()
	follows = newFollowStore()
	speech = newSpeechProvider()
	applyTranslateCfg()
	applyPersonalCfg()
	applyDataMode(cfg.DataMode, cfg.DataLimitKBps)
	startUsageLedger(nil)
	liveHealthDB.load()
	return nil
}

func trimForMobile(c *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.Categories {
		if strings.HasPrefix(c.Categories[i].Name, "⚡") {
			continue
		}
		if len(c.Categories[i].Sources) > mobileSourcesPerCategory {
			c.Categories[i].Sources = c.Categories[i].Sources[:mobileSourcesPerCategory]
		}
	}
	c.MaxPerCategory = 5
	c.RefreshMinutes = 60
}

func mobileReady() bool { return store != nil && cfg != nil }

func jsonOr(b []byte, err error, fallback string) string {
	if err != nil || len(b) == 0 {
		return fallback
	}
	return string(b)
}

// MobileRefresh fetches every source (blocking; call off the main thread).
func MobileRefresh() {
	if !mobileReady() {
		return
	}
	ensureLocalCategory(context.Background())
	store.Refresh(context.Background())
}

// MobileSetCountry tells the core the device's region (Locale country, ISO-3166).
func MobileSetCountry(code string) { sysCountryOverride = strings.ToUpper(strings.TrimSpace(code)) }

// MobileTranslate translates up to max titles (most important first) and
// re-clusters so cross-language duplicates merge. Blocking, bounded by secs.
func MobileTranslate(max, secs int) {
	if !mobileReady() {
		return
	}
	if secs <= 0 {
		secs = 45
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(secs)*time.Second)
	defer cancel()
	store.translateTitlesN(ctx, max)
	store.recluster()
}

// MobileLastFetch is the unix time of the last completed refresh (0 = never).
func MobileLastFetch() int64 {
	if !mobileReady() {
		return 0
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	if store.lastFetch.IsZero() {
		return 0
	}
	return store.lastFetch.Unix()
}

func MobileIsFetching() bool {
	if !mobileReady() {
		return false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.fetching
}

func MobileOverview() string {
	if !mobileReady() {
		return "{}"
	}
	b, err := store.overviewJSON()
	return jsonOr(b, err, "{}")
}

func MobileCategory(name string) string {
	if !mobileReady() {
		return "{}"
	}
	b, err := store.categoryJSON(name)
	return jsonOr(b, err, "{}")
}

func MobileStory(link string) string {
	if !mobileReady() {
		return `{"found":false}`
	}
	b, err := store.storyJSON(link)
	return jsonOr(b, err, `{"found":false}`)
}

func MobileSearch(q string) string {
	if !mobileReady() {
		return "{}"
	}
	b, err := store.searchJSON(q)
	return jsonOr(b, err, "{}")
}

func MobileBriefing() string {
	if !mobileReady() {
		return "{}"
	}
	b, err := store.briefingJSON()
	return jsonOr(b, err, "{}")
}

func MobileSourceHealth() string {
	if !mobileReady() {
		return "[]"
	}
	b, err := store.sourceHealthJSON()
	return jsonOr(b, err, "[]")
}

// MobileReadArticle fetches and cleans an article for the in-app reader.
func MobileReadArticle(link string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return string(fetchArticleJSON(ctx, link))
}

// MobileDiscover finds RSS/Atom feeds for a site address.
func MobileDiscover(site string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	return string(discoverJSON(ctx, site))
}

func MobileConfig() string {
	if !mobileReady() {
		return "{}"
	}
	b, err := cfg.toJSON()
	return jsonOr(b, err, "{}")
}

// MobileSaveConfig stores a full config JSON (as returned by MobileConfig,
// edited) and applies translation / personal filters immediately.
func MobileSaveConfig(js string) error {
	if !mobileReady() {
		return errors.New("çekirdek hazır değil")
	}
	if err := cfg.replaceFrom([]byte(js)); err != nil {
		return err
	}
	applyTranslateCfg()
	applyPersonalCfg()
	resetWeatherCache()
	return nil
}

// MobileRestoreAllSources adds every bundled source the config is missing.
func MobileRestoreAllSources() int {
	if !mobileReady() {
		return 0
	}
	cfg.mu.Lock()
	before := 0
	for _, c := range cfg.Categories {
		before += len(c.Sources)
	}
	cfg.Categories = mergeBundledSources(cfg.Categories, defaultConfig().Categories)
	after := 0
	for _, c := range cfg.Categories {
		after += len(c.Sources)
	}
	cfg.mu.Unlock()
	_ = cfg.save()
	return after - before
}

// ---- favourites, history, follows -----------------------------------------

func parseSaved(js string) (savedItem, bool) {
	var it savedItem
	if json.Unmarshal([]byte(js), &it) != nil || it.Link == "" {
		return it, false
	}
	return it, true
}

func MobileToggleFav(itemJSON string) bool {
	it, ok := parseSaved(itemJSON)
	if !ok || favs == nil {
		return false
	}
	return favs.toggleFav(it)
}

func MobileFavLinks() string {
	if favs == nil {
		return "[]"
	}
	m := favs.favLinks()
	arr := make([]string, 0, len(m))
	for k := range m {
		arr = append(arr, k)
	}
	b, _ := json.Marshal(arr)
	return string(b)
}

func MobileFavorites() string {
	if favs == nil {
		return "[]"
	}
	return string(favs.listJSON("favs"))
}

func MobileHistory() string {
	if favs == nil {
		return "[]"
	}
	return string(favs.listJSON("history"))
}

func MobileAddHistory(itemJSON string) {
	if it, ok := parseSaved(itemJSON); ok && favs != nil {
		favs.addHistory(it)
	}
}

func MobileClearHistory() {
	if favs != nil {
		favs.clearHistory()
	}
}

func MobileToggleFollowEvent(id string) bool {
	if follows == nil || id == "" {
		return false
	}
	return follows.toggleEvent(id)
}

// ---- notifications ----------------------------------------------------------

type mobileAlert struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Source   string `json:"source"`
	Category string `json:"category"`
	Breaking bool   `json:"breaking"`
	Score    int    `json:"score"`
	Sources  int    `json:"sources"`
	Followed bool   `json:"followed,omitempty"`
}

// alertCandidates lists stories that deserve a notification (breaking, or at
// least minScore), without touching any "already notified" state.
func (s *Store) alertCandidates(minScore int) []*Story {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Story
	for _, st := range s.important {
		if st.Link == "" || s.isHidden(st) || !s.countryAllowed(st) {
			continue
		}
		if st.Breaking || st.Score >= minScore {
			out = append(out, st)
		}
	}
	return out
}

// MobileNewAlerts returns (and remembers) stories worth a notification since
// the last call. Android runs refreshes in short-lived background jobs, so the
// "already notified" set is persisted; the very first call only seeds it.
// Quiet hours and the notification switch are honoured here.
func MobileNewAlerts() string {
	if !mobileReady() {
		return "[]"
	}
	cfg.mu.RLock()
	on, min := cfg.Notify, cfg.NotifyMinScore
	cfg.mu.RUnlock()
	if min <= 0 {
		min = 75
	}
	path := filepath.Join(appDataDir(), "notified.json")
	seen := map[string]int64{}
	b, err := os.ReadFile(path)
	firstEver := err != nil
	if err == nil {
		_ = json.Unmarshal(b, &seen)
	}
	now := time.Now()
	quiet := cfg.inQuietHours(now)
	out := []mobileAlert{}
	for _, st := range store.alertCandidates(min) {
		if _, ok := seen[st.Link]; ok {
			continue
		}
		seen[st.Link] = now.Unix()
		if firstEver || !on || quiet || len(out) >= 3 {
			continue
		}
		title := store.trTitleLocked(st.Title)
		src := ""
		if len(st.Sources) > 0 {
			src = st.Sources[0]
		}
		out = append(out, mobileAlert{Title: title, Link: st.Link, Source: src, Category: st.Category, Breaking: st.Breaking, Score: st.Score, Sources: st.IndependentSources})
	}
	cut := now.Add(-7 * 24 * time.Hour).Unix()
	for k, v := range seen {
		if v < cut {
			delete(seen, k)
		}
	}
	if nb, err := json.Marshal(seen); err == nil {
		_ = atomicWriteFile(path, nb)
	}
	// Takip edilen olaylardaki gelişmeler (sessiz saatlere uyar).
	if !firstEver && !quiet {
		for _, u := range store.syncFollows() {
			if len(out) >= 5 {
				break
			}
			out = append(out, mobileAlert{Title: fmt.Sprintf("Takip ettiğin olayda +%d yeni haber · %s", u.Added, u.Title), Link: u.Link, Source: "Takip", Score: 100, Followed: true})
		}
	}
	usage.flush()
	ob, _ := json.Marshal(out)
	return string(ob)
}

// trTitleLocked translates a title from the cache, taking the read lock.
func (s *Store) trTitleLocked(title string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.trTitle(title)
}

func MobileInQuietHours() bool { return mobileReady() && cfg.inQuietHours(time.Now()) }

// ---- boards that need the network --------------------------------------------

func MobileRadio() string {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	b, err := radioJSON(ctx)
	return jsonOr(b, err, `{"stations":[],"error":"Radyo dizinine ulaşılamadı."}`)
}

func MobileLiveTV() string { return string(liveTVJSON()) }

// MobileLiveHeaders returns the HTTP headers a live stream URL expects
// (Referer/Origin of the broadcaster's site, browser user agent).
func MobileLiveHeaders(streamURL string) string {
	ref := liveReferer(streamURL)
	if c, ok := liveHostHeader(streamURL); ok && c.R != "" {
		ref = c.R
	}
	origin := ""
	if u, err := url.Parse(ref); err == nil && u.Host != "" {
		origin = u.Scheme + "://" + u.Host
	}
	b, _ := json.Marshal(map[string]string{
		"referer":   ref,
		"origin":    origin,
		"userAgent": "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140 Mobile Safari/537.36",
	})
	return string(b)
}

func MobileWeather(place string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return string(weatherJSON(ctx, place))
}

func MobileFinance() string {
	if !mobileReady() {
		return "{}"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	return string(financeJSON(ctx, cfg))
}

// MobileStandings: Süper Lig ve Basketbol Süper Ligi puan durumları (30 dk önbellek).
func MobileStandings() string {
	if !mobileReady() {
		return "{}"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	return string(standingsJSON(ctx, cfg))
}

func MobileSports() string {
	if !mobileReady() {
		return "{}"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return string(sportsJSON(ctx, cfg))
}

// MobileNetStats reports NexNews' own traffic (rate, last 5 min, session).
func MobileNetStats() string {
	b, err := netStatsJSON()
	return jsonOr(b, err, "{}")
}

// MobileAddTraffic lets the app add bytes moved outside the Go core (image
// loading, media streaming) to the same meter.
func MobileAddTraffic(down int64) {
	if down > 0 {
		traffic.add(down, 0, true)
	}
}

func MobileVersion() string { return appVersion }
