//go:build amd64

package http

import (
	"bytes"

	"golang.org/x/sys/cpu"
)

//go:noescape
func toLower16Bytes(data []byte)

//go:noescape
func equalsSIMD(a, b []byte) bool

// SIMD-optimized lowercase conversion
func toLowerSIMD(data []byte) {
	// Simple fallback for small data or unsupported CPU
	if !cpu.X86.HasSSE2 || len(data) < 16 {
		toLowerScalar(data)
		return
	}

	// Process 16-byte chunks
	for len(data) >= 16 {
		toLower16Bytes(data[:16])
		data = data[16:]
	}

	// Handle remaining bytes
	if len(data) > 0 {
		toLowerScalar(data)
	}
}

func toLowerScalar(data []byte) {
	for i := range data {
		if data[i] >= 'A' && data[i] <= 'Z' {
			data[i] += 'a' - 'A'
		}
	}
}

func equalsFast(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	// Use standard library for small slices or unsupported CPU
	if !cpu.X86.HasSSE2 || len(a) < 16 {
		return bytes.Equal(a, b)
	}

	// For longer slices, try SIMD but fallback to standard on mismatch
	if len(a) >= 16 && equalsSIMD(a[:16], b[:16]) {
		// If first 16 bytes match, compare the rest
		return bytes.Equal(a[16:], b[16:])
	}

	return bytes.Equal(a, b)
}
