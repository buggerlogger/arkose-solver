# Deobfuscation + VM disassembly toolchain

Everything here operates on the frozen specimens in `../`. Nothing needs the
network except the capture step.

## 1. Deobfuscate a bundle

```bash
node deob3.js  ../api_fresh.js  api_deob.js  report.json   # resolve string tables
node clean.js  api_deob.js      api_clean.js               # fold concats, obj["x"] -> obj.x
```

`deob3.js` output on `api_fresh.js`: **7241 resolved, 0 out-of-range, 0
missing-table, 26 genuinely dynamic**.

### Why deob3 and not deob2

`deob2.js` (kept only for comparison — **do not use**) resolved decoder calls by
identifier name globally. That is wrong for this bundle:

* four different functions are named `S`, four are named `l`
* four different string tables are named `d`, two are named `w`

Resolving by name mixes them and emits confident nonsense — e.g. `screen.colorDepth`
came out as `screen["o(tm) " + "MYRIAD"]`. `deob3.js` keys everything by
declaration **node** and resolves identifiers through a real scope chain, so a
local binding always beats a same-named function elsewhere.

Two more traps it handles:

* **Tables are rotated at load time.** Each table has a `push(shift())` loop that
  spins until a checksum matches. Indices are meaningless until that is replayed,
  so each table is executed against *its own* rotation IIFE in a `vm` sandbox.
* **Decoder shape varies per bundle.** `api.js` uses
  `var n = TBL(); ... return n[t -= 439]`, `vendors.js` uses
  `var e = h(); ... return e[t -= 424]`. The matcher binds the variable and
  parameter names rather than hard-coding them.

`clean.js` is AST-driven on purpose. `fold.js` (also kept only for comparison,
**do not use**) did the same job with regexes and corrupted the file twice:
it rewrote single-element array literals `["foo"]` into `.foo`, and for a
parenthesised object it anchored on `object.range[1]`, which sits *before* the
closing paren, so `("a"+"b")["concat"]` lost its `)`. `clean.js` anchors on the
`[` token and skips numeric-literal objects (`5["toString"]` must not become
`5.toString`).

## 2. Diagnostics

```bash
node misses.js   ../api_fresh.js ../strtab2.json   # why any call stayed unresolved
node dumptabs.js ../api_fresh.js                   # every table string + fragment search
node scan.js     api_clean.js                      # big literals / numeric arrays / packed data
python deob.py   ../api_fresh.js strtab.json       # raw (pre-rotation) tables
node settle2.js  ../api_fresh.js strtab.json strtab2.json   # replay rotations standalone
```

`dumptabs.js` is what proved the crypto strings are *not* in `api.js`: none of
its 2348 table strings contains `RSA`, `OAEP`, `GCM`, `spki` or `importKey`.
They live in the VM program instead.

## 3. Capture the VM program

`api.js` embeds a stack VM. Its program is pre-linked into `Pe`, an array of
`[handler, operand, isAsync]`, so there is no raw byte blob to carve out — the
program has to be read at runtime.

Do **not** use a pausing breakpoint: the interpreter loop runs thousands of
times and the debugger session wedges. Use a **logpoint** — a conditional
breakpoint whose condition snapshots state and returns `false`, so it never
pauses.

Set it on this text in `api.js` (unique):

```
!(r=Pe[Re])[2]
```

with this condition:

```js
(function(){try{if(window.__VMDUMP)return false;var ids=new Map(),fns=[];
var ins=Pe.map(function(x,i){var f=x[0];if(!ids.has(f)){ids.set(f,fns.length);fns.push(String(f).slice(0,700));}
var op;try{op=JSON.stringify(x[1],function(k,v){return typeof v==='function'?'[fn]':v;});}catch(e){op='[unserializable]';}
return [i,ids.get(f),op,x[2]?1:0];});
window.__VMDUMP={count:Pe.length,ins:ins,fns:fns,je:je,Te:Te,Ie:Object.keys(Ie)};}catch(e){window.__VMDUMP={err:String(e)};}return false;})()
```

Then trigger a mint and read `window.__VMDUMP` **in the same evaluation** — the
page reloads readily and a separate read comes back empty.

## 4. Disassemble

```bash
python vmdisasm.py ../vm/vmdump.json ../vm/disasm.txt ../vm/opcodes.json
```

Produces `PC | MNEMONIC | decoded operands ; raw instruction` plus a machine
readable opcode table with per-opcode confidence.

## 5. Static VM extraction + disassembly + decompilation (no browser)

The VM program is the literal array `ye` in the deobfuscated bundle, and it is
byte-identical to the runtime `Te`. So once `api_clean.js` exists, the whole VM
analysis is offline:

```bash
python extract_program.py ../api_clean.js ../vm/program.json
python vmdisasm2.py  ../vm/program.json ../vm/disasm.txt ../vm/opcodes.json ../vm/pseudo.txt
python vmdecompile.py ../vm/program.json ../vm/decompiled.js
```

* `extract_program.py` — pulls `ye` out (handles JS `\u`/`\x`/`\r` escapes).
* `vmdisasm2.py` — **the definitive disassembler**. Complete 39-opcode table
  read from the VM's `Ke` handler map, every operand decrypted. Supersedes
  `vmdisasm.py` (which needed a runtime dump and mislabelled `GET_MEMBER`/`CALL`).
  Prints a coverage line: 954 instrs, 27 opcodes used, all mapped, 0 unknown.
* `vmdecompile.py` — lifts the stack ops back to readable pseudo-JavaScript by
  symbolic stack execution. Gotos are kept; the crypto envelope comes out as
  literal `crypto.subtle.importKey(...)` / `encrypt(...)` calls.

The old `vmdisasm.py` is retained only for comparison. Use `vmdisasm2.py`.

### The opcode encoding

Each `ye` line is `"<5-hex tag> <operand> <operand> ..."`. The tag selects a
handler; operands are either bare tokens (labels, ids, argc, numbers) or
single-quoted encrypted strings. Decryption keys are per-opcode
(`922`/`829`/`757`/`5`) and were lifted from the settled string table. All 371
encrypted operands in this program decode to printable text with zero residual.
