//go:build windows

package core

// Widget/window backdrops. Two Windows mechanisms are combined:
//
//   - DWM system backdrop (DWMWA_SYSTEMBACKDROP_TYPE, Windows 11 22H2+): the
//     official Acrylic/Mica. It only paints framed windows reliably, so it is
//     used for the main window.
//   - SetWindowCompositionAttribute + ACCENT_POLICY (Windows 10 1803+ and 11):
//     the long-standing undocumented API also used by the taskbar. It works on
//     the borderless widget popup and gives four looks: off (opaque), acrylic
//     ("cam"), plain blur ("bulanık") and a fully clear surface ("saydam").
//
// Everything is best-effort: unknown builds simply keep the opaque look.

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procSetWindowCompositionAttribute = user32.NewProc("SetWindowCompositionAttribute")

const (
	accentDisabled            = 0
	accentTransparentGradient = 2
	accentBlurBehind          = 3
	accentAcrylicBlurBehind   = 4

	wcaAccentPolicy = 19

	dwmwaWindowCornerPreference = 33
	dwmwcpRound                 = 2
	dwmwcpDefault               = 0
	dwmsbtNone                  = 1
)

type accentPolicy struct {
	AccentState   uint32
	AccentFlags   uint32
	GradientColor uint32 // AABBGGRR
	AnimationID   uint32
}

type winCompAttrData struct {
	Attrib uint32
	PvData unsafe.Pointer
	CbData uintptr
}

var (
	buildOnce sync.Once
	buildNum  uint32
)

// windowsBuild returns the OS build number (e.g. 19045, 22631), 0 if unknown.
func windowsBuild() uint32 {
	buildOnce.Do(func() {
		defer func() { _ = recover() }()
		if v := windows.RtlGetVersion(); v != nil {
			buildNum = v.BuildNumber
		}
	})
	return buildNum
}

func isWin11() bool        { return windowsBuild() >= 22000 }
func hasSysBackdrop() bool { return windowsBuild() >= 22621 }

func setAccent(hwnd uintptr, state uint32, color uint32) bool {
	defer func() { _ = recover() }()
	if hwnd == 0 || procSetWindowCompositionAttribute.Find() != nil {
		return false
	}
	ap := accentPolicy{AccentState: state, GradientColor: color}
	if state == accentTransparentGradient || state == accentAcrylicBlurBehind {
		ap.AccentFlags = 2 // honour GradientColor
	}
	data := winCompAttrData{Attrib: wcaAccentPolicy, PvData: unsafe.Pointer(&ap), CbData: unsafe.Sizeof(ap)}
	r, _, _ := procSetWindowCompositionAttribute.Call(hwnd, uintptr(unsafe.Pointer(&data)))
	return r != 0
}

func setSystemBackdrop(hwnd uintptr, kind int32) {
	defer func() { _ = recover() }()
	if hwnd == 0 {
		return
	}
	procDwmSetWindowAttribute.Call(hwnd, dwmwaSystemBackdropType, uintptr(unsafe.Pointer(&kind)), 4)
}

func setRoundCorners(hwnd uintptr, on bool) {
	defer func() { _ = recover() }()
	if hwnd == 0 || !isWin11() {
		return
	}
	pref := int32(dwmwcpDefault)
	if on {
		pref = dwmwcpRound
	}
	procDwmSetWindowAttribute.Call(hwnd, dwmwaWindowCornerPreference, uintptr(unsafe.Pointer(&pref)), 4)
}

// tintColor builds an AABBGGRR value for the dark (or light) widget tint with
// the given strength 0–100.
func tintColor(strength int, light bool) uint32 {
	if strength < 0 {
		strength = 0
	}
	if strength > 100 {
		strength = 100
	}
	a := uint32(strength * 255 / 100)
	// dark slate #121B2A or light #E5ECF6, stored as BGR
	var r, g, b uint32 = 0x12, 0x1B, 0x2A
	if light {
		r, g, b = 0xE5, 0xEC, 0xF6
	}
	return a<<24 | b<<16 | g<<8 | r
}

// applyWidgetBackdrop sets the widget popup's native backdrop.
//
//	mode: "opak" | "cam" | "bulanik" | "saydam"; tint: 0–100.
func applyWidgetBackdrop(hwnd uintptr, mode string, tint int, light bool) {
	defer func() { _ = recover() }()
	hwnd = nativeHostWindow(hwnd)
	if hwnd == 0 {
		return
	}
	// The DWM system backdrop and the accent policy must not fight each other.
	setSystemBackdrop(hwnd, dwmsbtNone)
	// RC86: saydam/cam/bulanık her zaman koyu tonla çizilir. Açık temada açık
	// ton + açık yüzey üst üste binince okunaksız gri bir kutu oluşuyordu;
	// beyaz yazı + koyu ton her masaüstü arka planında okunur kalır.
	light = false
	switch mode {
	case "opak":
		setAccent(hwnd, accentDisabled, 0)
	case "bulanik":
		if !setAccent(hwnd, accentBlurBehind, 0) {
			setAccent(hwnd, accentDisabled, 0)
		}
	case "saydam":
		// Fully clear unless a tint is requested.
		setAccent(hwnd, accentTransparentGradient, tintColor(tint*6/10, light))
	default: // "cam"
		c := tintColor(tint, light)
		if c>>24 == 0 {
			c |= 0x01 << 24 // alpha 0 renders black on some Windows 10 builds
		}
		if !setAccent(hwnd, accentAcrylicBlurBehind, c) {
			setAccent(hwnd, accentBlurBehind, 0)
		}
	}
}

// clearWidgetBackdrop removes the accent policy (back to a normal window).
func clearWidgetBackdrop(hwnd uintptr) {
	hwnd = nativeHostWindow(hwnd)
	setAccent(hwnd, accentDisabled, 0)
}

// useNativeRoundCorners reports whether the widget should rely on Windows 11's
// rounded corners instead of a clipping region (a region disables DWM effects
// such as blur and the drop shadow on some builds).
func useNativeRoundCorners() bool { return isWin11() }
