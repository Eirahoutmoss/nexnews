package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexnews/intelligence"
)

func TestEventDossierAPIReturnsScopedReadOnlyDossier(t *testing.T) {
	h := newHandler(testRepository(t))
	req := httptest.NewRequest(http.MethodGet, "/api/event-dossier?eventId=e1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET event dossier status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("event dossier must not be cached")
	}
	var dossier intelligence.EventDossier
	if err := json.Unmarshal(rec.Body.Bytes(), &dossier); err != nil {
		t.Fatal(err)
	}
	if dossier.Event.ID != "e1" || dossier.ArticleCount != 1 || len(dossier.Articles) != 1 {
		t.Fatalf("unexpected event dossier: %+v", dossier)
	}
	if dossier.Articles[0].ID != "a1" || dossier.Articles[0].CanonicalURL != "https://example.org/news" {
		t.Fatalf("dossier did not retain article provenance: %+v", dossier.Articles)
	}
}

func TestEventDossierAPIRejectsMissingUnknownIDsAndWrites(t *testing.T) {
	h := newHandler(testRepository(t))
	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/api/event-dossier", http.StatusBadRequest},
		{http.MethodGet, "/api/event-dossier?eventId=missing", http.StatusNotFound},
		{http.MethodPost, "/api/event-dossier?eventId=e1", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/event-dossier/extra?eventId=e1", http.StatusNotFound},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("%s %s status=%d, want %d: %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}
