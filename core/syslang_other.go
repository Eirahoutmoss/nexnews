//go:build !windows

package core

// Tests and other non-Windows builds fall back to English unless the Android
// host set sysLangOverride (device locale).
func systemLang() string {
	if sysLangOverride != "" {
		return sysLangOverride
	}
	return "en"
}
