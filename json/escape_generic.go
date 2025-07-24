//go:build !amd64
// +build !amd64

package json

// Fallback implementation for non-AMD64 architectures
func escapeStringASM(src []byte, dst []byte) (needsEscape bool, pos int) {
	for i, b := range src {
		// Check for characters that need escaping
		if b < 0x20 || b >= 0x80 || b == '"' || b == '\\' {
			return true, i
		}
		if i < len(dst) {
			dst[i] = b
		}
	}
	return false, len(src)
}

// Fast string escaping using Go implementation for non-AMD64
func escapeStringFast(s string, w writer) error {
	return escapeStringWriter(s, w)
}
