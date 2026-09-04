'use strict';

// Recover the site's RSA public key from the served api.js.
//
// The key is not a literal in the bundle. It lives inside the VM program api.js
// embeds, as a repeating-XOR encrypted operand that is itself assembled at
// runtime from string-table fragments.
//
// The decoder offset and the load-time table rotation are both index shifts, so
// they collapse into a single unknown that can be brute-forced. Each candidate
// is accepted only if it decrypts to a real RSA public key, which makes the
// search self-validating: no opcode tag, table name, offset or rotation
// checksum is hard-coded, so it survives a build rotation.

const crypto = require('crypto');

const SPKI_PREFIX = 'MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA';
const MAX_XOR_KEY_LEN = 16;

const PLAIN_KEY_RE = /["'`]([A-Za-z0-9+/]{300,}={0,2})["'`]/g;
const TABLE_RE = /function\s+(\w+)\(\)\s*\{\s*var\s+\w+\s*=\s*\[/g;
const TERM = '(?:\\w+\\(\\d+\\)|"(?:[^"\\\\]|\\\\.)*")';
const CHAIN_RE = new RegExp('(?:' + TERM + '\\s*\\+\\s*){30,}' + TERM, 'g');
const CALL_RE = /^(\w+)\((\d+)\)$/;
const B64_RE = /^[A-Za-z0-9+/]+={0,2}$/;

const ESCAPES = { n: '\n', t: '\t', r: '\r', b: '\b', f: '\f', v: '\v', 0: '\0' };

function parseJsString(src, i) {
  if (i >= src.length || (src[i] !== '"' && src[i] !== "'")) return [null, i];
  const quote = src[i];
  i++;
  let out = '';
  while (i < src.length) {
    const c = src[i];
    if (c === '\\' && i + 1 < src.length) {
      const n = src[i + 1];
      if (n === 'u' && i + 6 <= src.length) {
        out += String.fromCharCode(parseInt(src.slice(i + 2, i + 6), 16));
        i += 6;
        continue;
      }
      if (n === 'x' && i + 4 <= src.length) {
        out += String.fromCharCode(parseInt(src.slice(i + 2, i + 4), 16));
        i += 4;
        continue;
      }
      out += ESCAPES[n] !== undefined ? ESCAPES[n] : n;
      i += 2;
    } else if (c === quote) {
      return [out, i + 1];
    } else {
      out += c;
      i++;
    }
  }
  return [null, i];
}

function extractTables(src) {
  const tables = [];
  TABLE_RE.lastIndex = 0;
  let m;
  while ((m = TABLE_RE.exec(src)) !== null) {
    let i = src.indexOf('[', m.index + m[0].length - 1) + 1;
    const arr = [];
    let ok = true;
    while (i < src.length) {
      const c = src[i];
      if (c === ' ' || c === '\t' || c === '\r' || c === '\n' || c === ',') {
        i++;
      } else if (c === ']') {
        break;
      } else if (c === '"' || c === "'") {
        const [value, next] = parseJsString(src, i);
        if (value === null) { ok = false; break; }
        arr.push(value);
        i = next;
      } else {
        ok = false;
        break;
      }
    }
    if (ok && arr.length >= 20) tables.push(arr);
  }
  return tables;
}

function splitTerms(chain) {
  const terms = [];
  for (const part of chain.split('+')) {
    const p = part.trim();
    const m = CALL_RE.exec(p);
    if (m) { terms.push({ call: true, index: parseInt(m[2], 10) }); continue; }
    if (p.length > 1 && p[0] === '"') {
      const [value] = parseJsString(p, 0);
      if (value !== null) { terms.push({ call: false, lit: value }); continue; }
    }
    return null;
  }
  return terms;
}

function smallestPeriod(key) {
  for (let p = 1; p <= key.length; p++) {
    let ok = true;
    for (let i = p; i < key.length; i++) {
      if (key[i] !== key[i % p]) { ok = false; break; }
    }
    if (ok) return p;
  }
  return key.length;
}

function validSpki(b64) {
  if (!B64_RE.test(b64)) return false;
  let der;
  try { der = Buffer.from(b64, 'base64'); } catch (e) { return false; }
  if (der.length < 2 || der[0] !== 0x30 || der[1] !== 0x82) return false;
  try {
    const key = crypto.createPublicKey({ key: der, format: 'der', type: 'spki' });
    return key.asymmetricKeyType === 'rsa';
  } catch (e) {
    return false;
  }
}

function decryptSpki(ct) {
  if (ct.length < SPKI_PREFIX.length) return '';
  const raw = [];
  for (let i = 0; i < SPKI_PREFIX.length; i++) {
    raw.push(ct.charCodeAt(i) ^ SPKI_PREFIX.charCodeAt(i));
  }
  const p = smallestPeriod(raw);
  if (p > MAX_XOR_KEY_LEN) return '';
  let pt = '';
  for (let i = 0; i < ct.length; i++) {
    pt += String.fromCharCode(ct.charCodeAt(i) ^ raw[i % p]);
  }
  return validSpki(pt) ? pt : '';
}

function extractRsaKey(apiJs) {
  PLAIN_KEY_RE.lastIndex = 0;
  let m;
  while ((m = PLAIN_KEY_RE.exec(apiJs)) !== null) {
    if (validSpki(m[1])) return m[1];
  }

  const tables = extractTables(apiJs);
  if (!tables.length) return '';

  const chains = apiJs.match(CHAIN_RE) || [];
  for (const chain of chains) {
    const terms = splitTerms(chain);
    if (!terms || !terms.some((t) => t.call)) continue;
    for (const table of tables) {
      const n = table.length;
      for (let shift = 0; shift < n; shift++) {
        let s = '';
        for (const t of terms) s += t.call ? table[(t.index + shift) % n] : t.lit;
        const first = s.indexOf("'");
        const last = s.lastIndexOf("'");
        if (first < 0 || last <= first + 1) continue;
        const key = decryptSpki(s.slice(first + 1, last));
        if (key) return key;
      }
    }
  }
  return '';
}

module.exports = { extractRsaKey, validSpki };
