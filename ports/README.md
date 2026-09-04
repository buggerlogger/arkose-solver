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

## python/ — incomplete, not published

Status: the derivation layer is written and verified against
`testvectors.json`; the transport layer is not.

| Module | State |
|---|---|
| `hashes.py` | done — matches all `K`/`f`/`ife_hash` vectors |
| `webgl.py` | done — reproduces both webgl hashes and the hash input byte-for-byte |
| `vmkey.py` | done — extracts the RSA key from a live `api.js` in ~0.12 s, identical to Go |
| `devices.py` | done — samples the shared `devices.json` |
| `bda.py` | done — mirrors the Go 91-field assembly |
| `crypto.py` | done — RSA-OAEP + AES-GCM envelope |
| `solver.py` | **missing** — TLS-impersonating transport, header order, retry |
| `pyproject.toml` | **missing** |

It is therefore not installable and not on PyPI. Run the modules directly if
you want to use the derivations.

### The blocker worth knowing

Arkose reads the TLS fingerprint (JA3/JA4), so the transport cannot be pure
Python. The realistic options — `curl_cffi`, `tls-client` — are C- or Go-backed
under the hood, which is why the Go implementation remains the reference.

## node/

Not started.
