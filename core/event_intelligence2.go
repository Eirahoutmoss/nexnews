package core

import (
	"encoding/json"
	"sort"
	"time"
)

type intelligenceEvent struct {
	EventID            string  `json:"eventId"`
	Title              string  `json:"title"`
	Link               string  `json:"link"`
	Category           string  `json:"category"`
	Score              int     `json:"score"`
	Confidence         int     `json:"confidence"`
	IndependentSources int     `json:"independentSources"`
	PrimarySources     int     `json:"primarySources"`
	Velocity           float64 `json:"velocity"`
	Lifecycle          string  `json:"lifecycle"`
	Reason             string  `json:"reason"`
	EvidenceSummary    string  `json:"evidenceSummary"`
	AgeSeconds         int64   `json:"ageSeconds"`
}

type intelligenceBoard struct {
	Updated    int64               `json:"updated"`
	Total      int                 `json:"total"`
	Active     int                 `json:"active"`
	High       int                 `json:"high"`
	Confirmed  int                 `json:"confirmed"`
	Developing int                 `json:"developing"`
	Events     []intelligenceEvent `json:"events"`
}

func (s *Store) eventIntelligenceJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := map[string]bool{}
	items := make([]*Story, 0, len(s.important))
	for _, st := range s.important {
		if st == nil || s.isHidden(st) || st.EventID == "" || seen[st.EventID] {
			continue
		}
		seen[st.EventID] = true
		items = append(items, st)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].Published.After(items[j].Published)
	})

	out := make([]intelligenceEvent, 0, minInt(len(items), 8))
	board := intelligenceBoard{Updated: time.Now().Unix(), Total: len(items)}
	for _, st := range items {
		if st.Lifecycle == eventDeveloping {
			board.Developing++
		}
		if st.Lifecycle == eventConfirmed {
			board.Confirmed++
		}
		if st.Score >= 75 {
			board.High++
		}
		if st.Lifecycle != eventArchived {
			board.Active++
		}
		if len(out) >= 8 {
			continue
		}
		age := int64(0)
		if st.HasDate {
			age = int64(time.Since(st.Published).Seconds())
			if age < 0 {
				age = 0
			}
		}
		out = append(out, intelligenceEvent{
			EventID: st.EventID, Title: st.Title, Link: st.Link, Category: st.Category,
			Score: st.Score, Confidence: st.Confidence, IndependentSources: st.IndependentSources,
			PrimarySources: st.PrimarySources, Velocity: st.Velocity, Lifecycle: st.Lifecycle,
			Reason: eventReason(st), EvidenceSummary: evidenceForStory(st).Summary, AgeSeconds: age,
		})
	}
	board.Events = out
	return json.Marshal(board)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
