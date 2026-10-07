package core

// Shared safety helpers: URL scheme allow-listing (feed/article links are
// untrusted input) and a dialer that refuses private/loopback addresses so the
// local live-TV relay can never be used to reach the user's LAN.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

// safeWebURL returns the normalised URL when it is an absolute http(s) link,
// or "" for anything else (file:, javascript:, UNC paths, custom protocols…).
func safeWebURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, `\\`) {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.String()
	}
	return ""
}

// isPublicIP reports whether ip is a globally routable unicast address.
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		// 100.64.0.0/10 carrier-grade NAT, 0.0.0.0/8, 192.0.0.0/24, 198.18.0.0/15
		switch {
		case v4[0] == 0,
			v4[0] == 100 && v4[1]&0xC0 == 64,
			v4[0] == 192 && v4[1] == 0 && v4[2] == 0,
			v4[0] == 198 && (v4[1] == 18 || v4[1] == 19):
			return false
		}
	}
	return true
}

var errPrivateTarget = errors.New("özel/yerel ağ adresine erişim engellendi")

// publicHost resolves host and fails if it is, or resolves to, a non-public
// address (loopback, LAN, link-local, CGNAT…).
func publicHost(ctx context.Context, host string) error {
	if host == "" {
		return errPrivateTarget
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return errPrivateTarget
		}
		return nil
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".local") {
		return errPrivateTarget
	}
	rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(rctx, host)
	if err != nil {
		return err
	}
	for _, a := range addrs {
		if !isPublicIP(a.IP) {
			return errPrivateTarget
		}
	}
	return nil
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:2*n]
	}
	return hex.EncodeToString(b)
}
