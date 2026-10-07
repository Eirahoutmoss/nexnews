package core

// Wire-agency awareness. Seven sites republishing the same Anadolu Ajansı or
// Reuters copy are one origin, not seven independent confirmations. Each item
// gets an "origin key": the agency when the copy is attributed to one (or the
// item comes from the agency's own domain), the outlet domain otherwise, and
// near-identical summaries are folded together as echoes of a single text.

import (
	"regexp"
	"strings"
)

type wireAgency struct {
	key, label string
	domains    []string
	pattern    *regexp.Regexp
}

var wireAgencies = []wireAgency{
	{"aa", "AA", []string{"aa.com.tr"}, regexp.MustCompile(`(?i)(\(aa\)|\baa\s*[-–—]\s|anadolu ajans[ıi])`)},
	{"dha", "DHA", []string{"dha.com.tr"}, regexp.MustCompile(`(?i)(\(dha\)|demirören haber ajans[ıi])`)},
	{"iha", "İHA", []string{"iha.com.tr"}, regexp.MustCompile(`(?i)(\((iha|İha|İHA)\)|ihlas haber ajans[ıi]|i̇hlas haber ajans)`)},
	{"anka", "ANKA", []string{"ankahaber.net"}, regexp.MustCompile(`(?i)(\(anka\)|anka haber ajans[ıi])`)},
	{"reuters", "Reuters", []string{"reuters.com"}, regexp.MustCompile(`(?i)(\(reuters\)|\breuters\s*[-–—]\s|^reuters\b)`)},
	{"ap", "AP", []string{"apnews.com", "ap.org"}, regexp.MustCompile(`(\(AP\)|\bAP\s*[—–]\s|Associated Press)`)},
	{"afp", "AFP", []string{"afp.com"}, regexp.MustCompile(`(\(AFP\)|\bAFP\s*[-–—]\s|Agence France-Presse)`)},
	{"dpa", "dpa", []string{"dpa.com"}, regexp.MustCompile(`(\(dpa\)|\bdpa\s*[-–—]\s)`)},
	{"efe", "EFE", []string{"efe.com"}, regexp.MustCompile(`(\(EFE\)|\bEFE\s*[-–—]\s)`)},
	{"ansa", "ANSA", []string{"ansa.it"}, regexp.MustCompile(`(\(ANSA\)|\bANSA\s*[-–—]\s)`)},
}

// wireOf returns the agency key/label an item's copy comes from, or "".
func wireOf(it Item) (string, string) {
	d := sourceIdentity(it)
	for _, w := range wireAgencies {
		for _, dom := range w.domains {
			if d == dom || strings.HasSuffix(d, "."+dom) {
				return w.key, w.label
			}
		}
	}
	text := it.Title + " \n" + it.Summary
	for _, w := range wireAgencies {
		if w.pattern.MatchString(text) {
			return w.key, w.label
		}
	}
	return "", ""
}

// originKeys assigns every item of a story its origin key (same order as
// c.items) and reports, per origin, which agency label (if any) it is.
func originKeys(c *Story) []string {
	keys := make([]string, len(c.items))
	type echo struct {
		key string
		tok map[string]bool
	}
	var seenText []echo
	for i, it := range c.items {
		k := sourceIdentity(it)
		if wk, _ := wireOf(it); wk != "" {
			k = "wire:" + wk
		} else if tok := tokenize(it.Summary); len(tok) >= 12 {
			// Same body text under two mastheads = one origin.
			matched := false
			for _, e := range seenText {
				if e.key != k && jaccard(tok, e.tok) >= 0.8 {
					k = e.key
					matched = true
					break
				}
			}
			if !matched {
				seenText = append(seenText, echo{key: k, tok: tok})
			}
		}
		keys[i] = k
	}
	return keys
}

// syndicatedCount is the number of outlets whose item was folded into an
// origin other than their own domain (wire copy or echoed text).
func syndicatedCount(c *Story) (int, []string) {
	keys := originKeys(c)
	seenDomain := map[string]bool{}
	labels := []string{}
	seenLabel := map[string]bool{}
	n := 0
	for i, it := range c.items {
		d := sourceIdentity(it)
		if d == "" || seenDomain[d] {
			continue
		}
		seenDomain[d] = true
		if keys[i] != d {
			n++
			if _, l := wireOf(it); l != "" && !seenLabel[l] {
				seenLabel[l] = true
				labels = append(labels, l)
			}
		}
	}
	return n, labels
}
