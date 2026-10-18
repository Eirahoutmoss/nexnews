package core

import (
	"strings"
	"testing"
	"time"
)

func TestRewriteM3U8UsesBaseAndHeaders(t *testing.T) {
	body := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=640x360\nlow/index.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=5000000,RESOLUTION=1920x1080\nhd/index.m3u8\n"
	out := rewriteM3U8(body, "https://final.example.com/live/master.m3u8", liveHeaders{ref: "https://site.example/"}, 0)
	if !strings.Contains(out, "u=https%3A%2F%2Ffinal.example.com%2Flive%2Flow%2Findex.m3u8") {
		t.Fatalf("relative URI not resolved against final URL:\n%s", out)
	}
	if !strings.Contains(out, "&r=https%3A%2F%2Fsite.example%2F") {
		t.Fatalf("referer not propagated:\n%s", out)
	}
	lim := rewriteM3U8(body, "https://final.example.com/live/master.m3u8", liveHeaders{}, 480)
	if strings.Contains(lim, "1920x1080") || !strings.Contains(lim, "640x360") {
		t.Fatalf("limited mode should keep only <=480p:\n%s", lim)
	}
	// Hiçbiri uymuyorsa en küçük kalmalı.
	body2 := "#EXTM3U\n#EXT-X-STREAM-INF:RESOLUTION=1280x720\na.m3u8\n#EXT-X-STREAM-INF:RESOLUTION=1920x1080\nb.m3u8\n"
	lim2 := rewriteM3U8(body2, "https://x.example/m.m3u8", liveHeaders{}, 480)
	if !strings.Contains(lim2, "1280x720") || strings.Contains(lim2, "1920x1080") {
		t.Fatalf("smallest variant should survive:\n%s", lim2)
	}
}

func TestRepairAllowed(t *testing.T) {
	ok := []string{"https://tv-trthaber.medya.trt.com.tr/master.m3u8", "https://jmp2.uk/plu-1.m3u8"}
	bad := []string{"http://212.5.144.156:8080/history/index.m3u8", "https://nord.ayakkabiparti.lol/natgeo/index.m3u8", "https://stream8.cinerama.uz/1448/playlist.m3u8"}
	for _, u := range ok {
		if !repairAllowed(u) {
			t.Errorf("should allow %s", u)
		}
	}
	for _, u := range bad {
		if repairAllowed(u) {
			t.Errorf("should block %s", u)
		}
	}
}

func TestFollowMatchesGrownStory(t *testing.T) {
	f := &followStore{items: map[string]*followedEvent{}, topics: map[string]bool{}}
	st := &Story{Title: "İzmir'de 5,1 büyüklüğünde deprem", Link: "https://a/1", Category: "Türkiye", EventID: "x1", Count: 2}
	f.items["x1"] = &followedEvent{ID: "x1", Title: st.Title, Link: st.Link, Category: st.Category, Tokens: storyTokens(st), Count: 2}
	grown := &Story{Title: "İzmir'de 5,1 büyüklüğünde deprem: AFAD açıklama yaptı", Link: "https://b/2", Category: "Türkiye", EventID: "x2", Count: 6}
	if f.matchLocked(grown) != "x1" {
		t.Fatal("grown story with a new id/link should still match the followed event")
	}
	other := &Story{Title: "Merkez Bankası faiz kararını açıkladı", Link: "https://c/3", Category: "Türkiye", EventID: "x3"}
	if f.matchLocked(other) != "" {
		t.Fatal("unrelated story must not match")
	}
}

func TestByteLimiterPaces(t *testing.T) {
	applyDataMode(dataModeLimited, 64) // 64 KB/s
	defer applyDataMode(dataModeFull, 256)
	var l byteLimiter
	start := time.Now()
	for i := 0; i < 8; i++ {
		l.wait(16 * 1024) // 128 KB → ~2 s, 1 s burst allowed
	}
	if el := time.Since(start); el < 700*time.Millisecond {
		t.Fatalf("limiter too permissive: %v", el)
	}
}

func TestUsageQuota(t *testing.T) {
	u := newUsageLedger()
	u.add(900*1024*1024, 0)
	if k := u.quotaCheck(1000); k != "warn" {
		t.Fatalf("want warn, got %q", k)
	}
	u.add(200*1024*1024, 0)
	if k := u.quotaCheck(1000); k != "cap" {
		t.Fatalf("want cap, got %q", k)
	}
	if k := u.quotaCheck(1000); k != "" {
		t.Fatalf("cap must fire once, got %q", k)
	}
}

func TestProbePlaylistURIUsesNewestSegment(t *testing.T) {
	media := "#EXTM3U\n#EXTINF:6,\nseg1.ts\n#EXTINF:6,\nseg2.ts\n#EXTINF:6,\nseg3.ts\n"
	if u, m := probePlaylistURI(media); u != "seg3.ts" || m {
		t.Fatalf("got %q %v", u, m)
	}
	master := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1\na.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=2\nb.m3u8\n"
	if u, m := probePlaylistURI(master); u != "a.m3u8" || !m {
		t.Fatalf("got %q %v", u, m)
	}
}
