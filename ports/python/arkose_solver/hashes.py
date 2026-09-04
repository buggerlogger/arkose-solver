"""MurmurHash3 x64-128 and the BDA hash derivations.

The enforcement's `K` primitive is MurmurHash3 x64-128 emitted as
hex(h1) || hex(h2). Conformance vectors live in ports/testvectors.json.
"""

MASK64 = 0xFFFFFFFFFFFFFFFF
C1 = 0x87C37B91114253D5
C2 = 0x4CF5AD432745937F


def _rotl64(x, r):
    return ((x << r) | (x >> (64 - r))) & MASK64


def _fmix64(k):
    k = (k ^ (k >> 33)) & MASK64
    k = (k * 0xFF51AFD7ED558CCD) & MASK64
    k = (k ^ (k >> 33)) & MASK64
    k = (k * 0xC4CEB9FE1A85EC53) & MASK64
    return (k ^ (k >> 33)) & MASK64


def khash(data, seed=0):
    """K(data, seed) -> 32 lowercase hex chars."""
    if isinstance(data, str):
        data = data.encode("utf-8")
    length = len(data)
    nblocks = length // 16
    h1 = seed & MASK64
    h2 = seed & MASK64

    for i in range(nblocks):
        off = i * 16
        k1 = int.from_bytes(data[off:off + 8], "little")
        k2 = int.from_bytes(data[off + 8:off + 16], "little")

        k1 = (k1 * C1) & MASK64
        k1 = _rotl64(k1, 31)
        k1 = (k1 * C2) & MASK64
        h1 ^= k1

        h1 = _rotl64(h1, 27)
        h1 = (h1 + h2) & MASK64
        h1 = (h1 * 5 + 0x52DCE729) & MASK64

        k2 = (k2 * C2) & MASK64
        k2 = _rotl64(k2, 33)
        k2 = (k2 * C1) & MASK64
        h2 ^= k2

        h2 = _rotl64(h2, 31)
        h2 = (h2 + h1) & MASK64
        h2 = (h2 * 5 + 0x38495AB5) & MASK64

    tail = data[nblocks * 16:]
    k1 = 0
    k2 = 0
    n = len(tail)

    if n >= 15:
        k2 ^= tail[14] << 48
    if n >= 14:
        k2 ^= tail[13] << 40
    if n >= 13:
        k2 ^= tail[12] << 32
    if n >= 12:
        k2 ^= tail[11] << 24
    if n >= 11:
        k2 ^= tail[10] << 16
    if n >= 10:
        k2 ^= tail[9] << 8
    if n >= 9:
        k2 ^= tail[8]
        k2 = (k2 * C2) & MASK64
        k2 = _rotl64(k2, 33)
        k2 = (k2 * C1) & MASK64
        h2 ^= k2

    if n >= 8:
        k1 ^= tail[7] << 56
    if n >= 7:
        k1 ^= tail[6] << 48
    if n >= 6:
        k1 ^= tail[5] << 40
    if n >= 5:
        k1 ^= tail[4] << 32
    if n >= 4:
        k1 ^= tail[3] << 24
    if n >= 3:
        k1 ^= tail[2] << 16
    if n >= 2:
        k1 ^= tail[1] << 8
    if n >= 1:
        k1 ^= tail[0]
        k1 = (k1 * C1) & MASK64
        k1 = _rotl64(k1, 31)
        k1 = (k1 * C2) & MASK64
        h1 ^= k1

    h1 ^= length
    h2 ^= length
    h1 = (h1 + h2) & MASK64
    h2 = (h2 + h1) & MASK64
    h1 = _fmix64(h1)
    h2 = _fmix64(h2)
    h1 = (h1 + h2) & MASK64
    h2 = (h2 + h1) & MASK64

    return "%016x%016x" % (h1, h2)


def fe_values(fe):
    """Strip the "KEY:" prefix from each fe entry (the bundle's KQ())."""
    out = []
    for entry in fe:
        idx = entry.find(":")
        out.append(entry[idx + 1:] if idx >= 0 else entry)
    return out


def compute_f(fe):
    """bda["f"] = K(fe values joined by ";", seed 0)."""
    return khash(";".join(fe_values(fe)), 0)


def compute_ife_hash(fe):
    """bda["ife_hash"] = K(fe joined by ", ", seed 38)."""
    return khash(", ".join(fe), 38)
