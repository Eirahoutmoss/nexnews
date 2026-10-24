package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

var releasesAPI = "https://api.github.com/repos/Eirahoutmoss/nexnews/releases?per_page=30"

const releaseDownloadPrefix = "https://github.com/Eirahoutmoss/nexnews/releases/download/"

// releaseCandidates: yayın listesinden boyutu tutan .exe dosyalarının indirme adresleri.
func releaseCandidates(body []byte, size int64) []string {
	var rels []struct {
		Draft  bool `json:"draft"`
		Assets []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if json.Unmarshal(body, &rels) != nil {
		return nil
	}
	var out []string
	for _, r := range rels {
		if r.Draft {
			continue
		}
		for _, a := range r.Assets {
			if a.Size == size && strings.HasSuffix(strings.ToLower(a.Name), ".exe") && strings.HasPrefix(a.URL, releaseDownloadPrefix) {
				out = append(out, a.URL)
			}
		}
	}
	return out
}

// findReleaseAsset: imzalı manifestteki özeti tutan dosyayı GitHub yayınlarında arar.
func findReleaseAsset(ctx context.Context, a updApp) ([]byte, error) {
	body, err := updGet(ctx, releasesAPI, 2<<20)
	if err != nil {
		return nil, err
	}
	for _, u := range releaseCandidates(body, a.Size) {
		if b, err := updGet(ctx, u, a.Size); err == nil && sha256hex(b) == a.SHA256 {
			return b, nil
		}
	}
	return nil, errors.New("yayınlarda özeti tutan dosya bulunamadı")
}
