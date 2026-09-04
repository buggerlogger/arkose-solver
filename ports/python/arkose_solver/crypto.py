"""The BDA transport envelope.

c = b64(iv:12) || b64(gcmTag:16) || b64(rsaBlock:256) || b64(ciphertext)

The AES-256-GCM content key is wrapped with RSA-OAEP(SHA-256) under the site's
public key. Verified byte-for-byte against live capi 4.4.3 and 4.4.5 mints.
"""

import base64
import hashlib
import json
import os

from cryptography.hazmat.primitives import hashes as _hashes
from cryptography.hazmat.primitives.asymmetric import padding
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.serialization import load_der_public_key

GCM_TAG_LEN = 16


def encrypt(content, pubkey_b64):
    """Return the `c` field for a BDA plaintext and an SPKI base64 public key."""
    key = os.urandom(32)
    iv = os.urandom(12)

    sealed = AESGCM(key).encrypt(iv, content.encode("utf-8"), None)
    ciphertext, tag = sealed[:-GCM_TAG_LEN], sealed[-GCM_TAG_LEN:]

    public_key = load_der_public_key(base64.b64decode(pubkey_b64))
    wrapped = public_key.encrypt(
        key,
        padding.OAEP(
            mgf=padding.MGF1(algorithm=_hashes.SHA256()),
            algorithm=_hashes.SHA256(),
            label=None,
        ),
    )

    b64 = lambda b: base64.b64encode(b).decode("ascii")
    return b64(iv) + b64(tag) + b64(wrapped) + b64(ciphertext)


def md5_hex(s):
    return hashlib.md5(s.encode("utf-8")).hexdigest()


def marshal_bda(items):
    """Serialise the BDA the way the browser does: compact, non-ASCII escaped."""
    return json.dumps(items, separators=(",", ":"), ensure_ascii=True)
