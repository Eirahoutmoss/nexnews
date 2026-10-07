package core

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type pulseRegion struct {
	Name       string `json:"name"`
	Emoji      string `json:"emoji"`
	Events     int    `json:"events"`
	Score      int    `json:"score"`
	Confidence int    `json:"confidence"`
	Level      string `json:"level"`
}

type pulseEvent struct {
	EventID    string `json:"eventId"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Score      int    `json:"score"`
	Confidence int    `json:"confidence"`
	Sources    int    `json:"sources"`
	Lifecycle  string `json:"lifecycle"`
	Region     string `json:"region"`
	Emoji      string `json:"emoji"`
	TS         int64  `json:"ts"`
}

type worldPulse struct {
	Updated int64         `json:"updated"`
	Total   int           `json:"total"`
	Regions []pulseRegion `json:"regions"`
	Events  []pulseEvent  `json:"events"`
}

func pulseRegionForCategory(category string) (string, string) {
	c := strings.ToLower(category)
	switch {
	case strings.Contains(c, "türkiye") || strings.Contains(c, "turkey"):
		return "Türkiye", "🇹🇷"
	case strings.Contains(c, "asya") || strings.Contains(c, "asia"):
		return "Asya", "🌏"
	case strings.Contains(c, "afrika") || strings.Contains(c, "africa"):
		return "Afrika", "🌍"
	case strings.Contains(c, "avrupa") || strings.Contains(c, "europe"):
		return "Avrupa", "🇪🇺"
	case strings.Contains(c, "amerika") || strings.Contains(c, "america"):
		return "Amerika", "🌎"
	case strings.Contains(c, "okyanus") || strings.Contains(c, "oceania") || strings.Contains(c, "australia"):
		return "Okyanusya", "🌊"
	default:
		return "Küresel", "🌐"
	}
}

func pulseLevel(score int) string {
	switch {
	case score >= 75:
		return "high"
	case score >= 45:
		return "mid"
	default:
		return "low"
	}
}

func (s *Store) worldPulseJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	regions := map[string]*pulseRegion{}
	regionOrder := []string{"Türkiye", "Avrupa", "Asya", "Afrika", "Amerika", "Okyanusya", "Küresel"}
	emojis := map[string]string{"Türkiye": "🇹🇷", "Avrupa": "🇪🇺", "Asya": "🌏", "Afrika": "🌍", "Amerika": "🌎", "Okyanusya": "🌊", "Küresel": "🌐"}
	for _, name := range regionOrder {
		regions[name] = &pulseRegion{Name: name, Emoji: emojis[name]}
	}

	var candidates []*Story
	seen := map[string]bool{}
	for _, st := range s.important {
		if st == nil || s.isHidden(st) || !s.countryAllowed(st) || seen[st.EventID] {
			continue
		}
		seen[st.EventID] = true
		candidates = append(candidates, st)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Published.After(candidates[j].Published)
	})

	events := make([]pulseEvent, 0, 8)
	for _, st := range candidates {
		name, emoji := pulseRegionForCategory(st.Category)
		r := regions[name]
		r.Events++
		r.Score += st.Score
		r.Confidence += st.Confidence
		if len(events) < 8 {
			events = append(events, pulseEvent{EventID: st.EventID, Title: st.Title, Category: st.Category, Score: st.Score, Confidence: st.Confidence, Sources: st.IndependentSources, Lifecycle: st.Lifecycle, Region: name, Emoji: emoji, TS: st.Published.Unix()})
		}
	}

	outRegions := make([]pulseRegion, 0, len(regionOrder))
	for _, name := range regionOrder {
		r := *regions[name]
		if r.Events > 0 {
			r.Score = r.Score / r.Events
			r.Confidence = r.Confidence / r.Events
		}
		r.Level = pulseLevel(r.Score)
		outRegions = append(outRegions, r)
	}
	total := 0
	for _, r := range outRegions {
		total += r.Events
	}
	return json.Marshal(worldPulse{Updated: time.Now().Unix(), Total: total, Regions: outRegions, Events: events})
}
