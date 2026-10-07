package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const rss2Sample = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<title>Örnek Kaynak</title>
<item><title><![CDATA[Türkçe başlık: ışık &amp; gölge]]></title>
<link>https://example.com/a</link><pubDate>Mon, 02 Jan 2006 15:04:05 -0700</pubDate></item>
<item><title>İkinci haber</title><link>https://example.com/b</link>
<dc:date xmlns:dc="http://purl.org/dc/elements/1.1/">2024-05-01T10:00:00Z</dc:date></item>
</channel></rss>`

const rss2PipelineSample = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<title>Örnek Kaynak</title>
<item><title>Ankara'da örnek haber</title><link>https://example.com/a</link><pubDate>Mon, 02 Jan 2006 15:04:05 -0700</pubDate></item>
<item><title>İkinci Ankara haberi</title><link>https://example.com/b</link>
<dc:date xmlns:dc="http://purl.org/dc/elements/1.1/">2024-05-01T10:00:00Z</dc:date></item>
</channel></rss>`

const atomSample = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<title>Atom Kaynağı</title>
<entry><title>Atom haberi</title>
<link rel="alternate" href="https://example.com/atom1"/>
<link rel="self" href="https://example.com/self"/>
<published>2024-06-01T12:00:00Z</published></entry>
</feed>`

func TestParseRSS2(t *testing.T) {
	items, title, err := parseFeed([]byte(rss2Sample))
	if err != nil {
		t.Fatal(err)
	}
	if title != "Örnek Kaynak" {
		t.Errorf("feed title = %q", title)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}
	if items[0].Title != "Türkçe başlık: ışık & gölge" {
		t.Errorf("title[0] = %q", items[0].Title)
	}
	if items[0].Link != "https://example.com/a" || !items[0].HasDate {
		t.Errorf("item0 link/date wrong: %+v", items[0])
	}
	if !items[1].HasDate {
		t.Errorf("dc:date not parsed: %+v", items[1])
	}
}

func TestParseAtom(t *testing.T) {
	items, title, err := parseFeed([]byte(atomSample))
	if err != nil {
		t.Fatal(err)
	}
	if title != "Atom Kaynağı" {
		t.Errorf("title = %q", title)
	}
	if len(items) != 1 {
		t.Fatalf("got %d", len(items))
	}
	if items[0].Link != "https://example.com/atom1" {
		t.Errorf("atom link = %q (want alternate)", items[0].Link)
	}
}

// Non-UTF-8 feed (declared windows-1254) must decode via CharsetReader.
func TestParseCharset(t *testing.T) {
	// "İ" is 0xDD in windows-1254.
	body := "<?xml version=\"1.0\" encoding=\"windows-1254\"?><rss version=\"2.0\"><channel><title>K</title>" +
		"<item><title>\xddLK</title><link>https://x/1</link></item></channel></rss>"
	items, _, err := parseFeed([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !strings.HasPrefix(items[0].Title, "İLK") {
		t.Errorf("charset decode failed: %+v", items)
	}
}

func TestFetchPipelineUsesFeedTitleWhenSourceNameEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(rss2Sample))
	}))
	defer srv.Close()

	cfg := &Config{MaxPerCategory: 5, Categories: []Category{
		{Name: "Test", Sources: []Source{{URL: srv.URL}}},
	}}
	st := newStore(cfg)
	items := st.fetchOne(context.Background(), "Test", Source{URL: srv.URL})
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Source != "Örnek Kaynak" {
		t.Fatalf("source=%q, want feed title", items[0].Source)
	}
}

func TestFetchPipeline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(rss2PipelineSample))
	}))
	defer srv.Close()

	cfg := &Config{MaxPerCategory: 5, Categories: []Category{
		{Name: "Test", Sources: []Source{{URL: srv.URL}}},
	}}
	st := newStore(cfg)
	st.Refresh(context.Background())

	b, err := st.overviewJSON()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "Test") || !strings.Contains(s, "example.com/a") {
		t.Errorf("overview missing data: %s", s)
	}
	if !strings.Contains(s, "Örnek Kaynak") {
		t.Errorf("source name (feed title) not used: %s", s)
	}
}

func TestReaderExtract(t *testing.T) {
	page := `<html><head><title>Sayfa Başlık</title></head><body>
	<nav>menu menu menu</nav>
	<article><h1>Gerçek Başlık</h1>
	<p>` + strings.Repeat("Bu haberin gövdesidir ve yeterince uzundur. ", 20) + `</p>
	<p>` + strings.Repeat("İkinci paragraf da burada yer alır. ", 20) + `</p>
	<script>alert(1)</script></article>
	<footer>footer</footer></body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	b := fetchArticleJSON(context.Background(), srv.URL)
	s := string(b)
	if !strings.Contains(s, "Gerçek Başlık") {
		t.Errorf("title not extracted: %s", s)
	}
	if strings.Contains(s, "alert(1)") || strings.Contains(s, "menu menu") {
		t.Errorf("script/nav not stripped: %s", s)
	}
	if !strings.Contains(s, "gövdesidir") {
		t.Errorf("body not extracted: %s", s)
	}
}

func TestBlockedStubDetection(t *testing.T) {
	if !isBlockedStub("Please enable JS and disable any ad blocker") {
		t.Error("should detect JS/adblock stub")
	}
	if isBlockedStub("Bu uzun ve gerçek bir haber gövdesidir, engel yok.") {
		t.Error("false positive on normal text")
	}
}

func TestSummarize(t *testing.T) {
	out := summarize("<p>Merhaba <b>dünya</b> &amp; evren</p>")
	if out != "Merhaba dünya & evren" {
		t.Errorf("summarize=%q", out)
	}
}

func TestParseFeedImages(t *testing.T) {
	rss := []byte(`<?xml version="1.0"?><rss xmlns:media="http://search.yahoo.com/mrss/"><channel><title>T</title>
<item><title>A haber</title><link>https://a.example/1</link><media:thumbnail url="https://img.example/a.jpg"/></item>
<item><title>B haber</title><link>https://a.example/2</link><enclosure url="https://img.example/b.png" type="image/png"/></item>
<item><title>C haber</title><link>https://a.example/3</link><description><![CDATA[<p><img src="https://img.example/c.webp"> metin</p>]]></description></item>
<item><title>D haber</title><link>https://a.example/4</link><enclosure url="https://cdn.example/d.mp3" type="audio/mpeg"/></item>
<item><title>E haber</title><link>https://a.example/5</link><media:content url="javascript:alert(1)" medium="image"/></item>
</channel></rss>`)
	items, _, err := parseFeed(rss)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://img.example/a.jpg", "https://img.example/b.png", "https://img.example/c.webp", "", ""}
	for i, w := range want {
		if items[i].Image != w {
			t.Errorf("item %d image=%q want %q", i, items[i].Image, w)
		}
	}
}
