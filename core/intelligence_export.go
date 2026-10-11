package core

// Intelligence export types form a read-only boundary between NexNews' feed
// cache and consumers such as the separate NexNews Intelligence package.
// They intentionally do not expose mutable Store maps or Story internals.

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"time"
)

type IntelligenceArticleSnapshot struct {
	ItemID        string     `json:"itemId,omitempty"`
	CanonicalURL  string     `json:"canonicalUrl"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary,omitempty"`
	SourceName    string     `json:"sourceName"`
	FeedURL       string     `json:"feedUrl,omitempty"`
	PublishedAt   *time.Time `json:"publishedAt,omitempty"`
	RetrievedAt   time.Time  `json:"retrievedAt"`
	Language      string     `json:"language,omitempty"`
	Category      string     `json:"category,omitempty"`
	Breaking      bool       `json:"breaking,omitempty"`
}

type IntelligenceEventSnapshot struct {
	CandidateID          string     `json:"candidateId"`
	Title                string     `json:"title"`
	Category             string     `json:"category,omitempty"`
	Lifecycle            string     `json:"lifecycle,omitempty"`
	ArticleURLs          []string   `json:"articleUrls"`
	SourceNames          []string   `json:"sourceNames,omitempty"`
	IndependentSources   int        `json:"independentSources"`
	PrimarySources       int        `json:"primarySources"`
	ImportanceScore      int        `json:"importanceScore"`
	Breaking             bool       `json:"breaking,omitempty"`
	FirstPublishedAt     *time.Time `json:"firstPublishedAt,omitempty"`
	LastPublishedAt      *time.Time `json:"lastPublishedAt,omitempty"`
}

type IntelligenceSnapshot struct {
	CapturedAt time.Time                       `json:"capturedAt"`
	FeedCount  int                             `json:"feedCount"`
	Articles   []IntelligenceArticleSnapshot   `json:"articles"`
	Events     []IntelligenceEventSnapshot     `json:"events"`
}

// IntelligenceSnapshot returns a consistent, read-only view of cached feed
// items and current NexNews event candidates. It performs no network requests,
// applies no personal hide filters, and does not change feed-cache state.
func (s *Store) IntelligenceSnapshot() IntelligenceSnapshot {
	now := time.Now().UTC()
	out := IntelligenceSnapshot{
		CapturedAt: now,
		Articles:   []IntelligenceArticleSnapshot{},
		Events:     []IntelligenceEventSnapshot{},
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	feedURLs := make([]string, 0, len(s.cache))
	for feedURL := range s.cache {
		feedURLs = append(feedURLs, feedURL)
	}
	sort.Strings(feedURLs)
	out.FeedCount = len(feedURLs)

	articleIndex := make(map[string]int)
	for _, feedURL := range feedURLs {
		entry := s.cache[feedURL]
		retrievedAt := entry.FetchedAt
		if retrievedAt.IsZero() {
			retrievedAt = now
		}
		for _, it := range entry.Items {
			u := canonicalURL(it.Link)
			title := strings.TrimSpace(it.Title)
			if u == "" || title == "" {
				continue
			}
			sourceName := strings.TrimSpace(it.Source)
			if sourceName == "" {
				sourceName = strings.TrimSpace(entry.Name)
			}
			if sourceName == "" {
				sourceName = feedURL
			}
			var published *time.Time
			if it.HasDate && !it.Published.IsZero() {
				t := it.Published.UTC()
				published = &t
			}
			if idx, ok := articleIndex[u]; ok {
				// The same article can be cached through multiple category/feed
				// paths. Keep one document, enriching it only with available
				// metadata; never count duplicate cache entries as new articles.
				old := &out.Articles[idx]
				if old.Summary == "" && strings.TrimSpace(it.Summary) != "" {
					old.Summary = strings.TrimSpace(it.Summary)
				}
				if old.SourceName == "" && sourceName != "" {
					old.SourceName = sourceName
					old.FeedURL = feedURL
				}
				if old.PublishedAt == nil && published != nil {
					old.PublishedAt = published
				}
				if old.Category == "" && strings.TrimSpace(it.Category) != "" {
					old.Category = strings.TrimSpace(it.Category)
				}
				old.Breaking = old.Breaking || it.Breaking
				if retrievedAt.After(old.RetrievedAt) {
					old.RetrievedAt = retrievedAt
				}
				continue
			}
			articleIndex[u] = len(out.Articles)
			out.Articles = append(out.Articles, IntelligenceArticleSnapshot{
				ItemID: it.ID, CanonicalURL: u, Title: title,
				Summary: strings.TrimSpace(it.Summary), SourceName: sourceName,
				FeedURL: feedURL, PublishedAt: published, RetrievedAt: retrievedAt,
				Category: strings.TrimSpace(it.Category), Breaking: it.Breaking,
			})
		}
	}

	seenEvents := map[string]bool{}
	for _, st := range s.important {
		if st == nil || strings.TrimSpace(st.Title) == "" {
			continue
		}
		candidateID := strings.TrimSpace(st.EventID)
		if candidateID == "" {
			candidateID = "candidate:" + canonicalURL(st.Link)
		}
		if candidateID == "candidate:" || seenEvents[candidateID] {
			continue
		}
		seenEvents[candidateID] = true

		event := IntelligenceEventSnapshot{
			CandidateID: candidateID, Title: strings.TrimSpace(st.Title),
			Category: strings.TrimSpace(st.Category), Lifecycle: strings.TrimSpace(st.Lifecycle),
			ArticleURLs: []string{}, SourceNames: append([]string(nil), st.Sources...),
			IndependentSources: st.IndependentSources, PrimarySources: st.PrimarySources,
			ImportanceScore: st.Score, Breaking: st.Breaking,
		}
		seenURLs := map[string]bool{}
		var first, last time.Time
		for _, it := range st.items {
			u := canonicalURL(it.Link)
			if u == "" {
				continue
			}
			if !seenURLs[u] {
				seenURLs[u] = true
				event.ArticleURLs = append(event.ArticleURLs, u)
			}
			if !it.HasDate || it.Published.IsZero() {
				continue
			}
			t := it.Published.UTC()
			if first.IsZero() || t.Before(first) {
				first = t
			}
			if last.IsZero() || t.After(last) {
				last = t
			}
		}
		sort.Strings(event.ArticleURLs)
		sort.Strings(event.SourceNames)
		if !first.IsZero() {
			event.FirstPublishedAt = &first
		}
		if !last.IsZero() {
			event.LastPublishedAt = &last
		}
		out.Events = append(out.Events, event)
	}

	sort.Slice(out.Articles, func(i, j int) bool {
		return out.Articles[i].CanonicalURL < out.Articles[j].CanonicalURL
	})
	sort.Slice(out.Events, func(i, j int) bool {
		return out.Events[i].CandidateID < out.Events[j].CandidateID
	})
	return out
}


// CurrentIntelligenceSnapshot returns the active process-wide store's current
// read-only snapshot. It is safe to call before initialization and performs no
// network requests.
func CurrentIntelligenceSnapshot() IntelligenceSnapshot {
	if store == nil {
		return IntelligenceSnapshot{
			CapturedAt: time.Now().UTC(),
			Articles: []IntelligenceArticleSnapshot{},
			Events: []IntelligenceEventSnapshot{},
		}
	}
	return store.IntelligenceSnapshot()
}


// IntelligenceSnapshotFromFeedCacheFile reads NexNews' existing persistent
// feed-cache.json and constructs the same read-only snapshot without requiring
// an intermediate export file or making any network request. Event clusters
// are marked as NexNews candidates by the downstream adapter, not as verified
// Intelligence findings.
func IntelligenceSnapshotFromFeedCacheFile(path string) (IntelligenceSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return IntelligenceSnapshot{}, err
	}
	var cached persistentCache
	if err := json.Unmarshal(data, &cached); err != nil {
		return IntelligenceSnapshot{}, err
	}
	if cached.Version != cacheSchema {
		return IntelligenceSnapshot{}, errors.New("unsupported NexNews feed-cache schema")
	}
	captured := cached.SavedAt.UTC()
	if captured.IsZero() {
		captured = time.Now().UTC()
	}
	feedMap := make(map[string]FeedCacheEntry, len(cached.Feeds))
	byCategory := map[string][]Item{}
	for _, feed := range cached.Feeds {
		if strings.TrimSpace(feed.URL) == "" {
			continue
		}
		if feed.FetchedAt.IsZero() {
			feed.FetchedAt = captured
		}
		for i := range feed.Items {
			if strings.TrimSpace(feed.Items[i].Source) == "" {
				feed.Items[i].Source = strings.TrimSpace(feed.Name)
			}
			if strings.TrimSpace(feed.Items[i].Category) == "" {
				feed.Items[i].Category = strings.TrimSpace(feed.Category)
			}
			byCategory[feed.Items[i].Category] = append(byCategory[feed.Items[i].Category], feed.Items[i])
		}
		feedMap[feed.URL] = feed
	}
	var stories []*Story
	categories := make([]string, 0, len(byCategory))
	for category := range byCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		items := dedupeItems(byCategory[category])
		sortItemsNewest(items)
		cluster := clusterItems(items)
		rankStories(cluster)
		stories = append(stories, cluster...)
	}
	rankStories(stories)
	// Match Store.rebuildStories: when the same cached article appears in more
	// than one feed category, retain only the highest-ranked candidate. Without
	// this step the offline cache importer creates duplicate candidates and
	// every extracted statement becomes ambiguously assigned.
	stories = dedupeAcrossCategories(stories)
	s := &Store{cache: feedMap, important: stories}
	out := s.IntelligenceSnapshot()
	out.CapturedAt = captured
	return out, nil
}
