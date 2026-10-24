package core

// "Sen bakmayalı ne değişti?" (RC97).
//
// Bir olayı takip eden kişi her açışında aynı başlıkları baştan okumak zorunda
// kalmamalı. Olay sayfası her açıldığında ziyaret zamanı saklanır; bir sonraki
// açılışta önceki ziyaretten sonra gelen başlıklar işaretlenir. Aynı oturumda
// arka arkaya açılışlar (ör. panelin yeniden çizilmesi) yeni ziyaret sayılmaz.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	visitCap       = 600
	visitSameVisit = 10 * time.Minute // bu süre içindeki yeniden açılışlar aynı ziyaret
)

type visitRec struct {
	Prev int64 `json:"p"` // önceki ziyaretin zamanı (karşılaştırma tabanı)
	Cur  int64 `json:"c"` // son açılış
}

type visitStore struct {
	mu    sync.Mutex
	m     map[string]*visitRec
	path  string
	saved time.Time
	dirty bool
}

var visits = &visitStore{}

func visitKey(st *Story) string {
	if st == nil {
		return ""
	}
	if st.EventID != "" {
		return st.EventID
	}
	return st.Link
}

func (v *visitStore) loadLocked() {
	if v.m != nil {
		return
	}
	v.m = map[string]*visitRec{}
	v.path = filepath.Join(appDataDir(), "visits.json")
	if b, err := os.ReadFile(v.path); err == nil {
		_ = json.Unmarshal(b, &v.m)
		if v.m == nil {
			v.m = map[string]*visitRec{}
		}
	}
}

// touch: olay açıldı. Karşılaştırma için önceki ziyaretin zamanını döner
// (ilk ziyarette 0).
func (v *visitStore) touch(key string) int64 {
	if key == "" {
		return 0
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.loadLocked()
	now := time.Now().Unix()
	r := v.m[key]
	if r == nil {
		v.m[key] = &visitRec{Cur: now}
		v.trimLocked()
		v.saveLocked()
		return 0
	}
	if now-r.Cur > int64(visitSameVisit/time.Second) {
		r.Prev = r.Cur
	}
	r.Cur = now
	v.saveLocked()
	return r.Prev
}

// lastSeen: olayın en son ne zaman açıldığı (akış rozetleri için), yoksa 0.
func (v *visitStore) lastSeen(key string) int64 {
	if key == "" {
		return 0
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.loadLocked()
	if r := v.m[key]; r != nil {
		return r.Cur
	}
	return 0
}

func (v *visitStore) trimLocked() {
	if len(v.m) <= visitCap {
		return
	}
	// En eski ziyaretleri at.
	type kv struct {
		k string
		t int64
	}
	all := make([]kv, 0, len(v.m))
	for k, r := range v.m {
		all = append(all, kv{k, r.Cur})
	}
	for len(v.m) > visitCap*9/10 {
		oi := 0
		for i := range all {
			if all[i].t < all[oi].t {
				oi = i
			}
		}
		delete(v.m, all[oi].k)
		all[oi] = all[len(all)-1]
		all = all[:len(all)-1]
	}
}

func (v *visitStore) saveLocked() {
	v.dirty = true
	if time.Since(v.saved) < 20*time.Second {
		return
	}
	v.flushLocked()
}

func (v *visitStore) flushLocked() {
	if !v.dirty || v.m == nil {
		return
	}
	if b, err := json.Marshal(v.m); err == nil {
		_ = os.MkdirAll(filepath.Dir(v.path), 0o755)
		if atomicWriteFile(v.path, b) == nil {
			v.saved, v.dirty = time.Now(), false
		}
	}
}

// flushVisits: kapanışta bekleyen ziyaretleri yazar.
func flushVisits() {
	visits.mu.Lock()
	defer visits.mu.Unlock()
	visits.flushLocked()
}
