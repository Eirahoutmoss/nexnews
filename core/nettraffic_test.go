package core

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTrafficMeterWindow(t *testing.T) {
	m := newTrafficMeter()
	now := time.Unix(1_800_000_000, 0)
	m.now = func() time.Time { return now }
	m.add(1000, 100, false)
	now = now.Add(2 * time.Second)
	m.add(500, 0, true)
	d, u, w := m.sum(trafficWindow, false)
	if d != 1500 || u != 100 || w != 500 {
		t.Fatalf("sum=%d/%d/%d", d, u, w)
	}
	now = now.Add(time.Duration(trafficWindow+5) * time.Second)
	d, _, _ = m.sum(trafficWindow, false)
	if d != 0 {
		t.Fatalf("old buckets must expire, got %d", d)
	}
	if m.totalDown.Load() != 1500 {
		t.Fatalf("session total lost")
	}
}

func TestCountingTransportCountsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 10_000))
	}))
	defer srv.Close()
	before := traffic.totalDown.Load()
	c := &http.Client{}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if got := traffic.totalDown.Load() - before; got < 10_000 {
		t.Fatalf("expected >=10000 counted, got %d", got)
	}
}
