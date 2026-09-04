'use strict';

// The solve flow: fetch api.js, derive the RSA key, build and post the BDA.
//
// Transport uses node-tls-client's Chrome impersonation. Arkose reads the TLS
// fingerprint (JA3/JA4), so a plain fetch/axios request will not do -- the
// handshake has to look like Chrome even when the payload is perfect.

const { generateBda } = require('./bda');
const { encrypt, marshalBda } = require('./crypto');
const { extractRsaKey } = require('./vmkey');

const DEFAULT_UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 '
  + '(KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36';
const DEFAULT_CAPI_VERSION = '4.4.5';
const ESYNC_QUANTUM = 21600;

const ADX_RE = /data-adx="([^"]+)"/;
const BUILD_ID_RE = /"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"/;
const ENFORCEMENT_RE = /(\d+\.\d+\.\d+)\/enforcement\.([0-9a-f]{32})\.html/;
const CAPI_RES = [
  /f="v2\/api\.js",l="([\d.]+)"/,
  /,l="(\d+\.\d+\.\d+)",p=/,
  /"(\d+\.\d+\.\d+)\/enforcement\./,
];

const SAFE = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.-~/()';

function quotePython(s) {
  const bytes = Buffer.from(s, 'utf8');
  let out = '';
  for (const b of bytes) {
    const c = String.fromCharCode(b);
    out += SAFE.includes(c) ? c : '%' + b.toString(16).toUpperCase().padStart(2, '0');
  }
  return out;
}

const formEncode = (pairs) => pairs.map(([k, v]) => `${k}=${quotePython(v)}`).join('&');

const xArkEsyncValue = () =>
  String(Math.floor(Math.floor(Date.now() / 1000) / ESYNC_QUANTUM) * ESYNC_QUANTUM);

class SolveResult {
  constructor(token, suppressed, timings) {
    this.token = token;
    this.suppressed = suppressed;
    this.timings = timings;
  }
}

class Solver {
  constructor(opts = {}) {
    if (!opts.surl) throw new Error('surl (verify host) is required');
    if (!opts.publicKey) throw new Error('publicKey (site key) is required');

    this.surl = opts.surl.replace(/\/+$/, '');
    this.publicKey = opts.publicKey;
    this.site = opts.site || '';
    this.rsaPublicKey = opts.rsaPublicKey || '';
    this.proxy = opts.proxy || null;
    this.userAgent = opts.userAgent || DEFAULT_UA;
    this.language = opts.language || 'en-US';
    this.capiMode = opts.capiMode || 'lightbox';
    this.styleTheme = opts.styleTheme || 'default';
    this.title = opts.title || '';
    this.referer = opts.referer || this.site || this.surl;
    this.origin = opts.origin || this.site || this.surl;
    this.sitedataLocationHref = opts.sitedataLocationHref || this.site;
    this.documentReferrer = opts.documentReferrer || null;
    this.dataExchangeUrl = opts.dataExchangeUrl || null;
    this.dataExchangeRegex = opts.dataExchangeRegex
      ? new RegExp(opts.dataExchangeRegex) : ADX_RE;

    this.capiVersion = DEFAULT_CAPI_VERSION;
    this.buildId = '';
    this.enforcementHash = '';
    this.dataBlob = '';
    this._session = null;
  }

  async _client() {
    if (this._session) return this._session;
    let mod;
    try {
      mod = require('node-tls-client');
    } catch (e) {
      throw new Error(
        'node-tls-client is required for the Chrome TLS fingerprint: npm i node-tls-client');
    }
    // v2 keeps the native client in a module-level singleton that has to be
    // brought up before any Session is used, and torn down at the end.
    if (mod.initTLS && !Solver._tlsReady) {
      await mod.initTLS();
      Solver._tlsReady = true;
    }
    this._mod = mod;
    this._session = new mod.Session({
      clientIdentifier: mod.ClientIdentifier
        ? (mod.ClientIdentifier.chrome_131 || 'chrome_131')
        : 'chrome_131',
      timeout: 30000,
      ...(this.proxy ? { proxy: this.proxy } : {}),
    });
    return this._session;
  }

  _host() {
    return this.surl.includes('://') ? this.surl.split('://')[1] : this.surl;
  }

  async fetchCapi() {
    const session = await this._client();
    const url = `https://${this._host()}/v2/${this.publicKey}/api.js`;
    const res = await session.get(url, { headers: { 'user-agent': this.userAgent } });
    let content = await res.text();
    try {
      const wrapper = JSON.parse(content);
      if (wrapper && typeof wrapper.body === 'string') content = wrapper.body;
    } catch (e) { /* not wrapped */ }

    for (const re of CAPI_RES) {
      const m = re.exec(content);
      if (m) { this.capiVersion = m[1]; break; }
    }
    const b = BUILD_ID_RE.exec(content);
    if (b) this.buildId = b[1];
    const e = ENFORCEMENT_RE.exec(content);
    if (e) this.enforcementHash = e[2];
    if (!this.rsaPublicKey) this.rsaPublicKey = extractRsaKey(content);
    return this.rsaPublicKey;
  }

  async fetchDataExchange() {
    if (!this.dataExchangeUrl) return '';
    const session = await this._client();
    const res = await session.get(this.dataExchangeUrl, {
      headers: {
        'user-agent': this.userAgent,
        accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8',
        'accept-language': 'en-US,en;q=0.9',
        'sec-ch-ua-mobile': '?0',
        'sec-ch-ua-platform': '"Windows"',
        'sec-fetch-site': 'none',
        'sec-fetch-mode': 'navigate',
        'sec-fetch-user': '?1',
        'sec-fetch-dest': 'document',
        'upgrade-insecure-requests': '1',
      },
    });
    const m = this.dataExchangeRegex.exec(await res.text());
    this.dataBlob = m ? m[1] : '';
    return this.dataBlob;
  }

  _bdaConfig() {
    const href = this.sitedataLocationHref || this.site;
    return {
      surl: this.surl,
      language: this.language,
      title: this.title,
      windowLocationHref: href,
      sitedataLocationHref: href,
      documentReferrer: this.documentReferrer,
      enforcementHash: this.enforcementHash,
    };
  }

  _payload(encrypted) {
    const pairs = [
      [this.buildId ? 'c' : 'bda', encrypted],
      ['public_key', this.publicKey],
      ['site', this.site || this.surl],
      ['userbrowser', this.userAgent],
      ['capi_version', this.capiVersion],
      ['capi_mode', this.capiMode],
      ['style_theme', this.styleTheme],
      ['rnd', String(Math.random())],
      ['language', this.language],
    ];
    if (this.dataBlob) pairs.push(['data[blob]', this.dataBlob]);
    return formEncode(pairs);
  }

  _headers() {
    const referer = this.referer.endsWith('/') ? this.referer : this.referer + '/';
    const h = {
      accept: '*/*',
      'content-type': 'application/x-www-form-urlencoded; charset=UTF-8',
      origin: this.origin.replace(/\/+$/, ''),
      referer,
      'user-agent': this.userAgent,
      'sec-ch-ua-mobile': '?0',
      'sec-ch-ua-platform': '"Windows"',
      'x-ark-esync-value': xArkEsyncValue(),
    };
    if (this.buildId) h['ark-build-id'] = this.buildId;
    return h;
  }

  async solve() {
    const timings = {};
    let t = Date.now();
    await this.fetchCapi();
    timings.settings = Date.now() - t;
    if (!this.rsaPublicKey) {
      throw new Error('no RSA public key: none supplied and none extractable from api.js');
    }

    if (this.dataExchangeUrl) {
      t = Date.now();
      try { await this.fetchDataExchange(); } catch (e) { this.dataBlob = ''; }
      timings.dataexchange = Date.now() - t;
    }

    t = Date.now();
    const bda = generateBda(this._bdaConfig(), this.buildId);
    const encrypted = encrypt(marshalBda(bda), this.rsaPublicKey);
    timings.bda = Date.now() - t;

    t = Date.now();
    const base = this.surl.startsWith('http') ? this.surl : 'https://' + this.surl;
    const session = await this._client();
    const res = await session.post(`${base}/fc/gt2/public_key/${this.publicKey}`, {
      headers: this._headers(),
      body: this._payload(encrypted),
    });
    timings.gt2 = Date.now() - t;

    const text = await res.text();
    let token = '';
    try {
      let parsed = JSON.parse(text);
      if (typeof parsed.body === 'string') parsed = JSON.parse(parsed.body);
      token = parsed.token || '';
    } catch (e) { /* fall through */ }
    if (!token) {
      throw new Error(`gt2 returned no token (status=${res.status}, body=${text.slice(0, 300)})`);
    }
    return new SolveResult(token, token.includes('sup=1'), timings);
  }

  async close() {
    if (this._session && this._session.close) {
      try { await this._session.close(); } catch (e) { /* already down */ }
    }
    this._session = null;
  }

  static async shutdown() {
    if (Solver._tlsReady) {
      try { await require('node-tls-client').destroyTLS(); } catch (e) { /* ignore */ }
      Solver._tlsReady = false;
    }
  }
}

Solver._tlsReady = false;

const solve = async (opts) => {
  const s = new Solver(opts);
  try {
    return await s.solve();
  } finally {
    await s.close();
  }
};

module.exports = { Solver, SolveResult, solve, quotePython, formEncode, xArkEsyncValue };
