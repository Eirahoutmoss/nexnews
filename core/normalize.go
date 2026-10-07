package core

// Normalizer: turns a headline into a language-independent token set used for
// near-duplicate detection. Latin text splits on word boundaries (with a small
// stopword list); scripts without spaces (CJK) fall back to character bigrams so
// the same event clusters regardless of language.

import (
	"strings"
	"sync/atomic"
	"unicode"
)

// small multilingual stopword set; only high-frequency glue words.
var stopwords = map[string]bool{
	// Turkish
	"ve": true, "ile": true, "bir": true, "bu": true, "için": true, "ama": true,
	"da": true, "de": true, "ki": true, "mi": true, "çok": true, "daha": true,
	"sonra": true, "önce": true, "yeni": true, "oldu": true, "olan": true,
	// English
	"the": true, "a": true, "an": true, "of": true, "to": true, "in": true,
	"on": true, "for": true, "and": true, "is": true, "are": true, "as": true,
	"at": true, "by": true, "with": true, "after": true, "new": true, "says": true,
	"over": true, "from": true,
}

// tokenize returns the set of normalized tokens for a title.
func tokenize(title string) map[string]bool {
	title = strings.ToLower(strings.TrimSpace(title))
	set := map[string]bool{}

	var cur strings.Builder
	hasSpaceScript := false
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		w := cur.String()
		cur.Reset()
		if stopwords[w] {
			return
		}
		// keep tokens of length >= 3, or any token containing a digit (e.g. "6.8").
		if len([]rune(w)) >= 3 || hasDigit(w) {
			set[w] = true
		}
	}

	var cjk []rune
	for _, r := range title {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if isCJK(r) {
				cjk = append(cjk, r)
				flush() // break any latin run
			} else {
				hasSpaceScript = true
				cur.WriteRune(r)
			}
		default:
			flush()
		}
	}
	flush()

	// character bigrams for CJK runs
	for i := 0; i+1 < len(cjk); i++ {
		set[string(cjk[i:i+2])] = true
	}
	if len(cjk) == 1 {
		set[string(cjk)] = true
	}

	_ = hasSpaceScript
	return set
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func isCJK(r rune) bool {
	return (r >= 0x3040 && r <= 0x30ff) || // Hiragana/Katakana
		(r >= 0x3400 && r <= 0x4dbf) || // CJK ext A
		(r >= 0x4e00 && r <= 0x9fff) || // CJK unified
		(r >= 0xac00 && r <= 0xd7a3) // Hangul syllables
}

// overlapCoef returns (|A∩B| / min(|A|,|B|), |A∩B|). The overlap coefficient is
// more robust than Jaccard for same-event headlines of different lengths across
// outlets, where one title carries extra words the other lacks.
func overlapCoef(a, b map[string]bool) (float64, int) {
	if len(a) == 0 || len(b) == 0 {
		return 0, 0
	}
	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}
	inter := 0
	for t := range small {
		if large[t] {
			inter++
		}
	}
	return float64(inter) / float64(len(small)), inter
}

// jaccard returns the similarity of two token sets in [0,1].
func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	// iterate the smaller set
	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}
	for t := range small {
		if large[t] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// stemTokens folds inflected forms onto a shared prefix so that agglutinative
// Turkish ("deprem", "depremde", "depremin") and English plurals/tenses match
// during clustering. Tokens longer than stemLen runes are cut to stemLen;
// numbers and CJK bigrams are left untouched.
const stemLen = 6

func stemTokens(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for t := range in {
		r := []rune(t)
		if len(r) > stemLen && !hasDigit(t) && !isCJK(r[0]) {
			t = string(r[:stemLen])
		}
		out[t] = true
	}
	return out
}

// clusterTitleFn optionally maps a headline to the text used for clustering
// (the cached translation in the reading language), so the same event reported
// in different languages lands in one story. nil = original title.
var clusterTitleFn atomic.Value // func(string) string

func clusterText(title string) string {
	if f, ok := clusterTitleFn.Load().(func(string) string); ok && f != nil {
		if t := f(title); t != "" {
			return t
		}
	}
	return title
}

// clusterTokens is the token set used for near-duplicate detection.
func clusterTokens(title string) map[string]bool {
	return stemTokens(tokenize(clusterText(title)))
}
