package core

// Process-wide singletons shared by the platform-independent core. They are
// declared here (not in main_windows.go) so the test suite builds and runs on
// any OS.

import "strings"

const appVersion = "1.0.0-rc86"

// sysLangOverride is the device language set by the Android host.
var sysLangOverride string

var (
	store   *Store
	cfg     *Config
	favs    *favStore
	follows *followStore
	speech  SpeechProvider
)

// applyPersonalCfg pushes the interest / hide lists into the store.
func applyPersonalCfg() {
	cfg.mu.RLock()
	in, hide := cfg.Interests, cfg.HideWords
	cfg.mu.RUnlock()
	store.setPersonal(in, hide)
}

// applyTranslateCfg configures the store's translation from cfg, resolving an
// empty target to the system language.
func applyTranslateCfg() string {
	cfg.mu.RLock()
	target := strings.TrimSpace(cfg.TranslateLang)
	on := cfg.Translate
	cfg.mu.RUnlock()
	if target == "" {
		target = systemLang()
	}
	store.setTranslation(on, target)
	return target
}
