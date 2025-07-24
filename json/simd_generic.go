//go:build !amd64

package json

// Fallback implementations for non-AMD64 architectures

func scanForEscapeCharsSIMD(data []byte) int {
	return scanForEscapeCharsScalar(data)
}

func scanQuotesAndEscapesSIMD(data []byte) int {
	return scanQuotesAndEscapesScalar(data)
}

func scanForEscapeCharsScalar(data []byte) int {
	for i, b := range data {
		if b < 0x20 || b >= 0x80 || b == '"' || b == '\\' {
			return i
		}
	}
	return len(data)
}

func scanQuotesAndEscapesScalar(data []byte) int {
	for i, b := range data {
		if b == '"' || b == '\\' {
			return i
		}
	}
	return len(data)
}
