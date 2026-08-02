package arkose

type Config struct {
	PublicKey    string
	RSAPublicKey string
	Surl         string

	Site                 string
	Origin               string
	Referer              string
	SitedataLocationHref string
	WindowLocationHref   string
	DocumentReferrer     string
	SecFetchSite         string

	DataExchangeURL   string
	DataExchangeRegex string

	CapiVersion string
	CapiMode    string
	StyleTheme  string
	Language    string
	Title       string

	UserAgent string
	Proxy     string

	WindowAncestorOrigins []string
	WindowTreeIndex       []int
	WindowTreeStructure   string
	Jsbd                  string
}

type Option func(*Config)

func WithPublicKey(pk string) Option { return func(c *Config) { c.PublicKey = pk } }

func WithRSAPublicKey(spki string) Option { return func(c *Config) { c.RSAPublicKey = spki } }

func WithSurl(surl string) Option { return func(c *Config) { c.Surl = surl } }

func WithSite(site string) Option { return func(c *Config) { c.Site = site } }

func WithProxy(p string) Option { return func(c *Config) { c.Proxy = p } }

func WithUserAgent(ua string) Option { return func(c *Config) { c.UserAgent = ua } }

func WithLanguage(l string) Option { return func(c *Config) { c.Language = l } }

func WithTitle(t string) Option { return func(c *Config) { c.Title = t } }

func WithDataExchangeURL(u string) Option { return func(c *Config) { c.DataExchangeURL = u } }

func WithDataExchangeRegex(re string) Option { return func(c *Config) { c.DataExchangeRegex = re } }

func WithCapiMode(m string) Option { return func(c *Config) { c.CapiMode = m } }

func WithReferer(r string) Option { return func(c *Config) { c.Referer = r } }
func WithOrigin(o string) Option  { return func(c *Config) { c.Origin = o } }

func WithSitedataLocationHref(u string) Option {
	return func(c *Config) { c.SitedataLocationHref = u; c.WindowLocationHref = u }
}

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36"

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

	if c.WindowAncestorOrigins == nil {
		c.WindowAncestorOrigins = []string{}
	}
	if c.WindowTreeIndex == nil {
		c.WindowTreeIndex = []int{}
	}
	if c.WindowTreeStructure == "" {
		c.WindowTreeStructure = "[]"
	}

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
