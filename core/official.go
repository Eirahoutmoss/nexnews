package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type officialPublication struct {
	Country string `json:"country"`
	Flag    string `json:"flag"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Link    string `json:"link"`
	Date    string `json:"date"`
	Feed    bool   `json:"feed"`
}

type officialBoard struct {
	Updated      string                `json:"updated"`
	Country      string                `json:"country"`
	Flag         string                `json:"flag"`
	Publications []officialPublication `json:"publications"`
	Portals      []officialPublication `json:"portals"`
	Error        string                `json:"error,omitempty"`
}

type officialSource struct {
	Name   string
	Type   string
	URL    string
	Portal string
}

type officialContext struct {
	Country string
	Flag    string
	Sources []officialSource
}

var officialCache struct {
	sync.Mutex
	at    time.Time
	board []byte
}

func officialContextForLang(lang string) officialContext {
	l := strings.ToLower(strings.TrimSpace(lang))
	switch {
	case strings.HasPrefix(l, "tr"):
		return officialContext{"Türkiye", "🇹🇷", []officialSource{{"Resmî Gazete", "Resmî Gazete", "https://resmigazete.gov.tr/", "https://resmigazete.gov.tr/"}}}
	case strings.HasPrefix(l, "fr"):
		return officialContext{"France", "🇫🇷", []officialSource{{"Journal officiel", "Journal officiel", "https://www.legifrance.gouv.fr/jorf/jo", "https://www.legifrance.gouv.fr/jorf/jo"}}}
	case strings.HasPrefix(l, "de"):
		return officialContext{"Deutschland", "🇩🇪", []officialSource{{"Bundesgesetzblatt", "Bundesgesetzblatt", "https://www.recht.bund.de/", "https://www.recht.bund.de/"}, {"Bundesanzeiger", "Bundesanzeiger", "https://www.bundesanzeiger.de/", "https://www.bundesanzeiger.de/"}}}
	case strings.HasPrefix(l, "es"):
		return officialContext{"España", "🇪🇸", []officialSource{{"Boletín Oficial del Estado", "BOE", "https://www.boe.es/", "https://www.boe.es/"}}}
	case strings.HasPrefix(l, "it"):
		return officialContext{"Italia", "🇮🇹", []officialSource{{"Gazzetta Ufficiale", "Gazzetta Ufficiale", "https://www.gazzettaufficiale.it/", "https://www.gazzettaufficiale.it/"}}}
	case strings.HasPrefix(l, "pt"):
		return officialContext{"Portugal", "🇵🇹", []officialSource{{"Diário da República", "Diário da República", "https://diariodarepublica.pt/", "https://diariodarepublica.pt/"}}}
	case strings.HasPrefix(l, "en-gb"), strings.HasPrefix(l, "gb"):
		return officialContext{"United Kingdom", "🇬🇧", []officialSource{{"The Gazette", "Official notices", "https://www.thegazette.co.uk/", "https://www.thegazette.co.uk/"}, {"UK Legislation", "Legislation", "https://www.legislation.gov.uk/", "https://www.legislation.gov.uk/"}}}
	case strings.HasPrefix(l, "en-ca"), strings.HasPrefix(l, "fr-ca"), strings.HasPrefix(l, "ca"):
		return officialContext{"Canada", "🇨🇦", []officialSource{{"Canada Gazette Part I", "Official gazette", "https://www.gazette.gc.ca/rss/p1-eng.xml", "https://gazette.gc.ca/accueil-home-eng.html"}, {"Canada Gazette Part II", "Official regulations", "https://www.gazette.gc.ca/rss/p2-eng.xml", "https://gazette.gc.ca/accueil-home-eng.html"}, {"Canada Gazette Part III", "Acts of Parliament", "https://www.gazette.gc.ca/rss/en-ls-eng.xml", "https://gazette.gc.ca/accueil-home-eng.html"}}}
	case strings.HasPrefix(l, "en-au"), strings.HasPrefix(l, "au"):
		return officialContext{"Australia", "🇦🇺", []officialSource{{"Commonwealth Government Gazette", "Government gazette", "https://www.legislation.gov.au/", "https://www.legislation.gov.au/"}}}
	default:
		return officialContext{"United States", "🇺🇸", []officialSource{{"Federal Register", "Federal Register", "https://www.federalregister.gov/api/v1/documents.rss?per_page=20", "https://www.federalregister.gov/"}}}
	}
}

func officialPublicationsJSON(ctx context.Context) ([]byte, error) {
	officialCache.Lock()
	if len(officialCache.board) > 0 && time.Since(officialCache.at) < 5*time.Minute {
		b := append([]byte(nil), officialCache.board...)
		officialCache.Unlock()
		return b, nil
	}
	officialCache.Unlock()

	lang := systemLang()
	oc := officialContextForLang(lang)
	client := &http.Client{Timeout: 9 * time.Second}
	board := officialBoard{Updated: time.Now().UTC().Format(time.RFC3339), Country: oc.Country, Flag: oc.Flag}

	for _, src := range oc.Sources {
		items, ok := officialFeed(ctx, client, src)
		if ok {
			for _, it := range items {
				it.Country, it.Flag, it.Name, it.Type = oc.Country, oc.Flag, src.Name, src.Type
				board.Publications = append(board.Publications, it)
			}
		} else {
			board.Portals = append(board.Portals, officialPublication{Country: oc.Country, Flag: oc.Flag, Name: src.Name, Type: src.Type, Link: src.Portal})
		}
	}
	if len(board.Publications) > 8 {
		board.Publications = board.Publications[:8]
	}
	if len(board.Publications) == 0 && len(board.Portals) == 0 {
		board.Error = "Resmî yayın verisi alınamadı."
	}

	b, err := json.Marshal(board)
	if err != nil {
		return nil, err
	}
	officialCache.Lock()
	officialCache.at = time.Now()
	officialCache.board = append([]byte(nil), b...)
	officialCache.Unlock()
	return b, nil
}

func officialFeed(ctx context.Context, client *http.Client, src officialSource) ([]officialPublication, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("User-Agent", "NexNews/1.0 (+official-publication-reader)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, false
	}
	items, _, err := parseFeed(data)
	if err != nil || len(items) == 0 {
		if src.Name == "Resmî Gazete" {
			return officialGazetteHTML(data, src.Portal), true
		}
		return nil, false
	}
	out := make([]officialPublication, 0, minInt(len(items), 6))
	for _, it := range items[:minInt(len(items), 6)] {
		d := ""
		if it.HasDate {
			d = it.Published.Format(time.RFC3339)
		}
		out = append(out, officialPublication{Title: it.Title, Link: it.Link, Date: d, Feed: true})
	}
	return out, true
}

func officialGazetteHTML(data []byte, portal string) []officialPublication {
	s := string(data)
	re := regexp.MustCompile(`(?is)<title[^>]*>\s*([^<]{8,180})\s*</title>`)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return nil
	}
	title := strings.Join(strings.Fields(strings.TrimSpace(m[1])), " ")
	return []officialPublication{{Title: title, Link: portal, Feed: false}}
}

func (p officialPublication) String() string { return fmt.Sprintf("%s: %s", p.Name, p.Title) }
