package core

// Uzaktan güncelleme (RC93).
//
// Kanal listesi, arayüz çevirileri ve ülke haber siteleri exe'ye gömülü gelir,
// ama artık GitHub'daki updates/ klasöründen de tazelenebilir. Klasördeki
// manifest.json Ed25519 ile imzalanır; uygulama yalnızca gömülü açık anahtarla
// doğrulanan, dosya özetleri (SHA-256) tutan güncellemeleri kabul eder. Böylece
// biri depoya ya da bağlantıya müdahale etse bile uygulamaya sahte liste ya da
// sahte exe giremez. Manifest aynı zamanda yeni uygulama sürümünü de duyurur
// (bkz. selfupdate_windows.go).

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var updatesBase = "https://raw.githubusercontent.com/Eirahoutmoss/nexnews/main/updates/"

// updatesPubKey: güncelleme imzasının açık anahtarı (gizli anahtar yalnızca
// geliştiricinin bilgisayarında, depoda değil).
const updatesPubKey = "0e69a0659fb1d6832983e96b0741de0d9300b18e210cc869dd6d7aef64d54c94"

// Uzaktan değiştirilebilen dosyalar (başka ad kabul edilmez).
var updatableFiles = map[string]bool{"livetv.json": true, "i18n.json": true, "localnews.json": true, "ownership.json": true, "extras.json": true, "holidays.json": true}

type updFile struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type updApp struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
	Notes   string `json:"notes,omitempty"`
}

type updManifest struct {
	Seq    int64              `json:"seq"`
	Issued string             `json:"issued"`
	Files  map[string]updFile `json:"files"`
	App    *updApp            `json:"app,omitempty"`
}

type updStatus struct {
	Seq        int64   `json:"seq"`
	Checked    int64   `json:"checked"`
	Applied    int64   `json:"applied"`
	Changed    bool    `json:"changed"` // bu oturumda yeni listeler indi
	Err        string  `json:"err,omitempty"`
	App        *updApp `json:"app,omitempty"` // yalnızca kurulu sürümden yeniyse
	AppCurrent string  `json:"appCurrent"`
}

var (
	updMu     sync.Mutex
	updState  updStatus
	updCache  = map[string][]byte{}
	updLoaded bool
)

func updatesDir() string { return filepath.Join(appDataDir(), "updates") }

func verifyManifest(body, sig []byte) (*updManifest, error) {
	pub, err := hex.DecodeString(updatesPubKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("güncelleme anahtarı yok")
	}
	s, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil || !ed25519.Verify(ed25519.PublicKey(pub), body, s) {
		return nil, errors.New("imza doğrulanamadı")
	}
	var m updManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func sha256hex(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// loadLocalUpdates: diskteki (önceden doğrulanmış) güncellemeleri yeniden
// doğrulayarak belleğe alır. Bozuk ya da imzasız hiçbir şey kullanılmaz.
func loadLocalUpdates() {
	updMu.Lock()
	defer updMu.Unlock()
	if updLoaded {
		return
	}
	updLoaded = true
	dir := updatesDir()
	// İndirilmiş listeler yalnızca onları indiren uygulama sürümünde geçerlidir:
	// yeni sürümün içindeki listeler eski indirmelerden daha yenidir.
	if v, err := os.ReadFile(filepath.Join(dir, "appver.txt")); err != nil || strings.TrimSpace(string(v)) != appVersion {
		return
	}
	body, err1 := os.ReadFile(filepath.Join(dir, "manifest.json"))
	sig, err2 := os.ReadFile(filepath.Join(dir, "manifest.json.sig"))
	if err1 != nil || err2 != nil {
		return
	}
	m, err := verifyManifest(body, sig)
	if err != nil {
		return
	}
	for name, f := range m.Files {
		if !updatableFiles[name] {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil && sha256hex(b) == f.SHA256 && json.Valid(b) {
			updCache[name] = b
		}
	}
	updState.Seq = m.Seq
}

// updatedAsset: uzaktan gelmiş ve doğrulanmış dosya, yoksa nil.
func updatedAsset(name string) []byte {
	loadLocalUpdates()
	updMu.Lock()
	defer updMu.Unlock()
	return updCache[name]
}

func updGet(ctx context.Context, url string, max int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NexNews/"+appVersion)
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("HTTP " + strconv.Itoa(resp.StatusCode))
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, errors.New("dosya beklenenden büyük")
	}
	return b, nil
}

// checkUpdates: manifesti indirir, doğrular, yeni listeleri indirip kaydeder.
// Uygulama sürümü yeniyse durumda bildirir (indirme kullanıcı onayıyla yapılır).
func checkUpdates(ctx context.Context) updStatus {
	loadLocalUpdates()
	st := func(err error) updStatus {
		updMu.Lock()
		defer updMu.Unlock()
		updState.Checked = time.Now().Unix()
		updState.AppCurrent = appVersion
		if err != nil {
			updState.Err = err.Error()
		} else {
			updState.Err = ""
		}
		return updState
	}
	nonce := "?t=" + strconv.FormatInt(time.Now().Unix()/300, 10) // CDN önbelleğini 5 dk'da bir aş
	body, err := updGet(ctx, updatesBase+"manifest.json"+nonce, 256<<10)
	if err != nil {
		return st(err)
	}
	sig, err := updGet(ctx, updatesBase+"manifest.json.sig"+nonce, 4<<10)
	if err != nil {
		return st(err)
	}
	m, err := verifyManifest(body, sig)
	if err != nil {
		log.Printf("güncelleme: %v", err)
		return st(err)
	}
	updMu.Lock()
	cur := updState.Seq
	updMu.Unlock()
	if m.Seq > cur {
		got := map[string][]byte{}
		for name, f := range m.Files {
			if !updatableFiles[name] {
				continue
			}
			if f.Size <= 0 || f.Size > 8<<20 {
				return st(errors.New(name + ": geçersiz boyut"))
			}
			b, err := updGet(ctx, updatesBase+name+"?h="+f.SHA256[:12], f.Size)
			if err != nil {
				return st(errors.New(name + ": " + err.Error()))
			}
			if sha256hex(b) != f.SHA256 || !json.Valid(b) {
				return st(errors.New(name + ": özet tutmadı"))
			}
			got[name] = b
		}
		dir := updatesDir()
		_ = os.MkdirAll(dir, 0o755)
		for name, b := range got {
			if err := atomicWriteFile(filepath.Join(dir, name), b); err != nil {
				return st(err)
			}
		}
		// manifest en son yazılır: yarım kalan indirme eski durumu bozmaz.
		_ = atomicWriteFile(filepath.Join(dir, "manifest.json.sig"), sig)
		_ = atomicWriteFile(filepath.Join(dir, "manifest.json"), body)
		_ = atomicWriteFile(filepath.Join(dir, "appver.txt"), []byte(appVersion))
		updMu.Lock()
		for name, b := range got {
			updCache[name] = b
		}
		updState.Seq, updState.Applied, updState.Changed = m.Seq, time.Now().Unix(), true
		updMu.Unlock()
		applyLocalNewsUpdate()
		log.Printf("güncelleme: listeler seq=%d", m.Seq)
	}
	updMu.Lock()
	updState.App = nil
	if m.App != nil && versionNewer(m.App.Version, appVersion) && strings.HasPrefix(m.App.URL, "https://github.com/Eirahoutmoss/nexnews/releases/download/") {
		a := *m.App
		updState.App = &a
	}
	updMu.Unlock()
	return st(nil)
}

func updStatusJSON() string {
	updMu.Lock()
	defer updMu.Unlock()
	s := updState
	s.AppCurrent = appVersion
	b, _ := json.Marshal(s)
	return string(b)
}

// versionNewer: "1.0.0-rc93" > "1.0.0-rc92"; sayısal parçalar sayı olarak karşılaştırılır.
func versionNewer(a, b string) bool { return cmpVersion(a, b) > 0 }

func cmpVersion(a, b string) int {
	split := func(s string) []string {
		var out []string
		cur := ""
		isNum := false
		for _, r := range strings.ToLower(strings.TrimPrefix(s, "v")) {
			n := r >= '0' && r <= '9'
			if cur != "" && n != isNum {
				out = append(out, cur)
				cur = ""
			}
			if r == '.' || r == '-' || r == '+' {
				if cur != "" {
					out = append(out, cur)
				}
				cur = ""
				continue
			}
			cur += string(r)
			isNum = n
		}
		if cur != "" {
			out = append(out, cur)
		}
		return out
	}
	pa, pb := split(a), split(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		if i >= len(pa) {
			return -1
		}
		if i >= len(pb) {
			return 1
		}
		x, ex := strconv.Atoi(pa[i])
		y, ey := strconv.Atoi(pb[i])
		switch {
		case ex == nil && ey == nil:
			if x != y {
				if x < y {
					return -1
				}
				return 1
			}
		default:
			if c := strings.Compare(pa[i], pb[i]); c != 0 {
				return c
			}
		}
	}
	return 0
}

// ---- uzaktan güncellenebilir yerel haber kataloğu ----

type localNewsFile struct {
	Sites map[string][]string `json:"sites"`
	Names map[string]string   `json:"names"`
}

var localNewsOnce sync.Once

func applyLocalNewsUpdate() {
	b := updatedAsset("localnews.json")
	if b == nil {
		return
	}
	var f localNewsFile
	if json.Unmarshal(b, &f) != nil {
		return
	}
	localMu.Lock()
	defer localMu.Unlock()
	for k, v := range f.Sites {
		if len(k) == 2 && len(v) > 0 && len(v) <= 20 {
			localNewsSites[strings.ToUpper(k)] = v
		}
	}
	for k, v := range f.Names {
		if len(k) == 2 && v != "" {
			countryNameTR[strings.ToUpper(k)] = v
		}
	}
}

// localNewsCatalogJSON: depodaki updates/localnews.json'u üretmek için.
func localNewsCatalogJSON() []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	_ = enc.Encode(localNewsFile{Sites: localNewsSites, Names: countryNameTR})
	return buf.Bytes()
}

// startUpdateLoop: açılıştan 20 sn sonra, sonra 12 saatte bir denetler.
// notify, durum değiştiğinde (yeni listeler ya da yeni sürüm) çağrılır.
func startUpdateLoop(notify func(updStatus)) {
	localNewsOnce.Do(applyLocalNewsUpdate)
	go func() {
		time.Sleep(20 * time.Second)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			s := checkUpdates(ctx)
			cancel()
			if notify != nil && (s.Changed || s.App != nil) {
				notify(s)
			}
			time.Sleep(12 * time.Hour)
		}
	}()
}

// LocalNewsCatalogJSON (dışa açık): imzalama aracı depodaki updates/localnews.json'u bununla üretir.
func LocalNewsCatalogJSON() []byte { return localNewsCatalogJSON() }

// LiveTVAsset ve I18nAsset: imzalama aracı için gömülü dosyalar.
func LiveTVAsset() []byte { return liveTVRaw }
func I18nAsset() []byte   { return i18nJSON }
