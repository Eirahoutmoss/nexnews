package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type followFile struct {
	Events map[string]bool `json:"events"`
	Topics map[string]bool `json:"topics"`
}

type followStore struct {
	mu     sync.Mutex
	events map[string]bool
	topics map[string]bool
	path   string
}

func newFollowStore() *followStore {
	f := &followStore{events: map[string]bool{}, topics: map[string]bool{}, path: filepath.Join(filepath.Dir(configPath()), "follows.json")}
	if b, err := os.ReadFile(f.path); err == nil {
		var d followFile
		if json.Unmarshal(b, &d) == nil {
			if d.Events != nil {
				f.events = d.Events
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
	b, _ := json.MarshalIndent(followFile{Events: f.events, Topics: f.topics}, "", "  ")
	_ = os.WriteFile(f.path, b, 0o644)
}

func (f *followStore) toggleEvent(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.events[id] {
		delete(f.events, id)
		f.saveLocked()
		return false
	}
	f.events[id] = true
	f.saveLocked()
	return true
}
func (f *followStore) isEvent(id string) bool { f.mu.Lock(); defer f.mu.Unlock(); return f.events[id] }
func normalizeTopic(s string) string          { return strings.ToLower(strings.TrimSpace(s)) }
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
