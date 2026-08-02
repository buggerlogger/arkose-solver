# arkose-solver

A pure-Go **Arkose Labs / FunCaptcha** token solver. It builds a real-browser **BDA** fingerprint,
encrypts it with the site's **RSA-OAEP + AES-GCM** envelope, and posts it to
`/fc/gt2/public_key/{pk}` to obtain a **suppressed (`sup=1`) token** — no headless browser, no
Node sandbox, no PoW step. One HTTP round-trip, ~0.5s per token.

Works with any Arkose site: you provide the **site key**, the **RSA public key**, and the
**verify host**, and the solver does the rest.

```
GET /token  ->  {"token":"98718c6b...|r=eu-west-1|...|sup=1|...","suppressed":true,"elapsed_ms":482}
```

---

## Features

- **`sup=1` suppressed tokens** — the trusted, no-challenge token, produced directly.
- **dataExchange auto-fetch** — the single biggest reliability lever: point `WithDataExchangeURL`
  at the embedding page and every solve pulls a fresh `data[blob]` on the same session. Measured
  in practice: **with blob → reliably `sup=1`; without → mostly unsuppressed.** (See below.)
- **Ground-truth BDA fingerprint** — every field validated byte-for-byte against a live Chrome-150
  mint (the universal hashes are pinned to their real values, not fabricated).
- **A new, unique device every solve** — each solve synthesizes a fresh, internally-coherent
  Windows-Chrome machine (GPU from a ~900-device pool + matching RAM/cores/screen, random per-GPU
  WebGL hash), so no two tokens share a fingerprint and there's no fleet-constant tell.
- **Fresh session per solve** — each `Solve()` uses its own cookie jar so a spent, low-trust
  session can't poison the next token (inject your own client with `SetHTTPClient` to share one).
- **Correct crypto envelope** — RSA-2048-OAEP-wrapped AES-256-GCM (`iv|tag|rsa|ct`), matching the
  browser byte-for-byte. Legacy AES-CBC path kept for old `capi` versions.
- **Configurable per site** — site key, RSA key, verify host, origin, UA, language, proxy.
- **TLS impersonation** — Chrome fingerprint via `bogdanfinn/tls-client`.
- **Per-request proxy override** — send a `Proxy:` header to the server, or `WithProxy(...)`.
- **Library + CLI server** — import the package, or run the tiny HTTP server.

---

## Changelog: old (CBC port) → v1 (this repo)

This project began as a faithful Go port of a Python Arkose solver. That port **never produced
`sup=1`** — it was chasing the wrong two things. v1 fixes both, verified against a live mint.

| Area | Old (CBC port) | **v1 (this repo)** |
|---|---|---|
| **`c` envelope** | AES-256-**CBC**, emitted as `{"ct","s","iv"}` JSON | **RSA-OAEP + AES-256-GCM** hybrid `b64(iv12)+b64(tag16)+b64(rsa256)+b64(ct)` |
| **Server verdict** | token returned, **never `sup=1`** | **`sup=1` suppressed** token |
| **RSA public key** | assumed a "red herring", regex-scraped from api.js (**always empty** — key is VM-computed) | **required & configurable** (`WithRSAPublicKey`); captured once per site |
| **`f` (fp hash)** | `md5("")` — hash of empty | real hash from a genuine feature enumeration |
| **`fe` (features)** | `[]` (empty) | populated real Chrome `fe` (DNT/L/S/AS/JSF/P/… with correct comma-separated, sorted formats) |
| **Top-level BDA** | `…jsbd, vsadsa, basfas, lfasdgs` (extras misplaced at top level) | `…jsbd, c` (matches real; extras live only inside `enhanced_fp`) |
| **`enhanced_fp`** | 90 keys, missing `z87b89t5`; several wrong value formats | **91 keys, exact structure**; value formats corrected |
| **Value tells** | `L:en-us`, `ODB:true`, empty `document__referrer`, telemetry blobs | `L:en-US`, `ODB:false`, real referrer, minimal telemetry (`Ow==`) |
| **PoW** | stub → hard error on non-`sup=1` | irrelevant — `sup=1` means no challenge is demanded |
| **Shape** | single `package main` binary, hardcoded to one site + a checker | **reusable library** + CLI server, site-agnostic |

**The key insight:** `sup=1` requires **both** the RSA+GCM envelope **and** a real fingerprint.
CBC gets a token but never `sup=1`; a good fingerprint under CBC still never `sup=1`. Together they do.

---

## Install

```bash
go get github.com/buggerlogger/arkose-solver/arkose
```

Or build the server:

```bash
git clone https://github.com/buggerlogger/arkose-solver
cd arkose-solver
go build -ldflags="-s -w" -o arkose-server ./cmd/arkose-server
```

Requires Go 1.24+.

---

## CLI Usage (HTTP server)

```bash
arkose-server \
  -surl https://verify.example.com \
  -pk   00000000-0000-0000-0000-000000000000 \
  -rsa  MIIBIjAN...your-captured-key... \
  -site https://www.example.com \
  -proxy http://user:pass@host:port
```

Then request tokens:

```bash
curl http://127.0.0.1:8100/token
# {"token":"...|sup=1|...","suppressed":true,"elapsed_ms":482}
```

- Per-request proxy: `curl -H "Proxy: host:port:user:pass" http://127.0.0.1:8100/token`
- With a dataExchange blob: `http://127.0.0.1:8100/token?blob=<data-adx value>`

### CLI flags

| Flag | Meaning | Default |
|---|---|---|
| `-addr` | listen address | `127.0.0.1:8100` |
| `-surl` | Arkose verify host (**required**) | — |
| `-pk` | Arkose site key / `public_key` UUID (**required**) | — |
| `-rsa` | RSA public key, SPKI base64 (**required for `sup=1`**) | — |
| `-site` | origin the widget runs on | — |
| `-proxy` | default egress proxy (any form) | none |
| `-ua` | override User-Agent | Chrome 150 Win64 |
| `-lang` | language | `en-US` |
| `-mode` | `lightbox` or `inline` | `lightbox` |

---

## Library Usage (Go)

```go
package main

import (
	"fmt"
	"log"

	"github.com/buggerlogger/arkose-solver/arkose"
)

func main() {
	s, err := arkose.New(
		arkose.WithSurl("https://verify.example.com"),
		arkose.WithPublicKey("00000000-0000-0000-0000-000000000000"),
		arkose.WithRSAPublicKey("MIIBIjAN...your-captured-key..."),
		arkose.WithSite("https://www.example.com"),
		arkose.WithProxy("http://user:pass@host:port"), // optional
	)
	if err != nil {
		log.Fatal(err)
	}

	res, err := s.Solve()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Token, "suppressed:", res.Suppressed)
}
```

### Options

| Option | Purpose |
|---|---|
| `WithSurl(host)` | Arkose verify host, e.g. `https://verify.example.com` (**required**) |
| `WithPublicKey(uuid)` | Arkose site key (**required**) |
| `WithRSAPublicKey(spki)` | RSA SPKI base64 (**required for `sup=1`**) |
| `WithSite(url)` | origin the widget runs on (derives Origin/Referer) |
| `WithProxy(p)` | egress proxy — `http(s)://`, `socks5://`, `host:port:user:pass`, `user:pass@host:port` |
| `WithUserAgent(ua)` | override the UA (also reported in the BDA) |
| `WithLanguage(l)` | language (default `en-US`) |
| `WithCapiMode(m)` | `lightbox` (default) or `inline` |
| `WithReferer(r)` / `WithOrigin(o)` | override headers explicitly |
| `WithSitedataLocationHref(u)` | set the page URL reported in the BDA |
| `WithTitle(t)` | document.title of the embedding page (BDA `jsbd` `DT`) |
| `WithDataExchangeURL(u)` | **auto-fetch a fresh `data[blob]` per solve** from this page URL — the biggest `sup=1` lever (see below) |
| `WithDataExchangeRegex(re)` | override the blob extractor (default `data-adx="([^"]+)"`) |

### The dataExchange blob — the biggest `sup=1` lever

Many Arkose deployments embed a per-page-load **dataExchange blob** — a short-lived, server-minted
token — in the host page, and forward it as the `data[blob]` mint field. Supplying a **fresh,
same-session** blob is the single largest driver of suppression (back-to-back, same IP: with blob →
reliably `sup=1`; without → mostly unsuppressed).

`WithDataExchangeURL` makes this autonomous: before each solve the library GETs that page **on the
same TLS client** (which also warms the session cookies), extracts the blob, and forwards it — no
browser needed.

```go
s, _ := arkose.New(
    arkose.WithSurl("https://verify.example.com"),
    arkose.WithPublicKey("00000000-0000-0000-0000-000000000000"),
    arkose.WithRSAPublicKey(rsaKey),
    arkose.WithSite("https://www.example.com"),
    arkose.WithTitle("Sign in - Example"),
    arkose.WithDataExchangeURL("https://www.example.com/sign-in"),
)
res, _ := s.Solve() // fetches a fresh blob, then mints
```

Notes:
- The blob is **single-use**: it's re-fetched every `Solve()`.
- The page GET uses a full browser header set on purpose — a minimal request makes some origins
  serve a **blob-less** page variant. If the default `data-adx` extractor doesn't fit your site,
  pass `WithDataExchangeRegex`.
- Or supply a blob yourself per solve with `Solver.SetDataBlob(...)` (e.g. from a `data-adx` /
  `data-exchange` attribute you already have).
- **IP reputation is a separate axis:** even a perfect blob won't yield `sup=1` from an IP that
  Arkose has throttled after a burst. Spread solves out and/or rotate egress proxies for volume.

`Solve()` returns `*SolveResult{ Token, Suppressed, Timings }`. `Suppressed == true` means the
token carries `sup=1` and is accepted with no challenge.

---

## Getting your site's keys

Three values are site-specific. Two are trivial; one you capture once.

1. **`public_key` (site key)** — a UUID. It's in the target page's Arkose script URL:
   `…/v2/<PUBLIC_KEY>/api.js`, or in a `data-pkey` / `data-apk` attribute.
2. **`surl` (verify host)** — the host serving that `api.js`, e.g. `https://verify.example.com`.
3. **RSA public key** — **required for `sup=1`**. It is **not** a static string in `api.js`
   (the client reconstructs it at runtime inside an obfuscated VM), so you capture it once from a
   real browser mint. It's stable per site key for weeks.

### Option A — automated (recommended): the `capture-key` tool

A one-command tool that launches Chrome, injects the hook into **every frame before page scripts**,
**forces** the Arkose mint, catches the RSA key, writes it to a file, and closes Chrome:

```bash
go build -o capture-key ./cmd/capture-key
capture-key -url https://www.example.com/sign-in -out rsa_key.txt
```

Output:

```
Chrome open on https://www.example.com/sign-in
Waiting for the Arkose mint (auto-triggered)…

[OK] captured RSA key (392 chars) -> rsa_key.txt

MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB...
```

Then use it: `-rsa "$(cat rsa_key.txt)"` or `WithRSAPublicKey(...)`.

Flags: `-url` (page that uses Arkose, required), `-out` (default `rsa_key.txt`),
`-timeout` (default 90s), `-headless` (default off — leave off if a site is picky),
`-keep` (leave Chrome open for debugging). Chrome must be installed.

> **Why a plain console paste often fails:** Arkose runs its crypto inside a **cross-origin iframe**,
> and on suppressed sites the mint **never fires on its own** — it must be triggered.
> The tool handles both: it hooks all frames at document-start and calls the enforcement's
> `initSession()` to force the mint. That's why the manual snippet printed nothing.

### Option B — manual (one-off)

Switch the DevTools **console context** to the Arkose iframe (`verify.<site>.com` / `*.arkoselabs.com`
in the context dropdown), paste the hook below, then trigger the captcha. Or run it as a Tampermonkey
userscript with `@run-at document-start` + `@all-frames true`:

```js
const _i = crypto.subtle.importKey.bind(crypto.subtle);
crypto.subtle.importKey = function (fmt, kd) {
  try {
    if (fmt === 'spki' && kd && kd.byteLength)
      console.log('RSA_SPKI =', btoa(String.fromCharCode(...new Uint8Array(kd))));
  } catch (e) {}
  return _i.apply(this, arguments);
};
console.log('[hooked — now trigger the captcha]');
```

Copy the printed `RSA_SPKI = MIIBIjAN…` into `WithRSAPublicKey(...)` / `-rsa`.

> Without the RSA key the solver falls back to AES-CBC: the server still returns a token, but it
> will **not** be `sup=1`. The solver logs a warning in that case.

---

## How it works

1. **`GET /v2/{pk}/api.js`** — read the current `capi_version` and `ark-build-id`.
1a. **(optional) fetch the dataExchange blob** — if `WithDataExchangeURL` is set, GET that page on
   the same client and extract a fresh `data[blob]` (also warms session cookies). See above.
2. **Build the BDA** — a real Chrome fingerprint: `api_type, f, n, wh, enhanced_fp (91 fields),
   fe, ife_hash, jsbd, c`. WebGL/GPU, canvas, audio, codecs, and math fields rotate per request.
3. **Encrypt** — random AES-256 key + 12-byte IV; AES-GCM the BDA; RSA-OAEP(SHA-256) wrap the AES
   key with your site's RSA public key; concat `b64(iv)+b64(tag)+b64(rsa)+b64(ct)` into `c=`.
4. **`POST /fc/gt2/public_key/{pk}`** — form fields `c, public_key, site, userbrowser,
   capi_version, capi_mode, style_theme, rnd, language[, data[blob]]`, correct header order.
5. **Return the token** — if `sup=1`, done. Otherwise one ARID-warmup retry.

---

## Architecture

```
arkose-solver/
├── arkose/                 # the library (import this)
│   ├── config.go           # Config + functional options + defaults
│   ├── solver.go           # New(), Solve(), transport, payload/headers
│   ├── bda.go              # BDA + enhanced_fp + fe/f (ground-truth Chrome)
│   ├── encryption.go       # RSA-OAEP+AES-GCM (Encrypt) and AES-CBC (EncryptAES)
│   ├── rotators.go         # per-request rotating pools (chrome version, audio, langs, speech…)
│   ├── profiles.go         # coherent device profiles (real GPU+screen+RAM as one unit)
│   ├── gpu.go / gpu_data.go# WebGL vendor/renderer/hash pool (930 GPUs)
│   ├── logger.go           # phase logging
│   └── pow.go              # (unused — sup=1 means no PoW)
├── cmd/arkose-server/      # tiny HTTP server (GET /token, /health)
│   └── ...
├── cmd/capture-key/        # auto RSA-key capture: opens Chrome, hooks, saves to file, closes
└── examples/basic/         # minimal library example
```

---

## Notes & legality

For research and authorized testing only. You are responsible for complying with the terms of any
site you interact with and with applicable law. No affiliation with Arkose Labs.

## License

MIT — see [LICENSE](LICENSE).
