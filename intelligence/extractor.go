package intelligence

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const StatementExtractorVersion = "literal-segmenter/1"

// TextExcerpt identifies a verbatim region retrieved from a specific article.
// For title/summary fields, Text must match the corresponding Article field.
// full_text is allowed only when the article explicitly records full-text
// availability and the caller passes text actually obtained from that article.
type TextExcerpt struct {
	Field string `json:"field"`
	Text  string `json:"text"`
}

type ExtractionResult struct {
	ArticleID string     `json:"articleId"`
	Claims    []Claim    `json:"claims"`
	Evidence  []Evidence `json:"evidence"`
	Skipped   int        `json:"skipped"`
	Warnings  []string   `json:"warnings,omitempty"`
}

type sentenceRange struct {
	start int // byte offset, inclusive
	end   int // byte offset, exclusive
	text  string
}

// splitSourceSentences produces exact, byte-addressable segments. It is not a
// language model and does not paraphrase or decide whether a sentence is true.
// It avoids splitting decimal numbers and dots that are not followed by a
// boundary, but deliberately remains conservative for punctuation-heavy prose.
func splitSourceSentences(input string) []sentenceRange {
	var out []sentenceRange
	start := 0
	flush := func(end int) {
		if end <= start {
			start = end
			return
		}
		left, right := start, end
		for left < right {
			r, size := utf8.DecodeRuneInString(input[left:right])
			if !unicode.IsSpace(r) {
				break
			}
			left += size
		}
		for right > left {
			r, size := utf8.DecodeLastRuneInString(input[left:right])
			if !unicode.IsSpace(r) {
				break
			}
			right -= size
		}
		if left < right {
			out = append(out, sentenceRange{start: left, end: right, text: input[left:right]})
		}
		start = end
	}

	for i, r := range input {
		size := utf8.RuneLen(r)
		end := i + size
		if r == '\n' || r == '\r' {
			flush(i)
			start = end
			continue
		}
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		// A decimal point between digits is not a sentence boundary.
		if r == '.' && i > 0 && end < len(input) {
			prev, _ := utf8.DecodeLastRuneInString(input[:i])
			next, _ := utf8.DecodeRuneInString(input[end:])
			if unicode.IsDigit(prev) && unicode.IsDigit(next) {
				continue
			}
		}
		if end == len(input) {
			flush(end)
			start = end
			continue
		}
		next, _ := utf8.DecodeRuneInString(input[end:])
		if unicode.IsSpace(next) {
			flush(end)
			start = end
		}
	}
	if start < len(input) {
		flush(len(input))
	}
	return out
}

func normalizeStatement(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
	// Punctuation is retained in Evidence. For duplicate detection only, remove
	// terminal sentence punctuation so a headline and the same summary sentence
	// are not extracted twice just because the headline omits a final full stop.
	s = strings.TrimRight(s, ".!?;:,")
	return s
}

func meaningfulSourceStatement(s string) bool {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) < 18 {
		return false
	}
	letters := 0
	for _, r := range runes {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	// This is a noise filter, not a truth or grammar classifier.
	return letters >= 12
}

// ExtractSourceStatements creates literal, source-attributed statement
// candidates from exact text excerpts. The claim text is never normalized into
// a new proposition: it retains the source wording verbatim. Each evidence
// record points to the same exact quote and byte offsets. The result explicitly
// means "this source said/wrote this", not "this is independently verified".
func ExtractSourceStatements(article Article, sourceName string, excerpts []TextExcerpt, observedAt time.Time) (ExtractionResult, error) {
	if err := ValidateArticle(article); err != nil {
		return ExtractionResult{}, fmt.Errorf("invalid article: %w", err)
	}
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	} else {
		observedAt = observedAt.UTC()
	}
	out := ExtractionResult{
		ArticleID: article.ID,
		Claims:    []Claim{},
		Evidence:  []Evidence{},
	}
	seenStatement := map[string]bool{}
	seenField := map[string]bool{}

	for _, excerpt := range excerpts {
		field := strings.TrimSpace(excerpt.Field)
		text := excerpt.Text
		switch field {
		case "title":
			if text != article.Title {
				return ExtractionResult{}, errors.New("title excerpt must exactly match the stored article title")
			}
		case "summary":
			if article.ContentAvailability == ContentTitleOnly {
				return ExtractionResult{}, errors.New("summary excerpt is not allowed for a title-only article")
			}
			if text != article.Summary {
				return ExtractionResult{}, errors.New("summary excerpt must exactly match the stored article summary")
			}
		case "full_text":
			if article.ContentAvailability != ContentFullText {
				return ExtractionResult{}, errors.New("full-text excerpt requires article contentAvailability=full_text")
			}
			if strings.TrimSpace(text) == "" {
				out.Warnings = append(out.Warnings, "Full-text field was empty; no statement candidates were extracted.")
				continue
			}
		default:
			return ExtractionResult{}, fmt.Errorf("unsupported excerpt field %q", field)
		}
		if seenField[field] {
			return ExtractionResult{}, fmt.Errorf("excerpt field %q was provided more than once", field)
		}
		seenField[field] = true

		for _, part := range splitSourceSentences(text) {
			key := normalizeStatement(part.text)
			if key == "" || seenStatement[key] {
				continue
			}
			if !meaningfulSourceStatement(part.text) {
				out.Skipped++
				continue
			}
			seenStatement[key] = true

			claimID := stableID("claim", article.ID+"\x00"+key)
			evidenceID := stableID("evidence", article.ID+"\x00"+field+"\x00"+fmt.Sprint(part.start)+"\x00"+fmt.Sprint(part.end)+"\x00"+part.text)
			level := article.ContentAvailability
			if field == "full_text" {
				level = ContentFullText
			}
			start, end := part.start, part.end
			evidence := Evidence{
				ID: evidenceID, ArticleID: article.ID, ClaimID: claimID,
				Stance: EvidenceReports, ExcerptField: field, Quote: part.text,
				StartOffset: &start, EndOffset: &end,
				SourceURL: article.CanonicalURL, CapturedAt: observedAt,
				ContentLevel: level, ExtractionMethod: "literal-sentence-segmentation",
				ExtractionVersion: StatementExtractorVersion,
			}
			claim := Claim{
				ID: claimID, Kind: ClaimSourceStatement,
				Predicate: "source_statement", ObjectText: part.text, ClaimText: part.text,
				AttributedTo: strings.TrimSpace(sourceName), FirstSeenAt: observedAt,
				EvidenceIDs: []string{evidenceID}, Status: ClaimReported,
				StatusReason: "Birebir kaynak ifadesi. İddianın doğruluğu, bağlamı ve bağımsız teyidi bu işlemde değerlendirilmedi.",
			}
			if err := ValidateEvidence(evidence); err != nil {
				return ExtractionResult{}, fmt.Errorf("generated evidence failed validation: %w", err)
			}
			if err := ValidateClaim(claim, map[string]Evidence{evidence.ID: evidence}); err != nil {
				return ExtractionResult{}, fmt.Errorf("generated source statement failed validation: %w", err)
			}
			out.Claims = append(out.Claims, claim)
			out.Evidence = append(out.Evidence, evidence)
		}
	}
	return out, nil
}
