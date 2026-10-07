package core

import (
	"net"
	"strings"
	"testing"
)

func TestSafeWebURL(t *testing.T) {
	ok := []string{"https://example.com/a?b=1", "http://bbc.co.uk/x"}
	bad := []string{"file:///C:/Windows/system32/calc.exe", `\\evil\share\x.exe`, "javascript:alert(1)", "ms-msdt:/id", "mailto:a@b.c", "https:///nohost", "", "//example.com"}
	for _, u := range ok {
		if safeWebURL(u) == "" {
			t.Errorf("expected allowed: %s", u)
		}
	}
	for _, u := range bad {
		if safeWebURL(u) != "" {
			t.Errorf("expected blocked: %s", u)
		}
	}
}

func TestIsPublicIP(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.0.0.5", "192.168.1.1", "172.16.3.4", "169.254.1.1", "::1", "fe80::1", "100.64.1.1", "0.0.0.0"} {
		if isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be non-public", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "151.101.1.1", "2a00:1450:4001::1"} {
		if !isPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be public", s)
		}
	}
}

func TestPublicHostRejectsLocal(t *testing.T) {
	for _, h := range []string{"127.0.0.1", "192.168.1.1", "localhost", "router.local", "::1", ""} {
		if publicHost(t.Context(), h) == nil {
			t.Errorf("%q must be rejected", h)
		}
	}
	if err := publicHost(t.Context(), "8.8.8.8"); err != nil {
		t.Errorf("public IP rejected: %v", err)
	}
}

func TestSanitizeDropsScriptLinks(t *testing.T) {
	out := sanitizeArticleHTML(`<p>a <a href="javascript:alert(1)">x</a> <a href="https://ok.example/">y</a> <img src="data:image/svg+xml,<svg>"> <img src="https://img.example/a.jpg"></p>`)
	if strings.Contains(out, "javascript:") || strings.Contains(out, "data:image") {
		t.Fatalf("unsafe attr survived: %s", out)
	}
	if !strings.Contains(out, "https://ok.example/") || !strings.Contains(out, "https://img.example/a.jpg") {
		t.Fatalf("safe attr lost: %s", out)
	}
}
