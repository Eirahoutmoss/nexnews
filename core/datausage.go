package core

// Veri modu ve kullanım defteri (RC83).
//
//   - Tam veri akışı: NexNews olduğu gibi çalışır.
//   - Sınırlı veri akışı: NexNews'in indirme hızı ayarlanan değerle sınırlanır
//     (varsayılan 256 KB/s), canlı TV en çok 480p'ye düşürülür, otomatik yenileme
//     seyrekleşir ve arayüz küçük görselleri yüklemez.
//
// Her indirilen/yüklenen bayt günlük olarak usage.json dosyasına yazılır;
// NexNews'in açık kaldığı süre de gün gün tutulur. Böylece oturum, bugün, bu
// hafta, bu ay ve kurulumdan bu yana toplam trafik maliyeti görülebilir.
// İsteğe bağlı aylık kota dolunca uygulama kendiliğinden sınırlı moda geçer.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	dataModeFull    = "tam"
	dataModeLimited = "sinirli"
	defaultLimitKBs = 256
)

var (
	dataLimitedFlag atomic.Bool
	dataLimitBps    atomic.Int64 // bytes per second while limited
)

func dataLimited() bool { return dataLimitedFlag.Load() }

// applyDataMode sets the runtime limiter from the config values.
func applyDataMode(mode string, kbps int) {
	if kbps < 32 || kbps > 100000 {
		kbps = defaultLimitKBs
	}
	dataLimitBps.Store(int64(kbps) * 1024)
	dataLimitedFlag.Store(mode == dataModeLimited)
}

// ---- download limiter ---------------------------------------------------------

type byteLimiter struct {
	mu   sync.Mutex
	next time.Time
}

var downLimiter byteLimiter

// wait blocks long enough that the average download rate stays at the limit.
// One second of burst is allowed so short requests are not delayed at all.
func (l *byteLimiter) wait(n int) {
	bps := dataLimitBps.Load()
	if n <= 0 || bps <= 0 {
		return
	}
	d := time.Duration(float64(n) / float64(bps) * float64(time.Second))
	l.mu.Lock()
	now := time.Now()
	if l.next.Before(now.Add(-time.Second)) {
		l.next = now.Add(-time.Second)
	}
	l.next = l.next.Add(d)
	sleep := l.next.Sub(now)
	l.mu.Unlock()
	if sleep > 0 {
		time.Sleep(sleep)
	}
}

// ---- usage ledger ---------------------------------------------------------------

type usageDay struct {
	Down int64 `json:"down"`
	Up   int64 `json:"up"`
	Secs int64 `json:"secs"` // NexNews'in açık kaldığı süre
}

type usageFile struct {
	Since int64               `json:"since"` // ilk kayıt (kurulum)
	Days  map[string]usageDay `json:"days"`  // "2006-01-02"
	// Kota uyarılarının aynı ay içinde tekrar etmemesi için.
	WarnedMonth string `json:"warnedMonth,omitempty"`
	CapMonth    string `json:"capMonth,omitempty"`
}

type usageLedger struct {
	mu    sync.Mutex
	data  usageFile
	dirty bool
	path  string
	now   func() time.Time
}

var usage = newUsageLedger()

func newUsageLedger() *usageLedger {
	return &usageLedger{data: usageFile{Days: map[string]usageDay{}}, now: time.Now}
}

func (u *usageLedger) load() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.path = filepath.Join(filepath.Dir(configPath()), "usage.json")
	if b, err := os.ReadFile(u.path); err == nil {
		var d usageFile
		if json.Unmarshal(b, &d) == nil && d.Days != nil {
			u.data = d
		}
	}
	if u.data.Since == 0 {
		u.data.Since = u.now().Unix()
		u.dirty = true
	}
}

func dayKey(t time.Time) string { return t.Format("2006-01-02") }

func (u *usageLedger) add(down, up int64) {
	if down <= 0 && up <= 0 {
		return
	}
	u.mu.Lock()
	k := dayKey(u.now())
	d := u.data.Days[k]
	d.Down += down
	d.Up += up
	u.data.Days[k] = d
	u.dirty = true
	u.mu.Unlock()
}

func (u *usageLedger) addSecs(s int64) {
	u.mu.Lock()
	k := dayKey(u.now())
	d := u.data.Days[k]
	d.Secs += s
	u.data.Days[k] = d
	u.dirty = true
	u.mu.Unlock()
}

func (u *usageLedger) flush() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.dirty || u.path == "" {
		return
	}
	// 400 günden eski günleri tek bir "önceki" kaydında topla.
	if len(u.data.Days) > 400 {
		keys := make([]string, 0, len(u.data.Days))
		for k := range u.data.Days {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		old := u.data.Days["0000-00-00"]
		for _, k := range keys[:len(keys)-400] {
			if k == "0000-00-00" {
				continue
			}
			d := u.data.Days[k]
			old.Down, old.Up, old.Secs = old.Down+d.Down, old.Up+d.Up, old.Secs+d.Secs
			delete(u.data.Days, k)
		}
		u.data.Days["0000-00-00"] = old
	}
	b, err := json.Marshal(u.data)
	if err == nil {
		_ = os.MkdirAll(filepath.Dir(u.path), 0o755)
		if atomicWriteFile(u.path, b) == nil {
			u.dirty = false
		}
	}
}

func (u *usageLedger) reset() {
	u.mu.Lock()
	u.data = usageFile{Since: u.now().Unix(), Days: map[string]usageDay{}}
	u.dirty = true
	u.mu.Unlock()
	u.flush()
}

type usagePeriod struct {
	Down int64 `json:"down"`
	Up   int64 `json:"up"`
	Secs int64 `json:"secs"`
}

type usageDayOut struct {
	Day  string `json:"day"`
	Down int64  `json:"down"`
	Secs int64  `json:"secs"`
}

type usageStats struct {
	Mode        string        `json:"mode"`
	LimitKBps   int           `json:"limitKBps"`
	QuotaMB     int           `json:"quotaMB"`
	Session     usagePeriod   `json:"session"`
	Today       usagePeriod   `json:"today"`
	Week        usagePeriod   `json:"week"`  // Pazartesiden bu yana
	Month       usagePeriod   `json:"month"` // ayın 1'inden bu yana
	Last30      usagePeriod   `json:"last30"`
	All         usagePeriod   `json:"all"`
	Since       int64         `json:"since"`
	DailyAvg    int64         `json:"dailyAvg"` // açık kalınan günlerin ortalaması
	Days        []usageDayOut `json:"days"`     // son 30 gün (grafik)
	QuotaUsed   float64       `json:"quotaUsed"`
	AutoLimited bool          `json:"autoLimited"`
}

func (u *usageLedger) stats() usageStats {
	now := u.now()
	today := dayKey(now)
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	weekStart := dayKey(now.AddDate(0, 0, -(wd - 1)))
	monthStart := now.Format("2006-01") + "-01"
	last30 := dayKey(now.AddDate(0, 0, -29))

	u.mu.Lock()
	defer u.mu.Unlock()
	var s usageStats
	s.Since = u.data.Since
	active := 0
	for k, d := range u.data.Days {
		p := usagePeriod{Down: d.Down, Up: d.Up, Secs: d.Secs}
		addP(&s.All, p)
		if k == today {
			addP(&s.Today, p)
		}
		if k >= weekStart && k <= today {
			addP(&s.Week, p)
		}
		if k >= monthStart && k <= today {
			addP(&s.Month, p)
		}
		if k >= last30 && k <= today {
			addP(&s.Last30, p)
		}
		if k != "0000-00-00" && (d.Down > 0 || d.Secs > 0) {
			active++
		}
	}
	if active > 0 {
		s.DailyAvg = s.All.Down / int64(active)
	}
	for i := 29; i >= 0; i-- {
		k := dayKey(now.AddDate(0, 0, -i))
		d := u.data.Days[k]
		s.Days = append(s.Days, usageDayOut{Day: k, Down: d.Down, Secs: d.Secs})
	}
	s.AutoLimited = u.data.CapMonth == now.Format("2006-01")
	return s
}

func addP(dst *usagePeriod, p usagePeriod) {
	dst.Down += p.Down
	dst.Up += p.Up
	dst.Secs += p.Secs
}

// quotaCheck returns "warn" at 80 % and "cap" at 100 % of the monthly quota,
// once per month each.
func (u *usageLedger) quotaCheck(quotaMB int) string {
	if quotaMB <= 0 {
		return ""
	}
	st := u.stats()
	used := float64(st.Month.Down+st.Month.Up) / float64(int64(quotaMB)*1024*1024)
	month := u.now().Format("2006-01")
	u.mu.Lock()
	defer u.mu.Unlock()
	switch {
	case used >= 1 && u.data.CapMonth != month:
		u.data.CapMonth, u.dirty = month, true
		return "cap"
	case used >= 0.8 && u.data.WarnedMonth != month:
		u.data.WarnedMonth, u.dirty = month, true
		return "warn"
	}
	return ""
}

func usageStatsJSON() ([]byte, error) {
	s := usage.stats()
	cfg.mu.RLock()
	s.Mode, s.LimitKBps, s.QuotaMB = cfg.DataMode, cfg.DataLimitKBps, cfg.DataQuotaMB
	cfg.mu.RUnlock()
	if s.Mode == "" {
		s.Mode = dataModeFull
	}
	s.Session = usagePeriod{Down: traffic.totalDown.Load(), Up: traffic.totalUp.Load(), Secs: int64(time.Since(traffic.started).Seconds())}
	if s.QuotaMB > 0 {
		s.QuotaUsed = float64(s.Month.Down+s.Month.Up) / float64(int64(s.QuotaMB)*1024*1024)
	}
	return json.Marshal(s)
}

// startUsageLedger loads usage.json, counts open time and flushes periodically.
// onQuota is called (from the ticker goroutine) when a quota threshold is hit.
func startUsageLedger(onQuota func(kind string)) {
	usage.load()
	go func() {
		const step = 30
		t := time.NewTicker(step * time.Second)
		defer t.Stop()
		for range t.C {
			usage.addSecs(step)
			cfg.mu.RLock()
			q := cfg.DataQuotaMB
			cfg.mu.RUnlock()
			if k := usage.quotaCheck(q); k != "" && onQuota != nil {
				onQuota(k)
			}
			usage.flush()
		}
	}()
}
