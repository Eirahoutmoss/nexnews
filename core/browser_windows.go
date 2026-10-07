//go:build windows

package core

import (
	"errors"
	"os/exec"
	"syscall"
)

// openInBrowser opens url in the user's default browser without flashing a
// console window. Only absolute http(s) links are handed to the shell: feed
// links are untrusted, and FileProtocolHandler would happily launch file:,
// UNC paths or custom protocol handlers.
func openInBrowser(url string) error {
	safe := safeWebURL(url)
	if safe == "" {
		return errors.New("yalnızca http/https bağlantıları açılabilir")
	}
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", safe)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
