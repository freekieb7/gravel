package json

import (
	"strings"
	"testing"
)

// Test SIMD character scanning
func TestSIMDCharacterScanning(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"no special chars", "hello world", 11},
		{"quote at start", `"hello`, 0},
		{"backslash in middle", `hello\world`, 5},
		{"control char", "hello\nworld", 5},
		{"unicode char", "hello世界", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := scanForEscapeCharsSIMD([]byte(tt.input))
			if pos != tt.expected {
				t.Errorf("scanForEscapeCharsSIMD() = %d, want %d", pos, tt.expected)
			}
		})
	}
}

// Test assembly-optimized string escaping
func TestAssemblyStringEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple string", "hello", "hello"},
		{"with quotes", `hello"world`, `hello\"world`},
		{"with backslashes", `hello\world`, `hello\\world`},
		{"with newlines", "hello\nworld", `hello\nworld`},
		{"with tabs", "hello\tworld", `hello\tworld`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result strings.Builder
			w := &stringWriter{&result}

			err := escapeStringOptimized(tt.input, w)
			if err != nil {
				t.Fatalf("escapeStringOptimized() error = %v", err)
			}

			if result.String() != tt.expected {
				t.Errorf("escapeStringOptimized() = %q, want %q", result.String(), tt.expected)
			}
		})
	}
}

// Test compile-time encoder generation
func TestCompileTimeEncoder(t *testing.T) {
	demo := DemoStruct{
		ID:   42,
		Name: "test",
	}

	// Test that the compile-time encoder is registered and works
	result, err := Marshal(demo)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	expected := `{"id":42,"name":"test"}`
	if string(result) != expected {
		t.Errorf("Marshal() = %q, want %q", string(result), expected)
	}
}

// Benchmark SIMD vs scalar character scanning
func BenchmarkSIMDCharScanning(b *testing.B) {
	data := []byte("this is a test string with no special characters that need escaping in JSON format")

	b.Run("SIMD", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			scanForEscapeCharsSIMD(data)
		}
	})

	b.Run("Scalar", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			scanForEscapeCharsScalar(data)
		}
	})
}

// Benchmark assembly vs standard string escaping
func BenchmarkStringEscaping(b *testing.B) {
	testString := "this is a test string with some \"quotes\" and \\backslashes\\ and\nnewlines"

	b.Run("OptimizedEscaping", func(b *testing.B) {
		var result strings.Builder
		w := &stringWriter{&result}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result.Reset()
			escapeStringOptimized(testString, w)
		}
	})

	b.Run("OriginalEscaping", func(b *testing.B) {
		var result strings.Builder
		w := &stringWriter{&result}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result.Reset()
			escapeStringWriter_original(testString, w)
		}
	})
}

// Benchmark compile-time vs reflection-based encoding
func BenchmarkCompileTimeEncoder(b *testing.B) {
	demo := DemoStruct{
		ID:   42,
		Name: "benchmark test",
	}

	b.Run("CompileTime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := Marshal(demo)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Test without compile-time encoder by temporarily removing it
	b.Run("Reflection", func(b *testing.B) {
		// Create a different struct type that doesn't have a compile-time encoder
		type ReflectionStruct struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		reflectionDemo := ReflectionStruct{ID: 42, Name: "benchmark test"}

		for i := 0; i < b.N; i++ {
			_, err := Marshal(reflectionDemo)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Helper for testing
type stringWriter struct {
	builder *strings.Builder
}

func (w *stringWriter) WriteByte(b byte) error {
	return w.builder.WriteByte(b)
}

func (w *stringWriter) WriteString(s string) error {
	_, err := w.builder.WriteString(s)
	return err
}

func (w *stringWriter) WriteBytes(data []byte) error {
	_, err := w.builder.Write(data)
	return err
}
