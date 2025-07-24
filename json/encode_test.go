package json

import (
	"reflect"
	"testing"
)

func TestMarshal_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
		{"int8", int8(-128), "-128"},
		{"int16", int16(32767), "32767"},
		{"int32", int32(-2147483648), "-2147483648"},
		{"int64", int64(9223372036854775807), "9223372036854775807"},
		{"uint", uint(42), "42"},
		{"uint8", uint8(255), "255"},
		{"uint16", uint16(65535), "65535"},
		{"uint32", uint32(4294967295), "4294967295"},
		{"uint64", uint64(18446744073709551615), "18446744073709551615"},
		{"float32", float32(3.14), "3.14"},
		{"float64", 3.141592653589793, "3.141592653589793"},
		{"string", "hello", `"hello"`},
		{"empty string", "", `""`},
		{"nil pointer", (*int)(nil), "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Marshal() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestMarshal_StringEscaping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"quote", `"`, `"\""`},
		{"backslash", `\`, `"\\"`},
		{"newline", "\n", `"\n"`},
		{"tab", "\t", `"\t"`},
		{"carriage return", "\r", `"\r"`},
		{"backspace", "\b", `"\b"`},
		{"form feed", "\f", `"\f"`},
		{"control char", "\u0001", `"\u0001"`},
		{"unicode", "Hello 世界", `"Hello 世界"`},
		{"mixed", "line1\nline2\t\"quoted\"", `"line1\nline2\t\"quoted\""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Marshal() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestMarshal_Arrays(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"int array", [3]int{1, 2, 3}, "[1,2,3]"},
		{"empty array", [0]int{}, "[]"},
		{"string array", [2]string{"hello", "world"}, `["hello","world"]`},
		{"nested arrays", [2][2]int{{1, 2}, {3, 4}}, "[[1,2],[3,4]]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Marshal() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestMarshal_Slices(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"int slice", []int{1, 2, 3}, "[1,2,3]"},
		{"empty slice", []int{}, "[]"},
		{"nil slice", []int(nil), "null"},
		{"string slice", []string{"hello", "world"}, `["hello","world"]`},
		{"nested slices", [][]int{{1, 2}, {3, 4}}, "[[1,2],[3,4]]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Marshal() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestMarshal_Maps(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string // multiple valid JSON representations
	}{
		{
			"string map",
			map[string]int{"a": 1, "b": 2},
			[]string{`{"a":1,"b":2}`, `{"b":2,"a":1}`},
		},
		{
			"empty map",
			map[string]int{},
			[]string{"{}"},
		},
		{
			"nil map",
			map[string]int(nil),
			[]string{"null"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			resultStr := string(result)
			found := false
			for _, expected := range tt.expected {
				if resultStr == expected {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Marshal() = %q, want one of %v", resultStr, tt.expected)
			}
		})
	}
}

func TestMarshal_Structs(t *testing.T) {
	type Person struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email,omitempty"`
	}

	type StructWithoutTags struct {
		Name string
		Age  int
	}

	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			"basic struct",
			Person{Name: "John", Age: 30, Email: "john@example.com"},
			[]string{`{"name":"John","age":30,"email":"john@example.com"}`},
		},
		{
			"struct with omitempty",
			Person{Name: "Jane", Age: 25},
			[]string{`{"name":"Jane","age":25}`},
		},
		{
			"struct without tags",
			StructWithoutTags{Name: "Test", Age: 42},
			[]string{`{"Name":"Test","Age":42}`, `{"Age":42,"Name":"Test"}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			resultStr := string(result)
			found := false
			for _, expected := range tt.expected {
				if resultStr == expected {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Marshal() = %q, want one of %v", resultStr, tt.expected)
			}
		})
	}
}

func TestMarshal_StructSkipFields(t *testing.T) {
	type StructWithSkip struct {
		Public  string `json:"public"`
		private string
		Skip    string `json:"-"`
	}

	input := StructWithSkip{
		Public:  "visible",
		private: "hidden",
		Skip:    "ignored",
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	expected := `{"public":"visible"}`
	if string(result) != expected {
		t.Errorf("Marshal() = %q, want %q", string(result), expected)
	}
}

func TestMarshal_Pointers(t *testing.T) {
	name := "John"
	age := 30

	type Person struct {
		Name *string `json:"name,omitempty"`
		Age  *int    `json:"age,omitempty"`
	}

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			"non-nil pointers",
			Person{Name: &name, Age: &age},
			`{"name":"John","age":30}`,
		},
		{
			"nil pointers with omitempty",
			Person{},
			`{}`,
		},
		{
			"mixed pointers",
			Person{Name: &name},
			`{"name":"John"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Marshal() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

func TestMarshal_CustomMarshaler(t *testing.T) {
	// For this test, we'll create a custom type that implements Marshal
	customInput := customMarshalType{Value: "test"}
	result, err := Marshal(customInput)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	expected := `"custom:test"`
	if string(result) != expected {
		t.Errorf("Marshal() = %q, want %q", string(result), expected)
	}
}

// customMarshalType implements the Marshaler interface for testing
type customMarshalType struct {
	Value string
}

func (c customMarshalType) Marshal() ([]byte, error) {
	return []byte(`"custom:` + c.Value + `"`), nil
}

func TestMarshal_Errors(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			"non-string map key",
			map[int]string{1: "one"},
			true,
		},
		{
			"unsupported type",
			make(chan int),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseJSONTag(t *testing.T) {
	tests := []struct {
		tag        string
		fieldName  string
		expectName string
		expectOmit bool
	}{
		{"", "FieldName", "FieldName", false},
		{"customName", "FieldName", "customName", false},
		{"customName,omitempty", "FieldName", "customName", true},
		{",omitempty", "FieldName", "FieldName", true},
		{"name,omitempty,other", "FieldName", "name", true},
		{"-", "FieldName", "-", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			name, omit := parseJSONTag(tt.tag, tt.fieldName)
			if name != tt.expectName {
				t.Errorf("parseJSONTag() name = %q, want %q", name, tt.expectName)
			}
			if omit != tt.expectOmit {
				t.Errorf("parseJSONTag() omit = %v, want %v", omit, tt.expectOmit)
			}
		})
	}
}

func TestIsEmptyValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1}, false},
		{"nil slice", []int(nil), true},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
		{"nil map", map[string]int(nil), true},
		{"nil pointer", (*int)(nil), true},
		{"non-nil pointer", func() *int { i := 42; return &i }(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := reflect.ValueOf(tt.value)
			result := isEmptyValue(rv)
			if result != tt.expected {
				t.Errorf("isEmptyValue(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
