//go:build windows

package core

// Translucent ("cam") window: two pieces are needed for the desktop to show
// through. (1) The host window gets a DWM system backdrop (Acrylic/Mica) — this
// needs Windows 11 22H2+ for the nicest blur; it degrades silently on older
// builds. (2) The WebView2 control's default background is made transparent via
// ICoreWebView2Controller2.PutDefaultBackgroundColor(alpha=0) so the backdrop is
// visible behind the page. Everything here is best-effort and recovers on any
// failure, so an unsupported OS simply shows the opaque UI.

import (
	"reflect"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"github.com/jchv/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
)

var (
	dwmapi                    = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	dwmwaUseImmersiveDarkMode = 20
	dwmwaSystemBackdropType   = 38
	// system backdrop types
	dwmsbtTransient = 3 // Acrylic (translucent, blurred)
	dwmsbtMica      = 2 // Mica (subtle, tied to desktop wallpaper)
)

// enableGlass turns on the dark frame and an acrylic system backdrop for hwnd.
func enableGlass(hwnd uintptr) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	dark := int32(1)
	procDwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkMode,
		uintptr(unsafe.Pointer(&dark)), 4)

	// The WebView2 controller is transparent, so the system backdrop can paint
	// the client area directly. Extending the DWM frame here creates a visible
	// non-client strip on borderless widget windows, so do not extend it.
	if hasSysBackdrop() {
		setAccent(hwnd, accentDisabled, 0)
		bt := int32(dwmsbtTransient)
		procDwmSetWindowAttribute.Call(hwnd, dwmwaSystemBackdropType,
			uintptr(unsafe.Pointer(&bt)), 4)
		return
	}
	// Windows 10 / early Windows 11: no system backdrop API, use the accent
	// policy acrylic instead.
	setAccent(hwnd, accentAcrylicBlurBehind, tintColor(55, false))
}

// setGlass toggles the acrylic backdrop on or off at runtime.
func setGlass(hwnd uintptr, on bool) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	if on {
		enableGlass(hwnd)
		return
	}
	bt := int32(1) // DWMSBT_NONE
	procDwmSetWindowAttribute.Call(hwnd, dwmwaSystemBackdropType,
		uintptr(unsafe.Pointer(&bt)), 4)
	setAccent(hwnd, accentDisabled, 0)
}

// makeWebviewTransparent sets the WebView2 default background to fully
// transparent so the window's backdrop shows through the page.

// setWebviewBackground controls the WebView2 controller's fallback paint color.
// In widget mode we deliberately use an opaque fallback so the native host can
// never bleed through as a white/black rectangle around the page surface.
func setWebviewBackground(w webview2.WebView, a, r, g, b byte) {
	defer func() { _ = recover() }()
	ch := browserChromium(w)
	if ch == nil {
		return
	}
	ctrl := ch.GetController()
	if ctrl == nil {
		return
	}
	c2 := ctrl.GetICoreWebView2Controller2()
	if c2 == nil {
		return
	}
	_ = c2.PutDefaultBackgroundColor(edge.COREWEBVIEW2_COLOR{A: a, R: r, G: g, B: b})
}

func makeWebviewTransparent(w webview2.WebView) {
	defer func() { _ = recover() }()
	ch := browserChromium(w)
	if ch == nil {
		return
	}
	ctrl := ch.GetController()
	if ctrl == nil {
		return
	}
	c2 := ctrl.GetICoreWebView2Controller2()
	if c2 == nil {
		return
	}
	_ = c2.PutDefaultBackgroundColor(edge.COREWEBVIEW2_COLOR{A: 0, R: 0, G: 0, B: 0})
}

// browserChromium reaches the *edge.Chromium held in the (unexported) browser
// field of the high-level webview via reflection. The concrete type and its
// GetController method are exported, so this is only about reading one private
// field; it is wrapped in recover and returns nil on any mismatch.
func browserChromium(w webview2.WebView) *edge.Chromium {
	defer func() { _ = recover() }()
	rv := reflect.ValueOf(w)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil
	}
	f := rv.Elem().FieldByName("browser")
	if !f.IsValid() {
		return nil
	}
	f = reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	if ch, ok := f.Interface().(*edge.Chromium); ok {
		return ch
	}
	return nil
}
