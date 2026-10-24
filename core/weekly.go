package core

// Haftalık özet (RC95).
//
// Sakin görünümün haftalık karşılığı: her yenilemeden sonra önemli olaylar
// küçük bir günlüğe (weekly.json) yazılır; son 7 günün en önemli olayları,
// bölgelere dağılımı ve takip edilen olayların durumu tek sayfada özetlenir.
// Okumayı alışkanlık yapar ama bağımlılık yaratmaz: sonu belli, kısa bir sayfa.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type weekEntry struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Link     string `json:"link"`
	Category string `json:"category"`
	Score    int    `json:"score"`
	Sources  int    `json:"sources"`
	Breaking bool   `json:"breaking,omitempty"`
	First    int64  `json:"first"`
	Last     int64  `json:"last"`
}

type weekFile struct {
	Since   int64                 `json:"since"`
	Entries map[string]*weekEntry `json:"entries"`
}

var (
	weekMu    sync.Mutex
	weekData  *weekFile
	weekSaved time.Time
)

func weeklyPath() string { return filepath.Join(appDataDir(), "weekly.json") }

func weekLoad() *weekFile {
	if weekData != nil {
		return weekData
	}
	weekData = &weekFile{Entries: map[string]*weekEntry{}}
	if b, err := os.ReadFile(weeklyPath()); err == nil {
		_ = json.Unmarshal(b, weekData)
		if weekData.Entries == nil {
			weekData.Entries = map[string]*weekEntry{}
		}
	}
	if weekData.Since == 0 {
		weekData.Since = time.Now().Unix()
	}
	return weekData
}

// recordWeekly: güncel akıştaki önemli olayları günlüğe işler (yenilemeden sonra çağrılır).
func recordWeekly(s *Store) {
	if s == nil {
		return
	}
	now := time.Now()
	type snap struct {
		e   weekEntry
		key string
	}
	var picks []snap
	s.mu.RLock()
	for cat, list := range s.storiesByCat {
		if strings.HasPrefix(cat, "⚡") { // otomatik uyarı kategorileri (afet akışları) hariç
			continue
		}
		for _, st := range list {
			if st.Score < 65 && !st.Breaking && st.IndependentSources < 3 {
				continue
			}
			key := st.EventID
			if key == "" {
				key = st.Link
			}
			if key == "" {
				continue
			}
			first := st.FirstSeen.Unix()
			if !st.HasDate || first <= 0 {
				first = now.Unix()
			}
			picks = append(picks, snap{key: key, e: weekEntry{ID: st.EventID, Title: s.trTitle(st.Title), Link: st.Link, Category: st.Category,
				Score: st.Score, Sources: st.IndependentSources, Breaking: st.Breaking, First: first, Last: now.Unix()}})
		}
	}
	s.mu.RUnlock()

	weekMu.Lock()
	defer weekMu.Unlock()
	w := weekLoad()
	for _, p := range picks {
		if old, ok := w.Entries[p.key]; ok {
			if p.e.Score > old.Score {
				old.Score = p.e.Score
			}
			if p.e.Sources > old.Sources {
				old.Sources = p.e.Sources
			}
			old.Title, old.Link, old.Last = p.e.Title, p.e.Link, p.e.Last
			old.Breaking = old.Breaking || p.e.Breaking
			continue
		}
		e := p.e
		w.Entries[p.key] = &e
	}
	cut := now.Add(-8 * 24 * time.Hour).Unix()
	for k, e := range w.Entries {
		if e.Last < cut {
			delete(w.Entries, k)
		}
	}
	if time.Since(weekSaved) > 5*time.Minute {
		if b, err := json.Marshal(w); err == nil {
			_ = atomicWriteFile(weeklyPath(), b)
			weekSaved = time.Now()
		}
	}
}

type weekRegion struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type weekFollow struct {
	Title    string `json:"title"`
	Link     string `json:"link"`
	Sources  int    `json:"sources"`
	Count    int    `json:"count"`
	New      int    `json:"new"`
	LastSeen int64  `json:"lastSeen"`
}

type weeklyOut struct {
	From     int64        `json:"from"`
	To       int64        `json:"to"`
	Days     int          `json:"days"` // günlüğün kapsadığı gün sayısı (7'den azsa özet kısmi)
	Total    int          `json:"total"`
	Top      []weekEntry  `json:"top"`
	Regions  []weekRegion `json:"regions"`
	Breaking int          `json:"breaking"`
	Follows  []weekFollow `json:"follows"`
}

func weeklyJSON() ([]byte, error) {
	now := time.Now()
	from := now.Add(-7 * 24 * time.Hour).Unix()
	weekMu.Lock()
	w := weekLoad()
	var list []weekEntry
	for _, e := range w.Entries {
		if e.Last >= from {
			list = append(list, *e)
		}
	}
	since := w.Since
	weekMu.Unlock()

	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return a.Sources > b.Sources
	})
	out := weeklyOut{From: from, To: now.Unix(), Total: len(list)}
	days := int(now.Sub(time.Unix(since, 0)).Hours()/24) + 1
	if days > 7 {
		days = 7
	}
	out.Days = days
	reg := map[string]int{}
	for _, e := range list {
		reg[e.Category]++
		if e.Breaking {
			out.Breaking++
		}
	}
	for k, v := range reg {
		out.Regions = append(out.Regions, weekRegion{k, v})
	}
	sort.Slice(out.Regions, func(i, j int) bool { return out.Regions[i].Count > out.Regions[j].Count })
	// Aynı bölgeden en çok 4 olay: hafta tek bir gündeme boğulmasın.
	perCat := map[string]int{}
	for _, e := range list {
		if len(out.Top) >= 12 {
			break
		}
		if perCat[e.Category] >= 4 {
			continue
		}
		perCat[e.Category]++
		out.Top = append(out.Top, e)
	}
	if follows != nil {
		follows.mu.Lock()
		for _, f := range follows.items {
			if f == nil {
				continue
			}
			n := f.Count - f.Seen
			if n < 0 {
				n = 0
			}
			out.Follows = append(out.Follows, weekFollow{Title: f.Title, Link: f.Link, Sources: f.Sources, Count: f.Count, New: n, LastSeen: f.LastSeen})
		}
		follows.mu.Unlock()
		sort.Slice(out.Follows, func(i, j int) bool { return out.Follows[i].LastSeen > out.Follows[j].LastSeen })
	}
	return json.Marshal(out)
}
