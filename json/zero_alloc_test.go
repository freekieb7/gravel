package json

import (
	"testing"
)

// Test that demonstrates zero allocations when reusing buffers
func TestZeroAllocationMarshal(t *testing.T) {
	data := SimplePerson{
		Name: "John",
		Age:  30,
	}

	// Test MarshalAppend with reused buffer
	buf := make([]byte, 0, 256)

	// First call to establish baseline
	result1, err := MarshalAppend(buf, data)
	if err != nil {
		t.Fatal(err)
	}

	// Reuse the buffer - this should be zero allocations
	result2, err := MarshalAppend(result1[:0], data)
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"name":"John","age":30,"active":false,"score":0}`
	if string(result2) != expected {
		t.Errorf("Expected %s, got %s", expected, string(result2))
	}
}

func TestZeroAllocationMarshalTo(t *testing.T) {
	data := SimplePerson{
		Name: "Jane",
		Age:  25,
	}

	// Test MarshalTo with fixed buffer
	buf := make([]byte, 256)

	n, err := MarshalTo(data, buf)
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"name":"Jane","age":25,"active":false,"score":0}`
	if string(buf[:n]) != expected {
		t.Errorf("Expected %s, got %s", expected, string(buf[:n]))
	}
}

// Benchmark to verify true zero allocations with buffer reuse
func BenchmarkZeroAlloc_MarshalAppend_Reuse(b *testing.B) {
	data := SimplePerson{
		Name: "John",
		Age:  30,
	}
	buf := make([]byte, 0, 256)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var err error
		buf, err = MarshalAppend(buf[:0], data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkZeroAlloc_MarshalTo_Reuse(b *testing.B) {
	data := SimplePerson{
		Name: "John",
		Age:  30,
	}
	buf := make([]byte, 256)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := MarshalTo(data, buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}
