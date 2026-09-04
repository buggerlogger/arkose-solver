"""The solve flow: fetch api.js, derive the RSA key, build and post the BDA.

Transport uses curl_cffi's Chrome impersonation. Arkose reads the TLS
fingerprint (JA3/JA4), so a plain requests/httpx session will not do -- the
handshake has to look like Chrome even when the payload is perfect.
"""

import base64
import json
import random
import re
import time
import urllib.parse

from curl_cffi import requests as curl_requests

from .bda import generate_bda
from .crypto import encrypt, marshal_bda
from .vmkey import extract_rsa_key

DEFAULT_UA = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36"
)
DEFAULT_CAPI_VERSION = "4.4.5"
IMPERSONATE = "chrome"
ESYNC_QUANTUM = 21600

_ADX_RE = re.compile(r'data-adx="([^"]+)"')
_BUILD_ID_RE = re.compile(
    r'"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"')
_ENFORCEMENT_RE = re.compile(r"(\d+\.\d+\.\d+)/enforcement\.([0-9a-f]{32})\.html")
_CAPI_RES = [
    r'f="v2/api\.js",l="([\d.]+)"',
    r',l="(\d+\.\d+\.\d+)",p=',
    r'"(\d+\.\d+\.\d+)/enforcement\.',
]


class SolveResult:
    def __init__(self, token, suppressed, timings):
        self.token = token
        self.suppressed = suppressed
        self.timings = timings

    def __repr__(self):
        return "SolveResult(suppressed=%r, token=%r)" % (self.suppressed, self.token[:48] + "...")


def x_ark_esync_value():
    return str(int(time.time()) // ESYNC_QUANTUM * ESYNC_QUANTUM)


def quote_python(s):
    safe = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.-~/()"
    out = []
    for ch in s.encode("utf-8"):
        c = chr(ch)
        out.append(c if c in safe else "%%%02X" % ch)
    return "".join(out)


def form_encode(pairs):
    """Ordered, Python-quote-compatible form encoding (order is checked)."""
    return "&".join("%s=%s" % (k, quote_python(v)) for k, v in pairs)


class Solver:
    def __init__(self, surl, public_key, site=None, rsa_public_key=None,
                 proxy=None, user_agent=None, language="en-US", capi_mode="lightbox",
                 style_theme="default", title="", referer=None, origin=None,
                 sitedata_location_href=None, document_referrer=None,
                 data_exchange_url=None, data_exchange_regex=None):
        if not surl:
            raise ValueError("surl (verify host) is required")
        if not public_key:
            raise ValueError("public_key (site key) is required")

        self.surl = surl.rstrip("/")
        self.public_key = public_key
        self.site = site or ""
        self.rsa_public_key = rsa_public_key or ""
        self.proxy = proxy
        self.user_agent = user_agent or DEFAULT_UA
        self.language = language
        self.capi_mode = capi_mode
        self.style_theme = style_theme
        self.title = title
        self.referer = referer or (self.site or self.surl)
        self.origin = origin or (self.site or self.surl)
        self.sitedata_location_href = sitedata_location_href or self.site
        self.document_referrer = document_referrer
        self.data_exchange_url = data_exchange_url
        self.data_exchange_regex = re.compile(data_exchange_regex) if data_exchange_regex else _ADX_RE

        self.capi_version = DEFAULT_CAPI_VERSION
        self.build_id = ""
        self.enforcement_hash = ""
        self.data_blob = ""
        self._session = None

    def _client(self):
        if self._session is None:
            kwargs = {"impersonate": IMPERSONATE, "timeout": 30}
            if self.proxy:
                kwargs["proxies"] = {"http": self.proxy, "https": self.proxy}
            self._session = curl_requests.Session(**kwargs)
        return self._session

    def _host(self):
        h = self.surl
        if "://" in h:
            h = h.split("://", 1)[1]
        return h

    def fetch_capi(self):
        """GET api.js: capi version, build id, enforcement hash and the RSA key."""
        url = "https://%s/v2/%s/api.js" % (self._host(), self.public_key)
        r = self._client().get(url, headers={"user-agent": self.user_agent})
        content = r.text
        try:
            wrapper = json.loads(content)
            if isinstance(wrapper, dict) and isinstance(wrapper.get("body"), str):
                content = wrapper["body"]
        except Exception:
            pass

        for pat in _CAPI_RES:
            m = re.search(pat, content)
            if m:
                self.capi_version = m.group(1)
                break
        m = _BUILD_ID_RE.search(content)
        if m:
            self.build_id = m.group(1)
        m = _ENFORCEMENT_RE.search(content)
        if m:
            self.enforcement_hash = m.group(2)
        if not self.rsa_public_key:
            self.rsa_public_key = extract_rsa_key(content)
        return self.rsa_public_key

    def fetch_data_exchange(self):
        if not self.data_exchange_url:
            return ""
        headers = {
            "user-agent": self.user_agent,
            "accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
            "accept-language": "en-US,en;q=0.9",
            "sec-ch-ua-mobile": "?0",
            "sec-ch-ua-platform": '"Windows"',
            "sec-fetch-site": "none",
            "sec-fetch-mode": "navigate",
            "sec-fetch-user": "?1",
            "sec-fetch-dest": "document",
            "upgrade-insecure-requests": "1",
        }
        r = self._client().get(self.data_exchange_url, headers=headers)
        m = self.data_exchange_regex.search(r.text)
        self.data_blob = m.group(1) if m else ""
        return self.data_blob

    def _bda_config(self):
        href = self.sitedata_location_href or self.site
        return {
            "surl": self.surl,
            "language": self.language,
            "title": self.title,
            "window_location_href": href,
            "sitedata_location_href": href,
            "document_referrer": self.document_referrer,
            "enforcement_hash": self.enforcement_hash,
        }

    def _payload(self, encrypted):
        pairs = [
            ("c" if self.build_id else "bda", encrypted),
            ("public_key", self.public_key),
            ("site", self.site or self.surl),
            ("userbrowser", self.user_agent),
            ("capi_version", self.capi_version),
            ("capi_mode", self.capi_mode),
            ("style_theme", self.style_theme),
            ("rnd", repr(random.random())),
            ("language", self.language),
        ]
        if self.data_blob:
            pairs.append(("data[blob]", self.data_blob))
        return form_encode(pairs)

    def _headers(self):
        referer = self.referer if self.referer.endswith("/") else self.referer + "/"
        h = {
            "accept": "*/*",
            "content-type": "application/x-www-form-urlencoded; charset=UTF-8",
            "origin": self.origin.rstrip("/"),
            "referer": referer,
            "user-agent": self.user_agent,
            "sec-ch-ua-mobile": "?0",
            "sec-ch-ua-platform": '"Windows"',
            "x-ark-esync-value": x_ark_esync_value(),
        }
        if self.build_id:
            h["ark-build-id"] = self.build_id
        return h

    def solve(self):
        timings = {}

        t0 = time.time()
        self.fetch_capi()
        timings["settings"] = time.time() - t0
        if not self.rsa_public_key:
            raise RuntimeError(
                "no RSA public key: none supplied and none extractable from api.js")

        if self.data_exchange_url:
            t0 = time.time()
            try:
                self.fetch_data_exchange()
            except Exception:
                self.data_blob = ""
            timings["dataexchange"] = time.time() - t0

        t0 = time.time()
        bda = generate_bda(self._bda_config(), self.build_id)
        encrypted = encrypt(marshal_bda(bda), self.rsa_public_key)
        timings["bda"] = time.time() - t0

        t0 = time.time()
        url = "%s/fc/gt2/public_key/%s" % (
            self.surl if self.surl.startswith("http") else "https://" + self.surl,
            self.public_key)
        r = self._client().post(url, headers=self._headers(), data=self._payload(encrypted))
        timings["gt2"] = time.time() - t0

        token = ""
        try:
            parsed = r.json()
            if isinstance(parsed.get("body"), str):
                parsed = json.loads(parsed["body"])
            token = parsed.get("token", "")
        except Exception:
            pass
        if not token:
            raise RuntimeError("gt2 returned no token (status=%s, body=%s)"
                               % (r.status_code, r.text[:300]))

        return SolveResult(token, "sup=1" in token, timings)


def solve(surl, public_key, **kwargs):
    """One-shot convenience wrapper around Solver."""
    return Solver(surl, public_key, **kwargs).solve()
