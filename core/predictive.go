package core

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// Predictive Intelligence is a deterministic signal layer. It does not claim
// to forecast facts; it estimates which tracked events deserve attention next
// from recency, velocity, importance, corroboration and lifecycle.
type predictiveEvent struct {
	EventID    string  `json:"eventId"`
	Title      string  `json:"title"`
	Category   string  `json:"category"`
	Score      int     `json:"score"`
	Confidence int     `json:"confidence"`
	Sources    int     `json:"sources"`
	Velocity   float64 `json:"velocity"`
	Lifecycle  string  `json:"lifecycle"`
	Direction  string  `json:"direction"`
	Risk       int     `json:"risk"`
	WatchScore int     `json:"watchScore"`
	Window     string  `json:"window"`
	Reason     string  `json:"reason"`
	Link       string  `json:"link"`
	TS         int64   `json:"ts"`
}

type predictiveBoard struct {
	Updated    int64             `json:"updated"`
	Total      int               `json:"total"`
	Escalating int               `json:"escalating"`
	Emerging   int               `json:"emerging"`
	Stable     int               `json:"stable"`
	Declining  int               `json:"declining"`
	Events     []predictiveEvent `json:"events"`
}

func predictionDirection(st *Story) string {
	age := time.Since(st.Published)
	if age < 0 {
		age = 0
	}
	switch {
	case st.Lifecycle == eventDeveloping && st.Velocity >= 1.2:
		return "escalating"
	case st.Lifecycle == eventDetected || (age <= 3*time.Hour && st.Velocity >= 0.7):
		return "emerging"
	case st.Lifecycle == eventStabilizing || st.Lifecycle == eventConfirmed:
		if st.Velocity < 0.35 || age > 24*time.Hour {
			return "declining"
		}
		return "stable"
	default:
		return "stable"
	}
}

func predictionWindow(direction string) string {
	switch direction {
	case "escalating":
		return "önümüzdeki 1–3 saat"
	case "emerging":
		return "önümüzdeki 3–6 saat"
	case "stable":
		return "önümüzdeki 6–12 saat"
	default:
		return "önümüzdeki 12–24 saat"
	}
}

func predictionRisk(st *Story, direction string) int {
	r := st.Score/2 + st.Confidence/4 + st.IndependentSources*3
	switch direction {
	case "escalating":
		r += 18
	case "emerging":
		r += 10
	case "declining":
		r -= 8
	}
	if st.PrimarySources > 0 {
		r += 6
	}
	if r < 0 {
		r = 0
	}
	if r > 100 {
		r = 100
	}
	return r
}

func predictionReason(st *Story, direction string) string {
	parts := []string{}
	if st.Velocity >= 1.5 {
		parts = append(parts, "haber yoğunluğu yüksek")
	} else if st.Velocity >= 0.7 {
		parts = append(parts, "haber akışı hızlanıyor")
	}
	if st.IndependentSources >= 3 {
		parts = append(parts, "bağımsız kaynaklarla destekleniyor")
	} else if st.IndependentSources == 2 {
		parts = append(parts, "iki bağımsız kaynak bulundu")
	}
	if st.PrimarySources > 0 {
		parts = append(parts, "birincil kaynak sinyali var")
	}
	if st.Score >= 75 {
		parts = append(parts, "önem skoru yüksek")
	}
	if len(parts) == 0 {
		parts = append(parts, "mevcut olay sinyalleri izlemeye değer")
	}
	label := map[string]string{"escalating": "tırmanıyor", "emerging": "yükseliyor", "stable": "stabil", "declining": "zayıflıyor"}[direction]
	return "Durum " + label + ": " + strings.Join(parts, ", ") + "."
}

func predictiveForStory(st *Story) predictiveEvent {
	d := predictionDirection(st)
	return predictiveEvent{EventID: st.EventID, Title: st.Title, Category: st.Category, Score: st.Score, Confidence: st.Confidence, Sources: st.IndependentSources, Velocity: st.Velocity, Lifecycle: st.Lifecycle, Direction: d, Risk: predictionRisk(st, d), WatchScore: minInt(100, predictionRisk(st, d)+st.Score/5), Window: predictionWindow(d), Reason: predictionReason(st, d), Link: st.Link, TS: st.Published.Unix()}
}

func (s *Store) predictiveJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := map[string]bool{}
	out := make([]predictiveEvent, 0, 12)
	board := predictiveBoard{Updated: time.Now().Unix()}
	candidates := append([]*Story{}, s.important...)
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	for _, st := range candidates {
		if st == nil || st.EventID == "" || s.isHidden(st) || seen[st.EventID] {
			continue
		}
		seen[st.EventID] = true
		p := predictiveForStory(st)
		board.Total++
		switch p.Direction {
		case "escalating":
			board.Escalating++
		case "emerging":
			board.Emerging++
		case "stable":
			board.Stable++
		case "declining":
			board.Declining++
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WatchScore != out[j].WatchScore {
			return out[i].WatchScore > out[j].WatchScore
		}
		return out[i].TS > out[j].TS
	})
	if len(out) > 8 {
		out = out[:8]
	}
	board.Events = out
	return json.Marshal(board)
}
