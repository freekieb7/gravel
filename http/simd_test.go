package http

import (
	"bytes"
	"testing"
)

func TestSIMDLowercase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"CONTENT-LENGTH", "content-length"},
		{"TRANSFER-ENCODING", "transfer-encoding"},
		{"CONNECTION", "connection"},
		{"HOST", "host"},
		{"USER-AGENT", "user-agent"},
		{"MixedCase", "mixedcase"},
		{"already-lower", "already-lower"},
		{"A", "a"},
		{"", ""},
	}

	for _, tc := range testCases {
		data := []byte(tc.input)
		toLowerSIMD(data)

		if string(data) != tc.expected {
			t.Errorf("toLowerSIMD(%q) = %q, want %q", tc.input, data, tc.expected)
		}
	}
}

func TestSIMDEquals(t *testing.T) {
	testCases := []struct {
		a, b     string
		expected bool
	}{
		{"content-length", "content-length", true},
		{"content-length", "content-type", false},
		{"", "", true},
		{"a", "a", true},
		{"a", "b", false},
		{"hello world test", "hello world test", true},
		{"hello world test", "hello world xyz", false},
	}

	for _, tc := range testCases {
		result := equalsFast([]byte(tc.a), []byte(tc.b))

		if result != tc.expected {
			t.Errorf("equalsFast(%q, %q) = %v, want %v", tc.a, tc.b, result, tc.expected)
		}
	}
}

func BenchmarkLowercaseSIMD(b *testing.B) {
	data := make([]byte, len("TRANSFER-ENCODING"))
	copy(data, "TRANSFER-ENCODING")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(data, "TRANSFER-ENCODING")
		toLowerSIMD(data)
	}
}

func BenchmarkLowercaseScalar(b *testing.B) {
	data := make([]byte, len("TRANSFER-ENCODING"))
	copy(data, "TRANSFER-ENCODING")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(data, "TRANSFER-ENCODING")
		toLowerScalar(data)
	}
}

func BenchmarkEqualsSIMD(b *testing.B) {
	a := []byte("transfer-encoding")
	b1 := []byte("transfer-encoding")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		equalsFast(a, b1)
	}
}

func BenchmarkEqualsStandard(b *testing.B) {
	a := []byte("transfer-encoding")
	b1 := []byte("transfer-encoding")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytes.Equal(a, b1)
	}
}
