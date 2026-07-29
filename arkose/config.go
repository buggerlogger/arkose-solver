package arkose

// Config holds everything a solve needs. Construct it with New(...) + functional options;
// don't build it by hand. The three site-specific values you MUST provide are:
//
//   - PublicKey    : the Arkose site key (a UUID, e.g. "1A2B3C4D-...") from the target site.
//   - RSAPublicKey : the RSA-2048 SPKI (base64) the site's api.js uses to wrap the AES key.
//                    This is REQUIRED for a suppressed (sup=1) token. See the README for how
//                    to capture it once from a real browser mint.
//   - Surl         : the Arkose verify host for the site, e.g. "https://verify.example.com".
//
// Everything else has a working default (real Chrome-150 Windows fingerprint), and the
// fingerprint fields (WebGL/canvas/audio/etc.) rotate per request from built-in real pools.
type Config struct {
	// --- site-specific (required) ---
	PublicKey    string // Arkose site key (UUID)
	RSAPublicKey string // RSA SPKI base64 — REQUIRED for sup=1
	Surl         string // Arkose verify host, e.g. https://verify.example.com

	// --- site context (recommended) ---
	Site                 string // the origin the widget runs on, e.g. https://www.example.com
	Origin               string // Origin header (defaults to Site)
	Referer              string // Referer header (defaults to Site + "/")
	SitedataLocationHref string // page URL the widget was loaded on
	WindowLocationHref   string // window.location.href reported in the BDA
	DocumentReferrer     string // document.referrer reported in the BDA
	SecFetchSite         string // sec-fetch-site header (default "same-origin")

	// --- widget config (optional) ---
	CapiVersion string // fallback capi version if api.js parse fails (default "4.4.3")
	CapiMode    string // "lightbox" (default) or "inline"
	StyleTheme  string // "default"
	Language    string // "en-US"
	Title       string // document.title reported in the BDA

	// --- transport / identity (optional) ---
	UserAgent string // full UA string (default: Chrome 150 Win64)
	Proxy     string // egress proxy, any form (http(s)://, socks5://, host:port:user:pass, user:pass@host:port)

	// --- frame tree (rarely needed) ---
	WindowAncestorOrigins []string
	WindowTreeIndex       []int
	WindowTreeStructure   string
	Jsbd                  string
}

// Option mutates a Config. Pass any number to New().
type Option func(*Config)

// WithPublicKey sets the Arkose site key (UUID). Required.
func WithPublicKey(pk string) Option { return func(c *Config) { c.PublicKey = pk } }

// WithRSAPublicKey sets the RSA SPKI (base64) used to wrap the AES key. Required for sup=1.
func WithRSAPublicKey(spki string) Option { return func(c *Config) { c.RSAPublicKey = spki } }

// WithSurl sets the Arkose verify host, e.g. "https://verify.example.com".
func WithSurl(surl string) Option { return func(c *Config) { c.Surl = surl } }

// WithSite sets the origin the widget runs on and derives Origin/Referer/location hrefs
// from it when those are otherwise empty.
func WithSite(site string) Option { return func(c *Config) { c.Site = site } }

// WithProxy sets the egress proxy (any accepted form).
func WithProxy(p string) Option { return func(c *Config) { c.Proxy = p } }

// WithUserAgent overrides the User-Agent (and the UA reported in the BDA).
func WithUserAgent(ua string) Option { return func(c *Config) { c.UserAgent = ua } }

// WithLanguage sets the language field ("en-US" default).
func WithLanguage(l string) Option { return func(c *Config) { c.Language = l } }

// WithCapiMode sets "lightbox" (default) or "inline".
func WithCapiMode(m string) Option { return func(c *Config) { c.CapiMode = m } }

// WithReferer / WithOrigin let you override the headers explicitly.
func WithReferer(r string) Option { return func(c *Config) { c.Referer = r } }
func WithOrigin(o string) Option  { return func(c *Config) { c.Origin = o } }

// WithDataExchangeBlob is set per-solve via Solver.SetDataBlob; kept here for completeness
// if you want to bake a static blob into the config.
func WithSitedataLocationHref(u string) Option {
	return func(c *Config) { c.SitedataLocationHref = u; c.WindowLocationHref = u }
}

// defaultUA is a real Chrome 150 / Windows 10 x64 User-Agent.
const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36"

// applyDefaults fills any unset field with a sane default and derives dependent fields.
func (c *Config) applyDefaults() {
	if c.UserAgent == "" {
		c.UserAgent = defaultUA
	}
	if c.CapiVersion == "" {
		c.CapiVersion = "4.4.3"
	}
	if c.CapiMode == "" {
		c.CapiMode = "lightbox"
	}
	if c.StyleTheme == "" {
		c.StyleTheme = "default"
	}
	if c.Language == "" {
		c.Language = "en-US"
	}
	if c.SecFetchSite == "" {
		c.SecFetchSite = "same-origin"
	}
	if c.Jsbd == "" {
		c.Jsbd = `{"HL":9,"NCE":true,"DT":"","NWD":"false","DMTO":1,"DOTO":1}`
	}
	if c.WindowAncestorOrigins == nil {
		c.WindowAncestorOrigins = []string{}
	}
	if c.WindowTreeIndex == nil {
		c.WindowTreeIndex = []int{}
	}
	if c.WindowTreeStructure == "" {
		c.WindowTreeStructure = "[]"
	}
	// derive site-context defaults
	if c.Origin == "" && c.Site != "" {
		c.Origin = c.Site
	}
	if c.Referer == "" {
		if c.Site != "" {
			c.Referer = c.Site
		} else {
			c.Referer = c.Surl
		}
	}
	if c.SitedataLocationHref == "" {
		if c.WindowLocationHref != "" {
			c.SitedataLocationHref = c.WindowLocationHref
		} else if c.Site != "" {
			c.SitedataLocationHref = c.Site
		}
	}
	if c.WindowLocationHref == "" {
		c.WindowLocationHref = c.SitedataLocationHref
	}
}
