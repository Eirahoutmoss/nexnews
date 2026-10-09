package core

// Yerel lig puan durumları (Spor sekmesi).
//
// NexNews'in işi gündemden haberdar etmek; ayrıntılı istatistik değil. Bu yüzden
// yalnızca Türkiye'deki iki ana ligin puan tablosu gösterilir ve her tablo
// kaynağına bağlantı verir.
//
//   Futbol (Süper Lig): ESPN puan durumu (güncel), olmazsa İngilizce Wikipedia.
//   Basketbol (BSL):   İngilizce Wikipedia sezon sayfasındaki lig tablosu.
//
// Wikipedia tabloları gönüllülerce güncellenir; sayfadaki "Updated to match(es)
// played on ..." notu ayrıca okunup kullanıcıya "şu tarihe kadar" diye gösterilir.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type standingRow struct {
	Pos     int    `json:"pos"`
	Team    string `json:"team"`
	Played  int    `json:"played"`
	Won     int    `json:"won"`
	Drawn   int    `json:"drawn"`
	Lost    int    `json:"lost"`
	For     int    `json:"for"`
	Against int    `json:"against"`
	Diff    int    `json:"diff"`
	Points  int    `json:"points"`
	Note    string `json:"note,omitempty"`
}

type standingTable struct {
	Key       string        `json:"key"`   // futbol | basketbol
	Title     string        `json:"title"` // Süper Lig
	Season    string        `json:"season"`
	Source    string        `json:"source"` // ESPN | Wikipedia
	SourceURL string        `json:"sourceUrl"`
	AsOf      string        `json:"asOf,omitempty"` // Wikipedia "Updated to ..." tarihi
	HasDraws  bool          `json:"hasDraws"`
	Rows      []standingRow `json:"rows"`
	Error     string        `json:"error,omitempty"`
}

type standingsBoard struct {
	OK        bool            `json:"ok"`
	Updated   int64           `json:"updated"`
	Tables    []standingTable `json:"tables"`
	Favorites []string        `json:"favorites,omitempty"`
}

var (
	standingsMu    sync.Mutex
	standingsCache standingsBoard
	standingsAt    time.Time
)

const standingsTTL = 30 * time.Minute

var standingsClient = &http.Client{Timeout: 15 * time.Second}

const wikiUA = "NexNews/1.0 (https://github.com/Eirahoutmoss/nexnews; haber okuyucu)"

// seasonStartYear: Türkiye'de ligler ağustos-eylülde başlar.
func seasonStartYear(now time.Time) int {
	if now.Month() < time.August {
		return now.Year() - 1
	}
	return now.Year()
}

func seasonLabel(y int) string { return fmt.Sprintf("%d–%02d", y, (y+1)%100) }

func standingsJSON(ctx context.Context, cfg *Config) []byte {
	standingsMu.Lock()
	if time.Since(standingsAt) < standingsTTL && len(standingsCache.Tables) > 0 {
		out := standingsCache
		standingsMu.Unlock()
		out.Favorites = standingsFavorites(cfg)
		b, _ := json.Marshal(out)
		return b
	}
	standingsMu.Unlock()

	// Mobilde veriyi korumak için disk önbelleği (uygulama yeniden açıldığında).
	if disk, ok := loadStandingsDisk(); ok && time.Since(time.Unix(disk.Updated, 0)) < standingsTTL {
		standingsMu.Lock()
		standingsCache, standingsAt = disk, time.Unix(disk.Updated, 0)
		standingsMu.Unlock()
		disk.Favorites = standingsFavorites(cfg)
		b, _ := json.Marshal(disk)
		return b
	}

	y := seasonStartYear(time.Now())
	var wg sync.WaitGroup
	tables := make([]standingTable, 2)
	wg.Add(2)
	go func() { defer wg.Done(); tables[0] = footballStandings(ctx, y) }()
	go func() { defer wg.Done(); tables[1] = basketballStandings(ctx, y) }()
	wg.Wait()

	board := standingsBoard{OK: true, Updated: time.Now().Unix(), Tables: tables}
	good := 0
	for _, t := range tables {
		if len(t.Rows) > 0 {
			good++
		}
	}
	if good == 0 {
		board.OK = false
		// Ağ yoksa eski tabloyu göstermek boş ekrandan iyidir.
		if disk, ok := loadStandingsDisk(); ok && len(disk.Tables) > 0 {
			disk.Favorites = standingsFavorites(cfg)
			b, _ := json.Marshal(disk)
			return b
		}
	} else {
		standingsMu.Lock()
		standingsCache, standingsAt = board, time.Now()
		standingsMu.Unlock()
		saveStandingsDisk(board)
	}
	board.Favorites = standingsFavorites(cfg)
	b, _ := json.Marshal(board)
	return b
}

func standingsFavorites(cfg *Config) []string {
	if cfg == nil {
		return nil
	}
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	return append([]string(nil), cfg.SportsTeams...)
}

func standingsDiskPath() string { return filepath.Join(appDataDir(), "standings.json") }

func loadStandingsDisk() (standingsBoard, bool) {
	var b standingsBoard
	data, err := os.ReadFile(standingsDiskPath())
	if err != nil || json.Unmarshal(data, &b) != nil || len(b.Tables) == 0 {
		return b, false
	}
	return b, true
}

func saveStandingsDisk(b standingsBoard) {
	b.Favorites = nil
	if data, err := json.Marshal(b); err == nil {
		_ = atomicWriteFile(standingsDiskPath(), data)
	}
}

// ---------------------------------------------------------------- futbol

func footballStandings(ctx context.Context, y int) standingTable {
	t := standingTable{Key: "futbol", Title: "Süper Lig", Season: seasonLabel(y), HasDraws: true}
	if rows, season, err := espnStandings(ctx, "https://site.api.espn.com/apis/v2/sports/soccer/tur.1/standings"); err == nil && len(rows) > 0 {
		t.Rows, t.Source, t.SourceURL = rows, "ESPN", "https://www.espn.com/soccer/standings/_/league/tur.1"
		if season != "" {
			t.Season = season
		}
		return t
	} else if err != nil {
		t.Error = err.Error()
	}
	w := wikiStandings(ctx, y, "Süper Lig")
	if len(w.Rows) > 0 {
		w.Key, w.Title, w.HasDraws = t.Key, t.Title, true
		return w
	}
	if t.Error == "" {
		t.Error = w.Error
	}
	return t
}

func espnStandings(ctx context.Context, endpoint string) ([]standingRow, string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("User-Agent", "NexNews/1.0")
	resp, err := standingsClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, "", err
	}
	return parseESPNStandings(body)
}

func parseESPNStandings(body []byte) ([]standingRow, string, error) {
	var raw struct {
		Season struct {
			Year int `json:"year"`
		} `json:"season"`
		Children []struct {
			Standings struct {
				Entries []struct {
					Team struct {
						DisplayName string `json:"displayName"`
					} `json:"team"`
					Note struct {
						Description string `json:"description"`
					} `json:"note"`
					Stats []struct {
						Name  string   `json:"name"`
						Value *float64 `json:"value"`
					} `json:"stats"`
				} `json:"entries"`
			} `json:"standings"`
		} `json:"children"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, "", err
	}
	if len(raw.Children) == 0 {
		return nil, "", fmt.Errorf("puan durumu yok")
	}
	var rows []standingRow
	for _, e := range raw.Children[0].Standings.Entries {
		st := map[string]int{}
		for _, s := range e.Stats {
			if s.Value != nil {
				st[s.Name] = int(*s.Value)
			}
		}
		rows = append(rows, standingRow{
			Pos: st["rank"], Team: turkishTeamName(e.Team.DisplayName),
			Played: st["gamesPlayed"], Won: st["wins"], Drawn: st["ties"], Lost: st["losses"],
			For: st["pointsFor"], Against: st["pointsAgainst"], Diff: st["pointDifferential"],
			Points: st["points"], Note: noteTR(e.Note.Description),
		})
	}
	sortStandings(rows)
	season := ""
	if raw.Season.Year > 0 {
		// ESPN "year" sezonun başladığı yıldır (2026 = 2026-27).
		season = seasonLabel(raw.Season.Year)
	}
	return rows, season, nil
}

func sortStandings(rows []standingRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Pos > 0 && b.Pos > 0 && a.Pos != b.Pos {
			return a.Pos < b.Pos
		}
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		return a.Diff > b.Diff
	})
	for i := range rows {
		if rows[i].Pos == 0 {
			rows[i].Pos = i + 1
		}
	}
}

// ESPN takım adlarını Türkçe karakterlerle yazar (Fenerbahce -> Fenerbahçe).
var trTeamWords = map[string]string{
	"Fenerbahce": "Fenerbahçe", "Besiktas": "Beşiktaş", "Basaksehir": "Başakşehir",
	"Goztepe": "Göztepe", "Kasimpasa": "Kasımpaşa", "Eyupspor": "Eyüpspor",
	"Genclerbirligi": "Gençlerbirliği", "Caykur": "Çaykur", "Karagumruk": "Karagümrük",
	"Ankaragucu": "Ankaragücü", "Istanbulspor": "İstanbulspor", "Istanbul": "İstanbul",
	"Corum": "Çorum", "Igdir": "Iğdır", "Bandirmaspor": "Bandırmaspor",
	"Umraniyespor": "Ümraniyespor", "Erzurumspor": "Erzurumspor", "Keciorengucu": "Keçiörengücü",
	"Manisa": "Manisa", "Bodrumspor": "Bodrumspor", "Sivasspor": "Sivasspor",
}

func turkishTeamName(s string) string {
	parts := strings.Fields(s)
	for i, p := range parts {
		if r, ok := trTeamWords[p]; ok {
			parts[i] = r
		}
	}
	return strings.Join(parts, " ")
}

func noteTR(s string) string {
	l := strings.ToLower(s)
	switch {
	case l == "":
		return ""
	case strings.Contains(l, "champions league"):
		return "Şampiyonlar Ligi"
	case strings.Contains(l, "europa league"):
		return "Avrupa Ligi"
	case strings.Contains(l, "conference league"):
		return "Konferans Ligi"
	case strings.Contains(l, "relegation"):
		return "Küme düşme"
	case strings.Contains(l, "playoff") || strings.Contains(l, "play-off"):
		return "Play-off"
	}
	return ""
}

// ------------------------------------------------------------- basketbol

func basketballStandings(ctx context.Context, y int) standingTable {
	t := wikiStandings(ctx, y, "Basketbol Süper Ligi")
	if len(t.Rows) == 0 && t.Error != "" {
		// Sezon sayfası henüz açılmamışsa geçen sezonun son tablosu.
		if prev := wikiStandings(ctx, y-1, "Basketbol Süper Ligi"); len(prev.Rows) > 0 {
			t = prev
		}
	}
	t.Key, t.Title, t.HasDraws = "basketbol", "Basketbol Süper Ligi", false
	return t
}

// --------------------------------------------------------------- Wikipedia

func wikiStandings(ctx context.Context, y int, league string) standingTable {
	title := seasonLabel(y) + " " + league
	page := strings.ReplaceAll(title, " ", "_")
	t := standingTable{Season: seasonLabel(y), Source: "Wikipedia", SourceURL: "https://en.wikipedia.org/wiki/" + url.PathEscape(page)}
	endpoint := "https://en.wikipedia.org/w/index.php?action=render&title=" + url.QueryEscape(page)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("User-Agent", wikiUA)
	resp, err := standingsClient.Do(req)
	if err != nil {
		t.Error = err.Error()
		return t
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return t
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
	if err != nil {
		t.Error = err.Error()
		return t
	}
	rows, asOf, err := parseWikiStandings(string(body))
	if err != nil {
		t.Error = err.Error()
		return t
	}
	t.Rows, t.AsOf = rows, asOf
	return t
}

var (
	reWikiRef    = regexp.MustCompile(`\[[^\]]*\]`)
	reWikiMarker = regexp.MustCompile(`\s*\((?:[A-Z]{1,2}|[A-Z], ?[A-Z])\)\s*$`)
	reWikiAsOf   = regexp.MustCompile(`Updated to (?:match\(es\)|games?|matches) played on ([0-9]{1,2} [A-Z][a-z]+ [0-9]{4})`)
)

// parseWikiStandings Module:Sports table çıktısındaki ilk lig tablosunu okur
// (başlıkta Pld ve Pts olan ilk wikitable). rowspan'lı hücreler taşınır.
func parseWikiStandings(doc string) ([]standingRow, string, error) {
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return nil, "", err
	}
	asOf := ""
	var tables []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Table {
			tables = append(tables, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)

	for _, tb := range tables {
		trs := collectRows(tb)
		if len(trs) < 3 {
			continue
		}
		head := cellTexts(trs[0])
		col := map[string]int{}
		for i, h := range head {
			key := standingsHeaderKey(h)
			if key != "" {
				if _, dup := col[key]; !dup {
					col[key] = i
				}
			}
		}
		if _, ok := col["pld"]; !ok {
			continue
		}
		if _, ok := col["pts"]; !ok {
			continue
		}
		if _, ok := col["team"]; !ok {
			continue
		}
		rows := parseWikiRows(trs[1:], len(head), col)
		if len(rows) < 4 {
			continue
		}
		// Not tablonun hemen altında; tüm metinde ilk eşleşme yeterli.
		if m := reWikiAsOf.FindStringSubmatch(nodeText(root)); m != nil {
			asOf = trDate(m[1])
		}
		return rows, asOf, nil
	}
	return nil, "", fmt.Errorf("puan tablosu bulunamadı")
}

func collectRows(tb *html.Node) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Table && n != tb {
			return // iç içe tablo
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Tr {
			out = append(out, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(tb)
	return out
}

type wikiCell struct {
	text    string
	rowspan int
	colspan int
}

func rowCells(tr *html.Node) []wikiCell {
	var out []wikiCell
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || (c.DataAtom != atom.Td && c.DataAtom != atom.Th) {
			continue
		}
		wc := wikiCell{text: cleanCellText(c), rowspan: 1, colspan: 1}
		for _, a := range c.Attr {
			if v, err := strconv.Atoi(strings.TrimSpace(a.Val)); err == nil && v > 1 {
				if a.Key == "rowspan" {
					wc.rowspan = v
				}
				if a.Key == "colspan" {
					wc.colspan = v
				}
			}
		}
		out = append(out, wc)
	}
	return out
}

func cellTexts(tr *html.Node) []string {
	var out []string
	for _, c := range rowCells(tr) {
		for i := 0; i < c.colspan; i++ {
			out = append(out, c.text)
		}
	}
	return out
}

func parseWikiRows(trs []*html.Node, width int, col map[string]int) []standingRow {
	type carry struct {
		text string
		left int
	}
	pending := map[int]carry{}
	var rows []standingRow
	for _, tr := range trs {
		cells := rowCells(tr)
		grid := make([]string, 0, width)
		ci := 0
		for len(grid) < width && (ci < len(cells) || pending[len(grid)].left > 0) {
			pos := len(grid)
			if p, ok := pending[pos]; ok && p.left > 0 {
				grid = append(grid, p.text)
				p.left--
				pending[pos] = p
				continue
			}
			if ci >= len(cells) {
				break
			}
			c := cells[ci]
			ci++
			for k := 0; k < c.colspan; k++ {
				if c.rowspan > 1 {
					pending[len(grid)] = carry{text: c.text, left: c.rowspan - 1}
				}
				grid = append(grid, c.text)
			}
		}
		get := func(key string) string {
			i, ok := col[key]
			if !ok || i >= len(grid) {
				return ""
			}
			return grid[i]
		}
		team := get("team")
		pld, ok := atoiLoose(get("pld"))
		if team == "" || !ok {
			continue
		}
		r := standingRow{Team: cleanTeam(team), Played: pld}
		r.Pos, _ = atoiLoose(get("pos"))
		r.Won, _ = atoiLoose(get("w"))
		r.Drawn, _ = atoiLoose(get("d"))
		r.Lost, _ = atoiLoose(get("l"))
		r.For, _ = atoiLoose(get("for"))
		r.Against, _ = atoiLoose(get("against"))
		if d, ok := atoiLoose(get("diff")); ok {
			r.Diff = d
		} else {
			r.Diff = r.For - r.Against
		}
		r.Points, _ = atoiLoose(get("pts"))
		r.Note = noteTR(get("note"))
		rows = append(rows, r)
	}
	sortStandings(rows)
	return rows
}

func standingsHeaderKey(h string) string {
	h = strings.TrimSpace(h)
	low := strings.ToLower(h)
	switch {
	case low == "pos" || low == "rank" || low == "#":
		return "pos"
	case strings.HasPrefix(low, "team") || strings.HasPrefix(low, "club"):
		return "team"
	case low == "pld" || low == "gp" || low == "mp":
		return "pld"
	case low == "w":
		return "w"
	case low == "d":
		return "d"
	case low == "l":
		return "l"
	case low == "pf" || low == "gf" || low == "f":
		return "for"
	case low == "pa" || low == "ga" || low == "a":
		return "against"
	case low == "pd" || low == "gd":
		return "diff"
	case low == "pts":
		return "pts"
	case strings.HasPrefix(low, "qualification") || strings.HasPrefix(low, "relegation"):
		return "note"
	}
	return ""
}

func cleanCellText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Gezinme kutusu (v·t·e), dipnotlar ve gizli sıralama anahtarları atlanır.
			for _, a := range n.Attr {
				if a.Key == "class" && (strings.Contains(a.Val, "navbar") || strings.Contains(a.Val, "reference") || strings.Contains(a.Val, "sortkey")) {
					return
				}
				if a.Key == "style" && strings.Contains(strings.ReplaceAll(a.Val, " ", ""), "display:none") {
					return
				}
			}
			if n.DataAtom == atom.Sup || n.DataAtom == atom.Style || n.DataAtom == atom.Script {
				return
			}
			if n.DataAtom == atom.Br {
				b.WriteByte(' ')
			}
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(strings.ReplaceAll(b.String(), " ", " ")), " ")
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(strings.ReplaceAll(b.String(), " ", " ")), " ")
}

func cleanTeam(s string) string {
	s = reWikiRef.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	for {
		n := reWikiMarker.ReplaceAllString(s, "")
		if n == s {
			break
		}
		s = strings.TrimSpace(n)
	}
	return s
}

func atoiLoose(s string) (int, bool) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "−", "-") // Unicode eksi işareti
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.TrimPrefix(s, "+")
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0, false
	}
	v, err := strconv.Atoi(s)
	return v, err == nil
}

var trMonths = map[string]string{
	"January": "Ocak", "February": "Şubat", "March": "Mart", "April": "Nisan", "May": "Mayıs", "June": "Haziran",
	"July": "Temmuz", "August": "Ağustos", "September": "Eylül", "October": "Ekim", "November": "Kasım", "December": "Aralık",
}

func trDate(s string) string {
	f := strings.Fields(s)
	if len(f) == 3 {
		if m, ok := trMonths[f[1]]; ok {
			return f[0] + " " + m + " " + f[2]
		}
	}
	return s
}
