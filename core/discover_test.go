package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoverViaLinkTag(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(rss2Sample))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head>
		<link rel="alternate" type="application/rss+xml" title="Ana Akış" href="/feed.xml">
		</head><body>site</body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	b := discoverJSON(context.Background(), srv.URL)
	s := string(b)
	if !strings.Contains(s, "feed.xml") {
		t.Fatalf("link-tag feed not discovered: %s", s)
	}
	if !strings.Contains(s, "Örnek Kaynak") { // validated feed title wins over hint
		t.Errorf("feed not validated/titled: %s", s)
	}
}

func TestDiscoverViaCommonPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(rss2Sample))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head></head><body>no hints</body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	b := discoverJSON(context.Background(), srv.URL)
	if !strings.Contains(string(b), "/feed") {
		t.Fatalf("common-path feed not discovered: %s", string(b))
	}
}

func TestDiscoverNone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>nothing here</body></html>`))
	}))
	defer srv.Close()
	b := discoverJSON(context.Background(), srv.URL)
	if !strings.Contains(string(b), "bulunamadı") {
		t.Errorf("expected not-found message: %s", string(b))
	}
}
