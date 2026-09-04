# Arkose specimen + recovered payload rules — capi 4.4.5, 2026-09-04

Captured live from `https://www.istockphoto.com/sign-in?returnurl=%2F`
(site key `E14A2FF7-4C77-4B85-B296-5B328F433135`, host `verify.istockphoto.com`).
The mints used here returned `sup=1`, so this is a known-good reference.

| File | What it is |
|---|---|
| `api_fresh.js` | `v2/<pk>/api.js` as served, prettified |
| `api_final.js` | same, string tables resolved + concat-folded — **read this one** |
| `vm_eval_raw.js` | the bundle as the debugger sees it (same bytes as api.js) |
| `vm_final.js` | deobfuscated form of the above |
| `strtab2.json` | the 9 settled (post-rotation) string tables + decoder offsets |
| `bda445.json` | **plaintext BDA** lifted out of a live mint (91 enhanced_fp fields) |
| `bda445_spoofed_gpu.json` | second plaintext BDA with the GPU spoofed to an RTX 4070 |
| `live_fp_445.json` | `lr.fp` (the raw collected object) from the same paused mint |
| `webgl_hash_input_rtx4070.txt` | the exact string the browser hashed for `webgl_hash_webgl` |
| `mappers.json` | live sources of `Ht.K`, `Vt`, `Nr`, and the build constants |
| `gt2_fresh.json` | a full `POST /fc/gt2/public_key/` incl. body and `sup=1` response |
| `api_clean.js` | **fully deobfuscated api.js** — scope-correct, concats folded, parses clean |
| `vm/` | the stack VM: program dump, full disassembly, opcode table, sibling bundles |
| `tools/` | the deobfuscation + disassembly toolchain (see `tools/README.md`) |

## api.js embeds a stack VM

The fingerprint probes and the entire crypto envelope run as a **VM program**,
not as JavaScript. That is why `RSA-OAEP`, `AES-GCM`, `spki` and `importKey`
appear nowhere in the bundle — not as literals and not in any of its 2348
string-table entries. They are VM operands, encrypted with a per-opcode
repeating-key XOR.

954 instructions, 27 opcodes, 56 labels, fully disassembled in
[`vm/disasm.txt`](vm/disasm.txt). See [`vm/README.md`](vm/README.md) for the
interpreter, the state registers, the operand cipher and worked examples.

## Deobfuscation pipeline

The bundle uses obfuscator.io-style string tables: 9 decoders, each
`DEC(n) = TABLE[n - OFFSET]`, where every TABLE is **rotated at load time** by a
`push(shift())` loop that spins until a checksum matches. Indices are wrong
until you replay that rotation.

```bash
python tools/deob.py    api_fresh.js strtab.json                  # raw tables + offsets
node   tools/settle2.js api_fresh.js strtab.json strtab2.json     # replay the rotations
node   tools/deob2.js   api_fresh.js strtab2.json api_deob.js     # AST const-prop + inline
node   tools/fold.js    api_deob.js  api_final.js                 # fold "a"+"b", obj["x"]->obj.x
```

`deob2.js` does scope-aware constant propagation, which is required: the code
never calls `DEC(1234)` directly, it does `(R = 756), (I = yn), screen[I(R)]`.
Naive substitution yields plausible-looking garbage (`screen["o(tm) "+…]`).

## The payload rules (all reproduced in `arkose/`, all regression-tested)

`Ht.K` is MurmurHash3 x64-128 emitted as `hex(h1)||hex(h2)` — the full source is
in `mappers.json` and is textbook `x64hash128`.

`Vt(obj, asFE)` (also in `mappers.json`) builds both list shapes:
`Vt(fp, true)` → `["KEY:" + String(value), …]` (that's `fe`), and `Vt(ef)` →
`[{key, value}, …]` (that's `enhanced_fp`).

| Field | Rule | Where |
|---|---|---|
| `f` | `K(fe values joined by ";")`, seed 0 | `arkose/hashes.go` |
| `ife_hash` | `K(fe joined by ", ", 38)` | `arkose/hashes.go` |
| `webgl_hash_webgl` | `K(Object.entries(t).join(","))` where `t` is the 18 `webgl_*` fields **plus a trailing empty `webgl_hash_webgl`** | `arkose/webgl_hash.go` |
| `network_info_rtt_type` | `webgl_extensions_hash[:3] + webgl_hash_webgl[:3]` (both must be >12 chars, else `"abcdef"`) — decompiled from the VM, `vm/decompiled.js`; the "730" is just `7300c23f…[:3]` | `arkose/webgl_hash.go` |
| `screen_pixel_depth` | `3 × fe "D:"` (colorDepth) | `arkose/identity.go` |
| `6a62b2a558` | the enforcement build hash, i.e. the `<ver>/enforcement.<hash>.html` id | scraped in `arkose/solver.go` |
| `x-ark-esync-value` | `floor(now/21600)*21600` (`m.Jy = 21600`) | `arkose/solver.go` |

### The serialization-time transform (this is the important one)

The **collected** object and the **transmitted** payload are not the same. From
one paused mint, comparing `lr.fp.ef` against the BDA that same mint serialized:
86 of 88 fields identical, and exactly two rewritten.

| Field | collected (`lr.fp.ef`) | transmitted (BDA) |
|---|---|---|
| `network_info_rtt_type` | `null` | `"730442"` |
| `screen_pixel_depth` | `24` | `72` |

Reading only the collector is how you get this wrong: it stores
`h3(screen.pixelDepth)`, and `h3` is just `t => typeof t === "number" ? t : null`.
The ×3 and the rtt marker are applied later, on the way out. Three further
fields (`vsadsa`, `basfas`, `lfasdgs`) are appended by `Nr()` from `un(40)`,
taking 88 → 91.

`network_info_rtt_type` was confirmed by spoofing `WEBGL_debug_renderer_info`
and re-minting:

| GPU | `webgl_hash_webgl` | `network_info_rtt_type` |
|---|---|---|
| Intel UHD 4626 (real) | `44202604…` | `730442` |
| NVIDIA RTX 4070 (spoofed) | `a3fb76ae…` | `730a3f` |

## Crypto path

WebCrypto, in the top frame's main world. One mint emits exactly:

```
importKey(AES-GCM, raw, 32)      # content key
importKey(RSA-OAEP, spki, 294)   # the site's RSA public key
encrypt(RSA-OAEP, 32)            # wraps the content key
encrypt(AES-GCM, <bda bytes>)    # the payload
```

Envelope: `c = b64(iv:12) || b64(gcmTag:16) || b64(rsaBlock:256) || b64(ct)`,
verified across 4.4.3 and 4.4.5 captures. `arkose/encryption.go::Encrypt`
matches it.

**The RSA public key is not in the bundle.** It does not appear as base64, hex,
a decimal byte array, or as string-table fragments; the bundle contains no
`spki` / `RSA-OAEP` / `AES-GCM` literals either. It is stable per site key
across page loads *and* it has rotated: the key this build imports differs from
the one pinned in an older sibling project. So it must be re-acquired, not
hardcoded — `cmd/capture-key` hooks `importKey` at document-start, which is
proven to catch it.

## Instrumentation notes

- **A probe zero means nothing until you confirm a mint happened.** `initSession()`
  frequently no-ops, and `performance.getEntriesByType` misses the gt2 XHR — use
  the network log. Several "the VM avoids WebCrypto" conclusions here were
  drawn from runs that simply never minted.
- The mint runs in the main frame. The debugger reports the bundle with an empty
  `url` (it is the same bytes as `api.js`, md5 `8527d0e4…`), which makes it look
  like an eval'd script; `list_scripts` will not show it.
- Best access: `break_on_xhr` on `/fc/gt2/public_key/`, then evaluate in a frame
  around index 15 where `lr` (enforcement state), `Vt`, `Ht`, `m` and `pr` are
  all in scope. `lr.fp` holds the whole collected fingerprint.
- `Ht` is a live module singleton, so `Ht.K` can be wrapped at a pause to log
  every hash input — but only the *next* page load recomputes the fingerprint.
- The webgl hash input was recovered by hooking `String.prototype.charCodeAt`
  and keeping strings ≥ 24 chars: `K` consumes its input one char code at a time.
- `settings` (`v2/<pk>/settings`) is 2 bytes (`{}`) — no key material.

## Device generation

`arkose/devices.json` (234 GPUs, 105 screens, 13 core counts, 4 memory sizes,
62 connection profiles, all weighted) is embedded and sampled per solve. The
derived fields above are recomputed for each sampled device, so a fresh machine
never contradicts itself. Measured: 100 distinct `f` and ~72 distinct webgl
hashes per 100 solves.

Window geometry is sampled independently of the screen — a live mint ran at
`window_outer 1386x889` on a `1920x1032` available area, with
`inner = outer - 16 / outer - 95`. Always emitting a maximised window is a
fleet tell.

## Live oracle characterisation (2026-09-04, one IP, one site key, back-to-back)

A real solve from this repo returns `sup=1` in ~1.6s, and did so on 4/4 runs
with a different sampled device each time. Single-variable tamper arms against
the same endpoint:

**Still `sup=1` (not checked server-side):**

| Arm | Verdict |
|---|---|
| `f` pinned to the public-solver constant | sup=1 |
| `f` = literally `"garbage"` | sup=1 |
| `ife_hash` pinned | sup=1 |
| `webgl_hash_webgl` randomised | sup=1 |
| `network_info_rtt_type` left as the old literal | sup=1 |
| `screen_pixel_depth` un-multiplied (24) | sup=1 |
| `6a62b2a558` randomised | sup=1 |
| `api_type` = `"nonsense"` | sup=1 |
| User-Agent / `userbrowser` mismatch | sup=1 |
| no dataExchange blob | sup=1 |
| `fe` truncated to 5 entries | sup=1 |

**Loses `sup=1` (checked):**

| Arm | Verdict |
|---|---|
| `fe` emptied | not suppressed |
| `fe` values garbled, count kept | not suppressed |
| `enhanced_fp` emptied | not suppressed |
| `enhanced_fp` values → null, keys kept | not suppressed |
| `enhanced_fp` string values → `"x"` | not suppressed |
| `enhanced_fp` truncated to 45 fields | not suppressed |
| garbage (undecryptable) `c` envelope | not suppressed |

So suppression **is** a usable oracle, and what it grades is the *plausibility
of the fe and enhanced_fp values* plus a decryptable envelope — not the derived
hashes. `f`, `ife_hash`, `webgl_hash_webgl`, `network_info_rtt_type` and
`screen_pixel_depth` are **not currently recomputed server-side**.

Caveats before treating that as licence to pin them again:
- one IP, one session, one site key, arms run back-to-back; suppression is also
  reputation-weighted, and a colder IP may grade more strictly.
- these are exactly the checks a vendor turns on without shipping a new bundle;
  the values are cheap to compute correctly and free to keep correct.
- `fe` truncated to 5 passing while `fe` garbled fails says the grader looks at
  per-field value plausibility, not at the field set.

Reproduce with `LIVE_SOLVE=1` and `LIVE_{SURL,PK,RSA,SITE,HREF,TITLE}` set:
`arkose/zzlive_test.go` (one solve) and `arkose/zzcontrol_test.go`
(tamper / wall / graded arms).
