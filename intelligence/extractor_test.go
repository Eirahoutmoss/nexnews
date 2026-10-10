package intelligence

import (
	"strings"
	"testing"
	"time"
)

func sourceArticleForExtraction() Article {
	published := time.Date(2026, 10, 10, 7, 0, 0, 0, time.UTC)
	return Article{
		ID: "article-extract-1",
		CanonicalURL: "https://news.example.org/bridge",
		PublisherSourceID: "source-example",
		Title: "Authorities said the bridge reopened after repairs",
		Summary: "Authorities said the bridge reopened after repairs. Officials reported that repairs cost 2.5 million lira. This statement is retained as the source wording.",
		PublishedAt: &published,
		RetrievedAt: published.Add(time.Hour),
		FirstObservedAt: published.Add(time.Hour),
		LastObservedAt: published.Add(time.Hour),
		ContentAvailability: ContentTitleSummary,
	}
}

func TestExtractSourceStatementsKeepsLiteralTextAndExactOffsets(t *testing.T) {
	a := sourceArticleForExtraction()
	result, err := ExtractSourceStatements(a, "Example News", []TextExcerpt{
		{Field: "title", Text: a.Title},
		{Field: "summary", Text: a.Summary},
	}, a.RetrievedAt)
	if err != nil {
		t.Fatalf("ExtractSourceStatements failed: %v", err)
	}
	if len(result.Claims) != 3 {
		t.Fatalf("expected duplicate title/summary sentence to collapse and three statements to remain; got %d (%+v)", len(result.Claims), result.Claims)
	}
	if len(result.Evidence) != len(result.Claims) {
		t.Fatalf("each statement should have one directly linked evidence excerpt: claims=%d evidence=%d", len(result.Claims), len(result.Evidence))
	}
	if result.Claims[0].Kind != ClaimSourceStatement || result.Claims[0].Status != ClaimReported {
		t.Fatalf("source text must remain a reported statement candidate, got kind=%q status=%q", result.Claims[0].Kind, result.Claims[0].Status)
	}
	if result.Evidence[0].Stance != EvidenceReports && result.Evidence[0].Stance != EvidenceSourceText {
		t.Fatalf("literal source wording must not be marked as factual support, got %q", result.Evidence[0].Stance)
	}
	for _, e := range result.Evidence {
		if e.StartOffset == nil || e.EndOffset == nil {
			t.Fatalf("expected byte offsets: %+v", e)
		}
		var original string
		switch e.ExcerptField {
		case "title":
			original = a.Title
		case "summary":
			original = a.Summary
		default:
			t.Fatalf("unexpected field %q", e.ExcerptField)
		}
		if *e.StartOffset < 0 || *e.EndOffset > len(original) || original[*e.StartOffset:*e.EndOffset] != e.Quote {
			t.Fatalf("evidence quote does not match exact source offsets: field=%s quote=%q offsets=[%d,%d)", e.ExcerptField, e.Quote, *e.StartOffset, *e.EndOffset)
		}
		if e.SourceURL != a.CanonicalURL || e.ArticleID != a.ID {
			t.Fatalf("evidence provenance mismatch: %+v", e)
		}
	}
	if strings.Contains(result.Claims[0].StatusReason, "doğruluğu") == false {
		t.Fatal("claim status should clearly state that factual truth was not assessed")
	}
}

func TestExtractSourceStatementsRejectsTextNotMatchingArticle(t *testing.T) {
	a := sourceArticleForExtraction()
	_, err := ExtractSourceStatements(a, "Example News", []TextExcerpt{
		{Field: "summary", Text: "This text was not stored on this source record."},
	}, a.RetrievedAt)
	if err == nil {
		t.Fatal("expected non-verbatim summary text to be rejected")
	}
}

func TestExtractSourceStatementsDoesNotPretendSummaryIsFullText(t *testing.T) {
	a := sourceArticleForExtraction()
	_, err := ExtractSourceStatements(a, "Example News", []TextExcerpt{
		{Field: "full_text", Text: a.Summary},
	}, a.RetrievedAt)
	if err == nil || !strings.Contains(err.Error(), "full_text") {
		t.Fatalf("expected full-text access guard, got %v", err)
	}
}

func TestExtractSourceStatementsSkipsNoiseAndIsDeterministic(t *testing.T) {
	a := sourceArticleForExtraction()
	a.Summary = "OK. Officials confirmed that the north bridge will reopen on Friday."
	first, err := ExtractSourceStatements(a, "Example News", []TextExcerpt{{Field: "summary", Text: a.Summary}}, a.RetrievedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ExtractSourceStatements(a, "Example News", []TextExcerpt{{Field: "summary", Text: a.Summary}}, a.RetrievedAt.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if first.Skipped == 0 {
		t.Fatal("expected short noise fragment to be skipped")
	}
	if len(first.Claims) != 1 || len(second.Claims) != 1 {
		t.Fatalf("expected one meaningful source statement, got %d and %d", len(first.Claims), len(second.Claims))
	}
	if first.Claims[0].ID != second.Claims[0].ID || first.Evidence[0].ID != second.Evidence[0].ID {
		t.Fatal("record identity should be deterministic for identical article text and offsets")
	}
	if first.Claims[0].Status == ClaimCorroborated {
		t.Fatal("one sentence from one source must not be promoted to corroborated")
	}
}

func TestValidateClaimRejectsClaimWithoutTraceableEvidence(t *testing.T) {
	claim := Claim{
		ID: "claim-1", Kind: ClaimSourceStatement, Predicate: "source_statement",
		ClaimText: "The council approved a plan after a long public hearing.",
		Status: ClaimReported, EvidenceIDs: []string{"missing-evidence"},
	}
	if err := ValidateClaim(claim, map[string]Evidence{}); err == nil {
		t.Fatal("expected missing evidence reference to be rejected")
	}
}
