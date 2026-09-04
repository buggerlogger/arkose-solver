package arkose

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"regexp"
	"strconv"
	"strings"
)

const spkiPrefix = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA"

const maxXORKeyLen = 16

var (
	plainKeyRe = regexp.MustCompile("[\"'`]([A-Za-z0-9+/]{300,}={0,2})[\"'`]")
	tableRe    = regexp.MustCompile(`function\s+(\w+)\(\)\s*\{\s*var\s+\w+\s*=\s*\[`)
	termPat    = `(?:\w+\(\d+\)|"(?:[^"\\]|\\.)*")`
	chainRe    = regexp.MustCompile(`(?:` + termPat + `\s*\+\s*){30,}` + termPat)
	callRe     = regexp.MustCompile(`^(\w+)\((\d+)\)$`)
	b64Re      = regexp.MustCompile(`^[A-Za-z0-9+/]+={0,2}$`)
)

func parseJSString(s string, i int) (string, int, bool) {
	if i >= len(s) || (s[i] != '"' && s[i] != '\'') {
		return "", i, false
	}
	q := s[i]
	i++
	var b strings.Builder
	for i < len(s) {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s):
			n := s[i+1]
			switch n {
			case 'u':
				if i+6 <= len(s) {
					if v, err := strconv.ParseUint(s[i+2:i+6], 16, 32); err == nil {
						b.WriteRune(rune(v))
						i += 6
						continue
					}
				}
				i += 2
			case 'x':
				if i+4 <= len(s) {
					if v, err := strconv.ParseUint(s[i+2:i+4], 16, 32); err == nil {
						b.WriteRune(rune(v))
						i += 4
						continue
					}
				}
				i += 2
			case 'n':
				b.WriteByte('\n')
				i += 2
			case 't':
				b.WriteByte('\t')
				i += 2
			case 'r':
				b.WriteByte('\r')
				i += 2
			case 'b':
				b.WriteByte('\b')
				i += 2
			case 'f':
				b.WriteByte('\f')
				i += 2
			case 'v':
				b.WriteByte('\v')
				i += 2
			case '0':
				b.WriteByte(0)
				i += 2
			default:
				b.WriteByte(n)
				i += 2
			}
		case c == q:
			return b.String(), i + 1, true
		default:
			b.WriteByte(c)
			i++
		}
	}
	return "", i, false
}

func extractTables(src string) [][]string {
	var out [][]string
	for _, loc := range tableRe.FindAllStringIndex(src, -1) {
		i := strings.Index(src[loc[0]:loc[1]], "[")
		if i < 0 {
			continue
		}
		i += loc[0] + 1
		var arr []string
		ok := true
		for i < len(src) {
			c := src[i]
			switch {
			case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == ',':
				i++
			case c == ']':
				i = len(src)
			case c == '"' || c == '\'':
				v, ni, good := parseJSString(src, i)
				if !good {
					ok = false
					i = len(src)
					break
				}
				arr = append(arr, v)
				i = ni
			default:
				ok = false
				i = len(src)
			}
		}
		if ok && len(arr) >= 20 {
			out = append(out, arr)
		}
	}
	return out
}

type chainTerm struct {
	isCall bool
	index  int
	lit    string
}

func splitTerms(chain string) []chainTerm {
	var out []chainTerm
	for _, part := range strings.Split(chain, "+") {
		p := strings.TrimSpace(part)
		if m := callRe.FindStringSubmatch(p); m != nil {
			n, err := strconv.Atoi(m[2])
			if err != nil {
				return nil
			}
			out = append(out, chainTerm{isCall: true, index: n})
			continue
		}
		if len(p) > 1 && p[0] == '"' {
			if v, _, ok := parseJSString(p, 0); ok {
				out = append(out, chainTerm{lit: v})
				continue
			}
		}
		return nil
	}
	return out
}

func smallestPeriod(k []byte) int {
	for p := 1; p <= len(k); p++ {
		ok := true
		for i := p; i < len(k); i++ {
			if k[i] != k[i%p] {
				ok = false
				break
			}
		}
		if ok {
			return p
		}
	}
	return len(k)
}

func xorWith(s string, key []byte) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		out[i] = s[i] ^ key[i%len(key)]
	}
	return string(out)
}

func validSPKI(b64 string) bool {
	if !b64Re.MatchString(b64) {
		return false
	}
	der, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(der) < 2 || der[0] != 0x30 || der[1] != 0x82 {
		return false
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return false
	}
	_, ok := pub.(*rsa.PublicKey)
	return ok
}

func decryptSPKI(ct string) string {
	if len(ct) < len(spkiPrefix) {
		return ""
	}
	raw := make([]byte, len(spkiPrefix))
	for i := 0; i < len(spkiPrefix); i++ {
		raw[i] = ct[i] ^ spkiPrefix[i]
	}
	p := smallestPeriod(raw)
	if p > maxXORKeyLen {
		return ""
	}
	if pt := xorWith(ct, raw[:p]); validSPKI(pt) {
		return pt
	}
	return ""
}

func ExtractRSAKey(apiJS string) string {
	for _, m := range plainKeyRe.FindAllStringSubmatch(apiJS, -1) {
		if validSPKI(m[1]) {
			return m[1]
		}
	}

	tables := extractTables(apiJS)
	if len(tables) == 0 {
		return ""
	}

	for _, chain := range chainRe.FindAllString(apiJS, -1) {
		terms := splitTerms(chain)
		if terms == nil {
			continue
		}
		hasCall := false
		for _, t := range terms {
			if t.isCall {
				hasCall = true
				break
			}
		}
		if !hasCall {
			continue
		}
		for _, tab := range tables {
			n := len(tab)
			for shift := 0; shift < n; shift++ {
				var b strings.Builder
				for _, t := range terms {
					if t.isCall {
						b.WriteString(tab[(t.index+shift)%n])
					} else {
						b.WriteString(t.lit)
					}
				}
				s := b.String()
				first := strings.IndexByte(s, '\'')
				last := strings.LastIndexByte(s, '\'')
				if first < 0 || last <= first+1 {
					continue
				}
				if key := decryptSPKI(s[first+1 : last]); key != "" {
					return key
				}
			}
		}
	}
	return ""
}
