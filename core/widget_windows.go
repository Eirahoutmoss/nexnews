//go:build windows

package core

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procRedrawWindow       = user32.NewProc("RedrawWindow")
	procGetWindowLong      = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLong      = user32.NewProc("SetWindowLongPtrW")
	procReleaseCapture     = user32.NewProc("ReleaseCapture")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procShowWindow         = user32.NewProc("ShowWindow")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procMonitorFromWindow  = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW    = user32.NewProc("GetMonitorInfoW")
	procGetAncestor        = user32.NewProc("GetAncestor")
	procGetClientRect      = user32.NewProc("GetClientRect")
	procSetWindowRgn       = user32.NewProc("SetWindowRgn")
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	procCreateRoundRectRgn = gdi32.NewProc("CreateRoundRectRgn")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
)

const gaRoot = 2

// nativeHostWindow resolves the real top-level host window. Depending on the
// WebView2 wrapper, ui.Window() may refer to the embedded browser/control
// window rather than the outer native frame. Resizing only that child leaves
// the old host area exposed as a black/white rectangle. Always operate on the
// root host for frame, size, z-order and redraw operations.
func nativeHostWindow(hwnd uintptr) uintptr {
	defer func() { _ = recover() }()
	if hwnd == 0 {
		return 0
	}
	root, _, _ := procGetAncestor.Call(hwnd, gaRoot)
	if root != 0 {
		return root
	}
	return hwnd
}

// centerWindow moves+sizes the window to the middle of the primary screen, so
// returning from the dragged-around widget to full size never lands off-screen.
func placeWidgetBottomRight(hwnd uintptr, w, h, margin int) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	// Use the monitor containing the widget, not SM_CXWORKAREA. The latter
	// describes the primary work area and is wrong on multi-monitor/DPI setups.
	const monitorDefaultToNearest = 2
	mon, _, _ := procMonitorFromWindow.Call(hwnd, monitorDefaultToNearest)
	type rect struct{ Left, Top, Right, Bottom int32 }
	type monitorInfo struct {
		CbSize    uint32
		RcMonitor rect
		RcWork    rect
		Flags     uint32
	}
	mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	if mon != 0 {
		ret, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
		if ret != 0 {
			x := int(mi.RcWork.Right) - w - margin
			y := int(mi.RcWork.Bottom) - h - margin
			if x < int(mi.RcWork.Left) {
				x = int(mi.RcWork.Left)
			}
			if y < int(mi.RcWork.Top) {
				y = int(mi.RcWork.Top)
			}
			procSetWindowPos.Call(hwnd, 0, uintptr(int64(x)), uintptr(int64(y)), uintptr(w), uintptr(h), uintptr(swpNoZorder|swpNoActivate))
			return
		}
	}
	cx, _, _ := procGetSystemMetrics.Call(0x4)
	cy, _, _ := procGetSystemMetrics.Call(0x5)
	x := int(cx) - w - margin
	y := int(cy) - h - margin
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(swpNoZorder|swpNoActivate))
}

func centerWindow(hwnd uintptr, w, h int) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	cx, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	cy, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	x := (int(cx) - w) / 2
	y := (int(cy) - h) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(swpNoZorder))
}

const (
	gwlStyle        = ^uintptr(15) // -16
	wsPopup         = 0x80000000
	wsVisible       = 0x10000000
	wsOverlappedWin = 0x00CF0000
	swpFrameChanged = 0x0020
	swpNoZorder     = 0x0004
	wmNCLButtonDown = 0x00A1
	htCaption       = 2
	swMinimize      = 6
)

// setFrameless switches the window between the normal title-bar frame and a
// borderless popup (for the desktop-widget look). Removing the caption also
// stops Aero-snap from maximizing the window when it is dragged to a screen edge.
// setWidgetRegion clips the native popup itself to the same rounded shape as
// the visual widget. This is deliberately native, not CSS-only: if WebView2
// ever paints late, there is no rectangular native surface outside the panel.
func setWidgetRegion(hwnd uintptr, on bool, dims ...int) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	if !on {
		procSetWindowRgn.Call(hwnd, 0, 1)
		return
	}
	width, height := 460, 168
	if len(dims) >= 2 {
		width, height = dims[0], dims[1]
	}
	// SetWindowRgn takes window-relative coordinates and clips both the native
	// popup and client surface. Keep the radius visually stable while resizing.
	rgn, _, _ := procCreateRoundRectRgn.Call(0, 0, uintptr(width), uintptr(height), 24, 24)
	if rgn == 0 {
		return
	}
	if ret, _, _ := procSetWindowRgn.Call(hwnd, rgn, 1); ret == 0 {
		procDeleteObject.Call(rgn)
	}
}

func setFrameless(hwnd uintptr, on bool) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	var style uintptr
	if on {
		style = wsPopup | wsVisible
	} else {
		style = wsOverlappedWin | wsVisible
	}
	procSetWindowLong.Call(hwnd, gwlStyle, style)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0,
		uintptr(swpFrameChanged|swpNoMove|swpNoSize|swpNoZorder))
}

// dragMove starts the native move loop, so the borderless widget can be dragged
// from anywhere on its surface.
func dragMove(hwnd uintptr) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	procReleaseCapture.Call()
	procSendMessageW.Call(hwnd, wmNCLButtonDown, htCaption, 0)
}

func minimizeWindow(hwnd uintptr) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd != 0 {
		procShowWindow.Call(hwnd, swMinimize)
	}
}

// syncWebViewClient forces go-webview2's WebView2 controller to recompute its
// bounds after we remove the native frame. SetWindowLongPtr + SetWindowPos can
// change the client area without producing the WM_SIZE path that the library
// normally uses to resize the embedded WebView2 control. If that resize is
// missed, the old browser surface remains smaller than the native popup and
// the Windows default background becomes visible around it.
func syncWebViewClient(hwnd uintptr) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	type rect struct {
		Left, Top, Right, Bottom int32
	}
	r := rect{}
	ret, _, _ := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if ret == 0 {
		return
	}
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	if w <= 0 || h <= 0 {
		return
	}
	// WM_SIZE = 0x0005, SIZE_RESTORED = 0. The go-webview2 wndproc handles
	// WM_SIZE by calling its browser.Resize(), which makes WebView2 exactly
	// cover the native client area.
	const wmSize = 0x0005
	lParam := uintptr(uint32(w) | (uint32(h) << 16))
	procSendMessageW.Call(hwnd, wmSize, 0, lParam)
}

// resizeWidgetWindow changes only the native popup dimensions. Unlike WebView2 SetSize,
// this does not restore the normal overlapped frame, so the frameless single-surface
// geometry survives every drag.
func resizeWidgetWindow(hwnd uintptr, width, height int) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	if width < 300 {
		width = 300
	}
	if width > 1000 {
		width = 1000
	}
	if height < 60 {
		height = 60
	}
	if height > 500 {
		height = 500
	}
	procSetWindowPos.Call(hwnd, 0, 0, 0, uintptr(width), uintptr(height), uintptr(swpNoZorder|swpNoActivate|swpNoMove))
	if !useNativeRoundCorners() {
		setWidgetRegion(hwnd, true, width, height)
	}
	syncWebViewClient(hwnd)
	forceRedraw(hwnd)
}

const (
	rdwInvalidate  = 0x0001
	rdwErase       = 0x0004
	rdwFrame       = 0x0400
	rdwAllChildren = 0x0080
	rdwUpdateNow   = 0x0100
)

// forceRedraw erases and repaints the whole window tree. On a transparent/acrylic
// window a resize can leave stale "ghost" pixels (e.g. exiting widget mode back
// to full size); a full erase+repaint clears them.
func forceRedraw(hwnd uintptr) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	procRedrawWindow.Call(hwnd, 0, 0,
		uintptr(rdwInvalidate|rdwErase|rdwFrame|rdwAllChildren|rdwUpdateNow))
}

const (
	hwndTopmost   = ^uintptr(0)     // (HWND)-1
	hwndNoTopmost = ^uintptr(0) - 1 // (HWND)-2
	swpNoMove     = 0x0002
	swpNoSize     = 0x0001
	swpNoActivate = 0x0010
)

// setTopmost pins or unpins the window above all others (for widget mode).
func setTopmost(hwnd uintptr, on bool) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	after := hwndNoTopmost
	if on {
		after = hwndTopmost
	}
	procSetWindowPos.Call(hwnd, after, 0, 0, 0, 0,
		uintptr(swpNoMove|swpNoSize|swpNoActivate))
}
