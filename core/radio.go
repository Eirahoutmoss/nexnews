package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type radioStation struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Country   string `json:"country"`
	Code      string `json:"countryCode"`
	Language  string `json:"language"`
	Tags      string `json:"tags"`
	Continent string `json:"continent,omitempty"`
	Homepage  string `json:"homepage"`
	Stream    string `json:"stream"`
	Favicon   string `json:"favicon"`
}

type radioBoard struct {
	Updated            string         `json:"updated"`
	LocalCountry       string         `json:"localCountry"`
	LocalFlag          string         `json:"localFlag"`
	Stations           []radioStation `json:"stations"`
	LocalCount         int            `json:"localCount"`
	WorldCount         int            `json:"worldCount"`
	Provider           string         `json:"provider"`
	DirectoryStatus    string         `json:"directoryStatus"`
	PlayableCandidates int            `json:"playableCandidates"`
	Error              string         `json:"error,omitempty"`
}

var radioCache struct {
	sync.Mutex
	at    time.Time
	board []byte
}

func radioContext(lang string) (string, string) {
	l := strings.ToLower(strings.TrimSpace(lang))
	switch {
	case strings.HasPrefix(l, "tr"):
		return "TR", "🇹🇷"
	case strings.HasPrefix(l, "de"):
		return "DE", "🇩🇪"
	case strings.HasPrefix(l, "fr"):
		return "FR", "🇫🇷"
	case strings.HasPrefix(l, "es"):
		return "ES", "🇪🇸"
	case strings.HasPrefix(l, "it"):
		return "IT", "🇮🇹"
	case strings.HasPrefix(l, "pt"):
		return "PT", "🇵🇹"
	case strings.HasPrefix(l, "nl"):
		return "NL", "🇳🇱"
	case strings.HasPrefix(l, "pl"):
		return "PL", "🇵🇱"
	case strings.HasPrefix(l, "ja"):
		return "JP", "🇯🇵"
	case strings.HasPrefix(l, "ko"):
		return "KR", "🇰🇷"
	case strings.HasPrefix(l, "zh"):
		return "CN", "🇨🇳"
	case strings.HasPrefix(l, "ar"):
		return "SA", "🇸🇦"
	case strings.HasPrefix(l, "ru"):
		return "RU", "🇷🇺"
	case strings.HasPrefix(l, "hi"):
		return "IN", "🇮🇳"
	default:
		return "US", "🇺🇸"
	}
}

func radioJSON(ctx context.Context) ([]byte, error) {
	radioCache.Lock()
	if len(radioCache.board) > 0 && time.Since(radioCache.at) < 10*time.Minute {
		b := append([]byte(nil), radioCache.board...)
		radioCache.Unlock()
		return b, nil
	}
	radioCache.Unlock()

	code, flag := radioContext(systemLang())
	client := &http.Client{Timeout: 6 * time.Second}
	continentCodes := []struct {
		Name  string
		Codes []string
	}{
		{"Avrupa", []string{"GB", "DE", "FR", "IT", "ES", "PT", "NL", "SE", "NO", "PL", "CH", "AT", "GR", "IE", "BE"}},
		{"Asya", []string{"TR", "JP", "KR", "CN", "IN", "ID", "TH", "MY", "SG", "PH", "VN", "AE", "SA", "IL", "PK"}},
		{"Afrika", []string{"ZA", "NG", "EG", "MA", "DZ", "KE", "GH", "TZ", "UG", "SN", "TN", "ET"}},
		{"Kuzey Amerika", []string{"US", "CA", "MX", "CU", "JM", "DO", "CR", "PA", "GT", "PR"}},
		{"Güney Amerika", []string{"BR", "AR", "CL", "CO", "PE", "UY", "EC", "BO", "PY", "VE"}},
		{"Okyanusya", []string{"AU", "NZ", "FJ", "PG", "WS", "NC"}},
	}

	type job struct{ continent, code string }
	type result struct {
		continent string
		stations  []radioStation
	}
	jobs := make(chan job)
	results := make(chan result, 96)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				endpoint := fmt.Sprintf("https://de1.api.radio-browser.info/json/stations/bycountrycodeexact/%s?order=votes&reverse=true&limit=6&hidebroken=true", j.code)
				stations := radioFetch(ctx, client, endpoint)
				for i := range stations {
					stations[i].Continent = j.continent
				}
				results <- result{continent: j.continent, stations: stations}
			}
		}()
	}
	go func() {
		for _, c := range continentCodes {
			for _, code := range c.Codes {
				jobs <- job{continent: c.Name, code: code}
			}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	byContinent := map[string][]radioStation{}
	seen := map[string]bool{}
	for r := range results {
		for _, s := range r.stations {
			key := s.UUID
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(s.Name)) + "|" + s.Country
			}
			if seen[key] || strings.TrimSpace(s.Stream) == "" {
				continue
			}
			seen[key] = true
			if len(byContinent[r.continent]) < 20 {
				byContinent[r.continent] = append(byContinent[r.continent], s)
			}
		}
	}

	stations := make([]radioStation, 0, 120)
	for _, c := range continentCodes {
		stations = append(stations, byContinent[c.Name]...)
	}
	status := "healthy"
	if len(stations) == 0 {
		status = "error"
	} else if len(stations) < 30 {
		status = "degraded"
	}
	localCount := 0
	for _, s := range stations {
		if strings.EqualFold(s.Code, code) {
			localCount++
		}
	}
	board := radioBoard{
		Updated: time.Now().UTC().Format(time.RFC3339), LocalCountry: code, LocalFlag: flag,
		Stations: stations, LocalCount: localCount, WorldCount: len(stations) - localCount,
		Provider: "Radio Browser", DirectoryStatus: status, PlayableCandidates: len(stations),
	}
	if len(stations) == 0 {
		board.Error = "Canlı radyo istasyonları şu anda alınamadı."
	}
	b, err := json.Marshal(board)
	if err != nil {
		return nil, err
	}
	radioCache.Lock()
	radioCache.at = time.Now()
	radioCache.board = append([]byte(nil), b...)
	radioCache.Unlock()
	return b, nil
}

func radioFetch(ctx context.Context, client *http.Client, endpoint string) []radioStation {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "NexNews/1.0 (+world-radio)")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	if err != nil {
		return nil
	}
	var raw []struct {
		StationUUID string `json:"stationuuid"`
		Name        string `json:"name"`
		Country     string `json:"country"`
		CountryCode string `json:"countrycode"`
		Language    string `json:"language"`
		Tags        string `json:"tags"`
		Homepage    string `json:"homepage"`
		URLResolved string `json:"url_resolved"`
		URL         string `json:"url"`
		Favicon     string `json:"favicon"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	out := make([]radioStation, 0, len(raw))
	for _, x := range raw {
		stream := x.URLResolved
		if stream == "" {
			stream = x.URL
		}
		if stream == "" {
			continue
		}
		if u, err := url.Parse(stream); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		out = append(out, radioStation{UUID: x.StationUUID, Name: strings.TrimSpace(x.Name), Country: x.Country, Code: x.CountryCode, Language: x.Language, Tags: x.Tags, Homepage: x.Homepage, Stream: stream, Favicon: x.Favicon})
	}
	return out
}
