package core

import (
	"testing"
	"time"
)

func mkItem(title, src string, mins int) Item {
	return Item{Title: title, Link: "https://" + src + ".example/" + title, Source: src, Category: "Test",
		Published: time.Now().Add(-time.Duration(mins) * time.Minute), HasDate: true}
}

func TestTurkishInflectionsCluster(t *testing.T) {
	items := []Item{
		mkItem("Malatya'da 5,4 büyüklüğünde deprem: AFAD hasar tespiti yapıyor", "a", 1),
		mkItem("Malatya depreminin ardından AFAD hasar tespitine başladı", "b", 2),
		mkItem("İstanbul'da metro seferleri aksadı", "c", 3),
	}
	cl := clusterItems(items)
	if len(cl) != 2 {
		for _, c := range cl {
			t.Logf("%d %s", c.Count, c.Title)
		}
		t.Fatalf("expected 2 stories, got %d", len(cl))
	}
}

func TestCrossLanguageClusterViaTranslation(t *testing.T) {
	tr := map[string]string{
		"Les dirigeants européens s'accordent sur le pacte migratoire": "AB liderleri göç paketi üzerinde anlaştı",
		"EU leaders agree new migration package after summit":          "AB liderleri zirve sonrası yeni göç paketi üzerinde anlaştı",
	}
	clusterTitleFn.Store(func(s string) string { return tr[s] })
	defer clusterTitleFn.Store(func(string) string { return "" })
	items := []Item{
		mkItem("EU leaders agree new migration package after summit", "bbc", 1),
		mkItem("Les dirigeants européens s'accordent sur le pacte migratoire", "lemonde", 5),
	}
	if cl := clusterItems(items); len(cl) != 1 {
		t.Fatalf("expected the two languages to merge, got %d stories", len(cl))
	}
}
