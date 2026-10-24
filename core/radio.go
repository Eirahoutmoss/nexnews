package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

// RC94: radyo dizini büyütüldü. Yerel ülke (sistemin ülkesi ya da seçilen haber
// bölgesi) ayrı "Yerel" grubunda yüzlerce istasyonla, dünya 110+ ülkeden ülke
// başına 40 istasyonla gelir. Sonuç diske de yazılır; sonraki açılış anında
// gelir, dizin arka planda tazelenir. Radio Browser'ın birden çok sunucusu denenir.
var radioMirrors = []string{"de1", "nl1", "at1", "fi1", "de2"}

var radioRegions = []struct {
	Name  string
	Codes []string
}{
	{"Avrupa", []string{"GB", "DE", "FR", "IT", "ES", "PT", "NL", "BE", "LU", "IE", "CH", "AT", "SE", "NO", "DK", "FI", "IS", "PL", "CZ", "SK", "HU", "RO", "BG", "GR", "CY", "HR", "SI", "RS", "BA", "ME", "MK", "AL", "XK", "UA", "MD", "LT", "LV", "EE", "MT", "TR"}},
	{"Orta Doğu", []string{"AE", "SA", "QA", "KW", "BH", "OM", "JO", "LB", "IQ", "SY", "PS", "IL", "IR", "YE", "EG"}},
	{"Asya", []string{"JP", "KR", "CN", "TW", "HK", "IN", "PK", "BD", "LK", "NP", "ID", "TH", "MY", "SG", "PH", "VN", "KH", "MN", "KZ", "UZ", "KG", "AZ", "GE", "AM", "AF"}},
	{"Rusya", []string{"RU", "BY"}},
	{"Afrika", []string{"ZA", "NG", "MA", "DZ", "TN", "LY", "KE", "GH", "TZ", "UG", "SN", "ET", "CM", "CI", "SD", "SO", "AO", "MZ", "ZW", "RW", "CD", "NA", "BW", "MU"}},
	{"Kuzey Amerika", []string{"US", "CA", "MX", "CU", "JM", "DO", "HT", "CR", "PA", "GT", "HN", "SV", "NI", "PR", "TT", "BS"}},
	{"Güney Amerika", []string{"BR", "AR", "CL", "CO", "PE", "UY", "EC", "BO", "PY", "VE"}},
	{"Okyanusya", []string{"AU", "NZ", "FJ", "PG", "WS", "NC", "PF"}},
}

const (
	radioLocalLimit = 500
	radioWorldLimit = 40
	radioTTL        = 6 * time.Hour
)

func radioCachePath() string { return filepath.Join(appDataDir(), "radio.json") }

func radioJSON(ctx context.Context) ([]byte, error) {
	radioCache.Lock()
	if len(radioCache.board) == 0 {
		if b, err := os.ReadFile(radioCachePath()); err == nil && json.Valid(b) {
			radioCache.board = b
			if fi, err := os.Stat(radioCachePath()); err == nil {
				radioCache.at = fi.ModTime()
			}
		}
	}
	if len(radioCache.board) > 0 && time.Since(radioCache.at) < radioTTL {
		b := append([]byte(nil), radioCache.board...)
		radioCache.Unlock()
		return b, nil
	}
	radioCache.Unlock()

	code := strings.ToUpper(localNewsCountry())
	flag := countryFlag(code)
	if code == "" {
		code, flag = radioContext(systemLang())
	}
	client := &http.Client{Timeout: 9 * time.Second}

	type job struct {
		region, code string
		limit        int
	}
	type result struct {
		region   string
		stations []radioStation
	}
	var all []job
	all = append(all, job{"Yerel", code, radioLocalLimit})
	for _, r := range radioRegions {
		for _, c := range r.Codes {
			if c != code {
				all = append(all, job{r.Name, c, radioWorldLimit})
			}
		}
	}
	jobs := make(chan job)
	results := make(chan result, len(all))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for j := range jobs {
				var st []radioStation
				for k := 0; k < len(radioMirrors) && st == nil; k++ {
					m := radioMirrors[(w+k)%len(radioMirrors)]
					endpoint := fmt.Sprintf("https://%s.api.radio-browser.info/json/stations/bycountrycodeexact/%s?order=clickcount&reverse=true&limit=%d&hidebroken=true", m, j.code, j.limit)
					st = radioFetch(ctx, client, endpoint)
					if ctx.Err() != nil {
						break
					}
				}
				for i := range st {
					st[i].Continent = j.region
				}
				results <- result{region: j.region, stations: st}
			}
		}(i)
	}
	go func() {
		for _, j := range all {
			select {
			case jobs <- j:
			case <-ctx.Done():
			}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	byRegion := map[string][]radioStation{}
	seen := map[string]bool{}
	for r := range results {
		for _, s := range r.stations {
			key := s.UUID
			if key == "" {
				key = strings.ToLower(strings.TrimSpace(s.Name)) + "|" + s.Country
			}
			name := strings.ToLower(strings.TrimSpace(s.Name)) + "|" + strings.ToUpper(s.Code)
			if seen[key] || seen[name] || strings.TrimSpace(s.Stream) == "" || strings.TrimSpace(s.Name) == "" {
				continue
			}
			seen[key], seen[name] = true, true
			if len(s.Tags) > 80 {
				s.Tags = s.Tags[:80]
			}
			byRegion[r.region] = append(byRegion[r.region], s)
		}
	}
	stations := make([]radioStation, 0, 4000)
	stations = append(stations, byRegion["Yerel"]...)
	for _, r := range radioRegions {
		stations = append(stations, byRegion[r.Name]...)
	}
	localCount := len(byRegion["Yerel"])
	status := "healthy"
	if len(stations) == 0 {
		status = "error"
	} else if len(stations) < 300 {
		status = "degraded"
	}
	if len(stations) == 0 {
		// Dizine ulaşılamadıysa eski listeyi göster.
		radioCache.Lock()
		old := append([]byte(nil), radioCache.board...)
		radioCache.Unlock()
		if len(old) > 0 {
			return old, nil
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
	if len(stations) > 0 {
		radioCache.Lock()
		radioCache.at = time.Now()
		radioCache.board = append([]byte(nil), b...)
		radioCache.Unlock()
		_ = atomicWriteFile(radioCachePath(), b)
	}
	return b, nil
}

// countryFlag: "TR" → 🇹🇷
func countryFlag(code string) string {
	if len(code) != 2 {
		return "🌍"
	}
	r := []rune(strings.ToUpper(code))
	return string([]rune{0x1F1E6 + (r[0] - 'A'), 0x1F1E6 + (r[1] - 'A')})
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
