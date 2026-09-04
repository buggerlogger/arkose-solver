# Cross-language ports

## testvectors.json

Conformance vectors every implementation must reproduce exactly. Regenerate with:

```bash
go run ./ports/gen
```

The generator asserts each value against the captured ground truth **before**
writing the file, so a mismatch fails loudly instead of baking in a wrong
vector. It covers:

* `K` (MurmurHash3 x64-128) across several seeds
* `f` and `ife_hash` over the ground-truth `fe` array
* `webgl_hash_webgl` and `network_info_rtt_type` for two GPUs, plus the exact
  hash input string
* `screen_pixel_depth` factor and the `jsHeapSizeLimit` table

## python/ and node/

Both ports are complete, published, and reach `sup=1` against a live deployment.

```bash
pip install arkose-solver
npm install arkose-solver node-tls-client
```

| | Python | Node |
|---|---|---|
| package | [PyPI](https://pypi.org/project/arkose-solver/) | [npm](https://www.npmjs.com/package/arkose-solver) |
| requires | 3.10+ | 18+ |
| conformance | 162 checks pass | 163 checks pass |
| live solve | `sup=1` | `sup=1` |
| key extraction | ~0.12 s | ~0.03 s |
| transport | `curl_cffi` | `node-tls-client` |

Run the suites from a checkout with `python tests/conformance.py` and `npm test`.
All three languages are released at the same version.

### The blocker worth knowing

Arkose reads the TLS fingerprint (JA3/JA4), so neither transport can be pure.
`curl_cffi` is C-backed and `node-tls-client` is Go-backed; the Go
implementation remains the reference.

### What each port covers

| Module | Python | Node |
|---|---|---|
| MurmurHash3 x64-128, `f`, `ife_hash` | `hashes.py` | `hashes.js` |
| webgl fields, hash, `network_info_rtt_type` | `webgl.py` | `webgl.js` |
| the reversed hashes (`f58835f`, speech, codecs, …) | `derived.py` | `derived.js` |
| RSA key recovery from `api.js` | `vmkey.py` | `vmkey.js` |
| device sampling, `jsHeapSizeLimit` | `devices.py` | `devices.js` |
| 91-field enhanced_fp assembly | `bda.py` | `bda.js` |
| RSA-OAEP + AES-GCM envelope | `crypto.py` | `crypto.js` |
| transport | `solver.py` | `solver.js` |

Per-port usage docs are in `python/README.md` and `node/README.md`.
