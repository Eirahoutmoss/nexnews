package core

// Optional headline translation into the user's local language. Uses the public
// Google translate endpoint (no API key); it auto-detects the source language,
// so feeds in any language fold into one reading language. This is best-effort:
// if the endpoint is unreachable or rate-limits, originals are shown unchanged.
// Results are cached by (lang|text) so repeated refreshes don't re-translate.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type translator struct {
	mu       sync.RWMutex
	cache    map[string]string // key: target\x00source-text -> translated
	client   *http.Client
	localURL string

	diskPath string // persisted cache (translations.json); "" = memory only
	dirty    bool

	bmu     sync.Mutex
	backoff map[string]time.Time // provider -> skip until (after a failure)
}

func newTranslator() *translator {
	return &translator{
		cache:    map[string]string{},
		client:   &http.Client{Timeout: 8 * time.Second},
		localURL: "http://127.0.0.1:5000",
	}
}

func trKey(target, text string) string { return target + "\x00" + text }

func (t *translator) cached(target, text string) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	v, ok := t.cache[trKey(target, text)]
	return v, ok
}

func (t *translator) store(target, text, out string) {
	t.mu.Lock()
	t.cache[trKey(target, text)] = out
	t.dirty = true
	t.mu.Unlock()
}

const maxPersistedTranslations = 20000

type persistedTranslations struct {
	Version int               `json:"version"`
	Items   map[string]string `json:"items"`
}

// attachDisk loads a previously saved translation cache and remembers where to
// save it, so headlines are not re-translated on every start.
func (t *translator) attachDisk(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.diskPath = path
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var p persistedTranslations
	if json.Unmarshal(b, &p) != nil || p.Version != 1 {
		return
	}
	for k, v := range p.Items {
		if _, ok := t.cache[k]; !ok {
			t.cache[k] = v
		}
	}
}

// saveDisk writes the cache if it changed. Oldest-first ordering is not
// tracked, so when the cap is exceeded an arbitrary half is dropped; current
// headlines are re-translated on demand.
func (t *translator) saveDisk() error {
	t.mu.Lock()
	if t.diskPath == "" || !t.dirty {
		t.mu.Unlock()
		return nil
	}
	items := make(map[string]string, len(t.cache))
	for k, v := range t.cache {
		items[k] = v
	}
	if len(items) > maxPersistedTranslations {
		n := 0
		for k := range items {
			delete(items, k)
			n++
			if len(items) <= maxPersistedTranslations/2 {
				break
			}
		}
		t.cache = make(map[string]string, len(items))
		for k, v := range items {
			t.cache[k] = v
		}
	}
	t.dirty = false
	path := t.diskPath
	t.mu.Unlock()
	b, err := json.Marshal(persistedTranslations{Version: 1, Items: items})
	if err != nil {
		return err
	}
	return atomicWriteFile(path, b)
}

// providerDown/markProvider implement a small circuit breaker: a provider that
// failed is skipped for a while, so an unreachable mirror does not cost its
// full timeout for every headline.
func (t *translator) providerDown(name string) bool {
	t.bmu.Lock()
	defer t.bmu.Unlock()
	until, ok := t.backoff[name]
	return ok && time.Now().Before(until)
}

func (t *translator) markProvider(name string, err error) {
	t.bmu.Lock()
	defer t.bmu.Unlock()
	if t.backoff == nil {
		t.backoff = map[string]time.Time{}
	}
	if err == nil {
		delete(t.backoff, name)
		return
	}
	t.backoff[name] = time.Now().Add(10 * time.Minute)
}

// translate returns the translation of text into target, using the cache when
// possible. On any failure it returns the original text so the UI degrades
// gracefully.
func (t *translator) translate(ctx context.Context, text, target string) string {
	text = strings.TrimSpace(text)
	if text == "" || target == "" {
		return text
	}
	if v, ok := t.cached(target, text); ok {
		return v
	}
	out, _, err := t.fetch(ctx, text, target)
	if err != nil || out == "" {
		return text
	}
	t.store(target, text, out)
	return out
}

// probe runs one live translation for diagnostics and reports which provider
// answered (via) or the last error, so the settings panel can tell the user
// whether translation is reachable from this machine and through what route.
func (t *translator) probe(ctx context.Context, text, target string) (out, via string, err error) {
	return t.fetch(ctx, text, target)
}

// fetch tries several keyless providers in turn, so a network that blocks one
// family of hosts (common on Turkish home ISPs that throttle Google's
// translate endpoints) still gets results. It returns the provider that
// answered, or the last error seen across all providers.
func (t *translator) fetch(ctx context.Context, text, target string) (string, string, error) {
	var lastErr error = errURL
	try := func(name string, f func() (string, error)) (string, bool) {
		if t.providerDown(name) {
			return "", false
		}
		out, err := f()
		if err == nil && out == "" {
			err = errURL
		}
		if ctx.Err() == nil { // a cancelled pass says nothing about the provider
			t.markProvider(name, err)
		}
		if err != nil {
			lastErr = err
			return "", false
		}
		return out, true
	}
	// Prefer the local self-hosted LibreTranslate/Argos engine. This avoids
	// public-instance rate limits, CORS, and ISP throttling. If it is not
	// running, retain the legacy public-provider fallback for compatibility.
	if out, ok := try("local", func() (string, error) { return t.fetchLocal(ctx, text, target) }); ok {
		return out, "Local LibreTranslate", nil
	}
	// Google gtx format, across hosts that often differ in reachability.
	for _, host := range []string{"translate.googleapis.com", "clients5.google.com"} {
		h := host
		if out, ok := try(h, func() (string, error) { return t.fetchGTXHost(ctx, h, text, target) }); ok {
			return out, "Google (" + h + ")", nil
		}
	}
	// Lingva mirrors (keyless Google front-ends).
	for _, host := range []string{"lingva.lunar.icu", "lingva.garudalinux.org", "translate.plausibility.cloud", "lingva.ml"} {
		h := host
		if out, ok := try(h, func() (string, error) { return t.fetchLingva(ctx, h, text, target) }); ok {
			return out, "Lingva (" + h + ")", nil
		}
	}
	// SimplyTranslate instances (another keyless Google proxy family).
	for _, host := range []string{"simplytranslate.org", "st.tokhmi.xyz"} {
		h := host
		if out, ok := try(h, func() (string, error) { return t.fetchSimply(ctx, h, text, target) }); ok {
			return out, "SimplyTranslate (" + h + ")", nil
		}
	}
	return "", "", lastErr
}

func (t *translator) fetchLocal(ctx context.Context, text, target string) (string, error) {
	payload := struct {
		Q      string `json:"q"`
		Source string `json:"source"`
		Target string `json:"target"`
		Format string `json:"format"`
	}{Q: text, Source: "auto", Target: target, Format: "text"}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(t.localURL, "/") + "/translate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(b)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errURL
	}
	var out struct {
		Translated string `json:"translatedText"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return "", err
	}
	translated := strings.TrimSpace(out.Translated)
	if translated == "" {
		return "", errURL
	}
	return translated, nil
}

func (t *translator) get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 NexNews/0.1")
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errURL
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

func (t *translator) fetchGTXHost(ctx context.Context, host, text, target string) (string, error) {
	endpoint := "https://" + host + "/translate_a/single?client=gtx&sl=auto&tl=" +
		url.QueryEscape(target) + "&dt=t&q=" + url.QueryEscape(text)
	data, err := t.get(ctx, endpoint)
	if err != nil {
		return "", err
	}
	return parseTranslate(data)
}

// fetchSimply uses a SimplyTranslate instance (keyless): GET
// /api/translate/?engine=google&from=auto&to=<target>&text=<text>
// -> {"translated-text":"..."}.
func (t *translator) fetchSimply(ctx context.Context, host, text, target string) (string, error) {
	endpoint := "https://" + host + "/api/translate/?engine=google&from=auto&to=" +
		url.QueryEscape(target) + "&text=" + url.QueryEscape(text)
	data, err := t.get(ctx, endpoint)
	if err != nil {
		return "", err
	}
	var r struct {
		Translated string `json:"translated-text"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", err
	}
	if strings.TrimSpace(r.Translated) == "" {
		return "", errURL
	}
	return r.Translated, nil
}

// fetchLingva uses a Lingva Translate instance (a keyless Google Translate
// front-end): GET /api/v1/auto/<target>/<text> -> {"translation":"..."}.
func (t *translator) fetchLingva(ctx context.Context, host, text, target string) (string, error) {
	endpoint := "https://" + host + "/api/v1/auto/" + url.PathEscape(target) + "/" + url.PathEscape(text)
	data, err := t.get(ctx, endpoint)
	if err != nil {
		return "", err
	}
	var r struct {
		Translation string `json:"translation"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", err
	}
	if r.Translation == "" {
		return "", errURL
	}
	return r.Translation, nil
}

// parseTranslate extracts the translated text from the gtx JSON response, which
// looks like: [[["translated","original",...],["seg2","orig2",...]], ...].
func parseTranslate(data []byte) (string, error) {
	var root []json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil || len(root) == 0 {
		return "", errURL
	}
	var segs []json.RawMessage
	if err := json.Unmarshal(root[0], &segs); err != nil {
		return "", errURL
	}
	var b strings.Builder
	for _, s := range segs {
		var parts []json.RawMessage
		if err := json.Unmarshal(s, &parts); err != nil || len(parts) == 0 {
			continue
		}
		var chunk string
		if err := json.Unmarshal(parts[0], &chunk); err == nil {
			b.WriteString(chunk)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", errURL
	}
	return out, nil
}
