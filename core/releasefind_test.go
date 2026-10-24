package core

import "testing"

func TestReleaseCandidates(t *testing.T) {
	body := []byte(`[{"draft":true,"assets":[{"name":"x.exe","size":10,"browser_download_url":"https://github.com/Eirahoutmoss/nexnews/releases/download/a/x.exe"}]},
	{"draft":false,"assets":[{"name":"NexNews-rc95.exe","size":10,"browser_download_url":"https://github.com/Eirahoutmoss/nexnews/releases/download/v1.0.0.-rd95/NexNews-rc95.exe"},
	{"name":"other.apk","size":10,"browser_download_url":"https://github.com/Eirahoutmoss/nexnews/releases/download/v/o.apk"},
	{"name":"evil.exe","size":10,"browser_download_url":"https://evil.example/evil.exe"}]}]`)
	c := releaseCandidates(body, 10)
	if len(c) != 1 || c[0] != "https://github.com/Eirahoutmoss/nexnews/releases/download/v1.0.0.-rd95/NexNews-rc95.exe" {
		t.Fatalf("adaylar: %v", c)
	}
}
