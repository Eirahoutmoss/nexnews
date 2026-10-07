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
	if err := json.Compact(&b, liveTVRaw); err != nil {
		return []byte("[]")
	}
	return b.Bytes()
}
