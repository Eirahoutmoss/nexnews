package core

// In-app reader: fetch an article and return a cleaned, readable version. Most
// news sites block iframing (X-Frame-Options / frame-ancestors), so embedding
// the live page in a pane is unreliable — instead we extract the main content
// server-side with a small readability-style heuristic and render that. The
// user can always jump to the full page in their browser from the reader.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type article struct {
	Title string `json:"title"`
	HTML  string `json:"html"`
	URL   string `json:"url"`
	Err   string `json:"err,omitempty"`
}

func fetchArticleJSON(ctx context.Context, url string) []byte {
	a := article{URL: url}
	if safeWebURL(url) == "" {
		a.Err = "Geçersiz bağlantı."
		b, _ := json.Marshal(a)
		return b
	}
	base, _ := neturl.Parse(url)

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		a.Err = "Geçersiz bağlantı."
		b, _ := json.Marshal(a)
		return b
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) NexNews/0.1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		a.Err = "Sayfaya ulaşılamadı."
		b, _ := json.Marshal(a)
		return b
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	if err != nil {
		a.Err = "Sayfa okunamadı."
		b, _ := json.Marshal(a)
		return b
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		a.Err = "Sayfa çözümlenemedi."
		b, _ := json.Marshal(a)
		return b
	}

	a.Title = extractTitle(doc)
	a.HTML = extractContentBase(doc, base)

	// Many sites (NYT, WSJ…) gate the full text behind JS/paywalls, so server-side
	// extraction returns almost nothing or a "enable JS / disable ad blocker"
	// stub. Detect that and fall back to a clear message + the feed summary
	// (shown by the UI) instead of a near-empty reader.
	text := strings.TrimSpace(stripTags(a.HTML))
	if len([]rune(text)) < 220 || isBlockedStub(text) {
		a.HTML = ""
		a.Err = "Bu site tam metni uygulama içinde vermiyor (JS/abonelik gerekiyor)."
	}
	b, _ := json.Marshal(a)
	return b
}

// isBlockedStub spots the tell-tale phrases of JS-gated / anti-bot pages.
func isBlockedStub(text string) bool {
	low := strings.ToLower(text)
	for _, p := range []string{
		"enable js", "enable javascript", "please enable",
		"ad blocker", "adblock", "disable any ad",
		"are you a robot", "verify you are human", "access denied",
		"subscribe to continue", "subscription required", "abone",
	} {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

func extractTitle(doc *html.Node) string {
	// Prefer <h1>, fall back to <title>.
	if h1 := firstOfType(doc, atom.H1); h1 != nil {
		if t := strings.TrimSpace(textOf(h1)); t != "" {
			return t
		}
	}
	if t := firstOfType(doc, atom.Title); t != nil {
		return strings.TrimSpace(textOf(t))
	}
	return ""
}

// extractContent finds the densest block of paragraph text and returns a
// sanitized HTML fragment of it.
func extractContent(doc *html.Node) string { return extractContentBase(doc, nil) }

func extractContentBase(doc *html.Node, base *neturl.URL) string {
	// Prefer a semantic <article> if present and substantial.
	if art := firstOfType(doc, atom.Article); art != nil {
		if frag := sanitizeBase(art, base); textLen(frag) > 400 {
			return frag
		}
	}

	var best *html.Node
	bestScore := 0
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.DataAtom {
			case atom.Div, atom.Section, atom.Main, atom.Article:
				if sc := paragraphScore(n); sc > bestScore {
					bestScore = sc
					best = n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if best == nil {
		if body := firstOfType(doc, atom.Body); body != nil {
			best = body
		} else {
			return ""
		}
	}
	return sanitizeBase(best, base)
}

// paragraphScore sums the text length of direct-ish <p> descendants, which
// favors the real article body over navigation/sidebar containers.
func paragraphScore(n *html.Node) int {
	score := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.DataAtom == atom.P {
			l := len(strings.TrimSpace(textOf(c)))
			if l > 40 {
				score += l
			}
		} else if c.Type == html.ElementNode {
			// shallow look one level down for <p> wrapped in <div>
			for g := c.FirstChild; g != nil; g = g.NextSibling {
				if g.Type == html.ElementNode && g.DataAtom == atom.P {
					l := len(strings.TrimSpace(textOf(g)))
					if l > 40 {
						score += l
					}
				}
			}
		}
	}
	return score
}

// sanitize renders a node to HTML keeping only readable, safe elements.
var keepTags = map[atom.Atom]bool{
	atom.P: true, atom.Br: true, atom.H1: true, atom.H2: true, atom.H3: true,
	atom.H4: true, atom.Blockquote: true, atom.Ul: true, atom.Ol: true,
	atom.Li: true, atom.Strong: true, atom.B: true, atom.Em: true, atom.I: true,
	atom.A: true, atom.Figure: true, atom.Figcaption: true, atom.Img: true,
	atom.Pre: true, atom.Code: true,
}

var dropTags = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Nav: true,
	atom.Header: true, atom.Footer: true, atom.Aside: true, atom.Form: true,
	atom.Iframe: true, atom.Button: true, atom.Svg: true,
}

// sanitizeArticleHTML parses an HTML fragment and returns it through the same
// allow-list renderer the reader uses (exposed for tests).
func sanitizeArticleHTML(frag string) string {
	doc, err := html.Parse(strings.NewReader(frag))
	if err != nil {
		return ""
	}
	return sanitizeBase(doc, nil)
}

// resolveSafe resolves ref against base and keeps it only if the result is an
// http(s) URL. javascript:, data:, file: and friends are dropped.
func resolveSafe(base *neturl.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if base != nil {
		if u, err := neturl.Parse(ref); err == nil {
			ref = base.ResolveReference(u).String()
		}
	}
	return safeWebURL(ref)
}

func sanitize(n *html.Node) string { return sanitizeBase(n, nil) }

func sanitizeBase(n *html.Node, base *neturl.URL) string {
	var b strings.Builder
	var render func(n *html.Node)
	render = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(html.EscapeString(n.Data))
			return
		case html.ElementNode:
			if dropTags[n.DataAtom] {
				return
			}
			if keepTags[n.DataAtom] {
				tag := n.Data
				b.WriteByte('<')
				b.WriteString(tag)
				// Keep only href on <a> and src/alt on <img>.
				switch n.DataAtom {
				case atom.A:
					if v := resolveSafe(base, attrVal(n, "href")); v != "" {
						b.WriteString(` href="` + html.EscapeString(v) + `" target="_blank" rel="noopener"`)
					}
				case atom.Img:
					if v := resolveSafe(base, attrVal(n, "src")); v != "" {
						b.WriteString(` src="` + html.EscapeString(v) + `" loading="lazy"`)
					}
					if v := attrVal(n, "alt"); v != "" {
						b.WriteString(` alt="` + html.EscapeString(v) + `"`)
					}
				}
				b.WriteByte('>')
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					render(c)
				}
				if !isVoid(n.DataAtom) {
					b.WriteString("</" + tag + ">")
				}
				return
			}
			// Unknown/plain wrapper: keep children, drop the tag itself.
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				render(c)
			}
		default:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				render(c)
			}
		}
	}
	render(n)
	return b.String()
}

func isVoid(a atom.Atom) bool { return a == atom.Br || a == atom.Img }

// --- small DOM helpers ----------------------------------------------------

func firstOfType(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if r := firstOfType(c, a); r != nil {
			return r
		}
	}
	return nil
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func textLen(frag string) int { return len(strings.TrimSpace(stripTags(frag))) }

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}
