package core

// Persistent feed cache and source-health repository.
//
// Phase 1 deliberately keeps the storage backend dependency-free so the core
// can be built with CGO_ENABLED=0 as a single Windows executable. The data
// model is repository-shaped and can be moved to SQLite in a later release
// without changing the fetcher or UI contract.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type FeedCacheEntry struct {
	URL          string    `json:"url"`
	Category     string    `json:"category"`
	Name         string    `json:"name"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
	FetchedAt    time.Time `json:"fetchedAt"`
	Items        []Item    `json:"items"`
}

type SourceHealth struct {
	URL             string    `json:"url"`
	Name            string    `json:"name"`
	Status          string    `json:"status"` // healthy, stale, error
	LastAttempt     time.Time `json:"lastAttempt"`
	LastSuccess     time.Time `json:"lastSuccess"`
	LastChange      time.Time `json:"lastChange"`
	ConsecutiveFail int       `json:"consecutiveFail"`
	HTTPStatus      int       `json:"httpStatus"`
	LatencyMS       int64     `json:"latencyMs"`
	ItemCount       int       `json:"itemCount"`
	ParseErrors     int       `json:"parseErrors"`
	LastError       string    `json:"lastError,omitempty"`
}

type persistentCache struct {
	Version int              `json:"version"`
	SavedAt time.Time        `json:"savedAt"`
	Feeds   []FeedCacheEntry `json:"feeds"`
	Health  []SourceHealth   `json:"health"`
}

const cacheSchema = 1

type diskRepository struct {
	mu     sync.Mutex
	path   string
	feeds  map[string]FeedCacheEntry
	health map[string]SourceHealth
}

func newDiskRepository() *diskRepository {
	dir := appDataDir()
	r := &diskRepository{
		path:   filepath.Join(dir, "feed-cache.json"),
		feeds:  map[string]FeedCacheEntry{},
		health: map[string]SourceHealth{},
	}
	r.load()
	return r
}

func (r *diskRepository) load() {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, err := os.ReadFile(r.path)
	if err != nil {
		return
	}
	var p persistentCache
	if json.Unmarshal(b, &p) != nil || p.Version != cacheSchema {
		return
	}
	for _, f := range p.Feeds {
		if f.URL != "" {
			r.feeds[f.URL] = f
		}
	}
	for _, h := range p.Health {
		if h.URL != "" {
			r.health[h.URL] = h
		}
	}
}

func (r *diskRepository) snapshot() (map[string]FeedCacheEntry, map[string]SourceHealth) {
	r.mu.Lock()
	defer r.mu.Unlock()
	feeds := make(map[string]FeedCacheEntry, len(r.feeds))
	for k, v := range r.feeds {
		v.Items = append([]Item(nil), v.Items...)
		feeds[k] = v
	}
	health := make(map[string]SourceHealth, len(r.health))
	for k, v := range r.health {
		health[k] = v
	}
	return feeds, health
}

func (r *diskRepository) putFeed(entry FeedCacheEntry) {
	r.mu.Lock()
	r.feeds[entry.URL] = entry
	r.mu.Unlock()
}

func (r *diskRepository) putHealth(h SourceHealth) {
	r.mu.Lock()
	r.health[h.URL] = h
	r.mu.Unlock()
}

func (r *diskRepository) persist() error {
	r.mu.Lock()
	p := persistentCache{Version: cacheSchema, SavedAt: time.Now()}
	p.Feeds = make([]FeedCacheEntry, 0, len(r.feeds))
	for _, f := range r.feeds {
		p.Feeds = append(p.Feeds, f)
	}
	p.Health = make([]SourceHealth, 0, len(r.health))
	for _, h := range r.health {
		p.Health = append(p.Health, h)
	}
	r.mu.Unlock()

	sort.Slice(p.Feeds, func(i, j int) bool { return p.Feeds[i].URL < p.Feeds[j].URL })
	sort.Slice(p.Health, func(i, j int) bool { return p.Health[i].URL < p.Health[j].URL })
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(r.path, data)
}

func atomicWriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		// Windows does not replace an existing destination with Rename.
		// The temporary file still prevents a partial JSON document at the destination.
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			_ = os.Remove(tmp)
			return err
		}
		if retryErr := os.Rename(tmp, path); retryErr != nil {
			_ = os.Remove(tmp)
			return retryErr
		}
	}
	return nil
}
