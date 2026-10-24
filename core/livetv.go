package core

import (
	"bytes"
	_ "embed"
	"encoding/json"
)

// The live-TV channel list is shared by the desktop page and the Android app.
//
//go:embed assets/livetv.json
var liveTVRaw []byte

// liveTVJSON returns the channel list as compact JSON.
func liveTVJSON() []byte {
	var b bytes.Buffer
	if err := json.Compact(&b, liveTVSource()); err != nil {
		return []byte("[]")
	}
	return b.Bytes()
}

// liveTVSource: uzaktan güncellenmiş (imzası doğrulanmış) liste varsa o, yoksa gömülü liste.
func liveTVSource() []byte {
	if b := updatedAsset("livetv.json"); b != nil {
		return b
	}
	return liveTVRaw
}
