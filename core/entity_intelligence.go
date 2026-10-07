package core

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type entityMention struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Count    int      `json:"count"`
	Events   int      `json:"events"`
	Score    int      `json:"score"`
	EventIDs []string `json:"eventIds,omitempty"`
}

type entityRelation struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
	Type   string `json:"type"`
}

type entityBoard struct {
	Updated   int64            `json:"updated"`
	Total     int              `json:"total"`
	People    int              `json:"people"`
	Org       int              `json:"organizations"`
	Places    int              `json:"places"`
	Companies int              `json:"companies"`
	Entities  []entityMention  `json:"entities"`
	Relations []entityRelation `json:"relations"`
}

type entityLexeme struct {
	Name, Type string
	Terms      []string
}

var entityLexicon = []entityLexeme{
	{"Recep Tayyip Erdoğan", "person", []string{"recep tayyip erdoğan", "erdoğan"}},
	{"Donald Trump", "person", []string{"donald trump", "trump"}},
	{"Vladimir Putin", "person", []string{"vladimir putin", "putin"}},
	{"Emmanuel Macron", "person", []string{"emmanuel macron", "macron"}},
	{"Ursula von der Leyen", "person", []string{"ursula von der leyen"}},
	{"NATO", "organization", []string{"nato"}},
	{"Avrupa Birliği", "organization", []string{"avrupa birliği", "ab"}},
	{"Birleşmiş Milletler", "organization", []string{"birleşmiş milletler", "bm"}},
	{"Merkez Bankası", "organization", []string{"merkez bankası", "tcmb", "fed", "federal reserve"}},
	{"Hazine ve Maliye Bakanlığı", "organization", []string{"hazine ve maliye bakanlığı"}},
	{"Microsoft", "company", []string{"microsoft"}},
	{"Apple", "company", []string{"apple"}},
	{"Google", "company", []string{"google"}},
	{"NVIDIA", "company", []string{"nvidia"}},
	{"OpenAI", "company", []string{"openai"}},
	{"Tesla", "company", []string{"tesla"}},
	{"Türk Hava Yolları", "company", []string{"türk hava yolları", "thy"}},
	{"ASELSAN", "company", []string{"aselsan"}},
	{"Türkiye", "place", []string{"türkiye", "türkiye'de", "türkiye'nin"}},
	{"ABD", "place", []string{"abd", "amerika birleşik devletleri", "washington"}},
	{"Rusya", "place", []string{"rusya", "moskova"}},
	{"Ukrayna", "place", []string{"ukrayna", "kiev", "kyiv"}},
	{"İsrail", "place", []string{"israil", "kudüs", "tel aviv"}},
	{"İran", "place", []string{"iran", "tahran"}},
	{"Çin", "place", []string{"çin", "pekin"}},
	{"Almanya", "place", []string{"almanya", "berlin"}},
	{"Fransa", "place", []string{"fransa", "paris"}},
	{"İngiltere", "place", []string{"ingiltere", "londra"}},
	{"Suriye", "place", []string{"suriye", "şam", "şam"}},
	{"Irak", "place", []string{"irak", "bağdat"}},
}

func normEntity(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func entityMatches(text string) []entityLexeme {
	n := normEntity(text)
	out := make([]entityLexeme, 0, 8)
	for _, x := range entityLexicon {
		for _, term := range x.Terms {
			if strings.Contains(n, normEntity(term)) {
				out = append(out, x)
				break
			}
		}
	}
	return out
}

func (s *Store) entityIntelligenceJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	type acc struct {
		entityMention
		ids map[string]bool
	}
	byID := map[string]*acc{}
	pair := map[string]int{}
	seenEvents := map[string]bool{}
	candidates := append([]*Story{}, s.important...)
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	for _, st := range candidates {
		if st == nil || st.EventID == "" || s.isHidden(st) || seenEvents[st.EventID] {
			continue
		}
		seenEvents[st.EventID] = true
		matches := entityMatches(st.Title + " " + st.Category)
		local := map[string]bool{}
		for _, m := range matches {
			id := normEntity(m.Type + ":" + m.Name)
			a := byID[id]
			if a == nil {
				a = &acc{entityMention: entityMention{ID: id, Name: m.Name, Type: m.Type}, ids: map[string]bool{}}
				byID[id] = a
			}
			if !local[id] {
				a.Count++
				a.Score += st.Score
				a.ids[st.EventID] = true
				local[id] = true
			}
		}
		keys := make([]string, 0, len(local))
		for k := range local {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				pair[keys[i]+"\x00"+keys[j]]++
			}
		}
	}
	board := entityBoard{Updated: time.Now().Unix()}
	for _, a := range byID {
		a.Events = len(a.ids)
		a.EventIDs = make([]string, 0, len(a.ids))
		for id := range a.ids {
			a.EventIDs = append(a.EventIDs, id)
		}
		sort.Strings(a.EventIDs)
		board.Entities = append(board.Entities, a.entityMention)
	}
	sort.SliceStable(board.Entities, func(i, j int) bool {
		if board.Entities[i].Count != board.Entities[j].Count {
			return board.Entities[i].Count > board.Entities[j].Count
		}
		return board.Entities[i].Score > board.Entities[j].Score
	})
	if len(board.Entities) > 18 {
		board.Entities = board.Entities[:18]
	}
	for _, e := range board.Entities {
		board.Total++
		switch e.Type {
		case "person":
			board.People++
		case "organization":
			board.Org++
		case "place":
			board.Places++
		case "company":
			board.Companies++
		}
	}
	allowed := map[string]bool{}
	for _, e := range board.Entities {
		allowed[e.ID] = true
	}
	for k, w := range pair {
		p := strings.Split(k, "\x00")
		if len(p) != 2 || !allowed[p[0]] || !allowed[p[1]] || w < 2 {
			continue
		}
		board.Relations = append(board.Relations, entityRelation{Source: p[0], Target: p[1], Weight: w, Type: "co-occurrence"})
	}
	sort.SliceStable(board.Relations, func(i, j int) bool { return board.Relations[i].Weight > board.Relations[j].Weight })
	if len(board.Relations) > 24 {
		board.Relations = board.Relations[:24]
	}
	return json.Marshal(board)
}
