package core

// Network traffic meter. Every HTTP request NexNews makes goes through
// countingTransport (it replaces http.DefaultTransport at start-up and wraps
// the few clients with their own transport), so we know exactly how many bytes
// the app pulled from / pushed to the internet. Bytes are kept in one-second
// buckets; the UI shows the live rate and the total of the last 5 minutes.
// Traffic the WebView itself makes (images, map tiles, browser-side
// translation) is reported from the page via nnAddWebTraffic.

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const trafficWindow = 5 * 60 // seconds kept for the rolling total

type trafficBucket struct {
	sec      int64
	down, up int64
	webDown  int64
}

type trafficMeter struct {
	mu        sync.Mutex
	buckets   [trafficWindow + 2]trafficBucket
	totalDown atomic.Int64
	totalUp   atomic.Int64
	started   time.Time
	now       func() time.Time
}

var traffic = newTrafficMeter()

func newTrafficMeter() *trafficMeter {
	return &trafficMeter{started: time.Now(), now: time.Now}
}

func (m *trafficMeter) bucket(sec int64) *trafficBucket {
	b := &m.buckets[sec%int64(len(m.buckets))]
	if b.sec != sec {
		*b = trafficBucket{sec: sec}
	}
	return b
}

func (m *trafficMeter) add(down, up int64, web bool) {
	if down <= 0 && up <= 0 {
		return
	}
	sec := m.now().Unix()
	m.mu.Lock()
	b := m.bucket(sec)
	b.down += down
	b.up += up
	if web {
		b.webDown += down
	}
	m.mu.Unlock()
	m.totalDown.Add(down)
	m.totalUp.Add(up)
}

// sum totals the buckets of the last n seconds (excluding the running second
// when skipCurrent is set, so the live rate is not biased low).
func (m *trafficMeter) sum(n int64, skipCurrent bool) (down, up, web int64) {
	now := m.now().Unix()
	from, to := now-n+1, now
	if skipCurrent {
		from, to = now-n, now-1
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, b := range m.buckets {
		if b.sec >= from && b.sec <= to {
			down += b.down
			up += b.up
			web += b.webDown
		}
	}
	return
}

type netStats struct {
	AppDownBps  int64  `json:"appDownBps"` // NexNews download rate (bytes/s, last 3 s)
	AppUpBps    int64  `json:"appUpBps"`
	Down5m      int64  `json:"down5m"` // bytes NexNews downloaded in the last 5 min
	Up5m        int64  `json:"up5m"`
	Web5m       int64  `json:"web5m"`       // part of Down5m reported by the page itself
	SessionDown int64  `json:"sessionDown"` // since start-up
	SessionUp   int64  `json:"sessionUp"`
	SessionSecs int64  `json:"sessionSecs"`
	SysDownBps  int64  `json:"sysDownBps"` // whole computer, active adapter (-1 = unknown)
	SysUpBps    int64  `json:"sysUpBps"`
	Adapter     string `json:"adapter,omitempty"`
	LinkMbps    int64  `json:"linkMbps,omitempty"`
	Online      bool   `json:"online"`
}

func (m *trafficMeter) stats() netStats {
	d3, u3, _ := m.sum(3, true)
	d5, u5, w5 := m.sum(trafficWindow, false)
	s := netStats{
		AppDownBps:  d3 / 3,
		AppUpBps:    u3 / 3,
		Down5m:      d5,
		Up5m:        u5,
		Web5m:       w5,
		SessionDown: m.totalDown.Load(),
		SessionUp:   m.totalUp.Load(),
		SessionSecs: int64(m.now().Sub(m.started).Seconds()),
		SysDownBps:  -1,
		SysUpBps:    -1,
	}
	if a := currentAdapter(); a.ok {
		s.SysDownBps, s.SysUpBps = a.downBps, a.upBps
		s.Adapter, s.LinkMbps, s.Online = a.name, a.linkMbps, true
	}
	return s
}

func netStatsJSON() ([]byte, error) { return json.Marshal(traffic.stats()) }

// ---- counting transport --------------------------------------------------

type countingTransport struct{ base http.RoundTripper }

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	up := int64(len(req.Method) + len(req.URL.RequestURI()) + 12 + len(req.Host))
	for k, vs := range req.Header {
		for _, v := range vs {
			up += int64(len(k) + len(v) + 4)
		}
	}
	if req.ContentLength > 0 {
		up += req.ContentLength
	}
	traffic.add(0, up+64, false) // +64: implicit headers (UA, Accept-Encoding, TLS record overhead)
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	hdr := int64(len(resp.Status) + 12)
	for k, vs := range resp.Header {
		for _, v := range vs {
			hdr += int64(len(k) + len(v) + 4)
		}
	}
	traffic.add(hdr, 0, false)
	if resp.Body != nil {
		resp.Body = &countingBody{rc: resp.Body}
	}
	return resp, nil
}

type countingBody struct{ rc io.ReadCloser }

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.rc.Read(p)
	if n > 0 {
		traffic.add(int64(n), 0, false)
	}
	return n, err
}

func (b *countingBody) Close() error { return b.rc.Close() }

var installTrafficOnce sync.Once

// installTrafficMeter wraps http.DefaultTransport so every client that does
// not set its own Transport is counted.
func installTrafficMeter() {
	installTrafficOnce.Do(func() {
		if _, ok := http.DefaultTransport.(*countingTransport); !ok {
			http.DefaultTransport = &countingTransport{base: http.DefaultTransport}
		}
		startAdapterSampler()
	})
}

// countTransport makes sure a client with a custom transport is counted too.
func countTransport(c *http.Client) {
	if c == nil {
		return
	}
	switch c.Transport.(type) {
	case *countingTransport:
		return
	case nil:
		c.Transport = &countingTransport{base: http.DefaultTransport}
	default:
		c.Transport = &countingTransport{base: c.Transport}
	}
}

func init() { installTrafficMeter() }
