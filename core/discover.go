package core

// RSS/Atom autodiscovery: the user types a site address (e.g. "ntv.com.tr") and
// we find its feed(s). Strategy: (1) read the standard <link rel="alternate"
// type="application/rss+xml"> hints from the page head, (2) fall back to common
// feed paths, then (3) validate every candidate by actually fetching+parsing it,
// so only real feeds are returned. A site may expose several feeds, so we return
// a list and let the UI pick.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type foundFeed struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type discoverResult struct {
	Query string      `json:"query"`
	Feeds []foundFeed `json:"feeds"`
	Err   string      `json:"err,omitempty"`
}

var commonFeedPaths = []string{
	"/feed", "/feed/", "/rss", "/rss/", "/rss.xml", "/feed.xml",
	"/atom.xml", "/index.xml", "/feeds/posts/default?alt=rss",
	"/?feed=rss2", "/rss/all.xml", "/en/rss", "/rss/news",
}

func normalizeSite(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errURL
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	return url.Parse(raw)
}

var errURL = &discoverErr{"Geçersiz adres."}

type discoverErr struct{ s string }

func (e *discoverErr) Error() string { return e.s }

// discoverJSON runs the whole discovery and returns a JSON discoverResult.
func discoverJSON(ctx context.Context, query string) []byte {
	res := discoverResult{Query: query}

	base, err := normalizeSite(query)
	if err != nil {
		res.Err = "Geçersiz adres."
		b, _ := json.Marshal(res)
		return b
	}

	client := &http.Client{Timeout: 12 * time.Second}

	// 1) candidates from the page's <link> hints
	candidates := map[string]string{} // url -> title hint
	if page, err := httpGet(ctx, client, base.String()); err == nil {
		for u, t := range feedLinksFromHTML(page, base) {
			candidates[u] = t
		}
	}

	// 2) common path fallbacks (only add if not already present)
	for _, p := range commonFeedPaths {
		ref, err := url.Parse(p)
		if err != nil {
			continue
		}
		u := base.ResolveReference(ref).String()
		if _, ok := candidates[u]; !ok {
			candidates[u] = ""
		}
	}

	// 3) validate candidates concurrently; keep only real feeds
	type vr struct {
		url, title string
		ok         bool
	}
	ch := make(chan vr, len(candidates))
	var wg sync.WaitGroup
	for u, hint := range candidates {
		wg.Add(1)
		go func(u, hint string) {
			defer wg.Done()
			name, ok := validateFeed(ctx, client, u)
			if ok && name == "" {
				name = hint
			}
			ch <- vr{url: u, title: name, ok: ok}
		}(u, hint)
	}
	go func() { wg.Wait(); close(ch) }()

	seen := map[string]bool{}
	for r := range ch {
		if !r.ok || seen[r.url] {
			continue
		}
		seen[r.url] = true
		res.Feeds = append(res.Feeds, foundFeed{Title: r.title, URL: r.url})
	}

	if len(res.Feeds) == 0 && res.Err == "" {
		res.Err = "Bu adreste RSS/Atom akışı bulunamadı."
	}
	b, _ := json.Marshal(res)
	return b
}

func httpGet(ctx context.Context, client *http.Client, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) NexNews/0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errURL
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// feedLinksFromHTML extracts <link rel="alternate" type="application/rss+xml|atom+xml">
// hrefs (resolved absolute) mapped to their title attribute.
func feedLinksFromHTML(page []byte, base *url.URL) map[string]string {
	out := map[string]string{}
	doc, err := html.Parse(strings.NewReader(string(page)))
	if err != nil {
		return out
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Link {
			var rel, typ, href, title string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					rel = strings.ToLower(a.Val)
				case "type":
					typ = strings.ToLower(a.Val)
				case "href":
					href = a.Val
				case "title":
					title = a.Val
				}
			}
			isFeedType := strings.Contains(typ, "rss+xml") || strings.Contains(typ, "atom+xml")
			// Accept when the type matches; rel is usually "alternate" but some
			// sites omit it.
			if isFeedType && href != "" && (rel == "" || strings.Contains(rel, "alternate")) {
				if ref, err := url.Parse(href); err == nil {
					out[base.ResolveReference(ref).String()] = strings.TrimSpace(title)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// validateFeed fetches a candidate URL and confirms it parses as a feed with at
// least a title or one item. Returns the feed title when available.
func validateFeed(ctx context.Context, client *http.Client, u string) (string, bool) {
	data, err := httpGet(ctx, client, u)
	if err != nil {
		return "", false
	}
	// Cheap guard: must look like XML, not an HTML error page.
	head := strings.TrimSpace(string(data[:min(len(data), 200)]))
	low := strings.ToLower(head)
	if !strings.Contains(low, "<?xml") && !strings.Contains(low, "<rss") &&
		!strings.Contains(low, "<feed") && !strings.Contains(low, "<rdf") {
		return "", false
	}
	items, title, err := parseFeed(data)
	if err != nil {
		return "", false
	}
	if title == "" && len(items) == 0 {
		return "", false
	}
	return title, true
}
