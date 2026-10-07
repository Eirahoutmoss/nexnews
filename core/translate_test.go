package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseTranslate(t *testing.T) {
	in := []byte(`[[["Merhaba dünya","Hello world",null,null,10]],null,"en"]`)
	out, err := parseTranslate(in)
	if err != nil || out != "Merhaba dünya" {
		t.Fatalf("got %q err %v", out, err)
	}
	multi := []byte(`[[["A ","a"],["B","b"]],null,"en"]`)
	out, err = parseTranslate(multi)
	if err != nil || out != "A B" {
		t.Fatalf("multi: got %q err %v", out, err)
	}
	if _, err := parseTranslate([]byte(`garbage`)); err == nil {
		t.Errorf("expected error on garbage")
	}
}

func TestLocalLibreTranslate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/translate" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"translatedText":"Merhaba dünya"}`))
	}))
	defer srv.Close()

	tr := newTranslator()
	tr.localURL = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, via, err := tr.fetch(ctx, "Hello world", "tr")
	if err != nil {
		t.Fatalf("local fetch failed: %v", err)
	}
	if out != "Merhaba dünya" || via != "Local LibreTranslate" {
		t.Fatalf("got out=%q via=%q", out, via)
	}
}
