package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Status       string `json:"status"`
	StatusDetail string `json:"statusDetail"`
	Start        int64  `json:"start"`
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

func espnScoreboard(ctx context.Context, key, endpoint string) ([]sportsGame, error) {
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
				Type struct {
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
				home, hs = c.Team.DisplayName, c.Score
			} else {
				away, as = c.Team.DisplayName, c.Score
			}
		}
		start, _ := time.Parse(time.RFC3339, e.Date)
		league := key
		games = append(games, sportsGame{ID: e.ID, League: league, LeagueName: sportsLeagueName(key), Home: home, Away: away, HomeScore: hs, AwayScore: as, Status: e.Status.Type.State, StatusDetail: firstNonEmpty(e.Status.Type.ShortDetail, e.Status.Type.Detail), Start: start.Unix()})
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
