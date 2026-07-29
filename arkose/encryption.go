package arkose

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// Encrypt implements encryption.py::encrypt(content, pubkey).
// Layout: b64(iv) + b64(gcm_tag) + b64(rsa_oaep_sha256(aes_key)) + b64(aes_gcm_ciphertext)
// where aes_key is 32 random bytes, iv is 12 random bytes, and pubkey is base64-encoded DER (SPKI).
func Encrypt(content string, pubkeyB64 string) (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("rand key: %w", err)
	}
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("rand iv: %w", err)
	}

	// AES-256-GCM encrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}
	// Non-standard nonce size (12 is standard but explicit for safety).
	aesgcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	sealed := aesgcm.Seal(nil, iv, []byte(content), nil)
	// sealed = ciphertext || tag ; tag is 16 bytes
	if len(sealed) < 16 {
		return "", errors.New("gcm sealed too short")
	}
	ciphertext := sealed[:len(sealed)-16]
	tag := sealed[len(sealed)-16:]

	// RSA-OAEP-SHA256 encrypt the AES key
	pubDER, err := base64.StdEncoding.DecodeString(pubkeyB64)
	if err != nil {
		return "", fmt.Errorf("pubkey b64: %w", err)
	}
	pubKeyIfc, err := x509.ParsePKIXPublicKey(pubDER)
	if err != nil {
		return "", fmt.Errorf("parse SPKI: %w", err)
	}
	rsaPub, ok := pubKeyIfc.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("pubkey not RSA")
	}
	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, key, nil)
	if err != nil {
		return "", fmt.Errorf("rsa oaep: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString
	return b64(iv) + b64(tag) + b64(encryptedKey) + b64(ciphertext), nil
}

// genKeyGo mirrors encryption.py::_gen_key_go — MD5-based KDF that produces 32 bytes.
// key = md5(u) || md5(md5(u) || u) || md5(md5(md5(u) || u) || u), trimmed to 32 bytes.
func genKeyGo(userAgent, xArkValue, sValueHex string) ([]byte, error) {
	transformed, err := hex.DecodeString(sValueHex)
	if err != nil {
		return nil, err
	}
	u := append([]byte(userAgent+xArkValue), transformed...)

	s := make([][]byte, 3)
	h := md5.Sum(u)
	s[0] = h[:]
	f := make([]byte, 0, 48)
	f = append(f, s[0]...)
	for l := 1; l < 3; l++ {
		combined := append(append([]byte{}, s[l-1]...), u...)
		h := md5.Sum(combined)
		s[l] = h[:]
		f = append(f, s[l]...)
	}
	return f[:32], nil
}

// EncryptAES mirrors encryption.py::encrypt_aes — AES-256-CBC with MD5-KDF, PKCS7 padded.
// Output: compact JSON {"ct": b64(ct), "s": s_hex, "iv": iv_hex}
func EncryptAES(data, userAgent, xArkValue string) (string, error) {
	sBytes := make([]byte, 8)
	if _, err := rand.Read(sBytes); err != nil {
		return "", err
	}
	sHex := hex.EncodeToString(sBytes)

	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	key, err := genKeyGo(userAgent, xArkValue, sHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	// PKCS7 pad
	padded := pkcs7Pad([]byte(data), 16)
	ct := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ct, padded)

	// Emit fields in insertion order (ct, s, iv) — Python compact JSON preserves dict order.
	// Use ordered marshaling to guarantee output byte-order matches Python.
	type payload struct {
		Ct string `json:"ct"`
		S  string `json:"s"`
		Iv string `json:"iv"`
	}
	buf, err := json.Marshal(payload{
		Ct: base64.StdEncoding.EncodeToString(ct),
		S:  sHex,
		Iv: hex.EncodeToString(iv),
	})
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - (len(data) % blockSize)
	if pad == 0 {
		pad = blockSize
	}
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}
