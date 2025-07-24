package json

import (
	"strings"
	"testing"
	"time"
)

// Test two-stage parsing
func TestTwoStageParser(t *testing.T) {
	testJSON := `{"name":"John","age":30,"active":true,"scores":[95,87,92]}`

	parser, err := ParseFast([]byte(testJSON))
	if err != nil {
		t.Fatalf("ParseFast() error = %v", err)
	}

	if len(parser.tape) == 0 {
		t.Error("Expected non-empty tape")
	}

	if len(parser.strings) == 0 {
		t.Error("Expected non-empty strings pool")
	}
}

// Test zero-copy search
func TestZeroCopySearch(t *testing.T) {
	testJSON := `{
		"user": {
			"name": "Alice",
			"details": {
				"age": 25,
				"active": true
			}
		},
		"scores": [100, 95, 88]
	}`

	tests := []struct {
		name     string
		path     string
		expected interface{}
		method   string
	}{
		{"nested string", "user.name", "Alice", "string"},
		{"nested int", "user.details.age", int64(25), "int"},
		{"nested bool", "user.details.active", true, "bool"},
		{"array element", "scores.0", int64(100), "int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := ZeroCopySearch([]byte(testJSON), tt.path)
			if err != nil {
				t.Fatalf("ZeroCopySearch() error = %v", err)
			}

			switch tt.method {
			case "string":
				result, err := value.GetString()
				if err != nil {
					t.Fatalf("GetString() error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("GetString() = %v, want %v", result, tt.expected)
				}
			case "int":
				result, err := value.GetInt()
				if err != nil {
					t.Fatalf("GetInt() error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("GetInt() = %v, want %v", result, tt.expected)
				}
			case "bool":
				result, err := value.GetBool()
				if err != nil {
					t.Fatalf("GetBool() error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("GetBool() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

// Test convenience functions
func TestSearchConvenienceFunctions(t *testing.T) {
	testJSON := `{"name":"Bob","age":35,"height":5.9}`

	name, err := SearchString([]byte(testJSON), "name")
	if err != nil {
		t.Fatalf("SearchString() error = %v", err)
	}
	if name != "Bob" {
		t.Errorf("SearchString() = %v, want %v", name, "Bob")
	}

	age, err := SearchInt([]byte(testJSON), "age")
	if err != nil {
		t.Fatalf("SearchInt() error = %v", err)
	}
	if age != 35 {
		t.Errorf("SearchInt() = %v, want %v", age, 35)
	}

	height, err := SearchFloat([]byte(testJSON), "height")
	if err != nil {
		t.Fatalf("SearchFloat() error = %v", err)
	}
	if height != 5.9 {
		t.Errorf("SearchFloat() = %v, want %v", height, 5.9)
	}
}

// Test fast marshal path
func TestMarshalFast(t *testing.T) {
	demo := DemoStruct{ID: 123, Name: "FastTest"}

	result, err := MarshalFast(demo)
	if err != nil {
		t.Fatalf("MarshalFast() error = %v", err)
	}

	expected := `{"id":123,"name":"FastTest"}`
	if string(result) != expected {
		t.Errorf("MarshalFast() = %v, want %v", string(result), expected)
	}
}

// Benchmark comparisons: simdjson-inspired vs original
func BenchmarkTwoStageVsOriginal(b *testing.B) {
	largeJSON := generateLargeJSON(1000) // 1000 fields

	b.Run("TwoStage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := ParseFast(largeJSON)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Original", func(b *testing.B) {
		var result interface{}
		for i := 0; i < b.N; i++ {
			err := Unmarshal(largeJSON, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkZeroCopyVsSearch(b *testing.B) {
	testJSON := generateNestedJSON(5, 10) // 5 levels, 10 fields per level

	b.Run("ZeroCopy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := ZeroCopySearch(testJSON, "level5.level4.level3.level2")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Test data generators
func generateLargeJSON(numFields int) []byte {
	result := "{"
	for i := 0; i < numFields; i++ {
		if i > 0 {
			result += ","
		}
		result += `"field` + string(rune('0'+i%10)) + `":"value` + string(rune('0'+i%10)) + `"`
	}
	result += "}"
	return []byte(result)
}

func generateNestedJSON(levels, fieldsPerLevel int) []byte {
	if levels == 0 {
		return []byte(`"leaf_value"`)
	}

	result := "{"
	for i := 0; i < fieldsPerLevel; i++ {
		if i > 0 {
			result += ","
		}
		if i == fieldsPerLevel/2 { // Nest in the middle field
			result += `"level` + string(rune('0'+levels)) + `":` + string(generateNestedJSON(levels-1, fieldsPerLevel))
		} else {
			result += `"field` + string(rune('0'+i)) + `":"value` + string(rune('0'+i)) + `"`
		}
	}
	result += "}"
	return []byte(result)
}

// Test memory efficiency
func TestZeroCopyMemoryUsage(t *testing.T) {
	// Create a large valid string with repeated characters
	largeString := strings.Repeat("a", 1000)
	testJSON := `{"large_string":"` + largeString + `"}`

	// Measure original approach
	var originalResult interface{}
	start := time.Now()
	err := Unmarshal([]byte(testJSON), &originalResult)
	originalTime := time.Since(start)
	if err != nil {
		t.Fatalf("Original unmarshal failed: %v", err)
	}

	// Measure zero-copy approach
	start = time.Now()
	_, err = ZeroCopySearch([]byte(testJSON), "large_string")
	zeroCopyTime := time.Since(start)
	if err != nil {
		t.Fatalf("ZeroCopy search failed: %v", err)
	}

	t.Logf("Original: %v, ZeroCopy: %v, Speedup: %.2fx",
		originalTime, zeroCopyTime, float64(originalTime)/float64(zeroCopyTime))
}
