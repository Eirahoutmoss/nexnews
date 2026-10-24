package core

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestVersionNewer(t *testing.T) {
	cases := []struct {
		a, b string
		w    bool
	}{{"1.0.0-rc93", "1.0.0-rc92", true}, {"1.0.0-rc100", "1.0.0-rc99", true}, {"1.0.0-rc92", "1.0.0-rc92", false}, {"1.0.1", "1.0.0-rc99", true}, {"v1.0.0-rc9", "1.0.0-rc10", false}}
	for _, c := range cases {
		if versionNewer(c.a, c.b) != c.w {
			t.Fatalf("%s > %s should be %v", c.a, c.b, c.w)
		}
	}
}

func TestVerifyManifestRejectsForeignKey(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	body := []byte(`{"seq":5}`)
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, body))
	if _, err := verifyManifest(body, []byte(sig)); err == nil {
		t.Fatal("yabancı anahtarla imzalı manifest kabul edildi")
	}
}

func TestCheckUpdatesEndToEnd(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	srv := httptest.NewServer(http.StripPrefix("/u/", http.FileServer(http.Dir("../updates"))))
	defer srv.Close()
	old := updatesBase
	updatesBase = srv.URL + "/u/"
	defer func() { updatesBase = old }()
	updMu.Lock()
	updLoaded, updState, updCache = false, updStatus{}, map[string][]byte{}
	updMu.Unlock()
	s := checkUpdates(context.Background())
	if s.Err != "" || !s.Changed || s.Seq < 1 {
		t.Fatalf("güncelleme alınamadı: %+v", s)
	}
	if b := updatedAsset("livetv.json"); len(b) < 1000 {
		t.Fatal("livetv güncellemesi yok")
	}
	// kurcalanmış manifest reddedilmeli
	body, _ := os.ReadFile("../updates/manifest.json")
	sig, _ := os.ReadFile("../updates/manifest.json.sig")
	if _, err := verifyManifest(append(body, ' '), sig); err == nil {
		t.Fatal("kurcalanmış manifest kabul edildi")
	}
}

func TestCachedListsIgnoredAfterAppUpgrade(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := updatesDir()
	_ = os.MkdirAll(dir, 0o755)
	for _, f := range []string{"manifest.json", "manifest.json.sig", "livetv.json"} {
		b, _ := os.ReadFile("../updates/" + f)
		_ = os.WriteFile(dir+"/"+f, b, 0o644)
	}
	_ = os.WriteFile(dir+"/appver.txt", []byte("1.0.0-rc1"), 0o644)
	updMu.Lock()
	updLoaded, updState, updCache = false, updStatus{}, map[string][]byte{}
	updMu.Unlock()
	if updatedAsset("livetv.json") != nil {
		t.Fatal("eski sürümün indirdiği liste kullanıldı")
	}
}
