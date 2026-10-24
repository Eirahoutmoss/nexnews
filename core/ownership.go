package core

// Sahiplik şeffaflığı (RC95): kaynakların kamu yayıncısı / devlet medyası /
// ticari grup / bağımsız / resmî kurum sınıflandırması. Yalnızca herkese açık,
// yerleşik bilgiler; emin olunmayan kaynak etiketsiz kalır. Katalog uzaktan
// (imzalı güncelleme) tazelenebilir; sayfaya __NEXOWN__ olarak verilir.

import (
	"bytes"
	_ "embed"
	"encoding/json"
)

//go:embed assets/ownership.json
var ownershipRaw []byte

func ownershipSource() []byte {
	if b := updatedAsset("ownership.json"); b != nil {
		return b
	}
	return ownershipRaw
}

func ownershipJSON() []byte {
	var b bytes.Buffer
	if err := json.Compact(&b, ownershipSource()); err != nil {
		return []byte(`{"types":{},"d":{}}`)
	}
	return b.Bytes()
}

// OwnershipAsset: imzalama aracı için.
func OwnershipAsset() []byte { return ownershipRaw }
