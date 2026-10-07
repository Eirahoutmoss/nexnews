package core

import (
	"testing"
	"time"
)

func TestJaccardSimilar(t *testing.T) {
	a := tokenize("Japonya'da 6.8 büyüklüğünde deprem meydana geldi")
	b := tokenize("Japonya kıyılarında 6.8 deprem: tsunami uyarısı")
	c := tokenize("Yeni akıllı telefon modeli tanıtıldı")
	if jaccard(a, b) < 0.25 {
		t.Errorf("similar titles too low: %.2f", jaccard(a, b))
	}
	if jaccard(a, c) >= simThreshold {
		t.Errorf("unrelated titles too high: %.2f", jaccard(a, c))
	}
}

func TestClusterMergesSources(t *testing.T) {
	now := time.Now()
	items := []Item{
		{Title: "Japonya'da 6.8 büyüklüğünde deprem meydana geldi", Link: "a", Source: "BBC", Published: now, HasDate: true},
		{Title: "Japonya'da 6.8 deprem: tsunami uyarısı yayınlandı", Link: "b", Source: "Reuters", Published: now.Add(-10 * time.Minute), HasDate: true},
		{Title: "Yeni akıllı telefon modeli bugün tanıtıldı", Link: "c", Source: "Verge", Published: now.Add(-20 * time.Minute), HasDate: true},
	}
	cl := clusterItems(items)
	if len(cl) != 2 {
		t.Fatalf("got %d clusters, want 2", len(cl))
	}
	// find the quake cluster (2 sources)
	var quake *Story
	for _, c := range cl {
		if c.Count == 2 {
			quake = c
		}
	}
	if quake == nil {
		t.Fatalf("no merged cluster found: %+v", cl)
	}
	if len(quake.Sources) != 2 {
		t.Errorf("sources=%v want 2", quake.Sources)
	}
}

func TestScoreAndKind(t *testing.T) {
	now := time.Now()
	items := []Item{
		{Title: "Son dakika: büyük deprem", Link: "a", Source: "BBC", Published: now, HasDate: true, Breaking: true, Category: "Dünya"},
		{Title: "Son dakika büyük deprem bölgede", Link: "b", Source: "AP", Published: now, HasDate: true, Breaking: true, Category: "Dünya"},
	}
	cl := clusterItems(items)
	st := cl[0]
	if st.Kind != kindAcil {
		t.Errorf("kind=%s want acil", st.Kind)
	}
	if st.Score < 60 {
		t.Errorf("breaking multi-source recent score too low: %d", st.Score)
	}

	tech := []Item{{Title: "Yeni işlemci mimarisi duyuruldu", Link: "x", Source: "Verge", Published: now, HasDate: true, Category: "Teknoloji"}}
	ct := clusterItems(tech)[0]
	if ct.Kind != kindTeknoloji {
		t.Errorf("tech kind=%s", ct.Kind)
	}
}
