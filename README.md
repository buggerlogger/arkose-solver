# arkose-solver

A pure-Go **Arkose Labs / FunCaptcha** token solver. It synthesises a real-browser
**BDA** fingerprint, encrypts it with the site's **RSA-OAEP + AES-GCM** envelope, and
posts it to `/fc/gt2/public_key/{pk}` to obtain a **suppressed (`sup=1`) token** — no
headless browser, no Node sandbox, no key to capture. One HTTP round-trip.

```
GET /token  ->  {"token":"98718c6b...|r=eu-west-1|...|sup=1|...","suppressed":true,"elapsed_ms":1482}
```

You supply only the **site key** and the **verify host**. Everything else — including
the site's RSA public key — is derived from the served `api.js` on every solve.

---

## Install

```bash
go get github.com/buggerlogger/arkose-solver/arkose
```

Requires Go 1.24+.

## Library

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

`Solve()` returns `*SolveResult{ Token, Suppressed, Timings }`. `Suppressed == true`
means the token carries `sup=1` and is accepted with no challenge.

### Options

| Option | Purpose |
|---|---|
| `WithSurl(host)` | Arkose verify host, e.g. `https://verify.example.com` (**required**) |
| `WithPublicKey(uuid)` | Arkose site key (**required**) |
| `WithSite(url)` | origin the widget runs on (derives Origin/Referer) |
| `WithProxy(p)` | egress proxy — `http(s)://`, `socks5://`, `host:port:user:pass`, `user:pass@host:port` |
| `WithRSAPublicKey(spki)` | pin the RSA key instead of auto-extracting it (rarely needed) |
| `WithUserAgent(ua)` | override the UA (also reported in the BDA) |
| `WithLanguage(l)` | language (default `en-US`) |
| `WithCapiMode(m)` | `lightbox` (default) or `inline` |
| `WithReferer(r)` / `WithOrigin(o)` | override headers explicitly |
| `WithSitedataLocationHref(u)` | page URL reported in the BDA |
| `WithTitle(t)` | `document.title` of the embedding page (BDA `jsbd` `DT`) |
| `WithDataExchangeURL(u)` | auto-fetch a fresh `data[blob]` per solve from this page URL |
| `WithDataExchangeRegex(re)` | override the blob extractor (default `data-adx="([^"]+)"`) |

Inject your own transport with `Solver.SetHTTPClient(...)` to share a session, or
supply a blob per solve with `Solver.SetDataBlob(...)`.

## CLI server

```bash
go build -ldflags="-s -w" -o arkose-server ./cmd/arkose-server

arkose-server \
  -surl https://verify.example.com \
  -pk   00000000-0000-0000-0000-000000000000 \
  -site https://www.example.com

curl http://127.0.0.1:8100/token
```

Per-request proxy: `curl -H "Proxy: host:port:user:pass" http://127.0.0.1:8100/token`.
With a blob: `http://127.0.0.1:8100/token?blob=<data-adx value>`.

| Flag | Meaning | Default |
|---|---|---|
| `-addr` | listen address | `127.0.0.1:8100` |
| `-surl` | Arkose verify host (**required**) | — |
| `-pk` | site key / `public_key` UUID (**required**) | — |
| `-site` | origin the widget runs on | — |
| `-rsa` | pin the RSA key (optional; auto-extracted when unset) | auto |
| `-proxy` | default egress proxy | none |
| `-ua` | override User-Agent | Chrome 152 Win64 |
| `-lang` | language | `en-US` |
| `-mode` | `lightbox` or `inline` | `lightbox` |

Find your `public_key` in the page's Arkose script URL `…/v2/<PUBLIC_KEY>/api.js`
(or a `data-pkey` / `data-apk` attribute); `surl` is the host serving that script.

---

## The RSA key is derived, not captured

The envelope wraps the AES content key with the site's RSA public key. That key is
**not** a literal in `api.js` — it appears nowhere as base64, hex, a byte array, or
string-table fragments, and `settings` returns `{}`. Most solvers therefore capture
it once from a real browser and hard-code it, which breaks silently when it rotates.

It is actually stored **inside the VM program that `api.js` embeds**, as a
XOR-encrypted string operand assembled at runtime from string-table fragments.
`ExtractRSAKey` recovers it statically from the served bundle in ~40 ms:

1. parse every string table in the bundle;
2. find the long concatenation chains that build the VM program's instructions;
3. the decoder offset and the load-time table rotation are both index shifts, so
   they collapse into one unknown — brute-force it (≤ ~1000 tries per table);
4. for each candidate, recover the repeating-XOR key from the known SPKI prefix
   `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA` and accept only if the result
   parses as a real PKIX RSA public key.

Nothing about the opcode tags, offsets, table names or the rotation checksum is
hard-coded, so this survives a build rotation. `WithRSAPublicKey` still lets you
pin a key if you want to.

---

## How it works

1. **`GET /v2/{pk}/api.js`** — read `capi_version`, the enforcement build hash, and
   extract the RSA public key.
2. **(optional) fetch the dataExchange blob** on the same client.
3. **Build the BDA** — `api_type, f, n, wh, enhanced_fp (91), fe, ife_hash, jsbd, c`,
   with `fb` inserted when a font-probe flag is set. A fresh device is sampled per
   solve and every cross-checkable field is derived from it.
4. **Encrypt** — random AES-256 key + 12-byte IV, AES-GCM the BDA, RSA-OAEP(SHA-256)
   wrap the key, concat `b64(iv)+b64(tag)+b64(rsa)+b64(ct)`.
5. **`POST /fc/gt2/public_key/{pk}`** — correct field and header order.

### Derived fields

Nothing the enforcement can recompute is pinned. All recipes were lifted from the
live bundle and reproduce real mints byte-for-byte. `K` is MurmurHash3 x64-128
emitted as `hex(h1)||hex(h2)`.

| Field | Rule |
|---|---|
| `f` | `K(fe values joined by ";")`, seed 0 |
| `ife_hash` | `K(fe joined by ", ", 38)` |
| `webgl_hash_webgl` | `K(Object.entries(t).join(","))` over the 18 `webgl_*` fields plus a trailing empty `webgl_hash_webgl` |
| `network_info_rtt_type` | `webgl_extensions_hash[:3] + webgl_hash_webgl[:3]` (both >12 chars, else `"abcdef"`) |
| `screen_pixel_depth` | `3 × colorDepth` |
| `basfas` | `[0, performance.memory.jsHeapSizeLimit]` |
| `6a62b2a558` | the enforcement build hash, scraped from `api.js` |
| `x-ark-esync-value` | `floor(now/21600)*21600` |

`screen_pixel_depth` and `network_info_rtt_type` are rewritten by the VM at
serialization time — the collected object holds `24` and `null`, the transmitted
payload holds `72` and e.g. `"730442"`.

### Device generation

`arkose/devices.json` is an embedded weighted population — 234 GPUs, 105 screens,
13 core counts, 4 memory sizes, 62 connection profiles — sampled per solve, with
every derived field recomputed for that device. Measured: 100 distinct `f` per 100
solves. Window geometry is sampled independently of the screen; always emitting a
maximised window is a fleet tell.

---

## Reverse-engineering notes

[`specimen/`](specimen/) contains the deobfuscated bundle and the full analysis of
the stack VM that `api.js` embeds — the fingerprint probes, the RSA key and the
entire crypto envelope live there as bytecode, which is why `RSA-OAEP`, `AES-GCM`,
`spki` and `importKey` appear nowhere in the source.

* [`specimen/api_clean.js`](specimen/api_clean.js) — fully deobfuscated `api.js`
* [`specimen/vm/disasm.txt`](specimen/vm/disasm.txt) — 954-instruction disassembly, every operand decrypted
* [`specimen/vm/decompiled.js`](specimen/vm/decompiled.js) — the program lifted to readable JavaScript
* [`specimen/vm/opcodes.json`](specimen/vm/opcodes.json) — all 39 opcodes
* [`specimen/tools/`](specimen/tools/) — the deobfuscation + disassembly pipeline

---

## Notes & legality

For research and authorized testing only. You are responsible for complying with
the terms of any site you interact with and with applicable law. No affiliation
with Arkose Labs.

## License

MIT — see [LICENSE](LICENSE).
