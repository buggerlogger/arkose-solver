'use strict';

// The BDA transport envelope.
//
//   c = b64(iv:12) || b64(gcmTag:16) || b64(rsaBlock:256) || b64(ciphertext)
//
// The AES-256-GCM content key is wrapped with RSA-OAEP(SHA-256) under the
// site's public key. Verified byte-for-byte against live capi 4.4.3 and 4.4.5
// mints.

const crypto = require('crypto');

function encrypt(content, pubkeyB64) {
  const key = crypto.randomBytes(32);
  const iv = crypto.randomBytes(12);

  const cipher = crypto.createCipheriv('aes-256-gcm', key, iv);
  const ciphertext = Buffer.concat([cipher.update(content, 'utf8'), cipher.final()]);
  const tag = cipher.getAuthTag();

  const publicKey = crypto.createPublicKey({
    key: Buffer.from(pubkeyB64, 'base64'),
    format: 'der',
    type: 'spki',
  });
  const wrapped = crypto.publicEncrypt(
    { key: publicKey, padding: crypto.constants.RSA_PKCS1_OAEP_PADDING, oaepHash: 'sha256' },
    key,
  );

  const b64 = (b) => b.toString('base64');
  return b64(iv) + b64(tag) + b64(wrapped) + b64(ciphertext);
}

// Serialise the BDA the way the browser does: compact, non-ASCII escaped.
function marshalBda(items) {
  return JSON.stringify(items).replace(
    /[\u0080-\uffff]/g,
    (c) => '\\u' + c.charCodeAt(0).toString(16).padStart(4, '0'),
  );
}

module.exports = { encrypt, marshalBda };
