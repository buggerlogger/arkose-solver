# The Arkose stack VM (api.js, capi 4.4.5) — 100% mapped

`api.js` carries a small **stack VM**. The fingerprint probes and the whole
crypto envelope run as a VM program, not as JavaScript — which is why
`RSA-OAEP`, `AES-GCM`, `spki` and `importKey` appear nowhere in the bundle, not
as literals and not in any of its 2348 string-table entries. They are VM
operands, encrypted with a per-opcode repeating-key XOR.

**The program is static.** It is the literal array `ye` in `api.js`; extracting
it and comparing to a runtime snapshot of `Te` shows they are **byte-identical**.
So the entire disassembly is reproducible offline with no browser.

## Files

| File | What it is |
|---|---|
| `program.json` | the 954-instruction program, extracted from the static `ye` array |
| `disasm.txt` | full linear disassembly, every operand decrypted |
| `decompiled.js` | the program lifted back to readable pseudo-JavaScript |
| `pseudo.txt` | function-grouped intermediate view |
| `opcodes.json` | all 39 opcodes: tag, mnemonic, operand format, category, confidence |
| `vmdump.json` | a runtime snapshot (used only to prove `ye == Te`) |
| `vendors.js` | CryptoJS + a small obfuscated Arkose self-defence module |
| `vendors.forge.js` | node-forge, the pure-JS crypto fallback (not loaded on Chrome) |
| `enforcement.html` / `enforcement.js` | the challenge-iframe bundle (not the mint path) |

## Coverage

```
instructions 954 | opcodes used 27 | all mapped: True | unknown-tag instrs: 0
opcode table: 39 total (27 used, 12 defined-unused), all confirmed
encrypted-operand decode: 371/371 ok, 0 residual
static program == runtime Te: True
```

Every instruction is disassembled, every operand decrypts to printable text, and
every opcode maps to a handler read out of the deobfuscated source. The 12
defined-but-unused opcodes are the VM's full instruction set that this
particular program does not exercise (array-spread, object-assign, try/throw,
etc.); they are included in `opcodes.json` for completeness.

## Interpreter (`Ye`)

```js
while (Re < Pe.length) {
  try {
    const ins = Pe[Re];              // [handlerFn, operands, isAsync]
    ins[2] ? await ins[0](ins[1]) : ins[0](ins[1]);
    Re++;
  } catch (e) {
    if (Le.length) { Re = je[Le.pop() + "_CATCH_START"]; Me.error = e; }
    else throw e;
  }
}
```

Each `Pe[pc]` is pre-linked at load time from the raw text line `Te[pc]` by
looking the 5-hex opcode tag up in the handler map `He`/`Ke`.

| State | Meaning |
|---|---|
| `Ce` | operand stack |
| `ke` | call-frame stack: `{previousPosition, func, vars}` |
| `Re` | program counter |
| `Ie` | VM function table: `name -> {start, end}` |
| `De` | variable scope |
| `je` | label -> pc |
| `Le` | active try blocks |
| `Me` | `{name, error}` (active catch binding) |
| `xe` | staged `this`-chain for the next call |
| `Te` | raw instruction text |

## Calling convention

A call is two opcodes. `GET_MEMBER <depth>` walks a property path of `depth`
keys, pushes the resolved callee, and stages the receiver on `xe`. `CALL <argc>`
(handler `Ge`→`Fe`, always async) pops the callee off the top, pops `argc` args
below it, and does `callee.apply(this, args)` — the single host-call site at
`api.js:18:87767` that every `crypto.subtle.*` call flows through. A VM-defined
callee instead pushes a frame and jumps to `func.start`.

## Operand encryption

All string operands use repeating-key XOR:

```js
me = (s, key) => [...s].map((c, i) =>
  String.fromCharCode(c.charCodeAt(0) ^ key.charCodeAt(i % key.length))).join('');
```

Keys are per-opcode constants from the settled string table:

| Opcode | Key |
|---|---|
| `LOAD_NAME` | `922` |
| `PUSH_NAME` | `829` |
| `SET_PROP` | `757` |
| `PUSH_LIT` (typed literal via `be()`) | `5` |

Cross-check on the same table: the indices neighbouring those keys decode to
`map`, `join`, `fromCh`+`arCode`, `charCo`+`deAt` — exactly the methods `me`
itself uses.

## Full opcode table

All confirmed; handler bodies were read in the deobfuscated `Ke` map.

| Tag | Mnemonic | Operands | Effect |
|---|---|---|---|
| `a82bc` | DEFUN | name | scan to matching ENDFUN, register `Ie[name]={start,end}` |
| `b17bd` | ENDFUN | name | restore `Re` from the top call frame (return site) |
| `a0cd9` | RETURN | – | `Re = frame.func.end - 1` |
| `a5772` | CALL_NAME | name argc | call a named VM/host fn with argc popped args |
| `e73a3` | CALL | argc *(async)* | pop callee + argc args, `callee.apply(this, args)` |
| `ac4e4` | AWAIT | – | await the value on top of the stack |
| `b404d` | NEW | argc ctor | `new window[ctor](...args)` |
| `bb47b` | LOAD_NAME | name | push `De[n]` / `Ie[n]` / `window[n]` (or `Me.error` in catch) |
| `b1539` | PUSH_NAME | str [deref] | push a property-key string, or its deref if flagged |
| `c1888` / `c2b6b` | PUSH_LIT | val type | push a typed literal (`be()` decodes strings) |
| `e186f` | PUSH_RAW | val | push a raw operand |
| `a2b49` | NEW_OBJECT | – | push `{}` |
| `e8265` | NEW_ARRAY | n | pop n, push array |
| `b89af` | GET_MEMBER | depth | walk property path, stage `this` for a following call |
| `b1724` | INDEX_GET | – | `push(pop_obj[pop_key])` |
| `e776b` | SET_PROP | key | `obj[key]=val; push(obj)` |
| `e262e` | SET_PATH | – | assign through a staged `{target, props}` |
| `e9293` | JMP_LOGICAL | op label | peek top; for `\|\|` jump if truthy, for `&&` jump if falsy (value stays) |
| `ddde1` | POP | – | discard top |
| `b34bd` | PUSH_HOLE | – | grow stack by one hole |
| `ee884` | ARR_APPEND | – | append one element |
| `f70a4` | ARR_SPREAD | – | spread-concat two arrays |
| `e547b` | OBJ_ASSIGN | – | `Object.assign({}, a, b)` |
| `cea87` | STORE | name | pop, store to `Ie` (if fn-descriptor) or `De` |
| `e5b15` | DECL | name | same as STORE, used for declarations |
| `aac6f` | DELETE | name | `delete Ie[name]` |
| `ab1ed` | SET_ERRNAME | name | bind the active catch variable name |
| `c923e` | JMP | label | `Re = je[label]` |
| `d0c9a` | JMP_FALSE | label | pop; if falsy `Re = je[label]` |
| `b0280` | LABEL | label | jump target; clears `Me` on `_CATCH_END` |
| `ae498` | BINOP | op | arithmetic/comparison from the `Ee` table |
| `b0d77` | LOGIC | op | `&&` / `\|\|` from the `Se` table |
| `b5558` | UNARY | op | `! - + ~ typeof void` from the `we` table |
| `c9353` | UPDATE | op name deref | `++` / `--` from the `Oe` table |
| `d1f65` | TRY_BEGIN | label | `Le.push(label)` |
| `b5da0` | TRY_END | label | jump to `_CATCH_END`, `Le.pop()` |
| `fe342` | THROW | – | `throw pop()` |
| `bec34` | UNSUPPORTED | op | `throw new Error("Unsupported operation")` |

## Worked example — the crypto envelope, decompiled

Straight out of `decompiled.js` (`v_e38f0` is `crypto`):

```js
// AES content key
v_e38f0.getRandomValues(new Uint8Array(32));                 // 256-bit key
crypto.subtle.importKey("raw", key, { name: "AES-GCM" }, false, ["encrypt"]);

// 12-byte GCM iv, then encrypt the BDA
crypto.subtle.encrypt({ name: "AES-GCM", tagLength: 128, iv: iv }, aesKey, bdaBytes);

// RSA-wrap the AES key
key = await crypto.subtle.importKey("spki", spkiBytes,
        { name: "RSA-OAEP", hash: "SHA-256" }, false, ["encrypt"]);
crypto.subtle.encrypt({ name: "RSA-OAEP", hash: "SHA-256" }, key, aesKeyBytes);
```

This matches, call for call, what the runtime `crypto.subtle` hook recorded:
`importKey(AES-GCM,raw,32)`, `importKey(RSA-OAEP,spki,294)`,
`encrypt(RSA-OAEP,32)`, `encrypt(AES-GCM,<bda>)`. It also confirms the wire
format `iv | tag(16) | rsa(256) | ct` — `tagLength:128` is the 16-byte GCM tag.

The program opens with a `window.navigation.entries().length` probe (a
headless/automation signal) and seeds an `initPrng` +
`msCrypto.getRandomValues(new Uint8Array(48))` fallback RNG around pc 600.

## Worked example — the serialization-time transform, decompiled

This is the code that rewrites the collected fingerprint on its way out — the
step that turns `screen_pixel_depth` 24 into 72 and `network_info_rtt_type`
null into `"730442"`. It was only ever observed before; here it is as source
(from `decompiled.js`, `v_a14a1` is the `enhanced_fp` item):

```js
enhanced_fp.value.push({ key: "vsadsa",  value: v_b5eec() });   // navigation.entries().length, else null
enhanced_fp.value.push({ key: "basfas",  value: v_c57d6() });   // [0, performance.memory.jsHeapSizeLimit]
                                                                //   [1,null] if no performance, [2,null] if no .memory
enhanced_fp.value.push({ key: "lfasdgs", value: window.arkl.cbid });

v_e3df4 = ef[25];   // network_info_rtt_type
v_ee667 = ef[1];    // webgl_extensions_hash
v_c7431 = ef[18];   // webgl_hash_webgl
if (v_e3df4 && v_ee667 && v_c7431) {
  if (v_ee667.value && v_ee667.value.length > 12 && v_c7431.value && v_c7431.value.length > 12)
    ef[25] = { key: v_e3df4.key, value: v_ee667.value.slice(0, 3) + v_c7431.value.slice(0, 3) };
  else
    ef[25] = { key: v_e3df4.key, value: "abcdef" };
}

v_cef22 = ef[26];   // screen_pixel_depth
if (v_cef22 && v_cef22.value)
  ef[26] = { key: v_cef22.key, value: v_cef22.value * 3 };
```

Three consequences:

* `network_info_rtt_type` is **not** `"730" + hash[:3]`. The `"730"` is the first
  three characters of `webgl_extensions_hash` (`7300c23f…` on this Chrome build).
  `arkose/webgl_hash.go::computeRTTType` now takes both hashes.
* The `"abcdef"` fallback fires when either hash is missing or ≤12 chars — a
  cheap tell for any solver that leaves those fields empty.
* The 88→91 growth is exactly these three `push`es, in this order.

## Notes / limits of the decompiler

`decompiled.js` is a linear symbolic-stack lift; it does not thread values
across `goto` boundaries, so a handful of variables that are live across a jump
show as `= ?` (e.g. `v_e38f0 = ?`, which context makes obviously `crypto`).
Control flow is preserved faithfully as labels + gotos rather than being
re-structured into `if`/`while`. For exact semantics read `disasm.txt` (every
byte accounted for); `decompiled.js` is the readability layer.
