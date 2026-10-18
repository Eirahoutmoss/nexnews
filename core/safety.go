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

var (
	errPrivateTarget = errors.New("özel/yerel ağ adresine erişim engellendi")
	errDNSFail       = errors.New("alan adı çözülemedi (DNS)")
	errDNSSinkhole   = errors.New("DNS bu adresi engelliyor (0.0.0.0 / 127.x yanıtı)")
)

// resolvePublic resolves host and returns only its globally routable
// addresses. RC83: eskiden adreslerden biri bile yerel görünse (ör. DNS hem
// IPv6 ULA hem genel IPv4 döndürdüğünde) ya da ad hiç çözülemese "yerel ağ
// engellendi" deniyordu; artık genel adres varsa onunla bağlanılır ve hata
// nedeni ayrı ayrı bildirilir.
func resolvePublic(ctx context.Context, host string) ([]net.IP, error) {
	if host == "" {
		return nil, errPrivateTarget
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return nil, errPrivateTarget
		}
		return []net.IP{ip}, nil
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".local") {
		return nil, errPrivateTarget
	}
	rctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(rctx, host)
	if err != nil || len(addrs) == 0 {
		return nil, errDNSFail
	}
	var pub []net.IP
	sink := false
	for _, a := range addrs {
		if isPublicIP(a.IP) {
			pub = append(pub, a.IP)
		} else if a.IP.IsUnspecified() || a.IP.IsLoopback() {
			sink = true
		}
	}
	if len(pub) == 0 {
		if sink {
			return nil, errDNSSinkhole
		}
		return nil, errPrivateTarget
	}
	return pub, nil
}

// publicHost fails unless host has at least one public address.
func publicHost(ctx context.Context, host string) error {
	_, err := resolvePublic(ctx, host)
	return err
}

// publicDialContext connects only to the public addresses of the target, so a
// DNS answer can never steer the relay into the LAN. The configured HTTP(S)
// proxy (environment) is the one exception: it may live on the LAN.
func publicDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	d := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	if proxyHosts[strings.ToLower(host)] {
		return d.DialContext(ctx, network, addr)
	}
	ips, err := resolvePublic(ctx, host)
	if err != nil {
		return nil, err
	}
	var last error
	for _, ip := range ips {
		c, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return c, nil
		}
		last = err
	}
	return nil, last
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))[:2*n]
	}
	return hex.EncodeToString(b)
}
