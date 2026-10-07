package core

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type briefingItem struct {
	Rank       int    `json:"rank"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Score      int    `json:"score"`
	Confidence int    `json:"confidence"`
	Sources    int    `json:"sources"`
	Lifecycle  string `json:"lifecycle"`
	Direction  string `json:"direction,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Link       string `json:"link"`
	EventID    string `json:"eventId,omitempty"`
}

type briefingBoard struct {
	GeneratedAt int64          `json:"generatedAt"`
	WindowLabel string         `json:"windowLabel"`
	Headline    string         `json:"headline"`
	Summary     string         `json:"summary"`
	Items       []briefingItem `json:"items"`
	Total       int            `json:"total"`
}

func briefingWindowLabel(now time.Time) string {
	if now.Hour() < 12 {
		return "Sabah brifingi"
	}
	if now.Hour() < 18 {
		return "Gün ortası brifingi"
	}
	return "Akşam brifingi"
}

func briefingDirection(st *Story) string {
	p := predictiveForStory(st)
	return p.Direction
}

func (s *Store) briefingJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	candidates := append([]*Story{}, s.important...)
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Published.After(candidates[j].Published)
	})
	seen := map[string]bool{}
	out := briefingBoard{GeneratedAt: time.Now().Unix(), WindowLabel: briefingWindowLabel(time.Now()), Total: len(candidates)}
	for _, st := range candidates {
		if st == nil {
			continue
		}
		id := st.EventID
		if id == "" {
			id = st.Link
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out.Items = append(out.Items, briefingItem{
			Rank: len(out.Items) + 1, Title: s.trTitle(st.Title), Category: st.Category,
			Score: st.Score, Confidence: st.Confidence, Sources: st.IndependentSources,
			Lifecycle: st.Lifecycle, Direction: briefingDirection(st), Reason: eventReason(st),
			Link: st.Link, EventID: st.EventID,
		})
		if len(out.Items) >= 7 {
			break
		}
	}
	if len(out.Items) == 0 {
		out.Headline = "Dünyanın nabzında belirgin bir gelişme yok"
		out.Summary = "Yeni önemli olay sinyalleri oluştuğunda bu brifing otomatik olarak güncellenecek."
	} else {
		high, developing := 0, 0
		for _, it := range out.Items {
			if it.Score >= 75 {
				high++
			}
			if strings.EqualFold(it.Lifecycle, "developing") || strings.EqualFold(it.Direction, "escalating") {
				developing++
			}
		}
		out.Headline = out.Items[0].Title
		parts := []string{"Öncelikli gelişmeler derlendi."}
		if high > 0 {
			parts = append(parts, stringInt(high)+" yüksek önem sinyali var.")
		}
		if developing > 0 {
			parts = append(parts, stringInt(developing)+" olay gelişim/tırmanma sinyali taşıyor.")
		}
		out.Summary = strings.Join(parts, " ")
	}
	return json.Marshal(out)
}

func stringInt(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{digits[n%10]}, b...)
		n /= 10
	}
	return string(b)
}
