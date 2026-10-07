//go:build windows

package core

import (
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetUserDefaultLocaleName = windows.NewLazySystemDLL("kernel32.dll").
	NewProc("GetUserDefaultLocaleName")

// systemLang returns the user's Windows locale as a short language code (e.g. "tr").
func systemLang() string {
	defer func() { _ = recover() }()
	buf := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH
	r, _, _ := procGetUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return "en"
	}
	name := syscall.UTF16ToString(buf) // "tr-TR"
	if i := strings.IndexByte(name, '-'); i > 0 {
		name = name[:i]
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "en"
	}
	return name
}
