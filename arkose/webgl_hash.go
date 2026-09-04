package arkose

import "strings"

func webglHashInput(fields []Item) string {
	var b strings.Builder
	for _, it := range fields {
		b.WriteString(it.Key)
		b.WriteByte(',')
		if s, ok := it.Value.(string); ok {
			b.WriteString(s)
		}
		b.WriteByte(',')
	}

	b.WriteString("webgl_hash_webgl,")
	return b.String()
}

func computeWebGLHash(fields []Item) string {
	return KHash(webglHashInput(fields), 0)
}

func computeRTTType(extensionsHash, webglHash string) string {
	if len(extensionsHash) > 12 && len(webglHash) > 12 {
		return extensionsHash[:3] + webglHash[:3]
	}
	return rttTypeFallback
}

const rttTypeFallback = "abcdef"
