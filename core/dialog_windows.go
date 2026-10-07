//go:build windows

package core

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	comdlg32            = windows.NewLazySystemDLL("comdlg32.dll")
	procGetSaveFileName = comdlg32.NewProc("GetSaveFileNameW")
	procGetOpenFileName = comdlg32.NewProc("GetOpenFileNameW")
)

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

const (
	ofnOverwritePrompt = 0x00000002
	ofnPathMustExist   = 0x00000800
	ofnFileMustExist   = 0x00001000
	ofnExplorer        = 0x00080000
	ofnNoChangeDir     = 0x00000008
)

// filter must be the Windows double-NUL terminated form, e.g.
// "CSV\x00*.csv\x00Tümü\x00*.*\x00\x00".
func makeOFN(hwnd uintptr, filter, defExt, defName string) (*openFileNameW, []uint16) {
	buf := make([]uint16, 1024)
	if defName != "" {
		for i, r := range syscall.StringToUTF16(defName) {
			if i >= len(buf)-1 {
				break
			}
			buf[i] = r
		}
	}
	ofn := &openFileNameW{
		hwndOwner:   hwnd,
		lpstrFilter: &utf16z(filter)[0],
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		flags:       ofnExplorer | ofnNoChangeDir | ofnPathMustExist,
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(*ofn))
	if defExt != "" {
		ofn.lpstrDefExt = &utf16z(defExt)[0]
	}
	return ofn, buf
}

// utf16z encodes s as UTF-16 preserving embedded NULs (needed for the filter
// string, which StringToUTF16 would truncate at the first NUL).
func utf16z(s string) []uint16 {
	r := make([]uint16, 0, len(s)+1)
	for _, c := range s {
		r = append(r, uint16(c))
	}
	r = append(r, 0)
	return r
}

// saveFileDialog shows a native Save As dialog and returns the chosen path, or
// "" if cancelled/failed.
func saveFileDialog(hwnd uintptr, filter, defExt, defName string) string {
	defer func() { _ = recover() }()
	ofn, buf := makeOFN(hwnd, filter, defExt, defName)
	ofn.flags |= ofnOverwritePrompt
	r, _, _ := procGetSaveFileName.Call(uintptr(unsafe.Pointer(ofn)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

// openFileDialog shows a native Open dialog and returns the chosen path, or "".
func openFileDialog(hwnd uintptr, filter string) string {
	defer func() { _ = recover() }()
	ofn, buf := makeOFN(hwnd, filter, "", "")
	ofn.flags |= ofnFileMustExist
	r, _, _ := procGetOpenFileName.Call(uintptr(unsafe.Pointer(ofn)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
