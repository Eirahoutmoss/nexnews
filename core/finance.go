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

type marketQuote struct {
	Symbol      string    `json:"symbol"`
	Name        string    `json:"name"`
	Price       float64   `json:"price"`
	Prev        float64   `json:"prev"`
	Change      float64   `json:"change"`
	ChangePct   float64   `json:"changePct"`
	Change5dPct float64   `json:"change5dPct"`
	DayHigh     float64   `json:"dayHigh"`
	DayLow      float64   `json:"dayLow"`
	Volume      int64     `json:"volume"`
	History     []float64 `json:"history,omitempty"`
	Currency    string    `json:"currency"`
	Updated     int64     `json:"updated"`
	Error       string    `json:"error,omitempty"`
}

type goldQuote struct {
	Name        string  `json:"name"`
	WeightGram  float64 `json:"weightGram"`
	Purity      float64 `json:"purity"`
	ValueTRY    float64 `json:"valueTry"`
	Change5dPct float64 `json:"change5dPct"`
	Basis       string  `json:"basis"`
	Note        string  `json:"note,omitempty"`
}

type financeBoard struct {
	OK        bool          `json:"ok"`
	Updated   int64         `json:"updated"`
	Quotes    []marketQuote `json:"quotes"`
	Watchlist []string      `json:"watchlist"`
	Gainers   []string      `json:"gainers,omitempty"`
	Losers    []string      `json:"losers,omitempty"`
	Gold      []goldQuote   `json:"gold,omitempty"`
	Error     string        `json:"error,omitempty"`
}

var financeMu sync.Mutex
var financeCache financeBoard
var financeAt time.Time
var financeClient = &http.Client{Timeout: 10 * time.Second}

func financeSymbols(cfg *Config) []string {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	if len(cfg.MarketWatch) > 0 {
		return append([]string(nil), cfg.MarketWatch...)
	}
	return []string{"THYAO.IS", "ASELS.IS", "GARAN.IS", "BIMAS.IS", "USDTRY=X", "EURTRY=X", "GC=F"}
}

func yahooChart(ctx context.Context, symbol string) marketQuote {
	q := marketQuote{Symbol: symbol, Name: symbol}
	endpoint := "https://query1.finance.yahoo.com/v8/finance/chart/" + url.PathEscape(symbol) + "?range=5d&interval=1d"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("User-Agent", "NexNews/1.0")
	resp, err := financeClient.Do(req)
	if err != nil {
		q.Error = err.Error()
		return q
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		q.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return q
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		q.Error = err.Error()
		return q
	}
	var raw struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Symbol               string  `json:"symbol"`
					Currency             string  `json:"currency"`
					ShortName            string  `json:"shortName"`
					RegularMarketPrice   float64 `json:"regularMarketPrice"`
					PreviousClose        float64 `json:"previousClose"`
					RegularMarketTime    int64   `json:"regularMarketTime"`
					RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
					RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
					RegularMarketVolume  int64   `json:"regularMarketVolume"`
				} `json:"meta"`
				Indicators struct {
					Quote []struct {
						Close []*float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		q.Error = err.Error()
		return q
	}
	if len(raw.Chart.Result) == 0 {
		q.Error = "veri yok"
		return q
	}
	m := raw.Chart.Result[0].Meta
	q.Symbol = m.Symbol
	q.Name = m.ShortName
	q.Currency = m.Currency
	q.Price = m.RegularMarketPrice
	q.Prev = m.PreviousClose
	q.Updated = m.RegularMarketTime
	q.DayHigh = m.RegularMarketDayHigh
	q.DayLow = m.RegularMarketDayLow
	q.Volume = m.RegularMarketVolume
	if len(raw.Chart.Result[0].Indicators.Quote) > 0 {
		c := raw.Chart.Result[0].Indicators.Quote[0].Close
		vals := make([]float64, 0, len(c))
		for _, v := range c {
			if v != nil {
				vals = append(vals, *v)
			}
		}
		if len(vals) >= 2 && vals[0] != 0 {
			q.Change5dPct = (vals[len(vals)-1] - vals[0]) / vals[0] * 100
			if len(vals) > 12 {
				vals = vals[len(vals)-12:]
			}
			q.History = vals
		}
	}
	if q.Prev != 0 {
		q.Change = q.Price - q.Prev
		q.ChangePct = q.Change / q.Prev * 100
	}
	return q
}

func buildGoldQuotes(quotes []marketQuote) []goldQuote {
	var ounce, usdtry *marketQuote
	for i := range quotes {
		q := &quotes[i]
		s := strings.ToUpper(q.Symbol)
		if s == "GC=F" && q.Error == "" && q.Price > 0 {
			ounce = q
		}
		if s == "USDTRY=X" && q.Error == "" && q.Price > 0 {
			usdtry = q
		}
	}
	if ounce == nil || usdtry == nil {
		return nil
	}
	gram24 := ounce.Price * usdtry.Price / 31.1034768
	gold5d := 0.0
	if len(ounce.History) >= 2 && len(usdtry.History) >= 2 && ounce.History[0] > 0 && usdtry.History[0] > 0 {
		last := ounce.History[len(ounce.History)-1] * usdtry.History[len(usdtry.History)-1]
		first := ounce.History[0] * usdtry.History[0]
		if first > 0 {
			gold5d = (last - first) / first * 100
		}
	}
	items := []struct {
		name           string
		weight, purity float64
	}{
		{"Gram Altın", 1, 1},
		{"22 Ayar Gram", 1, 22.0 / 24.0},
		{"Çeyrek Altın", 1.754, 22.0 / 24.0},
		{"Yarım Altın", 3.608, 22.0 / 24.0},
		{"Tam Altın", 7.216, 22.0 / 24.0},
		{"Cumhuriyet Altını", 7.216, 22.0 / 24.0},
		{"22 Ayar Bilezik (1 g)", 1, 22.0 / 24.0},
	}
	out := make([]goldQuote, 0, len(items))
	for _, it := range items {
		out = append(out, goldQuote{Name: it.name, WeightGram: it.weight, Purity: it.purity, ValueTRY: gram24 * it.weight * it.purity, Change5dPct: gold5d, Basis: "Ons Altın + USD/TRY", Note: "Teorik referans değerdir; kuyumcu alış/satış fiyatı işçilik ve prim nedeniyle farklı olabilir."})
	}
	return out
}

func financeJSON(ctx context.Context, cfg *Config) []byte {
	symbols := financeSymbols(cfg)
	financeMu.Lock()
	if time.Since(financeAt) < 2*time.Minute && len(financeCache.Quotes) > 0 {
		out := financeCache
		financeMu.Unlock()
		b, _ := json.Marshal(out)
		return b
	}
	financeMu.Unlock()
	board := financeBoard{OK: true, Updated: time.Now().Unix(), Watchlist: symbols}
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 4)
	for _, s := range symbols {
		s := strings.TrimSpace(s)
		if s == "" {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			q := yahooChart(ctx, s)
			<-sem
			mu.Lock()
			board.Quotes = append(board.Quotes, q)
			mu.Unlock()
		}()
	}
	wg.Wait()
	board.Gold = buildGoldQuotes(board.Quotes)
	if len(board.Quotes) == 0 {
		board.OK = false
		board.Error = "Piyasa verisi alınamadı"
	}
	// Market intelligence: rank only BIST equities, excluding FX/commodities and benchmarks.
	type ranked struct {
		name string
		pct  float64
	}
	var rankedQuotes []ranked
	for _, q := range board.Quotes {
		s := strings.ToUpper(q.Symbol)
		if q.Error != "" || strings.Contains(s, "=") || s == "^XU100" || s == "^XU030" {
			continue
		}
		rankedQuotes = append(rankedQuotes, ranked{name: financeDisplayName(q), pct: q.ChangePct})
	}
	for i := 0; i < len(rankedQuotes); i++ {
		for j := i + 1; j < len(rankedQuotes); j++ {
			if rankedQuotes[j].pct > rankedQuotes[i].pct {
				rankedQuotes[i], rankedQuotes[j] = rankedQuotes[j], rankedQuotes[i]
			}
		}
	}
	for i := 0; i < len(rankedQuotes) && i < 3; i++ {
		board.Gainers = append(board.Gainers, fmt.Sprintf("%s|%.2f", rankedQuotes[i].name, rankedQuotes[i].pct))
	}
	for i, j := 0, len(rankedQuotes)-1; i < j; i, j = i+1, j-1 {
		rankedQuotes[i], rankedQuotes[j] = rankedQuotes[j], rankedQuotes[i]
	}
	for i := 0; i < len(rankedQuotes) && i < 3; i++ {
		board.Losers = append(board.Losers, fmt.Sprintf("%s|%.2f", rankedQuotes[i].name, rankedQuotes[i].pct))
	}
	financeMu.Lock()
	financeCache = board
	financeAt = time.Now()
	financeMu.Unlock()
	b, _ := json.Marshal(board)
	return b
}

func normalizeMarketSymbol(s string) string {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return ""
	}
	// Friendly Turkish inputs: THYAO, ASELS, GARAN are assumed BIST equities.
	if strings.IndexByte(s, '.') < 0 && !strings.Contains(s, "=") && !strings.Contains(s, "-") {
		if len(s) <= 6 {
			return s + ".IS"
		}
	}
	return s
}

func parseMarketWatch(v any) []string {
	a, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(a))
	for _, x := range a {
		if s, ok := x.(string); ok {
			if n := normalizeMarketSymbol(s); n != "" {
				out = append(out, n)
			}
		}
	}
	return out
}

func financeDisplayName(q marketQuote) string {
	s := strings.ToUpper(q.Symbol)
	names := map[string]string{"THYAO.IS": "THYAO", "ASELS.IS": "ASELS", "GARAN.IS": "GARAN", "BIMAS.IS": "BIMAS"}
	if n := names[s]; n != "" {
		return n
	}
	if q.Name != "" {
		return q.Name
	}
	return q.Symbol
}
