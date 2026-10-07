package core

import "testing"

func TestCanonicalTokenOverlap(t *testing.T) {
	a := tokenize("İstanbul'da 5.2 büyüklüğünde deprem")
	b := tokenize("5.2 büyüklüğünde İstanbul depremi")
	sim, shared := overlapCoef(a, b)
	if shared < 2 || sim < 0.5 {
		t.Fatalf("unexpected overlap: %.2f / %d", sim, shared)
	}
}
