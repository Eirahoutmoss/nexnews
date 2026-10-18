package core

import (
	"bufio"
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
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
	Transport: &countingTransport{base: liveTransport},
	Timeout:   90 * time.Second, // sınırlı veri modunda parçalar yavaş iner; asılı istekleri tarayıcı zaten iptal eder
	Jar:       liveJar,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return publicHost(req.Context(), req.URL.Hostname())
	},
}

// liveTransport dials only public addresses (see publicDialContext).
var liveTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           publicDialContext,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          64,
	MaxIdleConnsPerHost:   8,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 20 * time.Second,
}

// proxyHosts: the environment proxy may be on the LAN; allow dialing it.
var proxyHosts = func() map[string]bool {
	m := map[string]bool{}
	for _, k := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"} {
		if v := os.Getenv(k); v != "" {
			if u, err := url.Parse(v); err == nil && u.Hostname() != "" {
				m[strings.ToLower(u.Hostname())] = true
			}
		}
	}
	return m
}()

// Some CDNs (Akamai token auth, Wowza sessions) set a cookie on the master
// playlist and require it on every chunk; keep them per run.
var liveJar, _ = cookiejar.New(nil)

// liveHeaders is the optional per-channel Referer / User-Agent carried in the
// relay URL (r= / ua=) and inherited by every child playlist and segment.
type liveHeaders struct{ ref, ua string }

func (h liveHeaders) query() string {
	q := ""
	if h.ref != "" {
		q += "&r=" + url.QueryEscape(h.ref)
	}
	if h.ua != "" {
		q += "&ua=" + url.QueryEscape(h.ua)
	}
	return q
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
		return ""
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
	case strings.Contains(u.Host, "impresa.pt") && strings.Contains(u.Path, "sicnot"):
		return "https://sicnoticias.pt/"
	case strings.Contains(u.Host, "iblups.com"):
		return "http://tvperu.gob.pe/"
	case strings.Contains(u.Host, "cdnmedia.tv"):
		return "https://www.telecaribe.co/"
	case strings.Contains(u.Host, "bcovlive") && strings.Contains(u.Path, "6282251407001"):
		return "https://www.taiwanplus.com/"
	case strings.Contains(u.Host, "mangomolo.com"):
		return "https://www.sabcplus.com/"
	}
	// Bilinmeyen CDN: sahte bir Referer göndermek (eskiden google.com) birçok
	// sunucuda 403'e yol açıyordu; hiç göndermemek daha güvenli.
	return ""
}

func relayURL(raw string, h liveHeaders) string {
	return "http://" + liveRelayAddr + "/live?t=" + liveRelayToken + "&u=" + url.QueryEscape(raw) + h.query()
}

// rewriteM3U8 points every URI of the playlist at the relay. When maxHeight is
// > 0 (sınırlı veri modu) a master playlist keeps only the variants up to that
// height (or the smallest one when none qualifies).
func rewriteM3U8(body, baseURL string, h liveHeaders, maxHeight int) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return body
	}
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	drop := map[int]bool{}
	if maxHeight > 0 {
		drop = lowVariantsOnly(lines, maxHeight)
	}
	var out strings.Builder
	for i, line := range lines {
		if drop[i] {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#EXT-X-KEY:") || strings.HasPrefix(trimmed, "#EXT-X-MAP:") || strings.HasPrefix(trimmed, "#EXT-X-MEDIA:") || strings.HasPrefix(trimmed, "#EXT-X-I-FRAME-STREAM-INF:") {
			line = rewriteTagURI(line, base, h)
		} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			line = relayURL(resolveLiveURL(base, trimmed), h)
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

// lowVariantsOnly marks the #EXT-X-STREAM-INF lines (and their URI line) taller
// than maxHeight for removal. Nothing is removed when it would empty the list.
func lowVariantsOnly(lines []string, maxHeight int) map[int]bool {
	type variant struct{ tag, uri, height int }
	var vs []variant
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "#EXT-X-STREAM-INF:") {
			continue
		}
		h := 0
		if k := strings.Index(t, "RESOLUTION="); k >= 0 {
			r := t[k+len("RESOLUTION="):]
			if c := strings.IndexAny(r, ","); c >= 0 {
				r = r[:c]
			}
			if x := strings.IndexByte(r, 'x'); x >= 0 {
				fmt.Sscanf(r[x+1:], "%d", &h)
			}
		}
		for j := i + 1; j < len(lines); j++ {
			u := strings.TrimSpace(lines[j])
			if u != "" && !strings.HasPrefix(u, "#") {
				vs = append(vs, variant{tag: i, uri: j, height: h})
				break
			}
		}
	}
	drop := map[int]bool{}
	if len(vs) < 2 {
		return drop
	}
	keep, minH, minIdx := 0, 1<<30, -1
	for i, v := range vs {
		if v.height > 0 && v.height <= maxHeight {
			keep++
		}
		if v.height > 0 && v.height < minH {
			minH, minIdx = v.height, i
		}
	}
	for i, v := range vs {
		if v.height == 0 {
			continue // çözünürlüğü yazmayan (ör. yalnız ses) varyantlar kalır
		}
		if (keep > 0 && v.height > maxHeight) || (keep == 0 && i != minIdx) {
			drop[v.tag], drop[v.uri] = true, true
		}
	}
	return drop
}

func rewriteTagURI(line string, base *url.URL, h liveHeaders) string {
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
	return line[:start] + relayURL(abs, h) + line[end:]
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
	for _, k := range []string{"Content-Length", "Content-Range", "Accept-Ranges"} {
		if v := r.Header.Get(k); v != "" {
			w.Header().Set(k, v)
		}
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
	hdr := liveHeaders{ref: r.URL.Query().Get("r"), ua: r.URL.Query().Get("ua")}
	resp, err := liveFetch(r.Context(), target.String(), hdr, r.Header.Get("Range"))
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

	// Oynatma listesi mi? Content-Type ve uzantı yetmez (bazı sunucular
	// text/plain ya da octet-stream gönderir, bazı adreslerde .m3u8 yok):
	// gövdenin başına bakılır.
	br := bufio.NewReaderSize(resp.Body, 64*1024)
	head, _ := br.Peek(16)
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	isPlaylist := strings.Contains(ct, "mpegurl") || strings.HasPrefix(strings.TrimLeft(string(head), "\ufeff \r\n\t"), "#EXTM3U")
	if !isPlaylist {
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, br)
		return
	}

	data, err := io.ReadAll(io.LimitReader(br, 4*1024*1024))
	if err != nil {
		http.Error(w, "playlist read failed", http.StatusBadGateway)
		return
	}
	// Göreli adresler YÖNLENDİRME SONRASI adrese göre çözülmeli (wurl, amagi,
	// i.mjh.nz, jmp2.uk gibi FAST yayınları başka bir sunucuya yönlendirir).
	base := target.String()
	if resp.Request != nil && resp.Request.URL != nil {
		base = resp.Request.URL.String()
	}
	maxH := 0
	if dataLimited() {
		maxH = 480
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Del("Content-Length")
	w.WriteHeader(resp.StatusCode)
	_, _ = fmt.Fprint(w, rewriteM3U8(string(data), base, hdr, maxH))
}

// liveFetch GETs a stream URL with the channel's headers; a playlist request
// that fails transiently is retried once.
func liveFetch(ctx context.Context, raw string, hdr liveHeaders, rng string) (*http.Response, error) {
	ref := hdr.ref
	if ref == "" {
		ref = liveReferer(raw)
	}
	ua := hdr.ua
	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0"
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "*/*")
		if rng != "" {
			req.Header.Set("Range", rng)
		}
		if ref != "" {
			req.Header.Set("Referer", ref)
			if u, _ := url.Parse(ref); u != nil && u.Host != "" {
				req.Header.Set("Origin", u.Scheme+"://"+u.Host)
			}
		}
		resp, err := liveUpstreamClient().Do(req)
		if err == nil && (resp.StatusCode < 500 || attempt == 1) {
			return resp, nil
		}
		if err == nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("upstream %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		if ctx.Err() != nil {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	return nil, lastErr
}
