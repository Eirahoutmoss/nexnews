// nexsign: NexNews uzaktan güncelleme paketini hazırlar ve imzalar.
//
//	go run ./cmd/nexsign -key ..\imza\nexnews-guncelleme.key            (listeleri yayınla)
//	go run ./cmd/nexsign -key ... -app dist\NexNews.exe -version 1.0.0-rc93 -notes "..."
//
// updates/ klasörüne livetv.json, i18n.json, localnews.json yazar (gömülü
// güncel hâlleri), manifest.json'u bir artırılmış sıra numarasıyla üretir ve
// Ed25519 ile imzalar. Gizli anahtar asla depoya konmaz.
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nexnews/core"
)

type file struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type app struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
	Notes   string `json:"notes,omitempty"`
}
type manifest struct {
	Seq    int64           `json:"seq"`
	Issued string          `json:"issued"`
	Files  map[string]file `json:"files"`
	App    *app            `json:"app,omitempty"`
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "HATA:", err)
		os.Exit(1)
	}
}

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func main() {
	keyPath := flag.String("key", "", "gizli anahtar dosyası (base64 tohum)")
	dir := flag.String("dir", "updates", "yayın klasörü")
	appExe := flag.String("app", "", "yayınlanacak exe (isteğe bağlı)")
	version := flag.String("version", "", "exe sürümü, ör. 1.0.0-rc93")
	notes := flag.String("notes", "", "sürüm notu")
	appURL := flag.String("url", "", "exe'nin GitHub sürüm adresi (boşsa .../releases/download/v<sürüm>/NexNews.exe)")
	keepApp := flag.Bool("keep-app", true, "exe verilmezse önceki manifestteki uygulama bilgisini koru")
	flag.Parse()
	raw, err := os.ReadFile(*keyPath)
	must(err)
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	must(err)
	if len(seed) != ed25519.SeedSize {
		must(fmt.Errorf("anahtar boyutu yanlış"))
	}
	priv := ed25519.NewKeyFromSeed(seed)
	must(os.MkdirAll(*dir, 0o755))

	var old manifest
	if b, err := os.ReadFile(filepath.Join(*dir, "manifest.json")); err == nil {
		_ = json.Unmarshal(b, &old)
	}
	m := manifest{Seq: old.Seq + 1, Issued: time.Now().UTC().Format(time.RFC3339), Files: map[string]file{}}
	for name, b := range map[string][]byte{
		"livetv.json":    core.LiveTVAsset(),
		"i18n.json":      core.I18nAsset(),
		"localnews.json": core.LocalNewsCatalogJSON(),
		"ownership.json": core.OwnershipAsset(),
		"extras.json":    core.ExtrasAsset(),
		"holidays.json":  core.HolidaysAsset(),
	} {
		must(os.WriteFile(filepath.Join(*dir, name), b, 0o644))
		m.Files[name] = file{SHA256: sum(b), Size: int64(len(b))}
	}
	if *appExe != "" {
		if *version == "" {
			must(fmt.Errorf("-version gerekli"))
		}
		b, err := os.ReadFile(*appExe)
		must(err)
		u := *appURL
		if u == "" {
			u = "https://github.com/Eirahoutmoss/nexnews/releases/download/v" + *version + "/NexNews.exe"
		}
		m.App = &app{Version: *version, SHA256: sum(b), Size: int64(len(b)), Notes: *notes, URL: u}
	} else if *keepApp {
		m.App = old.App
	}
	body, err := json.MarshalIndent(m, "", " ")
	must(err)
	sig := ed25519.Sign(priv, body)
	must(os.WriteFile(filepath.Join(*dir, "manifest.json"), body, 0o644))
	must(os.WriteFile(filepath.Join(*dir, "manifest.json.sig"), []byte(base64.StdEncoding.EncodeToString(sig)+"\n"), 0o644))
	fmt.Printf("imzalandı: seq=%d, %d dosya", m.Seq, len(m.Files))
	if m.App != nil {
		fmt.Printf(", uygulama %s", m.App.Version)
	}
	fmt.Println()
	fmt.Println("Açık anahtar:", hex.EncodeToString(priv.Public().(ed25519.PublicKey)))
}
