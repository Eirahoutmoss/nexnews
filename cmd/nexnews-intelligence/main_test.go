package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexnews/core"
	"nexnews/intelligence"
)

func writeSnapshotFixture(t *testing.T, dir string) string {
	t.Helper()
	captured := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	published := captured.Add(-time.Hour)
	payload := core.IntelligenceSnapshot{
		CapturedAt: captured,
		FeedCount:  1,
		Articles: []core.IntelligenceArticleSnapshot{{
			ItemID: "fixture-item",
			CanonicalURL: "https://news.example.org/turkiye-iran",
			Title: "Türkiye ve İran temsilcileri görüşmeleri sürdürdü",
			Summary: "Türkiye ve İran temsilcileri bugün bölgesel gelişmeleri değerlendirmek üzere bir araya geldi.",
			SourceName: "Example News",
			FeedURL: "https://news.example.org/rss.xml",
			PublishedAt: &published,
			RetrievedAt: captured,
			Category: "World",
		}},
		Events: []core.IntelligenceEventSnapshot{{
			CandidateID: "fixture-event",
			Title: "Türkiye ve İran temsilcileri görüşmeleri sürdürdü",
			Category: "World",
			Lifecycle: "confirmed",
			ArticleURLs: []string{"https://news.example.org/turkiye-iran"},
			SourceNames: []string{"Example News"},
			IndependentSources: 1,
		}},
	}
	content, err := json.Marshal(payload)
	if err != nil { t.Fatal(err) }
	path := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(path, content, 0o600); err != nil { t.Fatal(err) }
	return path
}

func writeCacheFixture(t *testing.T, dir string) string {
	t.Helper()
	cache := map[string]any{
		"version": 1,
		"savedAt": "2026-10-10T12:00:00Z",
		"feeds": []any{
			map[string]any{
				"url": "https://news.example.org/rss.xml",
				"category": "World",
				"name": "Example News",
				"fetchedAt": "2026-10-10T12:00:00Z",
				"items": []any{
					map[string]any{
						"id": "fixture-item",
						"title": "Türkiye ve İran temsilcileri görüşmeleri sürdürdü",
						"link": "https://news.example.org/turkiye-iran",
						"canonical": "https://news.example.org/turkiye-iran",
						"source": "Example News",
						"category": "World",
						"published": "2026-10-10T11:00:00Z",
						"hasDate": true,
						"summary": "Türkiye ve İran temsilcileri bugün bölgesel gelişmeleri değerlendirmek üzere bir araya geldi.",
					},
				},
			},
		},
		"health": []any{},
	}
	content, err := json.Marshal(cache)
	if err != nil { t.Fatal(err) }
	path := filepath.Join(dir, "feed-cache.json")
	if err := os.WriteFile(path, content, 0o600); err != nil { t.Fatal(err) }
	return path
}

func assertSuccessfulReport(t *testing.T, stdout *bytes.Buffer, databasePath string) {
	t.Helper()
	var report commandReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode command report: %v; output=%s", err, stdout.String())
	}
	if !report.OK || report.Sync.ArticlesSaved != 1 {
		t.Fatalf("unexpected import report: %+v", report)
	}
	if report.Analysis.CompletedAt == nil || report.Analysis.Partial {
		t.Fatalf("analysis did not finish cleanly: %+v", report.Analysis)
	}
	if report.Analysis.Statements.ClaimsCreated == 0 || report.Analysis.Statements.EvidenceCreated == 0 {
		t.Fatalf("expected literal source statements with evidence: %+v", report.Analysis.Statements)
	}
	if report.Analysis.Entities.MentionsCreated < 2 {
		t.Fatalf("expected exact entity candidate mentions: %+v", report.Analysis.Entities)
	}
	db, err := intelligence.NewJSONRepository(databasePath)
	if err != nil { t.Fatal(err) }
	saved, err := db.Load()
	if err != nil { t.Fatal(err) }
	if len(saved.Relations) != 0 {
		t.Fatalf("analysis must not invent relations; got %d", len(saved.Relations))
	}
	if _, err := os.Stat(databasePath); err != nil {
		t.Fatalf("expected separate Intelligence database to be created: %v", err)
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-h"}, &stdout, &stderr)
	if code != 0 { t.Fatalf("expected help exit code 0, got %d", code) }
	if !strings.Contains(stderr.String(), "-cache") || !strings.Contains(stderr.String(), "-snapshot") {
		t.Fatalf("help should explain cache and snapshot modes: %s", stderr.String())
	}
}

func TestRunImportsAndAnalyzesSnapshot(t *testing.T) {
	dir := t.TempDir()
	snapshotPath := writeSnapshotFixture(t, dir)
	databasePath := filepath.Join(dir, "intelligence.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-snapshot", snapshotPath, "-db", databasePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got code %d; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	assertSuccessfulReport(t, &stdout, databasePath)
}

func TestRunImportsExistingNexNewsCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := writeCacheFixture(t, dir)
	databasePath := filepath.Join(dir, "intelligence.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-cache", cachePath, "-db", databasePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected success, got code %d; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	assertSuccessfulReport(t, &stdout, databasePath)
	if !strings.Contains(stdout.String(), cachePath) {
		t.Fatalf("report should identify source cache path: %s", stdout.String())
	}
}

func TestRunRejectsBothInputModes(t *testing.T) {
	dir := t.TempDir()
	snapshotPath := writeSnapshotFixture(t, dir)
	cachePath := writeCacheFixture(t, dir)
	databasePath := filepath.Join(dir, "intelligence.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-snapshot", snapshotPath, "-cache", cachePath, "-db", databasePath}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "either -snapshot or -cache") {
		t.Fatalf("expected mutually exclusive input mode error; code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunRejectsInvalidSnapshotAndDoesNotCreateDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil { t.Fatal(err) }
	databasePath := filepath.Join(dir, "intelligence.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-snapshot", path, "-db", databasePath}, &stdout, &stderr)
	if code == 0 { t.Fatal("invalid JSON must fail") }
	if _, err := os.Stat(databasePath); !os.IsNotExist(err) {
		t.Fatalf("database must not be created for invalid JSON; stat err=%v", err)
	}
}
