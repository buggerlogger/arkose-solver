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

type Solver struct {
	cfg            *Config
	dataBlob       string
	arkBuildID     string
	lastArid       string
	httpClient     tls_client.HttpClient
	externalClient bool
}

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

func (s *Solver) SetDataBlob(blob string) { s.dataBlob = blob }

func (s *Solver) SetHTTPClient(c tls_client.HttpClient) { s.httpClient = c; s.externalClient = true }

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

type SolveResult struct {
	Token      string
	Suppressed bool
	Timings    map[string]time.Duration
}

func (s *Solver) Solve() (*SolveResult, error) {
	timings := map[string]time.Duration{}
	res := &SolveResult{Timings: timings}

	if !s.externalClient {
		s.httpClient = nil
		s.lastArid = ""
	}

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

	if s.cfg.DataExchangeURL != "" {
		tb := time.Now()
		if blob, ferr := s.fetchDataExchange(); ferr != nil {
			logInfo("dataExchange fetch failed (minting without blob): %s", ferr.Error())
		} else {
			s.dataBlob = blob
			logPhase(1, 4, "dataExchange blob", time.Since(tb), fmt.Sprintf("%d chars", len(blob)))
		}
		timings["dataexchange"] = time.Since(tb)
	}

	t0 = time.Now()
	bda := generateBDA(s.cfg, s.arkBuildID)
	bdaJSON, err := marshalBDA(bda)
	if err != nil {
		timings["bda"] = time.Since(t0)
		logPhase(2, 4, "build bda + encrypt", timings["bda"], "FAILED")
		return nil, fmt.Errorf("marshal bda: %w", err)
	}

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

	payloadKey := "bda"
	if s.arkBuildID != "" {
		payloadKey = "c"
	}
	timings["bda"] = time.Since(t0)
	logPhase(2, 4, "build bda + encrypt", timings["bda"], fmt.Sprintf("%d B", len(encryptedBDA)))

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

	if strings.Contains(token, "sup=1") {
		logPhaseSkipped(4, 4, "suppress", "sup=1 (trusted)")
		res.Token, res.Suppressed = token, true
		return res, nil
	}

	logInfo("first token not suppressed — retrying with ARID warmup")
	bda2, _ := marshalBDA(generateBDA(s.cfg, s.arkBuildID))
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

var defaultAdxRe = regexp.MustCompile(`data-adx="([^"]+)"`)

func (s *Solver) fetchDataExchange() (string, error) {
	client, err := s.newHTTPClient()
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodGet, s.cfg.DataExchangeURL, nil)
	if err != nil {
		return "", err
	}
	req.Header = http.Header{
		"user-agent":                {s.cfg.UserAgent},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"accept-language":           {"en-US,en;q=0.9"},
		"accept-encoding":           {"gzip, deflate, br"},
		"sec-ch-ua":                 {`"Not;A=Brand";v="8", "Chromium";v="150", "Google Chrome";v="150"`},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"upgrade-insecure-requests": {"1"},
		http.HeaderOrderKey: {"user-agent", "accept", "accept-language", "accept-encoding",
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "sec-fetch-site",
			"sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest", "upgrade-insecure-requests"},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	re := defaultAdxRe
	if s.cfg.DataExchangeRegex != "" {
		re, err = regexp.Compile(s.cfg.DataExchangeRegex)
		if err != nil {
			return "", fmt.Errorf("dataexchange regex: %w", err)
		}
	}
	m := re.FindStringSubmatch(string(body))
	if len(m) < 2 {
		return "", fmt.Errorf("dataexchange blob not found at %s (status=%d, len=%d)", s.cfg.DataExchangeURL, resp.StatusCode, len(body))
	}
	return m[1], nil
}

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

func getXArkValue() string {
	return strconv.FormatInt((time.Now().Unix()/21600)*21600, 10)
}

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

	if s.lastArid != "" {
		hdrs.Set("x-ark-arid", `{"ls":"`+s.lastArid+`","idb":"`+s.lastArid+`"}`)
		order = append(order, "x-ark-arid")
	}
	hdrs[http.HeaderOrderKey] = order
	return hdrs
}

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

	if xa := resp.Header.Get("x-ark-arid"); xa != "" {
		var v struct{ Idb, Ls string }
		if json.Unmarshal([]byte(xa), &v) == nil {
			if v.Idb != "" {
				s.lastArid = v.Idb
			} else if v.Ls != "" {
				s.lastArid = v.Ls
			}
		}
	}

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
