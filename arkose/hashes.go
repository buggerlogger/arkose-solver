package arkose

import (
	"fmt"
	"strings"

	"github.com/spaolacci/murmur3"
)

func KHash(data string, seed uint32) string {
	h1, h2 := murmur3.Sum128WithSeed([]byte(data), seed)
	return fmt.Sprintf("%016x%016x", h1, h2)
}

func feValues(fe []string) []string {
	out := make([]string, len(fe))
	for i, s := range fe {
		if j := strings.IndexByte(s, ':'); j >= 0 {
			out[i] = s[j+1:]
		} else {
			out[i] = s
		}
	}
	return out
}

func ComputeF(fe []string) string {
	return KHash(strings.Join(feValues(fe), ";"), 0)
}

func ComputeIfeHash(fe []string) string {
	return KHash(strings.Join(fe, ", "), 38)
}
