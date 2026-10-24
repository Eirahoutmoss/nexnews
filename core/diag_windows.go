package core

// Widget saydamlık teşhisi (RC90). Widget görünümü her uygulandığında hangi
// adımın tuttuğunu exe'nin yanındaki widget-teshis.txt dosyasına yazar.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"github.com/jchv/go-webview2/pkg/edge"
	"golang.org/x/sys/windows"
)

var (
	diagMu              sync.Mutex
	diagLines           []string
	procGetWindowLongW  = user32.NewProc("GetWindowLongPtrW")
	procEnumChildWindow = user32.NewProc("EnumChildWindows")
	procGetClassNameW   = user32.NewProc("GetClassNameW")
)

func diagf(format string, a ...any) {
	diagMu.Lock()
	diagLines = append(diagLines, time.Now().Format("15:04:05.000")+" "+fmt.Sprintf(format, a...))
	if len(diagLines) > 400 {
		diagLines = diagLines[len(diagLines)-400:]
	}
	txt := strings.Join(diagLines, "\r\n")
	diagMu.Unlock()
	if exe, err := os.Executable(); err == nil {
		_ = os.WriteFile(filepath.Join(filepath.Dir(exe), "widget-teshis.txt"), []byte(txt), 0o644)
	}
}

func winLong(hwnd uintptr, idx int32) uintptr {
	r, _, _ := procGetWindowLongW.Call(hwnd, uintptr(idx))
	return r
}

func className(hwnd uintptr) string {
	buf := make([]uint16, 128)
	procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 128)
	return syscall.UTF16ToString(buf)
}

// diagWindow: pencere ve tüm alt pencerelerin sınıf/stil bilgisi.
func diagWindow(tag string, hwnd uintptr) {
	defer func() { _ = recover() }()
	host := nativeHostWindow(hwnd)
	gwlStyle, gwlEx := int32(-16), int32(-20)
	diagf("[%s] build=%d host=%#x class=%s style=%#x exstyle=%#x env=%q", tag, windowsBuild(), host, className(host),
		winLong(host, gwlStyle), winLong(host, gwlEx), os.Getenv("WEBVIEW2_DEFAULT_BACKGROUND_COLOR"))
	cb := windows.NewCallback(func(h, _ uintptr) uintptr {
		diagf("   child=%#x class=%s style=%#x exstyle=%#x", h, className(h), winLong(h, gwlStyle), winLong(h, gwlEx))
		return 1
	})
	procEnumChildWindow.Call(host, cb, 0)
}

// diagBackground: WebView2 arka planını okur.
func diagBackground(tag string, w webview2.WebView) {
	defer func() {
		if r := recover(); r != nil {
			diagf("[%s] arka plan okunamadı: %v", tag, r)
		}
	}()
	ch := browserChromium(w)
	if ch == nil {
		diagf("[%s] browserChromium=nil", tag)
		return
	}
	ctrl := ch.GetController()
	if ctrl == nil {
		diagf("[%s] controller=nil", tag)
		return
	}
	c2 := ctrl.GetICoreWebView2Controller2()
	if c2 == nil {
		diagf("[%s] controller2=nil", tag)
		return
	}
	err := c2.PutDefaultBackgroundColor(edge.COREWEBVIEW2_COLOR{})
	col, gerr := c2.GetDefaultBackgroundColor()
	diagf("[%s] put=%v get=%+v getErr=%v", tag, err, col, gerr)
}
