"""Recover the site's RSA public key from the served api.js.

The key is not a literal in the bundle. It lives inside the VM program api.js
embeds, as a repeating-XOR encrypted operand that is itself assembled at
runtime from string-table fragments.

The decoder offset and the load-time table rotation are both index shifts, so
they collapse into a single unknown that can be brute-forced. Each candidate is
accepted only if it decrypts to a real PKIX RSA public key, which makes the
search self-validating: no opcode tag, table name, offset or rotation checksum
is hard-coded, so this survives a build rotation.
"""

import re

from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.primitives.serialization import load_der_public_key

import base64

SPKI_PREFIX = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA"
MAX_XOR_KEY_LEN = 16

_PLAIN_KEY_RE = re.compile(r"""["'`]([A-Za-z0-9+/]{300,}={0,2})["'`]""")
_TABLE_RE = re.compile(r"function\s+(\w+)\(\)\s*\{\s*var\s+\w+\s*=\s*\[")
_TERM = r'(?:\w+\(\d+\)|"(?:[^"\\]|\\.)*")'
_CHAIN_RE = re.compile(r"(?:" + _TERM + r"\s*\+\s*){30,}" + _TERM)
_CALL_RE = re.compile(r"^(\w+)\((\d+)\)$")
_B64_RE = re.compile(r"^[A-Za-z0-9+/]+={0,2}$")

_ESCAPES = {"n": "\n", "t": "\t", "r": "\r", "b": "\b", "f": "\f", "v": "\v", "0": "\0"}


def _parse_js_string(src, i):
    """src[i] is a quote. Returns (value, index_after_close) or (None, i)."""
    if i >= len(src) or src[i] not in "\"'":
        return None, i
    quote = src[i]
    i += 1
    out = []
    while i < len(src):
        c = src[i]
        if c == "\\" and i + 1 < len(src):
            n = src[i + 1]
            if n == "u" and i + 6 <= len(src):
                try:
                    out.append(chr(int(src[i + 2:i + 6], 16)))
                    i += 6
                    continue
                except ValueError:
                    pass
            if n == "x" and i + 4 <= len(src):
                try:
                    out.append(chr(int(src[i + 2:i + 4], 16)))
                    i += 4
                    continue
                except ValueError:
                    pass
            out.append(_ESCAPES.get(n, n))
            i += 2
        elif c == quote:
            return "".join(out), i + 1
        else:
            out.append(c)
            i += 1
    return None, i


def _extract_tables(src):
    tables = []
    for m in _TABLE_RE.finditer(src):
        i = src.index("[", m.end() - 1) + 1
        arr = []
        ok = True
        while i < len(src):
            c = src[i]
            if c in " \t\r\n,":
                i += 1
            elif c == "]":
                break
            elif c in "\"'":
                value, i = _parse_js_string(src, i)
                if value is None:
                    ok = False
                    break
                arr.append(value)
            else:
                ok = False
                break
        if ok and len(arr) >= 20:
            tables.append(arr)
    return tables


def _split_terms(chain):
    terms = []
    for part in chain.split("+"):
        p = part.strip()
        m = _CALL_RE.match(p)
        if m:
            terms.append((True, int(m.group(2))))
            continue
        if len(p) > 1 and p[0] == '"':
            value, _ = _parse_js_string(p, 0)
            if value is not None:
                terms.append((False, value))
                continue
        return None
    return terms


def _smallest_period(key):
    for p in range(1, len(key) + 1):
        if all(key[i] == key[i % p] for i in range(len(key))):
            return p
    return len(key)


def _valid_spki(b64):
    if not _B64_RE.match(b64):
        return False
    try:
        der = base64.b64decode(b64, validate=True)
    except Exception:
        return False
    if len(der) < 2 or der[0] != 0x30 or der[1] != 0x82:
        return False
    try:
        return isinstance(load_der_public_key(der), rsa.RSAPublicKey)
    except Exception:
        return False


def _decrypt_spki(ct):
    if len(ct) < len(SPKI_PREFIX):
        return ""
    raw = bytes(ord(ct[i]) ^ ord(SPKI_PREFIX[i]) for i in range(len(SPKI_PREFIX)))
    p = _smallest_period(raw)
    if p > MAX_XOR_KEY_LEN:
        return ""
    key = raw[:p]
    pt = "".join(chr(ord(ct[i]) ^ key[i % p]) for i in range(len(ct)))
    return pt if _valid_spki(pt) else ""


def extract_rsa_key(api_js):
    """Return the site's RSA public key as SPKI base64, or "" if not found."""
    for m in _PLAIN_KEY_RE.finditer(api_js):
        if _valid_spki(m.group(1)):
            return m.group(1)

    tables = _extract_tables(api_js)
    if not tables:
        return ""

    for chain in _CHAIN_RE.findall(api_js):
        terms = _split_terms(chain)
        if not terms or not any(is_call for is_call, _ in terms):
            continue
        for table in tables:
            n = len(table)
            for shift in range(n):
                parts = []
                for is_call, value in terms:
                    parts.append(table[(value + shift) % n] if is_call else value)
                s = "".join(parts)
                first = s.find("'")
                last = s.rfind("'")
                if first < 0 or last <= first + 1:
                    continue
                key = _decrypt_spki(s[first + 1:last])
                if key:
                    return key
    return ""
