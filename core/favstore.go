package core

// Favorites and reading history, persisted as small JSON files next to the
// config. History is capped; favorites are unbounded (user-curated).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type savedItem struct {
	Link   string `json:"link"`
	Title  string `json:"title"`
	Source string `json:"source"`
	Cat    string `json:"cat,omitempty"`
	TS     int64  `json:"ts"`    // story time
	Added  int64  `json:"added"` // when favorited / read
}

type favStore struct {
	mu       sync.Mutex
	favs     []savedItem
	hist     []savedItem
	favPath  string
	histPath string
}

const historyCap = 200

func newFavStore() *favStore {
	dir := filepath.Dir(configPath())
	f := &favStore{
		favPath:  filepath.Join(dir, "favorites.json"),
		histPath: filepath.Join(dir, "history.json"),
	}
	f.favs = loadItems(f.favPath)
	f.hist = loadItems(f.histPath)
	return f
}

func loadItems(p string) []savedItem {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var out []savedItem
	_ = json.Unmarshal(data, &out)
	return out
}

func saveItems(p string, items []savedItem) {
	if data, err := json.MarshalIndent(items, "", "  "); err == nil {
		_ = os.WriteFile(p, data, 0o644)
	}
}

// toggleFav adds the item if absent, removes it if present; returns true if now
// favorited.
func (f *favStore) toggleFav(it savedItem) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, x := range f.favs {
		if x.Link == it.Link {
			f.favs = append(f.favs[:i], f.favs[i+1:]...)
			saveItems(f.favPath, f.favs)
			return false
		}
	}
	it.Added = time.Now().Unix()
	f.favs = append([]savedItem{it}, f.favs...)
	saveItems(f.favPath, f.favs)
	return true
}

func (f *favStore) isFav(link string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.favs {
		if x.Link == link {
			return true
		}
	}
	return false
}

func (f *favStore) favLinks() map[string]bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := make(map[string]bool, len(f.favs))
	for _, x := range f.favs {
		m[x.Link] = true
	}
	return m
}

// addHistory records an opened article (most-recent first, deduped, capped).
func (f *favStore) addHistory(it savedItem) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, x := range f.hist {
		if x.Link == it.Link {
			f.hist = append(f.hist[:i], f.hist[i+1:]...)
			break
		}
	}
	it.Added = time.Now().Unix()
	f.hist = append([]savedItem{it}, f.hist...)
	if len(f.hist) > historyCap {
		f.hist = f.hist[:historyCap]
	}
	saveItems(f.histPath, f.hist)
}

func (f *favStore) listJSON(which string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := f.favs
	if which == "history" {
		items = f.hist
	}
	if items == nil {
		items = []savedItem{}
	}
	b, _ := json.Marshal(items)
	return b
}

func (f *favStore) clearHistory() {
	f.mu.Lock()
	f.hist = nil
	saveItems(f.histPath, f.hist)
	f.mu.Unlock()
}

// readStats returns the set of read links and, for reads newer than window,
// how many stories were read per category.
func (f *favStore) readStats(window time.Duration) (map[string]bool, map[string]int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	read := make(map[string]bool, len(f.hist))
	byCat := map[string]int{}
	cut := time.Now().Add(-window).Unix()
	for _, h := range f.hist {
		read[h.Link] = true
		if h.Added >= cut && h.Cat != "" {
			byCat[h.Cat]++
		}
	}
	return read, byCat
}
