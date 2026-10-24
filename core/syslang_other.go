//go:build !windows

package core

import (
	"os"
	"strings"
)

// Tests and other non-Windows builds fall back to English unless the Android
// host set sysLangOverride (device locale).
func systemLang() string {
	if sysLangOverride != "" {
		return sysLangOverride
	}
	return "en"
}

// sysCountryOverride is the device country set by the Android host.

// systemCountry returns the device/user country (ISO alpha-2) when known:
// Android sets it; elsewhere it is read from LANG ("ar_QA.UTF-8").
func systemCountry() string {
	if sysCountryOverride != "" {
		return strings.ToUpper(sysCountryOverride)
	}
	l := os.Getenv("LANG")
	if i := strings.IndexByte(l, '.'); i > 0 {
		l = l[:i]
	}
	if i := strings.IndexByte(l, '_'); i > 0 && len(l)-i-1 == 2 {
		return strings.ToUpper(l[i+1:])
	}
	return ""
}
