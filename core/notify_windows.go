//go:build windows

package core

// Smart desktop notifications. A small tray icon is registered once; when a
// genuinely important NEW event appears (breaking / high score, not seen before)
// a balloon toast is shown via Shell_NotifyIcon. This is the "NexNews masaüstü
// radarı" — the app brings important news to the user without them opening it.

import (
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32          = windows.NewLazySystemDLL("shell32.dll")
	procShellNotifyW = shell32.NewProc("Shell_NotifyIconW")
	procLoadImageW   = user32.NewProc("LoadImageW")
	procGetModuleH   = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetModuleHandleW")
)

const (
	nimAdd    = 0x0
	nimModify = 0x1
	nimDelete = 0x2

	nifIcon = 0x02
	nifTip  = 0x04
	nifInfo = 0x10

	niifInfo = 0x1
)

type notifyIconDataW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UTimeoutVersion  uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

var (
	trayMu    sync.Mutex
	trayHWND  uintptr
	trayAdded bool
	trayIcon  uintptr
)

func loadAppIcon() uintptr {
	mod, _, _ := procGetModuleH.Call(0)
	// LoadImageW(hinst, MAKEINTRESOURCE(1), IMAGE_ICON, 0, 0, LR_SHARED|LR_DEFAULTSIZE)
	h, _, _ := procLoadImageW.Call(mod, 1, 1, 0, 0, 0x8000|0x0040)
	return h
}

// addTray registers the tray icon once.
func addTray(hwnd uintptr) {
	defer func() { _ = recover() }()
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayAdded || hwnd == 0 {
		return
	}
	trayHWND = hwnd
	trayIcon = loadAppIcon()
	var nid notifyIconDataW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwnd
	nid.UID = 1
	nid.UFlags = nifIcon | nifTip
	nid.HIcon = trayIcon
	copyU16(nid.SzTip[:], "NexNews")
	if r, _, _ := procShellNotifyW.Call(nimAdd, uintptr(unsafe.Pointer(&nid))); r != 0 {
		trayAdded = true
	}
}

// showToast shows a balloon notification (adds the tray icon first if needed).
func showToast(hwnd uintptr, title, msg string) {
	defer func() { _ = recover() }()
	if hwnd == 0 {
		return
	}
	addTray(hwnd)
	trayMu.Lock()
	defer trayMu.Unlock()
	var nid notifyIconDataW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = hwnd
	nid.UID = 1
	nid.UFlags = nifInfo | nifIcon
	nid.HIcon = trayIcon
	nid.DwInfoFlags = niifInfo
	copyU16(nid.SzInfoTitle[:], title)
	copyU16(nid.SzInfo[:], msg)
	procShellNotifyW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func copyU16(dst []uint16, s string) {
	u := syscall.StringToUTF16(s)
	for i := 0; i < len(u) && i < len(dst)-1; i++ {
		dst[i] = u[i]
	}
}
