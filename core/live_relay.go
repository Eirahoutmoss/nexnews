package core

import (
	"bufio"
	"crypto/subtle"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const liveRelayAddr = "127.0.0.1:39765"

var liveRelayOnce sync.Once

// liveRelayToken is a per-run secret. Only the NexNews page knows it (it is
// injected into the HTML at startup), so other web pages open in the user's
// browser cannot use the localhost relay as a proxy.
var liveRelayToken = randomToken(16)

// liveUpstream fetches stream playlists/segments. Targets (and redirects) are
// checked by publicHost before connecting, so the relay never reaches the LAN;
// the normal transport is kept so a corporate HTTP proxy still works.
var liveUpstream = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return publicHost(req.Context(), req.URL.Hostname())
	},
}

func startLiveRelay() {
	liveRelayOnce.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusNoContent)
		})
		mux.HandleFunc("/live", liveRelayHandler)
		ln, err := net.Listen("tcp", liveRelayAddr)
		if err != nil {
			log.Printf("NexNews Live Relay başlatılamadı: %v", err)
			return
		}
		srv := &http.Server{Handler: mux, ReadHeaderTimeout: 8 * time.Second}
		go func() {
			if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("NexNews Live Relay durdu: %v", err)
			}
		}()
	})
}

func liveUpstreamClient() *http.Client { return liveUpstream }

func liveReferer(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "https://www.google.com/"
	}
	switch {
	case strings.Contains(u.Host, "daioncdn.net") && strings.Contains(u.Path, "/kanald/"):
		return "https://www.kanald.com.tr/canli-yayin/"
	case strings.Contains(u.Host, "daioncdn.net") && strings.Contains(u.Path, "/showtv/"):
		return "https://www.showtv.com.tr/canli-yayin"
	case strings.Contains(u.Host, "daioncdn.net") && strings.Contains(u.Path, "/startv/"):
		return "https://www.startv.com.tr/canli-yayin"
	case strings.Contains(u.Host, "daioncdn.net") && strings.Contains(u.Path, "/haberturktv/"):
		return "https://tv.haberturk.com/"
	case strings.Contains(u.Host, "daioncdn.net") && strings.Contains(u.Path, "/tv100/"):
		return "https://www.tv100.com/canli-yayin"
	case strings.Contains(u.Host, "bbci.co.uk") || strings.Contains(u.Host, "fastly.md.bbci.co.uk"):
		return "https://www.bbc.com/"
	case strings.Contains(u.Host, "akamaized.net") && strings.Contains(u.Host, "dwamdstream"):
		return "https://www.dw.com/"
	case strings.Contains(u.Host, "france24.com"):
		return "https://www.france24.com/"
	case strings.Contains(u.Host, "cdn-euronews.akamaized.net"):
		return "https://www.euronews.com/"
	case strings.Contains(u.Host, "ard-mcdn.de") || strings.Contains(u.Host, "tagesschau.akamaized.net"):
		return "https://www.ardmediathek.de/"
	case strings.Contains(u.Host, "zdf-hls-"):
		return "https://www.zdf.de/"
	case strings.Contains(u.Host, "artesimulcast.akamaized.net"):
		return "https://www.arte.tv/"
	case strings.Contains(u.Host, "bfmtv.bct.nextradiotv.com"):
		return "https://www.bfmtv.com/"
	case strings.Contains(u.Host, "cloudfront.net") && strings.Contains(u.Path, "/RAINEWS24/"):
		return "https://www.rainews.it/"
	case strings.Contains(u.Host, "d3749synfikwkv.cloudfront.net"):
		return "https://www.la7.it/"
	case strings.Contains(u.Host, "streaming-live.rtp.pt"):
		return "https://www.rtp.pt/"
	case strings.Contains(u.Host, "impresa.pt"):
		return "https://sicnoticias.pt/"
	case strings.Contains(u.Host, "netplus.ch"):
		return "https://www.srf.ch/"
	case strings.Contains(u.Host, "svt-live-channel.akamaized.net"):
		return "https://www.svtplay.se/"
	case strings.Contains(u.Host, "nrk-nrk1.akamaized.net"):
		return "https://tv.nrk.no/"
	case strings.Contains(u.Host, "nhkwlive-ojp.akamaized.net") || strings.Contains(u.Host, "nhkworld.jp"):
		return "https://www3.nhk.or.jp/nhkworld/en/live/"
	case strings.Contains(u.Host, "arirang") || strings.Contains(u.Host, "ctnd-com.akamaized.net"):
		return "https://www.arirang.com/live"
	case strings.Contains(u.Host, "kbsworld-ott.akamaized.net"):
		return "https://kbsworld.kbs.co.kr/"
	case strings.Contains(u.Host, "cna") || strings.Contains(u.Host, "cloudfront.net") && strings.Contains(u.Path, "/7782e205e72f43aeb4a48ec97f66ebbe/"):
		return "https://www.channelnewsasia.com/watch/live-tv-1441426"
	case strings.Contains(u.Host, "getaj.net") || strings.Contains(u.Host, "thehlive.com"):
		return "https://www.aljazeera.com/video/live/"
	case strings.Contains(u.Host, "cgtn.com"):
		return "https://www.cgtn.com/tv"
	case strings.Contains(u.Host, "ntv.co.jp"):
		return "https://www.news24.jp/livestream/"
	case strings.Contains(u.Host, "alarabiya.net") || strings.Contains(u.Host, "edgenextcdn.net"):
		return "https://www.alarabiya.net/"
	case strings.Contains(u.Host, "asharq.com"):
		return "https://asharq.com/live/"
	case strings.Contains(u.Host, "5centscdn.com") || strings.Contains(u.Host, "tamashaweb.com"):
		return "https://www.geo.tv/live"
	case strings.Contains(u.Host, "akamaized.net") && strings.Contains(u.Path, "/indiatoday/"):
		return "https://www.indiatoday.in/livetv"
	case strings.Contains(u.Host, "indiatvnews.akamaized.net"):
		return "https://www.indiatvnews.com/livetv"
	case strings.Contains(u.Host, "ndtv24x7elemarchana.akamaized.net") || strings.Contains(u.Host, "ndtvprofit.akamaized.net"):
		return "https://www.ndtv.com/livetv"
	case strings.Contains(u.Host, "n18syndication.akamaized.net") || strings.Contains(u.Host, "nw18live.cdn.jio.com"):
		return "https://hindi.news18.com/livetv/"
	case strings.Contains(u.Host, "media.nic.in"):
		return "https://sansadtv.nic.in/live"
	}
	return "https://www.google.com/"
}

func relayURL(raw string) string {
	return "http://" + liveRelayAddr + "/live?t=" + liveRelayToken + "&u=" + url.QueryEscape(raw)
}

func rewriteM3U8(body, baseURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return body
	}
	var out strings.Builder
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#EXT-X-KEY:") || strings.HasPrefix(trimmed, "#EXT-X-MAP:") || strings.HasPrefix(trimmed, "#EXT-X-MEDIA:") {
			line = rewriteTagURI(line, base)
		} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			line = rewritePlaylistURI(trimmed, base)
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

func rewriteTagURI(line string, base *url.URL) string {
	marker := "URI=\""
	i := strings.Index(line, marker)
	if i < 0 {
		return line
	}
	start := i + len(marker)
	end := strings.Index(line[start:], "\"")
	if end < 0 {
		return line
	}
	end += start
	ref := line[start:end]
	abs := resolveLiveURL(base, ref)
	return line[:start] + relayURL(abs) + line[end:]
}

func rewritePlaylistURI(ref string, base *url.URL) string {
	return relayURL(resolveLiveURL(base, ref))
}

func resolveLiveURL(base *url.URL, ref string) string {
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ref
	}
	if u.IsAbs() {
		return u.String()
	}
	// Some CDN playlists contain root-relative paths. ResolveReference handles both.
	return base.ResolveReference(u).String()
}

func copyLiveHeaders(w http.ResponseWriter, r *http.Response) {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if cl := r.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
}

func liveRelayHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("t")), []byte(liveRelayToken)) != 1 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("u"))
	if raw == "" {
		http.Error(w, "missing u", http.StatusBadRequest)
		return
	}
	target, err := url.Parse(raw)
	if err != nil || (target.Scheme != "https" && target.Scheme != "http") || target.Host == "" {
		http.Error(w, "invalid upstream", http.StatusBadRequest)
		return
	}

	if err := publicHost(r.Context(), target.Hostname()); err != nil {
		http.Error(w, "forbidden upstream", http.StatusForbidden)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/154 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", liveReferer(target.String()))
	req.Header.Set("Origin", func() string {
		u, _ := url.Parse(liveReferer(target.String()))
		if u != nil {
			return u.Scheme + "://" + u.Host
		}
		return ""
	}())

	resp, err := liveUpstreamClient().Do(req)
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	copyLiveHeaders(w, resp)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.WriteHeader(resp.StatusCode)
		return
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	isPlaylist := strings.Contains(ct, "mpegurl") || strings.HasSuffix(strings.ToLower(target.Path), ".m3u8")
	if !isPlaylist {
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		return
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		http.Error(w, "playlist read failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Del("Content-Length")
	w.WriteHeader(resp.StatusCode)
	_, _ = fmt.Fprint(w, rewriteM3U8(string(data), target.String()))
}
