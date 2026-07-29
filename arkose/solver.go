// Package arkose is a pure-Go Arkose Labs / FunCaptcha token solver. It builds a real-browser
// BDA fingerprint, encrypts it with the site's RSA-OAEP + AES-GCM envelope, and posts it to
// /fc/gt2/public_key/{pk} to obtain a suppressed (sup=1) token — no browser, no PoW.
//
// Basic use:
//
//	s, err := arkose.New(
//	    arkose.WithSurl("https://verify.example.com"),
//	    arkose.WithPublicKey("YOUR-SITE-KEY-UUID"),
//	    arkose.WithRSAPublicKey("MIIBIjAN..."),   // see README: capture once
//	    arkose.WithSite("https://www.example.com"),
//	    arkose.WithProxy("http://user:pass@host:port"),
//	)
//	res, err := s.Solve()
//	fmt.Println(res.Token)
package arkose

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// capiCacheEntry stores a per-public-key /v2/{pk}/api.js lookup: capi version + build id.
type capiCacheEntry struct {
	Version   string
	VMKey     string
	BuildID   string
	FetchedAt time.Time
}

var (
	capiCache   = map[string]*capiCacheEntry{}
	capiCacheMu sync.Mutex
	capiCacheTL = 5 * time.Minute
)

// Solver holds a solving context bound to one Config. Construct with New(). It is safe to
// reuse across sequential Solve() calls; for concurrency, create one Solver per goroutine
// (each keeps its own TLS client + cookie jar).
type Solver struct {
	cfg        *Config
	dataBlob   string // optional data[blob] (dataExchange) from the target page
	arkBuildID string // from api.js, sent as ark-build-id header
	httpClient tls_client.HttpClient
}

// New builds a Solver from options. It returns an error if a required field is missing.
func New(opts ...Option) (*Solver, error) {
	cfg := &Config{}
	for _, o := range opts {
		o(cfg)
	}
	cfg.applyDefaults()
	if cfg.PublicKey == "" {
		return nil, errors.New("arkose: PublicKey (site key) is required — use WithPublicKey")
	}
	if cfg.Surl == "" {
		return nil, errors.New("arkose: Surl (verify host) is required — use WithSurl, e.g. https://verify.example.com")
	}
	return &Solver{cfg: cfg}, nil
}

// SetDataBlob supplies the per-request dataExchange blob (the value of a data-adx / data[blob]
// attribute the target page embeds). Optional; only some sites require it.
func (s *Solver) SetDataBlob(blob string) { s.dataBlob = blob }

// Config returns the resolved config (with defaults applied). Read-only use.
func (s *Solver) Config() Config { return *s.cfg }

func (s *Solver) newHTTPClient() (tls_client.HttpClient, error) {
	if s.httpClient != nil {
		return s.httpClient, nil
	}
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_146),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
	}
	if s.cfg.Proxy != "" {
		opts = append(opts, tls_client.WithProxyUrl(s.cfg.Proxy))
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, err
	}
	s.httpClient = c
	return c, nil
}

// SolveResult reports what the solve produced. Token is non-empty on success. Suppressed is
// true when the token carries sup=1 (trusted, no challenge required — the goal).
type SolveResult struct {
	Token      string
	Suppressed bool                     // token contains sup=1
	Timings    map[string]time.Duration // named phase durations
}

// Solve runs the pipeline: fetch api.js settings, build+encrypt the BDA, POST /fc/gt2, and
// (if the first token wasn't suppressed) retry once with the ARID warmup cookie.
func (s *Solver) Solve() (*SolveResult, error) {
	timings := map[string]time.Duration{}
	res := &SolveResult{Timings: timings}

	// Phase 1: fetch api.js → capi_version + build_id
	t0 := time.Now()
	capiVersion, vmKey, buildID, err := s.getCAPI()
	timings["settings"] = time.Since(t0)
	if err != nil {
		logPhase(1, 4, "fetch settings", timings["settings"], "FAILED")
		return nil, fmt.Errorf("settings: %w", err)
	}
	if buildID != "" {
		s.arkBuildID = buildID
	}
	logPhase(1, 4, "fetch settings", timings["settings"], fmt.Sprintf("capi=%s bid=%s", capiVersion, truncate(buildID, 8)))

	// Phase 2: build BDA + encrypt
	t0 = time.Now()
	bda := generateBDA(s.cfg)
	bdaJSON, err := marshalBDA(bda)
	if err != nil {
		timings["bda"] = time.Since(t0)
		logPhase(2, 4, "build bda + encrypt", timings["bda"], "FAILED")
		return nil, fmt.Errorf("marshal bda: %w", err)
	}

	// Prefer the RSA key from config (captured once); the api.js regex almost never finds it
	// because the key is VM-reconstructed at runtime, not a static literal.
	if vmKey == "" {
		vmKey = s.cfg.RSAPublicKey
	}
	if vmKey == "" && !isOldCapiVersion(capiVersion) {
		logError("RSAPublicKey is empty -> AES-CBC fallback: token will NOT be sup=1. " +
			"Set WithRSAPublicKey to get suppressed tokens (see README).")
	}

	encryptedBDA, err := s.encryptBDA(string(bdaJSON), vmKey, capiVersion)
	if err != nil {
		timings["bda"] = time.Since(t0)
		logPhase(2, 4, "build bda + encrypt", timings["bda"], "FAILED")
		return nil, err
	}
	// Modern Arkose uses "c" when a build id is present, "bda" otherwise.
	payloadKey := "bda"
	if s.arkBuildID != "" {
		payloadKey = "c"
	}
	timings["bda"] = time.Since(t0)
	logPhase(2, 4, "build bda + encrypt", timings["bda"], fmt.Sprintf("%d B", len(encryptedBDA)))

	// Phase 3: POST /fc/gt2/public_key/{pk}
	t0 = time.Now()
	data := s.buildPayload(encryptedBDA, payloadKey, capiVersion)
	encoded := arkoseFormEncode(data)
	headers := s.buildHeaders()
	token, respBody, status, err := s.postGT2(headers, encoded)
	timings["gt2"] = time.Since(t0)
	if err != nil {
		logPhase(3, 4, "POST /fc/gt2/public_key", timings["gt2"], "FAILED")
		return nil, fmt.Errorf("gt2: %w (status=%d, body=%s)", err, status, truncate(respBody, 300))
	}
	if token == "" {
		logPhase(3, 4, "POST /fc/gt2/public_key", timings["gt2"], "no token")
		return nil, fmt.Errorf("gt2 returned no token (status=%d, body=%s)", status, truncate(respBody, 300))
	}
	logPhase(3, 4, "POST /fc/gt2/public_key", timings["gt2"], "token acquired")

	// Phase 4: check suppression; one ARID-warmup retry if not suppressed.
	if strings.Contains(token, "sup=1") {
		logPhaseSkipped(4, 4, "suppress", "sup=1 (trusted)")
		res.Token, res.Suppressed = token, true
		return res, nil
	}

	logInfo("first token not suppressed — retrying with ARID warmup")
	bda2, _ := marshalBDA(generateBDA(s.cfg))
	enc2, err := s.encryptBDA(string(bda2), vmKey, capiVersion)
	if err == nil {
		data2 := s.buildPayload(enc2, payloadKey, capiVersion)
		token2, _, _, err2 := s.postGT2(s.buildHeaders(), arkoseFormEncode(data2))
		if err2 == nil && token2 != "" {
			if strings.Contains(token2, "sup=1") {
				logPhase(4, 4, "suppress", 0, "sup=1 via ARID warmup")
				res.Token, res.Suppressed = token2, true
				return res, nil
			}
			token = token2
		}
	}

	logPhase(4, 4, "suppress", 0, "token NOT suppressed (no sup=1)")
	res.Token = token
	return res, nil
}

// encryptBDA picks the RSA-OAEP+AES-GCM envelope (modern, gets sup=1) or the legacy AES-CBC
// path (old capi, or when no RSA key is available).
func (s *Solver) encryptBDA(bdaJSON, vmKey, capiVersion string) (string, error) {
	if vmKey != "" && !isOldCapiVersion(capiVersion) {
		enc, err := Encrypt(bdaJSON, vmKey)
		if err != nil {
			return "", fmt.Errorf("encrypt (rsa+gcm): %w", err)
		}
		return enc, nil
	}
	enc, err := EncryptAES(bdaJSON, s.cfg.UserAgent, getXArkValue())
	if err != nil {
		return "", fmt.Errorf("encrypt (aes-cbc): %w", err)
	}
	return base64.StdEncoding.EncodeToString([]byte(enc)), nil
}

// getCAPI fetches /v2/{pk}/api.js and pulls (capi_version, build_id) from it (cached 5 min).
// vm_key extraction is attempted but usually misses (see comment in Solve); the config key wins.
func (s *Solver) getCAPI() (version, vmKey, buildID string, err error) {
	pk := s.cfg.PublicKey
	capiCacheMu.Lock()
	if entry, ok := capiCache[pk]; ok && time.Since(entry.FetchedAt) < capiCacheTL {
		v, k, b := entry.Version, entry.VMKey, entry.BuildID
		capiCacheMu.Unlock()
		return v, k, b, nil
	}
	capiCacheMu.Unlock()

	host := s.cfg.Surl
	if strings.Contains(host, "://") {
		host = strings.SplitN(host, "://", 2)[1]
	}
	url := fmt.Sprintf("https://%s/v2/%s/api.js", host, pk)

	client, err := s.newHTTPClient()
	if err != nil {
		return "", "", "", err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header = http.Header{
		"user-agent":        {s.cfg.UserAgent},
		http.HeaderOrderKey: {"user-agent"},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", err
	}
	content := string(body)
	var wrapper map[string]interface{}
	if json.Unmarshal(body, &wrapper) == nil {
		if b, ok := wrapper["body"].(string); ok {
			content = b
		}
	}

	version = s.cfg.CapiVersion
	for _, re := range []string{
		`f="v2/api\.js",l="([\d\.]+)"`,
		`,l="(\d+\.\d+\.\d+)",p=`,
		`"(\d+\.\d+\.\d+)/enforcement\.`,
		`="(\d+\.\d+\.\d+)"`,
	} {
		if m := regexp.MustCompile(re).FindStringSubmatch(content); len(m) > 1 {
			version = m[1]
			break
		}
	}

	if m := regexp.MustCompile(`"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`).FindStringSubmatch(content); len(m) > 1 {
		buildID = m[1]
	}

	// best-effort vm_key scrape (usually empty for modern deployments)
	vmKeyRe := regexp.MustCompile("[\"'`]([A-Za-z0-9+/]{200,}={0,2})[\"'`]")
	for _, m := range vmKeyRe.FindAllStringSubmatch(content, -1) {
		if dec, e := base64.StdEncoding.DecodeString(m[1]); e == nil && len(dec) >= 2 && dec[0] == 0x30 && dec[1] == 0x82 {
			vmKey = m[1]
			break
		}
	}

	capiCacheMu.Lock()
	capiCache[pk] = &capiCacheEntry{Version: version, VMKey: vmKey, BuildID: buildID, FetchedAt: time.Now()}
	capiCacheMu.Unlock()
	return version, vmKey, buildID, nil
}

func isOldCapiVersion(v string) bool {
	if v == "" || !isDigit(v[0]) {
		return false
	}
	n, err := strconv.Atoi(strings.SplitN(v, ".", 2)[0])
	if err != nil {
		return false
	}
	return n < 4
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// getXArkValue returns the ms timestamp truncated to a 21600-ms bucket (x-ark-esync-value).
func getXArkValue() string {
	return strconv.FormatInt((time.Now().UnixMilli()/21600)*21600, 10)
}

// buildPayload assembles the form for /fc/gt2/public_key/{pk}.
func (s *Solver) buildPayload(encryptedBDA, payloadKey, capiVersion string) *orderedMap {
	m := newOrderedMap()
	m.Set(payloadKey, encryptedBDA)
	m.Set("public_key", s.cfg.PublicKey)

	site := s.cfg.Site
	if site == "" {
		site = s.cfg.DocumentReferrer
	}
	if site == "" {
		site = s.cfg.Surl
	}
	m.Set("site", site)
	m.Set("userbrowser", s.cfg.UserAgent)
	m.Set("capi_version", capiVersion)
	m.Set("capi_mode", ifEmpty(s.cfg.CapiMode, "lightbox"))
	m.Set("style_theme", ifEmpty(s.cfg.StyleTheme, "default"))
	m.Set("rnd", strconv.FormatFloat(rand.Float64(), 'f', -1, 64))
	if !isOldCapiVersion(capiVersion) {
		m.Set("language", ifEmpty(s.cfg.Language, "en"))
	}
	if s.dataBlob != "" {
		m.Set("data[blob]", s.dataBlob)
	}
	return m
}

func ifEmpty(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// arkoseFormEncode joins fields as urllib.parse.quote(v, safe='()') would.
func arkoseFormEncode(m *orderedMap) string {
	var b strings.Builder
	for i, k := range m.Keys() {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(quotePython(m.Get(k)))
	}
	return b.String()
}

func quotePython(s string) string {
	const safe = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.-~/()"
	set := make(map[byte]bool, len(safe))
	for i := 0; i < len(safe); i++ {
		set[safe[i]] = true
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if set[s[i]] {
			b.WriteByte(s[i])
		} else {
			fmt.Fprintf(&b, "%%%02X", s[i])
		}
	}
	return b.String()
}

// buildHeaders sets the request headers in the order Arkose expects (HTTP/2 header order).
func (s *Solver) buildHeaders() http.Header {
	referer := s.cfg.Referer
	if referer == "" {
		referer = s.cfg.Surl
	}
	if referer != "" && !strings.HasSuffix(referer, "/") {
		referer += "/"
	}
	origin := s.cfg.Origin
	if origin == "" {
		origin = strings.TrimRight(referer, "/")
	}
	hdrs := http.Header{
		"accept":             {"*/*"},
		"content-type":       {"application/x-www-form-urlencoded; charset=UTF-8"},
		"origin":             {origin},
		"referer":            {referer},
		"sec-ch-ua":          {`"Not;A=Brand";v="8", "Chromium";v="150", "Google Chrome";v="150"`},
		"sec-ch-ua-mobile":   {"?0"},
		"sec-ch-ua-platform": {`"Windows"`},
		"user-agent":         {s.cfg.UserAgent},
		"x-ark-esync-value":  {getXArkValue()},
	}
	order := []string{"accept", "content-type", "origin", "referer", "user-agent"}
	if s.arkBuildID != "" {
		hdrs.Set("ark-build-id", s.arkBuildID)
		order = append(order, "ark-build-id")
	}
	order = append(order, "sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "x-ark-esync-value")
	hdrs[http.HeaderOrderKey] = order
	return hdrs
}

// postGT2 posts the form and returns the parsed token, raw body, status, and transport error.
func (s *Solver) postGT2(headers http.Header, encoded string) (token, body string, status int, err error) {
	surl := s.cfg.Surl
	if !strings.HasPrefix(surl, "http") {
		surl = "https://" + surl
	}
	url := fmt.Sprintf("%s/fc/gt2/public_key/%s", surl, s.cfg.PublicKey)

	client, err := s.newHTTPClient()
	if err != nil {
		return "", "", 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(encoded))
	if err != nil {
		return "", "", 0, err
	}
	req.Header = headers
	resp, err := client.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", resp.StatusCode, err
	}
	body, status = string(raw), resp.StatusCode

	var parsed map[string]interface{}
	if json.Unmarshal(raw, &parsed) == nil {
		if inner, ok := parsed["body"].(string); ok {
			var inner2 map[string]interface{}
			if json.Unmarshal([]byte(inner), &inner2) == nil {
				parsed = inner2
			}
		}
		if t, ok := parsed["token"].(string); ok {
			token = t
		}
	}
	return token, body, status, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// orderedMap keeps insertion order for the form fields.
type orderedMap struct {
	keys []string
	m    map[string]string
}

func newOrderedMap() *orderedMap { return &orderedMap{m: map[string]string{}} }
func (o *orderedMap) Set(k, v string) {
	if _, ok := o.m[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.m[k] = v
}
func (o *orderedMap) Get(k string) string { return o.m[k] }
func (o *orderedMap) Keys() []string      { return o.keys }
