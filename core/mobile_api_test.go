package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMobileAPIEndToEnd(t *testing.T) {
	now := time.Now().UTC()
	feed := func(src string) string {
		return fmt.Sprintf(`<?xml version="1.0"?><rss><channel><title>%s</title>
<item><title>Malatya'da 5,4 büyüklüğünde deprem meydana geldi</title><link>https://%s.example/a</link><pubDate>%s</pubDate></item>
<item><title>Merkez Bankası faiz kararını açıkladı</title><link>https://%s.example/b</link><pubDate>%s</pubDate></item>
</channel></rss>`, src, src, now.Format(time.RFC1123Z), src, now.Add(-time.Hour).Format(time.RFC1123Z))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(feed(strings.Trim(r.URL.Path, "/"))))
	}))
	defer srv.Close()

	oldStore, oldCfg, oldFavs, oldFollows := store, cfg, favs, follows
	defer func() {
		store, cfg, favs, follows = oldStore, oldCfg, oldFavs, oldFollows
		dataDirOverride, sysLangOverride = "", ""
	}()
	store, cfg = nil, nil

	dir := t.TempDir()
	if err := MobileInit(dir, "tr-TR"); err != nil {
		t.Fatal(err)
	}
	if systemLang() != "tr" {
		t.Fatalf("lang = %q", systemLang())
	}
	// First run trims bundled sources for mobile data.
	var c Config
	if err := json.Unmarshal([]byte(MobileConfig()), &c); err != nil {
		t.Fatal(err)
	}
	for _, cat := range c.Categories {
		if !strings.HasPrefix(cat.Name, "⚡") && len(cat.Sources) > mobileSourcesPerCategory {
			t.Fatalf("%s has %d sources on first mobile run", cat.Name, len(cat.Sources))
		}
	}
	// Point the config at the local test feeds.
	c.Categories = []Category{{Name: "Türkiye", Sources: []Source{{Name: "A", URL: srv.URL + "/aa"}, {Name: "B", URL: srv.URL + "/bb"}, {Name: "C", URL: srv.URL + "/cc"}}}}
	c.Translate = false
	c.Notify = true
	c.NotifyMinScore = 1
	raw, _ := json.Marshal(&c)
	if err := MobileSaveConfig(string(raw)); err != nil {
		t.Fatal(err)
	}
	MobileRefresh()
	if MobileLastFetch() == 0 {
		t.Fatal("no fetch recorded")
	}
	var ov overview
	if err := json.Unmarshal([]byte(MobileOverview()), &ov); err != nil {
		t.Fatal(err)
	}
	if len(ov.Categories) != 1 || ov.Categories[0].Total == 0 {
		t.Fatalf("overview empty: %+v", ov.Categories)
	}
	// First alert call only seeds; second returns nothing new.
	if a := MobileNewAlerts(); a != "[]" {
		t.Fatalf("first call must only seed, got %s", a)
	}
	if a := MobileNewAlerts(); a != "[]" {
		t.Fatalf("nothing new expected, got %s", a)
	}
	link := ov.Categories[0].Items[0].Link
	if !strings.Contains(MobileStory(link), `"found":true`) {
		t.Fatal("story not found")
	}
	item := fmt.Sprintf(`{"link":%q,"title":"x","cat":"Türkiye"}`, link)
	if !MobileToggleFav(item) || !strings.Contains(MobileFavLinks(), link) {
		t.Fatal("favourite not stored")
	}
	MobileAddHistory(item)
	if !strings.Contains(MobileHistory(), link) {
		t.Fatal("history not stored")
	}
	if !strings.Contains(MobileLiveTV(), "trt-haber") {
		t.Fatal("live tv list missing")
	}
	if !strings.Contains(MobileLiveHeaders("https://fastly.md.bbci.co.uk/x.m3u8"), "bbc.com") {
		t.Fatal("live headers")
	}
	if n := MobileRestoreAllSources(); n <= 0 {
		t.Fatalf("restore added %d", n)
	}
}
