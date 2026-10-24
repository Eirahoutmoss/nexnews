//go:build windows

package core

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	webview2 "github.com/jchv/go-webview2"
)

//go:embed assets/index.html
var indexHTML string

//go:embed assets/bg.b64
var bgB64 string

//go:embed assets/logo.b64
var logoB64 string

// pageHTML injects the embedded brand logo and premium background (as data URIs)
// into the page before it is shown.
func pageHTML() string {
	h := indexHTML
	h = strings.ReplaceAll(h, "__NEXLOGO__", "data:image/png;base64,"+strings.TrimSpace(logoB64))
	h = strings.ReplaceAll(h, "__NEXBG__", "data:image/jpeg;base64,"+strings.TrimSpace(bgB64))
	h = strings.ReplaceAll(h, "__NEXRELAYTOKEN__", liveRelayToken)
	h = strings.ReplaceAll(h, "__NEXLIVETV__", string(liveTVJSON()))
	return h
}

// appTitle carries the slogan in the window title bar. The N icon is resource
// id 1 (rsrc assigns the group-icon id first when no manifest is embedded), so
// IconId: 1 gives the window its real title-bar icon instead of the generic one.
const appTitle = "NexNews — Dünyanın nabzı, tek akışta"

var ui webview2.WebView

// widgetOn mirrors whether the window is currently in widget mode.
var widgetOn atomic.Bool

// Main window size. 1360 px keeps the three-column layout (left nav, feed,
// right panel) visible; the CSS hides the right panel only below 1100 px.
const (
	mainW = 1360
	mainH = 860
)

// RunDesktop starts the Windows desktop application (WebView2 window).
func RunDesktop() {
	startLiveRelay()
	cfg = loadConfig()
	applyDataMode(cfg.DataMode, cfg.DataLimitKBps)
	startUsageLedger(onDataQuota)
	liveHealthDB.load()
	defer usage.flush()
	store = newStore(cfg)
	favs = newFavStore()
	follows = newFollowStore()
	speech = newSpeechProvider()

	ui = webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  appTitle,
			Width:  mainW,
			Height: mainH,
			Center: true,
			IconId: 1, // N logo embedded via rsrc (assets/nexnews.ico)
		},
	})
	if ui == nil {
		log.Fatalln("WebView2 başlatılamadı. Microsoft Edge WebView2 Runtime kurulu olmalı.")
	}
	defer ui.Destroy()
	ui.SetSize(mainW, mainH, webview2.HintNone)

	bindAll()

	// Translucent window: make the WebView2 background transparent (so any
	// backdrop shows through) and, if enabled, turn on the acrylic backdrop.
	makeWebviewTransparent(ui)
	if cfg.Glass {
		enableGlass(uintptr(ui.Window()))
	}

	applyTranslateCfg()
	applyPersonalCfg()

	// First paint shows the shell immediately; data streams in from the
	// background refresh below so the UI is never blocked on the network.
	go func() {
		store.Refresh(context.Background())
		pushData()
		translatePass()
		startAutoRefresh()
		// Kullanıcının ülkesine göre "📍 Yerel" kategorisi (ilk açılışta / ülke değişince).
		if ensureLocalCategory(context.Background()) {
			store.Refresh(context.Background())
			pushData()
		}
	}()

	ui.SetHtml(pageHTML())
	ui.Run()
}

// pushData notifies the page that fresh headlines are in the cache.
func pushData() {
	if ui == nil {
		return
	}
	ui.Dispatch(func() { ui.Eval("window.nnOnData && window.nnOnData()") })
	notifyAlerts()   // desktop toast for newly-appeared important events
	notifyFollowed() // takip edilen olaylardaki gelişmeler
}

// notifyFollowed toasts growth in followed events (quiet hours respected).
func notifyFollowed() {
	ups := store.syncFollows()
	if len(ups) == 0 || cfg.inQuietHours(time.Now()) {
		return
	}
	hwnd := uintptr(ui.Window())
	for i, u := range ups {
		if i >= 3 {
			break
		}
		title := "NexNews — Takip ettiğin olayda gelişme"
		msg := u.Title
		if u.Added > 0 {
			msg = fmt.Sprintf("+%d yeni haber · %s", u.Added, u.Title)
		}
		ui.Dispatch(func() { showToast(hwnd, title, msg) })
	}
}

// notifyAlerts shows desktop toasts for newly-appeared important events.
func notifyAlerts() {
	cfg.mu.RLock()
	on, min := cfg.Notify, cfg.NotifyMinScore
	cfg.mu.RUnlock()
	if !on {
		return
	}
	if min <= 0 {
		min = 75
	}
	if cfg.inQuietHours(time.Now()) {
		store.alerts(min) // mark as seen so they don't all fire at the end of quiet hours
		return
	}
	for _, st := range store.alerts(min) {
		title := "NexNews — Önemli"
		if st.Breaking {
			title = "NexNews — Son Dakika ⚡"
		}
		msg := st.Title
		hwnd := uintptr(ui.Window())
		ui.Dispatch(func() { showToast(hwnd, title, msg) })
	}
}

// translatePass fills translations in the background (if enabled) then refreshes
// the UI so titles switch to the local language.
func translatePass() {
	if !cfg.Translate {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	store.translateTitles(ctx)
	store.recluster() // translated titles let cross-language duplicates merge
	pushData()
}

func startAutoRefresh() {
	mins := cfg.RefreshMinutes
	if mins <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(time.Duration(mins) * time.Minute)
		defer t.Stop()
		last := time.Now()
		for range t.C {
			// Sınırlı veri modunda yenileme aralığı iki katına (en az 30 dk) çıkar.
			if dataLimited() {
				gap := time.Duration(maxInt(2*mins, 30)) * time.Minute
				if time.Since(last) < gap-time.Minute {
					continue
				}
			}
			last = time.Now()
			store.Refresh(context.Background())
			pushData()
			translatePass()
		}
	}()
}

func bindAll() {
	// Cache readers — fast, safe on the UI thread.
	must(ui.Bind("nnOverview", func() (string, error) {
		b, err := store.overviewJSON()
		return string(b), err
	}))
	must(ui.Bind("nnCategory", func(name string) (string, error) {
		b, err := store.categoryJSON(name)
		return string(b), err
	}))
	must(ui.Bind("nnGetConfig", func() (string, error) {
		b, err := cfg.toJSON()
		return string(b), err
	}))
	must(ui.Bind("nnVersion", func() (string, error) { return appVersion, nil }))
	must(ui.Bind("nnSpeechSupport", func() (string, error) {
		if speech != nil && speech.Available() {
			return "on", nil
		}
		return "off", nil
	}))
	must(ui.Bind("nnSpeak", func(text string) (string, error) {
		if speech == nil || !speech.Available() {
			return "off", nil
		}
		if err := speech.Speak(text); err != nil {
			return "", err
		}
		return "on", nil
	}))
	must(ui.Bind("nnStopSpeech", func() (string, error) {
		if speech != nil {
			_ = speech.Stop()
		}
		return "ok", nil
	}))
	must(ui.Bind("nnSourceHealth", func() (string, error) { b, err := store.sourceHealthJSON(); return string(b), err }))
	must(ui.Bind("nnSourceHealthSummary", func() (string, error) { b, err := store.sourceHealthSummaryJSON(); return string(b), err }))
	must(ui.Bind("nnContentIntelligence", func() (string, error) { b, err := store.contentIntelligenceJSON(); return string(b), err }))
	// Olay takibi: arayüz olayın bağlantısını (ve kimliğini) gönderir; olay
	// önbellekte bulunursa başlık/kelime imzasıyla birlikte saklanır.
	must(ui.Bind("nnToggleFollowEvent", func(payload string) (string, error) {
		var p struct{ ID, Link string }
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			p.ID = payload
		}
		if st := store.storyForFollow(p.Link, p.ID); st != nil {
			if follows.toggleStory(st) {
				return "on", nil
			}
			return "off", nil
		}
		if p.ID == "" {
			return "off", nil
		}
		if follows.toggleEvent(p.ID) {
			return "on", nil
		}
		return "off", nil
	}))
	must(ui.Bind("nnIsFollowedEvent", func(id string) (string, error) {
		if follows.isEvent(id) {
			return "on", nil
		}
		return "off", nil
	}))
	must(ui.Bind("nnFollowed", func() (string, error) { b, err := store.followedJSON(); return string(b), err }))
	must(ui.Bind("nnFollowedSeen", func() (string, error) { store.markFollowedSeen(); return "ok", nil }))
	must(ui.Bind("nnUnfollow", func(link string) (string, error) { follows.unfollowLink(link); return "ok", nil }))
	must(ui.Bind("nnFollowCount", func() (int, error) { return follows.count(), nil }))
	must(ui.Bind("nnToggleFollowTopic", func(topic string) (string, error) {
		if follows.toggleTopic(topic) {
			return "on", nil
		}
		return "off", nil
	}))

	// Event file (a story's timeline) and advanced search — fast cache reads.
	must(ui.Bind("nnStory", func(link string) (string, error) {
		b, err := store.storyJSON(link)
		return string(b), err
	}))
	must(ui.Bind("nnSearch", func(query string) (string, error) {
		b, err := store.searchJSON(query)
		return string(b), err
	}))
	must(ui.Bind("nnDeepSearch", func(query string) (string, error) {
		b, err := store.deepSearchJSON(query)
		return string(b), err
	}))

	// Favorites + reading history.
	must(ui.Bind("nnToggleFav", func(payload string) (string, error) {
		var it savedItem
		if err := json.Unmarshal([]byte(payload), &it); err != nil {
			return "", err
		}
		if favs.toggleFav(it) {
			return "on", nil
		}
		return "off", nil
	}))
	must(ui.Bind("nnFavLinks", func() (string, error) {
		m := favs.favLinks()
		arr := make([]string, 0, len(m))
		for k := range m {
			arr = append(arr, k)
		}
		b, _ := json.Marshal(arr)
		return string(b), nil
	}))
	must(ui.Bind("nnFavs", func() (string, error) { return string(favs.listJSON("favs")), nil }))
	must(ui.Bind("nnHistory", func() (string, error) { return string(favs.listJSON("history")), nil }))
	must(ui.Bind("nnAddHistory", func(payload string) (string, error) {
		var it savedItem
		if err := json.Unmarshal([]byte(payload), &it); err != nil {
			return "", err
		}
		favs.addHistory(it)
		return "ok", nil
	}))
	must(ui.Bind("nnClearHistory", func() (string, error) { favs.clearHistory(); return "ok", nil }))

	// Open the article in the default browser (fast: spawns a process).
	must(ui.Bind("nnOpen", func(url string) (string, error) {
		if err := openInBrowser(url); err != nil {
			return "", err
		}
		return "ok", nil
	}))

	// Refresh — kick off in the background, notify via nnOnData when done.
	must(ui.Bind("nnRefresh", func() (string, error) {
		go func() {
			store.Refresh(context.Background())
			pushData()
		}()
		return "started", nil
	}))

	// In-app reader — network bound, so run async and push the result to JS via
	// window.nnOnArticle to avoid blocking the UI thread.
	must(ui.Bind("nnRead", func(url string) (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			b := fetchArticleJSON(ctx, url)
			ui.Dispatch(func() { ui.Eval("window.nnOnArticle && window.nnOnArticle(" + string(b) + ")") })
		}()
		return "started", nil
	}))

	// Autodiscover feeds from a site address; result pushed via nnOnDiscover.
	must(ui.Bind("nnDiscover", func(query string) (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
			defer cancel()
			b := discoverJSON(ctx, query)
			ui.Dispatch(func() { ui.Eval("window.nnOnDiscover && window.nnOnDiscover(" + string(b) + ")") })
		}()
		return "started", nil
	}))

	// Widget mode: shrink to a compact always-on-top ticker, or restore.
	must(ui.Bind("nnSetWidget", func(on bool) (string, error) {
		hwnd := uintptr(ui.Window())
		cfg.mu.RLock()
		top := cfg.WidgetTopmost
		cfg.mu.RUnlock()
		cfg.mu.RLock()
		widgetW, widgetH := cfg.WidgetWidth, cfg.WidgetHeight
		mode, tint, light := cfg.WidgetMode, cfg.WidgetTint, cfg.Theme == "acik"
		cfg.mu.RUnlock()
		if widgetW < 300 {
			widgetW = 460
		}
		if widgetH < 60 {
			widgetH = 120
		}
		widgetOn.Store(on)
		ui.Dispatch(func() {
			host := nativeHostWindow(hwnd)
			if on {
				// Initial entry still uses WebView2 SetSize once, then permanently
				// switches to the frameless native popup. Live resizing never calls
				// SetSize again, so Windows cannot reintroduce the frame.
				ui.SetSize(widgetW, widgetH, webview2.HintNone)
				setFrameless(host, true)
				// Windows 11 rounds the popup natively (keeps blur + shadow);
				// Windows 10 needs a clipping region instead.
				if useNativeRoundCorners() {
					setWidgetRegion(host, false)
					setRoundCorners(host, true)
				} else {
					setWidgetRegion(host, true, widgetW, widgetH)
				}
				// RC81: native backdrop chosen by the user (opak/cam/bulanık/
				// saydam). The CSS surface opacity (CONFIG.widgetOpacity)
				// decides how much of it shows through.
				applyWidgetBackdrop(host, mode, tint, light)
				makeWebviewTransparent(ui)
				// Removing the frame changes the client area. go-webview2's normal
				// WM_SIZE path is not guaranteed to run for this style-only change,
				// so explicitly force the WebView2 controller to fill the new client
				// rectangle. This is the missing piece behind the white side strips.
				syncWebViewClient(host)
				placeWidgetBottomRight(host, widgetW, widgetH, 18)
				// Positioning can produce another size notification on some Windows
				// builds, so finish with one deterministic controller resize.
				syncWebViewClient(host)
				setTopmost(host, top)
			} else {
				setTopmost(host, false)
				setWidgetRegion(host, false)
				clearWidgetBackdrop(host)
				setRoundCorners(host, false)
				ui.SetSize(mainW, mainH, webview2.HintNone)
				setFrameless(host, false)
				centerWindow(host, mainW, mainH)
				makeWebviewTransparent(ui)
				setGlass(host, cfg.Glass)
				syncWebViewClient(host)
			}
			forceRedraw(host)
		})
		return "ok", nil
	}))

	// Live widget look (mode / tint / surface opacity) from the settings or the
	// widget's right-click menu. Persisted immediately.
	must(ui.Bind("nnSetWidgetLook", func(mode string, tint, opacity int) (string, error) {
		cfg.mu.Lock()
		cfg.WidgetMode, cfg.WidgetTint, cfg.WidgetOpacity = mode, tint, opacity
		if cfg.WidgetOpacity < 0 || cfg.WidgetOpacity > 100 {
			cfg.WidgetOpacity = 78
		}
		cfg.normalizeV11()
		mode, tint, light := cfg.WidgetMode, cfg.WidgetTint, cfg.Theme == "acik"
		cfg.mu.Unlock()
		_ = cfg.save()
		if widgetOn.Load() {
			hwnd := uintptr(ui.Window())
			ui.Dispatch(func() {
				host := nativeHostWindow(hwnd)
				applyWidgetBackdrop(host, mode, tint, light)
				forceRedraw(host)
			})
		}
		return mode, nil
	}))

	// Remember the right-panel tab without triggering a refresh.
	must(ui.Bind("nnSetRightTab", func(tab string) (string, error) {
		cfg.mu.Lock()
		cfg.RightTab = tab
		cfg.normalizeV11()
		cfg.mu.Unlock()
		return "ok", cfg.save()
	}))

	// Veri modu ve kullanım defteri.
	must(ui.Bind("nnUsage", func() (string, error) { b, err := usageStatsJSON(); return string(b), err }))
	must(ui.Bind("nnSetDataMode", func(mode string, kbps, quotaMB int) (string, error) {
		if mode != dataModeLimited {
			mode = dataModeFull
		}
		if kbps < 32 || kbps > 100000 {
			kbps = defaultLimitKBs
		}
		if quotaMB < 0 {
			quotaMB = 0
		}
		cfg.mu.Lock()
		cfg.DataMode, cfg.DataLimitKBps, cfg.DataQuotaMB = mode, kbps, quotaMB
		cfg.mu.Unlock()
		applyDataMode(mode, kbps)
		return mode, cfg.save()
	}))
	// Canlı TV: aday adresleri dene, sonucu kaydet; hepsi çalışmazsa iptv-org
	// dizininden lisanslı güncel adres ara (otomatik onarım).
	must(ui.Bind("nnLiveHealth", func() (string, error) { return string(liveHealthDB.json()), nil }))
	// Kanal denetimi ağda 45 sn'ye kadar sürebilir. WebView2 bağlamaları arayüz
	// iş parçacığında çalıştığından (RC84'te pencereyi donduruyordu) iş arka
	// planda yapılır, sonuç window.nnOnAsync ile sayfaya iletilir.
	must(ui.Bind("nnProbeLive", func(ticket, id, candsJSON, tvg, files string) (string, error) {
		go func() {
			b, err := probeLiveJSON(id, candsJSON, tvg, files)
			asyncReply(ticket, string(b), err)
		}()
		return "started", nil
	}))
	must(ui.Bind("nnGetLiveState", func() (string, error) { return liveStateJSON(), nil }))
	// RC87: kullanıcının kendi kanal listeleri (yalnızca istenince güncellenir).
	must(ui.Bind("nnLiveLists", func() (string, error) { return liveListsJSON(), nil }))
	must(ui.Bind("nnLiveListAdd", func(ticket, name, url, text string) (string, error) {
		go func() {
			b, err := liveListAdd(context.Background(), name, url, text)
			asyncReply(ticket, string(b), err)
		}()
		return "started", nil
	}))
	must(ui.Bind("nnLiveListRefresh", func(ticket, id string) (string, error) {
		go func() {
			b, err := liveListRefresh(context.Background(), id)
			asyncReply(ticket, string(b), err)
		}()
		return "started", nil
	}))
	must(ui.Bind("nnLiveListDelete", func(id string) (string, error) { return "ok", liveListDelete(id) }))
	must(ui.Bind("nnLiveListRename", func(id, name string) (string, error) { return "ok", liveListRename(id, name) }))
	must(ui.Bind("nnLiveBuiltin", func(on bool) (string, error) { return "ok", liveSetBuiltin(on) }))
	must(ui.Bind("nnSetLiveState", func(stateJSON string) (string, error) { return "ok", setLiveState(stateJSON) }))
	must(ui.Bind("nnResetUsage", func() (string, error) { usage.reset(); return "ok", nil }))

	// Network meter: live rate + last-5-minutes data (cheap, no network).
	must(ui.Bind("nnNetStats", func() (string, error) {
		b, err := netStatsJSON()
		return string(b), err
	}))
	// Bytes the WebView loaded itself (images, map tiles, browser-side
	// translation), reported by the page's PerformanceObserver.
	must(ui.Bind("nnAddWebTraffic", func(bytes int) (string, error) {
		if bytes > 0 && bytes < 1<<30 {
			traffic.add(int64(bytes), 0, true)
		}
		return "ok", nil
	}))

	// Drag the borderless widget from anywhere on its surface.
	must(ui.Bind("nnDragMove", func() (string, error) {
		hwnd := uintptr(ui.Window())
		ui.Dispatch(func() { dragMove(hwnd) })
		return "ok", nil
	}))
	// Live widget resize. The JS resize grip sends the desired outer dimensions;
	// native resizing keeps the borderless window and rounded region intact.
	must(ui.Bind("nnResizeWidget", func(width, height int) (string, error) {
		hwnd := uintptr(ui.Window())
		if width < 300 {
			width = 300
		}
		if width > 1000 {
			width = 1000
		}
		if height < 60 {
			height = 60
		}
		if height > 500 {
			height = 500
		}
		cfg.mu.Lock()
		cfg.WidgetWidth, cfg.WidgetHeight = width, height
		cfg.mu.Unlock()
		ui.Dispatch(func() { resizeWidgetWindow(hwnd, width, height) })
		return "ok", nil
	}))

	// Persist the final dimensions once the user releases the resize grip.
	must(ui.Bind("nnSaveWidgetSize", func() (string, error) {
		return "ok", cfg.save()
	}))

	// Toggle always-on-top (the widget's right-click "Üstte kal").
	must(ui.Bind("nnSetTopmost", func(on bool) (string, error) {
		hwnd := uintptr(ui.Window())
		cfg.mu.Lock()
		cfg.WidgetTopmost = on
		cfg.mu.Unlock()
		_ = cfg.save()
		ui.Dispatch(func() { setTopmost(hwnd, on) })
		return "ok", nil
	}))
	must(ui.Bind("nnMinimize", func() (string, error) {
		hwnd := uintptr(ui.Window())
		ui.Dispatch(func() { minimizeWindow(hwnd) })
		return "ok", nil
	}))

	// Toggle the acrylic backdrop live from the settings UI.
	must(ui.Bind("nnSetGlass", func(on bool) (string, error) {
		hwnd := uintptr(ui.Window())
		ui.Dispatch(func() { setGlass(hwnd, on) })
		return "ok", nil
	}))

	// Save config from the settings UI, then refresh.
	must(ui.Bind("nnSaveConfig", func(payload string) (string, error) {
		if err := cfg.replaceFrom([]byte(payload)); err != nil {
			return "", err
		}
		applyTranslateCfg()
		applyPersonalCfg()
		resetWeatherCache()
		go func() {
			ensureLocalCategory(context.Background())
			store.Refresh(context.Background())
			pushData()
			translatePass()
		}()
		return "ok", nil
	}))

	// Weather panel data (network) — pushed via nnOnWeather.
	must(ui.Bind("nnFinance", func() (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			b := financeJSON(ctx, cfg)
			ui.Dispatch(func() { ui.Eval("window.nnOnFinance && window.nnOnFinance(" + string(b) + ")") })
		}()
		return "started", nil
	}))

	must(ui.Bind("nnBriefing", func() (string, error) {
		b, err := store.briefingJSON()
		return string(b), err
	}))
	must(ui.Bind("nnPersonalIntelligence", func() (string, error) {
		b, err := store.personalIntelligenceJSON()
		return string(b), err
	}))
	must(ui.Bind("nnSmartAlerts", func() (string, error) {
		b, err := store.smartAlertsJSON()
		return string(b), err
	}))
	// Network-bound boards: bindings run on the UI thread, so the work happens
	// in a goroutine and the result is pushed back via a window callback.
	must(ui.Bind("nnOfficialPublications", func() (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			b, err := officialPublicationsJSON(ctx)
			if err != nil || len(b) == 0 {
				b = []byte(`{"publications":[],"portals":[],"error":"Resmî yayınlar alınamadı."}`)
			}
			pushJS("nnOnOfficialPublications", b)
		}()
		return "started", nil
	}))
	must(ui.Bind("nnRadio", func() (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			b, err := radioJSON(ctx)
			if err != nil || len(b) == 0 {
				b = []byte(`{"stations":[],"error":"Radyo dizinine ulaşılamadı."}`)
			}
			pushJS("nnOnRadio", b)
		}()
		return "started", nil
	}))
	must(ui.Bind("nnWorldPulse", func() (string, error) {
		b, err := store.worldPulseJSON()
		return string(b), err
	}))
	must(ui.Bind("nnEventIntelligence", func() (string, error) {
		b, err := store.eventIntelligenceJSON()
		return string(b), err
	}))
	must(ui.Bind("nnPredictive", func() (string, error) {
		b, err := store.predictiveJSON()
		return string(b), err
	}))
	must(ui.Bind("nnEventGraph", func() (string, error) {
		b, err := store.eventGraphJSON()
		return string(b), err
	}))
	must(ui.Bind("nnEntityIntelligence", func() (string, error) {
		b, err := store.entityIntelligenceJSON()
		return string(b), err
	}))
	must(ui.Bind("nnSports", func() (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
			defer cancel()
			b := sportsJSON(ctx, cfg)
			ui.Dispatch(func() { ui.Eval("window.nnOnSports && window.nnOnSports(" + string(b) + ")") })
		}()
		return "started", nil
	}))

	must(ui.Bind("nnStandings", func() (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			b := standingsJSON(ctx, cfg)
			ui.Dispatch(func() { ui.Eval("window.nnOnStandings && window.nnOnStandings(" + string(b) + ")") })
		}()
		return "started", nil
	}))

	must(ui.Bind("nnWeather", func() (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
			defer cancel()
			cfg.mu.RLock()
			place := cfg.WeatherPlace
			cfg.mu.RUnlock()
			b := weatherJSON(ctx, place)
			ui.Dispatch(func() { ui.Eval("window.nnOnWeather && window.nnOnWeather(" + string(b) + ")") })
		}()
		return "started", nil
	}))

	// Coordinates for the radar map (network) — pushed via nnOnGeo.
	must(ui.Bind("nnGeo", func() (string, error) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
			defer cancel()
			cfg.mu.RLock()
			place := cfg.WeatherPlace
			cfg.mu.RUnlock()
			b := geoJSON(ctx, place)
			ui.Dispatch(func() { ui.Eval("window.nnOnGeo && window.nnOnGeo(" + string(b) + ")") })
		}()
		return "started", nil
	}))

	// System default language code (e.g. "tr"), for the settings display.
	must(ui.Bind("nnSysLang", func() (string, error) { return systemLang(), nil }))

	// Toggle translation live; returns the active target code. The live
	// reachability test and the headline translation now run in the WebView2
	// browser context (see translateDOM/trFetch in index.html), whose network
	// stack reaches external hosts even where Go's client is throttled.
	must(ui.Bind("nnSetTranslate", func(on bool, lang string) (string, error) {
		cfg.mu.Lock()
		cfg.Translate = on
		cfg.TranslateLang = strings.TrimSpace(lang)
		cfg.mu.Unlock()
		_ = cfg.save()
		target := applyTranslateCfg()
		go translatePass()
		pushData()
		return target, nil
	}))

	// Export sources/settings to a file chosen by the user.
	must(ui.Bind("nnExport", func(kind string) (string, error) {
		hwnd := uintptr(ui.Window())
		var data []byte
		var filter, defExt, defName string
		switch kind {
		case "json":
			b, _ := cfg.toJSON()
			data = b
			filter = "JSON\x00*.json\x00Tümü\x00*.*\x00\x00"
			defExt, defName = "json", "nexnews-yedek.json"
		default: // csv
			cats, _ := cfg.snapshot()
			data = sourcesToCSV(cats)
			filter = "CSV\x00*.csv\x00Metin\x00*.txt\x00Tümü\x00*.*\x00\x00"
			defExt, defName = "csv", "nexnews-kaynaklar.csv"
		}
		path := saveFileDialog(hwnd, filter, defExt, defName)
		if path == "" {
			return "", nil // cancelled
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return "", err
		}
		return path, nil
	}))

	// Import sources (CSV/TXT) or restore a full backup (JSON).
	must(ui.Bind("nnImport", func() (string, error) {
		hwnd := uintptr(ui.Window())
		path := openFileDialog(hwnd, "Desteklenen\x00*.csv;*.txt;*.json\x00Tümü\x00*.*\x00\x00")
		if path == "" {
			return "İçe aktarma iptal edildi.", nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		var msg string
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json":
			if err := cfg.replaceFrom(data); err != nil {
				return "", err
			}
			msg = "Yedek geri yüklendi."
		case ".txt":
			n := mergeCategories(cfg, []Category{{Name: "İçe Aktarılan", Sources: txtToSources(data)}})
			_ = cfg.save()
			msg = strconv.Itoa(n) + " kaynak eklendi."
		default: // csv
			cats, err := csvToCategories(data)
			if err != nil {
				return "", err
			}
			n := mergeCategories(cfg, cats)
			_ = cfg.save()
			msg = strconv.Itoa(n) + " kaynak eklendi."
		}
		applyTranslateCfg()
		go func() {
			store.Refresh(context.Background())
			pushData()
			translatePass()
		}()
		return msg, nil
	}))
}

// pushJS hands a JSON payload to window.<fn> on the UI thread.
func pushJS(fn string, payload []byte) {
	if ui == nil {
		return
	}
	js := "window." + fn + " && window." + fn + "(" + string(payload) + ")"
	ui.Dispatch(func() { ui.Eval(js) })
}

func must(err error) {
	if err != nil {
		log.Fatalln("bind:", err)
	}
}

// onDataQuota: aylık kotanın %80'inde uyarır, dolunca sınırlı moda geçer.
func onDataQuota(kind string) {
	if ui == nil {
		return
	}
	title, msg := "NexNews — Veri kotası", "Bu ayki veri kotanın %80'i kullanıldı."
	if kind == "cap" {
		cfg.mu.Lock()
		if cfg.DataMode != dataModeLimited {
			cfg.DataMode = dataModeLimited
		}
		mode, kbps := cfg.DataMode, cfg.DataLimitKBps
		cfg.mu.Unlock()
		applyDataMode(mode, kbps)
		_ = cfg.save()
		msg = "Aylık veri kotası doldu — NexNews sınırlı veri akışına geçti."
	}
	hwnd := uintptr(ui.Window())
	ui.Dispatch(func() {
		showToast(hwnd, title, msg)
		ui.Eval("window.nnOnDataMode && window.nnOnDataMode()")
	})
}

// asyncReply resolves a promise created by the page's nnAsync() helper.
func asyncReply(ticket, val string, err error) {
	ok := "true"
	if err != nil {
		ok, val = "false", err.Error()
	}
	t, _ := json.Marshal(ticket)
	v, _ := json.Marshal(val)
	js := "window.nnOnAsync && window.nnOnAsync(" + string(t) + "," + ok + "," + string(v) + ")"
	if ui != nil {
		ui.Dispatch(func() { ui.Eval(js) })
	}
}
