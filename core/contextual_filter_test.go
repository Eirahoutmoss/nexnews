package core

import "testing"

func TestContextualTopicSuppression(t *testing.T) {
	cases := []struct {
		name string
		text string
		hide bool
	}{
		{"crypto market", "Bitcoin fiyatı bugün yükseldi", true},
		{"crypto investment", "Kripto yatırım fonuna yoğun ilgi", true},
		{"crypto laundering", "Şebekenin kripto para ile kara para akladığı ortaya çıktı", false},
		{"crypto fraud", "Kripto para dolandırıcılığı operasyonunda 12 kişi gözaltına alındı", false},
		{"crypto cybercrime", "Siber saldırıda kripto cüzdanlardan para çalındı", false},
		{"crypto seizure", "Kripto varlıklara el konuldu", false},
		{"crypto seizure confiscation", "Kripto varlıklara müsadere kararı verildi", false},
		{"crypto court", "Kripto dolandırıcılığı davasında karar açıklandı", false},
		{"crypto price", "Ethereum fiyatı yüzde 5 düştü", true},
	}
	for _, tc := range cases {
		got := contextualTopicSuppressed("kripto", tc.text, "genel")
		if got != tc.hide {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.hide)
		}
	}
}
