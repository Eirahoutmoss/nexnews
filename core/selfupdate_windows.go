package core

// Kendiliğinden güncelleme (RC93, Windows).
//
// İmzalı manifest yeni bir sürüm duyurursa sayfada "Yeni sürüm" düğmesi
// çıkar. Kullanıcı onaylayınca exe GitHub Releases'tan indirilir, boyutu ve
// SHA-256 özeti manifestle karşılaştırılır. Çalışan exe'nin adı değiştirilir
// (Windows çalışan dosyanın adının değişmesine izin verir), yenisi yerine
// konur, yeni sürüm başlatılır ve bu süreç kapanır. Bir sonraki açılışta eski
// dosya temizlenir. Klasöre yazılamıyorsa sürüm sayfası tarayıcıda açılır.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// cleanupOldExe: önceki güncellemeden kalan .old dosyasını siler.
func cleanupOldExe() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = os.Remove(exe + ".old")
	_ = os.Remove(exe + ".new")
}

// installAppUpdate indirir, doğrular ve yerine koyar. Başarılıysa yeni exe'nin yolunu döndürür.
func installAppUpdate(ctx context.Context) (string, error) {
	updMu.Lock()
	var a *updApp
	if updState.App != nil {
		x := *updState.App
		a = &x
	}
	updMu.Unlock()
	if a == nil {
		return "", errors.New("Yeni sürüm bilgisi yok.")
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	if a.Size <= 0 || a.Size > 200<<20 {
		return "", errors.New("Geçersiz sürüm boyutu.")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	b, err := updGet(ctx, a.URL, a.Size)
	if err != nil {
		return "", errors.New("İndirilemedi: " + err.Error())
	}
	if int64(len(b)) != a.Size || sha256hex(b) != a.SHA256 {
		return "", errors.New("İndirilen dosyanın özeti tutmadı; kurulmadı.")
	}
	if len(b) < 2 || b[0] != 'M' || b[1] != 'Z' {
		return "", errors.New("İndirilen dosya bir Windows programı değil.")
	}
	newPath := exe + ".new"
	if err := os.WriteFile(newPath, b, 0o755); err != nil {
		return "", errors.New("Program klasörüne yazılamıyor: " + err.Error())
	}
	_ = os.Remove(exe + ".old")
	if err := os.Rename(exe, exe+".old"); err != nil {
		_ = os.Remove(newPath)
		return "", errors.New("Çalışan dosya değiştirilemedi: " + err.Error())
	}
	if err := os.Rename(newPath, exe); err != nil {
		_ = os.Rename(exe+".old", exe) // geri al
		return "", errors.New("Yeni dosya yerine konamadı: " + err.Error())
	}
	return exe, nil
}

func restartInto(exe string) error {
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func releasePageURL(v string) string {
	return "https://github.com/Eirahoutmoss/nexnews/releases/tag/v" + strings.TrimPrefix(v, "v")
}
