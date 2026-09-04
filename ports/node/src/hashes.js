'use strict';

// MurmurHash3 x64-128 emitted as hex(h1) || hex(h2) -- the enforcement's `K`.
// Conformance vectors live in ports/testvectors.json.

const MASK = 0xffffffffffffffffn;
const C1 = 0x87c37b91114253d5n;
const C2 = 0x4cf5ad432745937fn;

function rotl64(x, r) {
  const n = BigInt(r);
  return ((x << n) | (x >> (64n - n))) & MASK;
}

function fmix64(k) {
  k = (k ^ (k >> 33n)) & MASK;
  k = (k * 0xff51afd7ed558ccdn) & MASK;
  k = (k ^ (k >> 33n)) & MASK;
  k = (k * 0xc4ceb9fe1a85ec53n) & MASK;
  return (k ^ (k >> 33n)) & MASK;
}

function readLE64(buf, off) {
  let v = 0n;
  for (let i = 7; i >= 0; i--) v = (v << 8n) | BigInt(buf[off + i]);
  return v;
}

function khash(data, seed = 0) {
  const buf = Buffer.isBuffer(data) ? data : Buffer.from(String(data), 'utf8');
  const len = buf.length;
  const nblocks = Math.floor(len / 16);
  let h1 = BigInt(seed) & MASK;
  let h2 = BigInt(seed) & MASK;

  for (let i = 0; i < nblocks; i++) {
    const off = i * 16;
    let k1 = readLE64(buf, off);
    let k2 = readLE64(buf, off + 8);

    k1 = (k1 * C1) & MASK;
    k1 = rotl64(k1, 31);
    k1 = (k1 * C2) & MASK;
    h1 ^= k1;

    h1 = rotl64(h1, 27);
    h1 = (h1 + h2) & MASK;
    h1 = (h1 * 5n + 0x52dce729n) & MASK;

    k2 = (k2 * C2) & MASK;
    k2 = rotl64(k2, 33);
    k2 = (k2 * C1) & MASK;
    h2 ^= k2;

    h2 = rotl64(h2, 31);
    h2 = (h2 + h1) & MASK;
    h2 = (h2 * 5n + 0x38495ab5n) & MASK;
  }

  const tail = buf.subarray(nblocks * 16);
  const n = tail.length;
  let k1 = 0n;
  let k2 = 0n;

  if (n >= 15) k2 ^= BigInt(tail[14]) << 48n;
  if (n >= 14) k2 ^= BigInt(tail[13]) << 40n;
  if (n >= 13) k2 ^= BigInt(tail[12]) << 32n;
  if (n >= 12) k2 ^= BigInt(tail[11]) << 24n;
  if (n >= 11) k2 ^= BigInt(tail[10]) << 16n;
  if (n >= 10) k2 ^= BigInt(tail[9]) << 8n;
  if (n >= 9) {
    k2 ^= BigInt(tail[8]);
    k2 = (k2 * C2) & MASK;
    k2 = rotl64(k2, 33);
    k2 = (k2 * C1) & MASK;
    h2 ^= k2;
  }

  if (n >= 8) k1 ^= BigInt(tail[7]) << 56n;
  if (n >= 7) k1 ^= BigInt(tail[6]) << 48n;
  if (n >= 6) k1 ^= BigInt(tail[5]) << 40n;
  if (n >= 5) k1 ^= BigInt(tail[4]) << 32n;
  if (n >= 4) k1 ^= BigInt(tail[3]) << 24n;
  if (n >= 3) k1 ^= BigInt(tail[2]) << 16n;
  if (n >= 2) k1 ^= BigInt(tail[1]) << 8n;
  if (n >= 1) {
    k1 ^= BigInt(tail[0]);
    k1 = (k1 * C1) & MASK;
    k1 = rotl64(k1, 31);
    k1 = (k1 * C2) & MASK;
    h1 ^= k1;
  }

  h1 ^= BigInt(len);
  h2 ^= BigInt(len);
  h1 = (h1 + h2) & MASK;
  h2 = (h2 + h1) & MASK;
  h1 = fmix64(h1);
  h2 = fmix64(h2);
  h1 = (h1 + h2) & MASK;
  h2 = (h2 + h1) & MASK;

  return h1.toString(16).padStart(16, '0') + h2.toString(16).padStart(16, '0');
}

// Strip the "KEY:" prefix from each fe entry (the bundle's KQ()).
function feValues(fe) {
  return fe.map((entry) => {
    const i = entry.indexOf(':');
    return i >= 0 ? entry.slice(i + 1) : entry;
  });
}

const computeF = (fe) => khash(feValues(fe).join(';'), 0);
const computeIfeHash = (fe) => khash(fe.join(', '), 38);

module.exports = { khash, feValues, computeF, computeIfeHash };
