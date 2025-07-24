//go:build amd64

package json

import (
	"golang.org/x/sys/cpu"
)

//go:noescape
func scanJSONChars16(data []byte) int

//go:noescape
func scanQuotesAndEscapes16(data []byte) int

// SIMD-optimized scanning for JSON special characters
func scanForEscapeCharsSIMD(data []byte) int {
	// Fallback for small data or unsupported CPU
	if !cpu.X86.HasSSE2 || len(data) < 16 {
		return scanForEscapeCharsScalar(data)
	}

	// Process 16-byte chunks with SIMD
	processed := 0
	for len(data) >= 16 {
		pos := scanJSONChars16(data[:16])
		if pos < 16 {
			return processed + pos
		}
		processed += 16
		data = data[16:]
	}

	// Handle remaining bytes
	if len(data) > 0 {
		pos := scanForEscapeCharsScalar(data)
		if pos < len(data) {
			return processed + pos
		}
		processed += len(data)
	}

	return processed
}

// SIMD-optimized scanning for quotes and escape characters
func scanQuotesAndEscapesSIMD(data []byte) int {
	// Fallback for small data or unsupported CPU
	if !cpu.X86.HasSSE2 || len(data) < 16 {
		return scanQuotesAndEscapesScalar(data)
	}

	// Process 16-byte chunks
	processed := 0
	for len(data) >= 16 {
		pos := scanQuotesAndEscapes16(data[:16])
		if pos < 16 {
			return processed + pos
		}
		processed += 16
		data = data[16:]
	}

	// Handle remaining bytes
	if len(data) > 0 {
		pos := scanQuotesAndEscapesScalar(data)
		if pos < len(data) {
			return processed + pos
		}
		processed += len(data)
	}

	return processed
}

// Scalar fallback implementations
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
