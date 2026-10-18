package core

// Olay / konu takibi.
//
// RC83: Bir olayın kimliği (EventID) başlık kelimelerinden üretildiği için olay
// büyüdükçe temsilci başlık değişir ve kimlik kayabilir. Eski sürümde yalnızca
// kimlik saklandığından "Olayı takip et" pratikte hiçbir şey yapmıyordu: takip
// edilen olay bir sonraki yenilemede kayboluyor, hiçbir yerde listelenmiyor ve
// bildirim üretmiyordu. Artık her takip; başlık, bağlantı, kategori ve kelime
// imzasıyla saklanır, güncel akıştaki olayla bulanık eşleştirilir, "Takip
// ettiklerim" sayfasında listelenir ve yeni kaynak geldiğinde bildirim verir.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type followedEvent struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Link     string   `json:"link"`
	Category string   `json:"category"`
	Tokens   []string `json:"tokens,omitempty"`
	Sources  int      `json:"sources"`  // bağımsız kaynak sayısı (son görülen)
	Count    int      `json:"count"`    // toplam başlık sayısı (son görülen)
	Seen     int      `json:"seen"`     // kullanıcının en son gördüğü başlık sayısı
	Added    int64    `json:"added"`    // takibe alınma zamanı
	LastSeen int64    `json:"lastSeen"` // akışta en son eşleştiği zaman
}

type followFile struct {
	Events map[string]bool           `json:"events"` // eski biçim (yalnız kimlik)
	Items  map[string]*followedEvent `json:"items"`
	Topics map[string]bool           `json:"topics"`
}

type followStore struct {
	mu     sync.Mutex
	items  map[string]*followedEvent
	topics map[string]bool
	path   string
}

func newFollowStore() *followStore {
	f := &followStore{items: map[string]*followedEvent{}, topics: map[string]bool{}, path: filepath.Join(filepath.Dir(configPath()), "follows.json")}
	if b, err := os.ReadFile(f.path); err == nil {
		var d followFile
		if json.Unmarshal(b, &d) == nil {
			for id, fe := range d.Items {
				if fe != nil && id != "" {
					fe.ID = id
					f.items[id] = fe
				}
			}
			// Eski kayıtlar: yalnızca kimlik var; başlık ilk eşleşmede dolar.
			for id, on := range d.Events {
				if on && id != "" && f.items[id] == nil {
					f.items[id] = &followedEvent{ID: id, Added: time.Now().Unix()}
				}
			}
			if d.Topics != nil {
				f.topics = d.Topics
			}
		}
	}
	return f
}

func (f *followStore) saveLocked() {
	_ = os.MkdirAll(filepath.Dir(f.path), 0o755)
	b, _ := json.MarshalIndent(followFile{Items: f.items, Topics: f.topics}, "", "  ")
	_ = atomicWriteFile(f.path, b)
}

func storyTokens(st *Story) []string {
	toks := tokenize(st.Title)
	out := make([]string, 0, len(toks))
	for t := range toks {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// toggleStory follows/unfollows the story; returns true when now followed.
func (f *followStore) toggleStory(st *Story) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id := f.matchLocked(st); id != "" {
		delete(f.items, id)
		f.saveLocked()
		return false
	}
	now := time.Now().Unix()
	f.items[st.EventID] = &followedEvent{
		ID: st.EventID, Title: st.Title, Link: st.Link, Category: st.Category,
		Tokens: storyTokens(st), Sources: st.IndependentSources, Count: st.Count, Seen: st.Count,
		Added: now, LastSeen: now,
	}
	f.saveLocked()
	return true
}

// toggleEvent keeps the old id-only API (used when the story is not in cache).
func (f *followStore) toggleEvent(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.items[id] != nil {
		delete(f.items, id)
		f.saveLocked()
		return false
	}
	f.items[id] = &followedEvent{ID: id, Added: time.Now().Unix()}
	f.saveLocked()
	return true
}

func (f *followStore) isEvent(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.items[id] != nil
}

// isStory reports whether the story matches any followed event.
func (f *followStore) isStory(st *Story) bool {
	if f == nil || st == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.matchLocked(st) != ""
}

// matchLocked finds the followed record this story continues (same id, same
// link, or a strongly overlapping headline in the same category).
func (f *followStore) matchLocked(st *Story) string {
	if st == nil {
		return ""
	}
	if f.items[st.EventID] != nil {
		return st.EventID
	}
	var stTok map[string]bool
	for id, fe := range f.items {
		if fe.Link != "" {
			if fe.Link == st.Link {
				return id
			}
			for _, it := range st.items {
				if it.Link == fe.Link {
					return id
				}
			}
		}
		if len(fe.Tokens) == 0 || (fe.Category != "" && !strings.EqualFold(fe.Category, st.Category)) {
			continue
		}
		if stTok == nil {
			stTok = tokenize(st.Title)
		}
		ft := make(map[string]bool, len(fe.Tokens))
		for _, t := range fe.Tokens {
			ft[t] = true
		}
		if c, n := overlapCoef(ft, stTok); n >= 3 && c >= 0.6 {
			return id
		}
		if jaccard(ft, stTok) >= 0.5 {
			return id
		}
	}
	return ""
}

func (f *followStore) count() int {
	if f == nil {
		return 0
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items)
}

func normalizeTopic(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
func (f *followStore) toggleTopic(topic string) bool {
	topic = normalizeTopic(topic)
	if topic == "" {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.topics[topic] {
		delete(f.topics, topic)
		f.saveLocked()
		return false
	}
	f.topics[topic] = true
	f.saveLocked()
	return true
}
func (f *followStore) isTopic(topic string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.topics[normalizeTopic(topic)]
}

// ---- store side -----------------------------------------------------------

type followUpdate struct {
	Link  string
	Title string
	Added int // new headlines since the last notification
}

// syncFollows matches every followed record against the current stories,
// refreshes their metadata and returns the ones that grew (for toasts).
func (s *Store) syncFollows() []followUpdate {
	if follows == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	follows.mu.Lock()
	defer follows.mu.Unlock()
	if len(follows.items) == 0 {
		return nil
	}
	var ups []followUpdate
	changed := false
	now := time.Now().Unix()
	for _, cl := range s.storiesByCat {
		for _, st := range cl {
			id := follows.matchLocked(st)
			if id == "" {
				continue
			}
			fe := follows.items[id]
			if fe.Title == "" { // eski kayıt: ilk eşleşmede doldur
				fe.Title, fe.Link, fe.Category, fe.Count, fe.Seen = st.Title, st.Link, st.Category, st.Count, st.Count
				fe.Tokens = storyTokens(st)
			}
			if st.Count > fe.Count {
				ups = append(ups, followUpdate{Link: st.Link, Title: s.trTitle(st.Title), Added: st.Count - fe.Count})
				fe.Count = st.Count
			}
			fe.Sources = st.IndependentSources
			fe.Link = st.Link // en güncel temsilci haberi aç
			fe.LastSeen = now
			changed = true
		}
	}
	if changed {
		follows.saveLocked()
	}
	return ups
}

type outFollowed struct {
	Item     outItem `json:"item"`
	Live     bool    `json:"live"`     // still in the current feed
	New      int     `json:"new"`      // headlines added since last opened
	Added    int64   `json:"added"`    // followed at
	LastSeen int64   `json:"lastSeen"` // last matched
}

// followedJSON lists followed events, live ones first, newest activity first.
func (s *Store) followedJSON() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	type rec struct {
		fe followedEvent
		st *Story
	}
	var recs []rec
	follows.mu.Lock()
	found := map[string]*Story{}
	for _, cl := range s.storiesByCat {
		for _, st := range cl {
			if id := follows.matchLocked(st); id != "" {
				if old := found[id]; old == nil || st.Count > old.Count {
					found[id] = st
				}
			}
		}
	}
	for id, fe := range follows.items {
		recs = append(recs, rec{fe: *fe, st: found[id]})
	}
	follows.mu.Unlock() // storyOut → isStory takes the lock again

	out := make([]outFollowed, 0, len(recs))
	for _, r := range recs {
		fe := r.fe
		o := outFollowed{Added: fe.Added, LastSeen: fe.LastSeen}
		if st := r.st; st != nil {
			o.Item = s.storyOut(st)
			o.Item.Followed = true
			o.Live = true
			if st.Count > fe.Seen {
				o.New = st.Count - fe.Seen
			}
		} else {
			if fe.Title == "" {
				continue
			}
			o.Item = outItem{Title: s.trTitle(fe.Title), Link: fe.Link, Category: fe.Category, TS: fe.LastSeen, Followed: true, SourceCount: fe.Sources}
		}
		out = append(out, o)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Live != out[j].Live {
			return out[i].Live
		}
		return out[i].Item.TS > out[j].Item.TS
	})
	return json.Marshal(out)
}

// markFollowedSeen clears the "new" badge of every followed event.
func (s *Store) markFollowedSeen() {
	follows.mu.Lock()
	defer follows.mu.Unlock()
	for _, fe := range follows.items {
		fe.Seen = fe.Count
	}
	follows.saveLocked()
}

// unfollowLink removes the followed record whose last link matches.
func (f *followStore) unfollowLink(link string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, fe := range f.items {
		if fe.Link == link || id == link {
			delete(f.items, id)
		}
	}
	f.saveLocked()
}
