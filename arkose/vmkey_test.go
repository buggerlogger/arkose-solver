package arkose

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func syntheticDollarTableBundle(t *testing.T) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	want := base64.StdEncoding.EncodeToString(der)

	instruction := "f9c78 '" + xorWith(want, []byte("70")) + "' string"
	const chunkSize = 8
	var chunks, calls []string
	for start := 0; start < len(instruction); start += chunkSize {
		end := start + chunkSize
		if end > len(instruction) {
			end = len(instruction)
		}
		quoted, err := json.Marshal(instruction[start:end])
		if err != nil {
			t.Fatal(err)
		}
		chunks = append(chunks, string(quoted))
		calls = append(calls, "de("+strconv.Itoa(len(calls))+")")
	}

	source := "function $e(){var $t=[" + strings.Join(chunks, ",") +
		"];return $t;}var ye=[" + strings.Join(calls, "+") + "];"
	return source, want
}

func TestExtractRSAKeyAllowsDollarIdentifiers(t *testing.T) {
	source, want := syntheticDollarTableBundle(t)
	if got := ExtractRSAKey(source); got != want {
		t.Fatalf("ExtractRSAKey() returned a different key: got length %d, want %d", len(got), len(want))
	}
}
