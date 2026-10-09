package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type sportsGame struct {
	ID           string `json:"id"`
	League       string `json:"league"`
	LeagueName   string `json:"leagueName"`
	Home         string `json:"home"`
	Away         string `json:"away"`
	HomeScore    string `json:"homeScore"`
	AwayScore    string `json:"awayScore"`
	Status       string `json:"status"` // prematch | in | post
	StatusDetail string `json:"statusDetail"`
	Start        int64  `json:"start"`
	Fav          bool   `json:"fav,omitempty"`
}

type sportsBoard struct {
	OK        bool         `json:"ok"`
	Updated   int64        `json:"updated"`
	Games     []sportsGame `json:"games"`
	Favorites []string     `json:"favorites,omitempty"`
	Error     string       `json:"error,omitempty"`
}

var sportsMu sync.Mutex
var sportsCache sportsBoard
var sportsAt time.Time

var sportsEndpoints = map[string]string{
	"futbol":     "https://site.api.espn.com/apis/site/v2/sports/soccer/tur.1/scoreboard",
	"premier":    "https://site.api.espn.com/apis/site/v2/sports/soccer/eng.1/scoreboard",
	"ucl":        "https://site.api.espn.com/apis/site/v2/sports/soccer/uefa.champions/scoreboard",
	"nba":        "https://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard",
	"euroleague": "https://site.api.espn.com/apis/site/v2/sports/basketball/eurleague/scoreboard",
	"nfl":        "https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard",
}

// sportsWindow: dünden önceki günden 8 gün sonrasına kadar (Türkiye saati).
// ESPN'in tarihsiz uç noktası yalnızca "bugünü" ya da ligin sıradaki gününü
// verdiği için milli maç aralarında hep aynı maçlar görünüyordu.
func sportsWindow(now time.Time) string {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		loc = time.FixedZone("TRT", 3*3600)
	}
	n := now.In(loc)
	return n.AddDate(0, 0, -2).Format("20060102") + "-" + n.AddDate(0, 0, 8).Format("20060102")
}

func espnScoreboard(ctx context.Context, key, endpoint string) ([]sportsGame, error) {
	games, err := espnScoreboardURL(ctx, key, endpoint+"?limit=300&dates="+sportsWindow(time.Now()))
	if err != nil || len(games) == 0 {
		// Tarih aralığını desteklemeyen lig: günlük uç noktaya dön.
		if g2, err2 := espnScoreboardURL(ctx, key, endpoint); err2 == nil && len(g2) > 0 {
			return g2, nil
		}
	}
	return games, err
}

func espnScoreboardURL(ctx context.Context, key, endpoint string) ([]sportsGame, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NexNews/1.0")
	resp, err := financeClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var raw struct {
		Events []struct {
			ID     string `json:"id"`
			Date   string `json:"date"`
			Name   string `json:"name"`
			Status struct {
				DisplayClock string `json:"displayClock"`
				Period       int    `json:"period"`
				Type         struct {
					Name        string `json:"name"`
					ShortDetail string `json:"shortDetail"`
					Detail      string `json:"detail"`
					State       string `json:"state"`
				} `json:"type"`
			} `json:"status"`
			Competitions []struct {
				Competitors []struct {
					Team struct {
						DisplayName string `json:"displayName"`
					} `json:"team"`
					Score    string `json:"score"`
					HomeAway string `json:"homeAway"`
				} `json:"competitors"`
			} `json:"competitions"`
		} `json:"events"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	games := make([]sportsGame, 0, len(raw.Events))
	for _, e := range raw.Events {
		if len(e.Competitions) == 0 {
			continue
		}
		var home, away, hs, as string
		for _, c := range e.Competitions[0].Competitors {
			if strings.EqualFold(c.HomeAway, "home") {
				home, hs = turkishTeamName(c.Team.DisplayName), c.Score
			} else {
				away, as = turkishTeamName(c.Team.DisplayName), c.Score
			}
		}
		start, _ := time.Parse(time.RFC3339, e.Date)
		league := key
		if start.IsZero() {
			start, _ = time.Parse("2006-01-02T15:04Z", e.Date)
		}
		st, detail := sportsStatusTR(e.Status.Type.State, e.Status.Type.Name, firstNonEmpty(e.Status.Type.ShortDetail, e.Status.Type.Detail), e.Status.DisplayClock)
		if st == "prematch" {
			hs, as = "", ""
		}
		games = append(games, sportsGame{ID: e.ID, League: league, LeagueName: sportsLeagueName(key), Home: home, Away: away, HomeScore: hs, AwayScore: as, Status: st, StatusDetail: detail, Start: start.Unix()})
	}
	return games, nil
}

func sportsLeagueName(k string) string {
	switch k {
	case "futbol":
		return "Süper Lig"
	case "premier":
		return "Premier League"
	case "ucl":
		return "Şampiyonlar Ligi"
	case "nba":
		return "NBA"
	case "euroleague":
		return "EuroLeague"
	case "nfl":
		return "NFL"
	}
	return k
}

// sportsStatusTR: ESPN durumlarını Türkçeleştirir. "pre" maçlarda ayrıntı boş
// bırakılır; arayüz başlama saatini kendisi yazar.
func sportsStatusTR(state, name, short, clock string) (string, string) {
	n := strings.ToUpper(name)
	switch {
	case strings.Contains(n, "POSTPONED"):
		return "post", "Ertelendi"
	case strings.Contains(n, "CANCELED") || strings.Contains(n, "CANCELLED"):
		return "post", "İptal"
	case strings.Contains(n, "ABANDONED") || strings.Contains(n, "SUSPENDED"):
		return "post", "Yarıda kaldı"
	}
	switch strings.ToLower(state) {
	case "pre":
		return "prematch", ""
	case "in":
		switch {
		case strings.Contains(n, "HALFTIME"):
			return "in", "Devre arası"
		case strings.TrimSpace(clock) != "" && clock != "0'" && clock != "0:00":
			return "in", clock
		}
		return "in", "Oynanıyor"
	case "post":
		switch {
		case strings.Contains(n, "PEN") || strings.Contains(strings.ToUpper(short), "PEN"):
			return "post", "Penaltılarla bitti"
		case strings.Contains(n, "AET") || strings.Contains(strings.ToUpper(short), "AET") || strings.Contains(strings.ToUpper(short), "OT"):
			return "post", "Uzatmada bitti"
		}
		return "post", "Bitti"
	}
	return "prematch", ""
}

// foldTR: favori eşleşmesi için Türkçe harfleri sadeleştirir (Beşiktaş = Besiktas).
func foldTR(s string) string {
	r := strings.NewReplacer("ç", "c", "ğ", "g", "ı", "i", "İ", "i", "ö", "o", "ş", "s", "ü", "u", "Ç", "c", "Ğ", "g", "Ö", "o", "Ş", "s", "Ü", "u", "â", "a", "î", "i", "û", "u")
	return strings.ToLower(r.Replace(strings.TrimSpace(s)))
}

func isFavoriteGame(g sportsGame, favorites []string) bool {
	h, a := foldTR(g.Home), foldTR(g.Away)
	for _, f := range favorites {
		f = foldTR(f)
		if len(f) >= 3 && (strings.Contains(h, f) || strings.Contains(a, f)) {
			return true
		}
	}
	return false
}

// sortSportsGames: favoriler, canlı maçlar, yaklaşan maçlar (en yakın önce),
// sonra sonuçlar (en yeni önce).
func sortSportsGames(games []sportsGame) {
	rank := func(g sportsGame) int {
		r := map[string]int{"in": 0, "prematch": 1, "post": 2}[g.Status]
		if g.Fav {
			r -= 10
		}
		return r
	}
	sort.SliceStable(games, func(i, j int) bool {
		a, b := games[i], games[j]
		if rank(a) != rank(b) {
			return rank(a) < rank(b)
		}
		if a.Status == "post" {
			return a.Start > b.Start
		}
		return a.Start < b.Start
	})
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func sportsJSON(ctx context.Context, cfg *Config) []byte {
	sportsMu.Lock()
	if time.Since(sportsAt) < 90*time.Second && len(sportsCache.Games) > 0 {
		out := sportsCache
		sportsMu.Unlock()
		b, _ := json.Marshal(out)
		return b
	}
	sportsMu.Unlock()
	board := sportsBoard{OK: true, Updated: time.Now().Unix()}
	cfg.mu.RLock()
	watch := append([]string(nil), cfg.SportsWatch...)
	favorites := append([]string(nil), cfg.SportsTeams...)
	cfg.mu.RUnlock()
	board.Favorites = favorites
	if len(watch) == 0 {
		watch = []string{"futbol", "nba", "euroleague", "premier", "ucl"}
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 3)
	for _, key := range watch {
		ep, ok := sportsEndpoints[key]
		if !ok {
			continue
		}
		key, ep := key, ep
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			games, err := espnScoreboard(ctx, key, ep)
			<-sem
			if err != nil {
				return
			}
			mu.Lock()
			board.Games = append(board.Games, games...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	for i := range board.Games {
		board.Games[i].Fav = isFavoriteGame(board.Games[i], favorites)
	}
	sortSportsGames(board.Games)
	if len(board.Games) == 0 {
		board.OK = false
		board.Error = "Spor verisi alınamadı"
	}
	sportsMu.Lock()
	sportsCache = board
	sportsAt = time.Now()
	sportsMu.Unlock()
	b, _ := json.Marshal(board)
	return b
}
