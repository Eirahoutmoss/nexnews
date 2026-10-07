package core

// Language- and source-independent RSS / Atom / RDF(RSS 1.0) parser built only
// on the standard library (plus x/net charset for non-UTF-8 feeds). Keeping the
// parser dependency-light is deliberate: it keeps NexDesk-style single-.exe
// builds simple and sidesteps the heavier feed libraries.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

// Item is one normalized headline, regardless of the source feed dialect.
type Item struct {
	ID        string `json:"id,omitempty"`
	Title     string
	Link      string
	Canonical string `json:"canonical,omitempty"`
	Source    string
	Category  string
	Published time.Time
	HasDate   bool
	Breaking  bool
	Summary   string // short plain-text summary from the feed (fallback reader text)
	Image     string `json:"image,omitempty"` // thumbnail from media:* / enclosure / first <img>
}

// --- raw XML shapes -------------------------------------------------------
//
// encoding/xml matches on the element's *local* name when the struct tag has no
// namespace, so a single set of tags captures RSS 2.0 (<item>), RSS 1.0/RDF
// (<item> directly under the root, dc:date → local "date") and Atom (<entry>).

type xmlFeed struct {
	// RSS 2.0: <rss><channel>…</channel></rss>
	Channel xmlChannel `xml:"channel"`
	// RSS 1.0 / RDF: <item> elements are siblings of <channel> under the root.
	RDFItems []xmlItem `xml:"item"`
	// Atom: <feed><title/> <entry/>…</feed>
	AtomTitle   string     `xml:"title"`
	AtomEntries []xmlEntry `xml:"entry"`
}

type xmlChannel struct {
	Title string    `xml:"title"`
	Items []xmlItem `xml:"item"`
}

type xmlItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	DCDate      string `xml:"date"` // dc:date (RSS 1.0 / some RSS 2.0)
	Description string `xml:"description"`
	Encoded     string `xml:"encoded"` // content:encoded (local name "encoded")
	mediaFields
}

// mediaFields captures the usual places feeds put a picture: media:thumbnail,
// media:content (optionally inside media:group), <enclosure> and <image>.
type mediaFields struct {
	Thumbs    []mediaRef `xml:"thumbnail"`
	Media     []mediaRef `xml:"content"`
	Group     mediaGroup `xml:"group"`
	Enclosure []mediaRef `xml:"enclosure"`
	ImageTag  string     `xml:"image"`
}

type mediaGroup struct {
	Thumbs []mediaRef `xml:"thumbnail"`
	Media  []mediaRef `xml:"content"`
}

type mediaRef struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Medium string `xml:"medium,attr"`
}

var imgSrcRe = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)

func isImageRef(m mediaRef) bool {
	t, md := strings.ToLower(m.Type), strings.ToLower(m.Medium)
	if md == "image" || strings.HasPrefix(t, "image/") {
		return true
	}
	if md != "" || t != "" {
		return false
	}
	low := strings.ToLower(m.URL)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif"} {
		if strings.Contains(low, ext) {
			return true
		}
	}
	return false
}

// pickImage returns the first usable http(s) image URL from the media fields,
// falling back to the first <img> in the HTML description.
func (m mediaFields) pickImage(htmlDesc ...string) string {
	try := func(refs []mediaRef, any bool) string {
		for _, r := range refs {
			if (any || isImageRef(r)) && safeWebURL(r.URL) != "" {
				return safeWebURL(r.URL)
			}
		}
		return ""
	}
	for _, u := range []string{
		try(m.Thumbs, true), try(m.Group.Thumbs, true),
		try(m.Media, false), try(m.Group.Media, false), try(m.Enclosure, false),
	} {
		if u != "" {
			return u
		}
	}
	if u := safeWebURL(strings.TrimSpace(m.ImageTag)); u != "" {
		return u
	}
	for _, d := range htmlDesc {
		if mm := imgSrcRe.FindStringSubmatch(d); len(mm) == 2 {
			if u := safeWebURL(html.UnescapeString(mm[1])); u != "" {
				return u
			}
		}
	}
	return ""
}

type xmlEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Updated   string     `xml:"updated"`
	Published string     `xml:"published"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Thumbs    []mediaRef `xml:"thumbnail"`
	Group     mediaGroup `xml:"group"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// parseFeed turns raw feed bytes into normalized items. It never returns an
// error for an empty-but-valid document; callers treat an empty slice as "no
// headlines" rather than a hard failure.
func parseFeed(data []byte) ([]Item, string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	// Makes ISO-8859-*, windows-125x and other declared encodings work, which is
	// what "dil bağımsız" really requires in the wild.
	dec.CharsetReader = charset.NewReaderLabel
	dec.Strict = false

	var f xmlFeed
	if err := dec.Decode(&f); err != nil {
		return nil, "", err
	}

	feedTitle := strings.TrimSpace(clean(f.Channel.Title))
	if feedTitle == "" {
		feedTitle = strings.TrimSpace(clean(f.AtomTitle))
	}

	var items []Item

	add := func(title, link, d1, d2, desc, image string) {
		title = clean(title)
		link = strings.TrimSpace(link)
		canonical := canonicalURL(link)
		if title == "" || canonical == "" {
			return
		}
		ts, ok := parseTime(d1)
		if !ok {
			ts, ok = parseTime(d2)
		}
		items = append(items, Item{
			ID:        articleID(title, canonical),
			Title:     title,
			Link:      link,
			Canonical: canonical,
			Published: ts,
			HasDate:   ok,
			Summary:   summarize(desc),
			Image:     image,
		})
	}

	for _, it := range f.Channel.Items {
		add(it.Title, it.Link, it.PubDate, it.DCDate, pick(it.Description, it.Encoded), it.pickImage(it.Description, it.Encoded))
	}
	for _, it := range f.RDFItems {
		add(it.Title, it.Link, it.PubDate, it.DCDate, pick(it.Description, it.Encoded), it.pickImage(it.Description, it.Encoded))
	}
	for _, e := range f.AtomEntries {
		img := mediaFields{Thumbs: e.Thumbs, Group: e.Group}.pickImage(e.Summary, e.Content)
		if img == "" {
			for _, l := range e.Links {
				if strings.EqualFold(l.Rel, "enclosure") && strings.HasPrefix(strings.ToLower(l.Type), "image/") {
					img = safeWebURL(l.Href)
					break
				}
			}
		}
		add(e.Title, pickAtomLink(e.Links), e.Published, e.Updated, pick(e.Summary, e.Content), img)
	}

	return items, feedTitle, nil
}

func pick(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// summarize turns a feed description/summary (often HTML) into a short plain
// paragraph used as a fallback when the full article can't be extracted.
func summarize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, "<&") {
		s = stripTags(s)
		s = html.UnescapeString(s)
	}
	s = strings.Join(strings.Fields(s), " ")
	const max = 500
	if len([]rune(s)) > max {
		r := []rune(s)[:max]
		// trim back to the last space for a clean cut
		cut := string(r)
		if i := strings.LastIndex(cut, " "); i > max-80 {
			cut = cut[:i]
		}
		s = cut + "…"
	}
	return s
}

// pickAtomLink prefers rel="alternate" (the human-readable article), else the
// first link that is not a self/enclosure reference, else the first href.
func pickAtomLink(links []atomLink) string {
	for _, l := range links {
		if strings.EqualFold(l.Rel, "alternate") && l.Href != "" {
			return l.Href
		}
	}
	for _, l := range links {
		if l.Rel == "" && l.Href != "" {
			return l.Href
		}
	}
	for _, l := range links {
		if l.Href != "" && !strings.EqualFold(l.Rel, "self") {
			return l.Href
		}
	}
	return ""
}

// clean strips stray markup/entities and collapses whitespace in a title.
func clean(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Some feeds wrap titles in CDATA with inline HTML; decode entities and drop
	// any tags so the headline reads cleanly in the list.
	if strings.ContainsAny(s, "<&") {
		s = stripTags(s)
		s = html.UnescapeString(s)
	}
	return strings.Join(strings.Fields(s), " ")
}

func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func canonicalURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if h := u.Hostname(); h != "" {
		u.Host = h
		if p := u.Port(); p != "" && !((u.Scheme == "http" && p == "80") || (u.Scheme == "https" && p == "443")) {
			u.Host += ":" + p
		}
	}
	q := u.Query()
	for key := range q {
		lk := strings.ToLower(key)
		if strings.HasPrefix(lk, "utm_") || lk == "fbclid" || lk == "gclid" || lk == "mc_cid" || lk == "mc_eid" || lk == "ref" || lk == "ref_" || lk == "source" {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

func articleID(title, canonical string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(title)) + "\x00" + canonical))
	return hex.EncodeToString(h[:16])
}

// dateLayouts covers the formats feeds actually use across languages/servers.
var dateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC3339,
	time.RFC3339Nano,
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"02 Jan 2006 15:04:05 -0700",
}

func parseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
