package core

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type personalSignal struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Kind  string `json:"kind"`
}

type personalReason struct {
	Text  string `json:"text"`
	Score int    `json:"score"`
}

type personalBoard struct {
	Updated       int64            `json:"updated"`
	PersonalScore int              `json:"personalScore"`
	Explicit      []string         `json:"explicit"`
	Topics        []personalSignal `json:"topics"`
	Entities      []personalSignal `json:"entities"`
	Categories    []personalSignal `json:"categories"`
	Reasons       []personalReason `json:"reasons"`
	TopStories    []outItem        `json:"topStories"`
}

func containsAnyPersonal(s string, terms []string) bool {
	for _, t := range terms {
		if t != "" && strings.Contains(s, t) {
			return true
		}
	}
	return false
}

func (s *Store) personalIntelligenceJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	board := personalBoard{Updated: time.Now().Unix()}
	for _, v := range s.interests {
		v = strings.TrimSpace(v)
		if v != "" {
			board.Explicit = append(board.Explicit, v)
		}
	}

	catScore := map[string]int{}
	catHits := map[string]int{}
	entityScore := map[string]int{}
	entityHits := map[string]int{}
	topicScore := map[string]int{}
	topicHits := map[string]int{}
	var candidates []*Story
	for _, st := range s.important {
		if st == nil || s.isHidden(st) {
			continue
		}
		low := strings.ToLower(st.Title + " " + st.Category)
		match := 0
		for _, interest := range s.interests {
			if interest != "" && strings.Contains(low, interest) {
				match++
			}
		}
		if match == 0 {
			continue
		}
		personal := st.Score + match*12
		if personal > 100 {
			personal = 100
		}
		candidates = append(candidates, st)
		catScore[st.Category] += personal
		catHits[st.Category] += match
		for _, interest := range s.interests {
			if strings.Contains(low, interest) {
				topicScore[interest] += personal
				topicHits[interest]++
			}
		}
		for _, ent := range entityMatches(st.Title + " " + st.Category) {
			if containsAnyPersonal(strings.ToLower(ent.Name), s.interests) || containsAnyPersonal(low, ent.Terms) {
				key := ent.Name + "|" + ent.Type
				entityScore[key] += personal
				entityHits[key]++
			}
		}
	}

	for name, score := range topicScore {
		board.Topics = append(board.Topics, personalSignal{Name: name, Score: normalizePersonal(score, topicHits[name]), Kind: "topic"})
	}
	for name, score := range catScore {
		board.Categories = append(board.Categories, personalSignal{Name: name, Score: normalizePersonal(score, catHits[name]), Kind: "category"})
	}
	for key, score := range entityScore {
		parts := strings.SplitN(key, "|", 2)
		name := parts[0]
		board.Entities = append(board.Entities, personalSignal{Name: name, Score: normalizePersonal(score, entityHits[key]), Kind: parts[1]})
	}

	sort.Slice(board.Topics, func(i, j int) bool { return board.Topics[i].Score > board.Topics[j].Score })
	sort.Slice(board.Categories, func(i, j int) bool { return board.Categories[i].Score > board.Categories[j].Score })
	sort.Slice(board.Entities, func(i, j int) bool { return board.Entities[i].Score > board.Entities[j].Score })
	if len(board.Topics) > 6 {
		board.Topics = board.Topics[:6]
	}
	if len(board.Categories) > 6 {
		board.Categories = board.Categories[:6]
	}
	if len(board.Entities) > 8 {
		board.Entities = board.Entities[:8]
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return personalStoryScore(candidates[i], s.interests) > personalStoryScore(candidates[j], s.interests)
	})
	seen := map[string]bool{}
	for _, st := range candidates {
		if seen[st.Link] {
			continue
		}
		seen[st.Link] = true
		board.TopStories = append(board.TopStories, s.storyOut(st))
		board.PersonalScore += personalStoryScore(st, s.interests)
		if len(board.TopStories) >= 5 {
			break
		}
	}
	if len(board.TopStories) > 0 {
		board.PersonalScore /= len(board.TopStories)
	}
	if board.PersonalScore > 100 {
		board.PersonalScore = 100
	}

	if len(board.Explicit) == 0 {
		board.Reasons = append(board.Reasons, personalReason{Text: "Henüz kişisel konu tercihi tanımlanmadı. Ayarlar'dan ilgi alanları ekleyebilirsin.", Score: 0})
	} else {
		board.Reasons = append(board.Reasons, personalReason{Text: "Açıkça belirttiğin ilgi alanları haber önceliğine ağırlık katıyor.", Score: 100})
		if len(board.Topics) > 0 {
			board.Reasons = append(board.Reasons, personalReason{Text: "İlgi alanlarınla eşleşen ve önem skoru yüksek olaylar öne çıkarılıyor.", Score: board.Topics[0].Score})
		}
		if len(board.Entities) > 0 {
			board.Reasons = append(board.Reasons, personalReason{Text: "İlgili olaylarda tekrar eden kişi, kurum, ülke ve şirketler ayrıca ağırlıklandırılıyor.", Score: board.Entities[0].Score})
		}
	}
	return json.Marshal(board)
}

func normalizePersonal(total, hits int) int {
	if hits <= 0 {
		return 0
	}
	v := total / hits
	if v > 100 {
		v = 100
	}
	return v
}

func personalStoryScore(st *Story, interests []string) int {
	if st == nil {
		return 0
	}
	low := strings.ToLower(st.Title + " " + st.Category)
	hits := 0
	for _, interest := range interests {
		if interest != "" && strings.Contains(low, interest) {
			hits++
		}
	}
	v := st.Score + hits*12
	if v > 100 {
		v = 100
	}
	return v
}
