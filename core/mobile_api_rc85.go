package core

// RC85: Android'e masaüstüyle aynı özellikler — olay takibi, akıllı uyarılar,
// veri modu ve kullanım defteri, canlı TV kanal durumu / denetimi.

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
)

func MobileFollowed() string {
	if !mobileReady() {
		return "[]"
	}
	b, err := store.followedJSON()
	return jsonOr(b, err, "[]")
}

func MobileFollowedSeen() {
	if mobileReady() {
		store.markFollowedSeen()
	}
}

func MobileUnfollow(link string) {
	if follows != nil {
		follows.unfollowLink(link)
	}
}

// MobileToggleFollowStory follows the story by its link (falls back to id).
func MobileToggleFollowStory(link, id string) bool {
	if !mobileReady() || follows == nil {
		return false
	}
	if st := store.storyForFollow(link, id); st != nil {
		return follows.toggleStory(st)
	}
	if id == "" {
		return false
	}
	return follows.toggleEvent(id)
}

func MobileFollowCount() int {
	if follows == nil {
		return 0
	}
	return follows.count()
}

func MobileSmartAlerts() string {
	if !mobileReady() {
		return "{}"
	}
	b, err := store.smartAlertsJSON()
	return jsonOr(b, err, "{}")
}

func MobileUsage() string {
	if !mobileReady() {
		return "{}"
	}
	b, err := usageStatsJSON()
	return jsonOr(b, err, "{}")
}

func MobileSetDataMode(mode string, kbps, quotaMB int) {
	if !mobileReady() {
		return
	}
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
	_ = cfg.save()
}

func MobileResetUsage() { usage.reset() }

func MobileLiveState() string {
	if !mobileReady() {
		return "{}"
	}
	return liveStateJSON()
}

func MobileSetLiveState(js string) error {
	if !mobileReady() {
		return nil
	}
	liveHostReset()
	return setLiveState(js)
}

func MobileLiveHealth() string { return string(liveHealthDB.json()) }

// MobileProbeLive blocks (≤45 s); Android calls it from a background thread.
func MobileProbeLive(id, candsJSON, tvg, files string) string {
	b, err := probeLiveJSON(id, candsJSON, tvg, files)
	return jsonOr(b, err, `{"ok":false,"reason":"denetlenemedi"}`)
}

// ---- kanal başına Referer / User-Agent (Android ExoPlayer için) -------------

var (
	liveHostMu  sync.Mutex
	liveHostIdx map[string]liveCandidate
)

func liveHostReset() {
	liveHostMu.Lock()
	liveHostIdx = nil
	liveHostMu.Unlock()
}

// liveHostHeader finds the channel candidate whose host matches the stream URL
// and that carries its own Referer / User-Agent.
func liveHostHeader(raw string) (liveCandidate, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return liveCandidate{}, false
	}
	liveHostMu.Lock()
	defer liveHostMu.Unlock()
	if liveHostIdx == nil {
		liveHostIdx = map[string]liveCandidate{}
		add := func(list []json.RawMessage) {
			for _, raw := range list {
				var ch struct {
					URLs []json.RawMessage `json:"urls"`
				}
				if json.Unmarshal(raw, &ch) != nil {
					continue
				}
				for _, x := range ch.URLs {
					var c liveCandidate
					if json.Unmarshal(x, &c) != nil || c.U == "" || (c.R == "" && c.UA == "") {
						continue
					}
					if cu, err := url.Parse(c.U); err == nil {
						liveHostIdx[strings.ToLower(cu.Host)] = c
					}
				}
			}
		}
		var all []json.RawMessage
		_ = json.Unmarshal(liveTVSource(), &all)
		add(all)
		if cfg != nil {
			cfg.mu.RLock()
			add(cfg.LiveCustom)
			cfg.mu.RUnlock()
		}
	}
	c, ok := liveHostIdx[strings.ToLower(u.Host)]
	return c, ok
}
