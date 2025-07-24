package json

import (
	stdjson "encoding/json"
	"testing"
)

// Simple test structures for benchmarking
type SimplePerson struct {
	Name   string  `json:"name"`
	Age    int     `json:"age"`
	Active bool    `json:"active"`
	Score  float64 `json:"score"`
}

type BenchmarkData struct {
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Email   string   `json:"email"`
	Active  bool     `json:"active"`
	Score   float64  `json:"score"`
	Tags    []string `json:"tags"`
	Address *Address `json:"address"`
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
	ZipCode string `json:"zip_code"`
}

// Additional types for large benchmark tests
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Settings struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
	Timeout  int    `json:"timeout"`
}

type LargeData struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Active   bool      `json:"active"`
	Score    float64   `json:"score"`
	Tags     []string  `json:"tags"`
	Items    []Item    `json:"items"`
	Settings *Settings `json:"settings"`
}

func createBenchmarkData() BenchmarkData {
	return BenchmarkData{
		Name:   "John Doe",
		Age:    30,
		Email:  "john@example.com",
		Active: true,
		Score:  95.5,
		Tags:   []string{"developer", "golang", "json"},
		Address: &Address{
			Street:  "123 Main St",
			City:    "San Francisco",
			Country: "USA",
			ZipCode: "94102",
		},
	}
}

func BenchmarkGravel_Marshal(b *testing.B) {
	data := createBenchmarkData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Marshal(b *testing.B) {
	data := createBenchmarkData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := stdjson.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Marshal_Simple(b *testing.B) {
	data := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		Name: "John",
		Age:  30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Marshal_Simple(b *testing.B) {
	data := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		Name: "John",
		Age:  30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := stdjson.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Marshal_Array(b *testing.B) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Marshal_Array(b *testing.B) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := stdjson.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Marshal_Map(b *testing.B) {
	data := map[string]interface{}{
		"name":   "John",
		"age":    30,
		"active": true,
		"score":  95.5,
		"tags":   []string{"a", "b", "c"},
		"nested": map[string]int{"x": 1, "y": 2},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Marshal_Map(b *testing.B) {
	data := map[string]interface{}{
		"name":   "John",
		"age":    30,
		"active": true,
		"score":  95.5,
		"tags":   []string{"a", "b", "c"},
		"nested": map[string]int{"x": 1, "y": 2},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := stdjson.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Unmarshal Benchmarks

func BenchmarkGravel_Unmarshal(b *testing.B) {
	data := createBenchmarkData()
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchmarkData
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal(b *testing.B) {
	data := createBenchmarkData()
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchmarkData
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Unmarshal_Simple(b *testing.B) {
	data := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		Name: "John",
		Age:  30,
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal_Simple(b *testing.B) {
	data := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		Name: "John",
		Age:  30,
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Unmarshal_Array(b *testing.B) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result []int
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal_Array(b *testing.B) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result []int
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Unmarshal_Map(b *testing.B) {
	data := map[string]interface{}{
		"name":   "John",
		"age":    30,
		"active": true,
		"score":  95.5,
		"tags":   []string{"a", "b", "c"},
		"nested": map[string]int{"x": 1, "y": 2},
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal_Map(b *testing.B) {
	data := map[string]interface{}{
		"name":   "John",
		"age":    30,
		"active": true,
		"score":  95.5,
		"tags":   []string{"a", "b", "c"},
		"nested": map[string]int{"x": 1, "y": 2},
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Unmarshal_Large(b *testing.B) {
	// Create a larger dataset for stress testing
	largeData := LargeData{
		ID:     12345,
		Name:   "Large Test Dataset",
		Email:  "test@example.com",
		Active: true,
		Score:  98.7,
		Tags:   []string{"performance", "benchmark", "json", "golang", "testing"},
		Items: []Item{
			{ID: 1, Name: "Item One", Value: "value1"},
			{ID: 2, Name: "Item Two", Value: "value2"},
			{ID: 3, Name: "Item Three", Value: "value3"},
			{ID: 4, Name: "Item Four", Value: "value4"},
			{ID: 5, Name: "Item Five", Value: "value5"},
		},
		Settings: &Settings{
			Theme:    "dark",
			Language: "en-US",
			Timeout:  30,
		},
	}

	jsonData, _ := stdjson.Marshal(largeData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result LargeData
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal_Large(b *testing.B) {
	// Same large dataset structure
	largeData := LargeData{
		ID:     12345,
		Name:   "Large Test Dataset",
		Email:  "test@example.com",
		Active: true,
		Score:  98.7,
		Tags:   []string{"performance", "benchmark", "json", "golang", "testing"},
		Items: []Item{
			{ID: 1, Name: "Item One", Value: "value1"},
			{ID: 2, Name: "Item Two", Value: "value2"},
			{ID: 3, Name: "Item Three", Value: "value3"},
			{ID: 4, Name: "Item Four", Value: "value4"},
			{ID: 5, Name: "Item Five", Value: "value5"},
		},
		Settings: &Settings{
			Theme:    "dark",
			Language: "en-US",
			Timeout:  30,
		},
	}

	jsonData, _ := stdjson.Marshal(largeData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result LargeData
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Additional focused unmarshal benchmarks for different data patterns

func BenchmarkGravel_Unmarshal_Numbers(b *testing.B) {
	data := struct {
		Int     int     `json:"int"`
		Float   float64 `json:"float"`
		Uint    uint    `json:"uint"`
		Int64   int64   `json:"int64"`
		Float32 float32 `json:"float32"`
	}{
		Int:     42,
		Float:   3.14159,
		Uint:    123,
		Int64:   9223372036854775807,
		Float32: 2.718,
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			Int     int     `json:"int"`
			Float   float64 `json:"float"`
			Uint    uint    `json:"uint"`
			Int64   int64   `json:"int64"`
			Float32 float32 `json:"float32"`
		}
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal_Numbers(b *testing.B) {
	data := struct {
		Int     int     `json:"int"`
		Float   float64 `json:"float"`
		Uint    uint    `json:"uint"`
		Int64   int64   `json:"int64"`
		Float32 float32 `json:"float32"`
	}{
		Int:     42,
		Float:   3.14159,
		Uint:    123,
		Int64:   9223372036854775807,
		Float32: 2.718,
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			Int     int     `json:"int"`
			Float   float64 `json:"float"`
			Uint    uint    `json:"uint"`
			Int64   int64   `json:"int64"`
			Float32 float32 `json:"float32"`
		}
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Unmarshal_Strings(b *testing.B) {
	data := struct {
		Short  string `json:"short"`
		Medium string `json:"medium"`
		Long   string `json:"long"`
		Empty  string `json:"empty"`
	}{
		Short:  "test",
		Medium: "This is a medium length string for testing",
		Long:   "This is a much longer string that contains more content to test the performance of string parsing and allocation behavior during JSON unmarshaling operations",
		Empty:  "",
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			Short  string `json:"short"`
			Medium string `json:"medium"`
			Long   string `json:"long"`
			Empty  string `json:"empty"`
		}
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal_Strings(b *testing.B) {
	data := struct {
		Short  string `json:"short"`
		Medium string `json:"medium"`
		Long   string `json:"long"`
		Empty  string `json:"empty"`
	}{
		Short:  "test",
		Medium: "This is a medium length string for testing",
		Long:   "This is a much longer string that contains more content to test the performance of string parsing and allocation behavior during JSON unmarshaling operations",
		Empty:  "",
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			Short  string `json:"short"`
			Medium string `json:"medium"`
			Long   string `json:"long"`
			Empty  string `json:"empty"`
		}
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Unmarshal_Booleans(b *testing.B) {
	data := struct {
		True    bool `json:"true"`
		False   bool `json:"false"`
		Default bool `json:"default"`
	}{
		True:    true,
		False:   false,
		Default: false,
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			True    bool `json:"true"`
			False   bool `json:"false"`
			Default bool `json:"default"`
		}
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal_Booleans(b *testing.B) {
	data := struct {
		True    bool `json:"true"`
		False   bool `json:"false"`
		Default bool `json:"default"`
	}{
		True:    true,
		False:   false,
		Default: false,
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			True    bool `json:"true"`
			False   bool `json:"false"`
			Default bool `json:"default"`
		}
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_Unmarshal_NestedStruct(b *testing.B) {
	data := struct {
		User struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		} `json:"user"`
		Active bool `json:"active"`
	}{
		User: struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{
			Name: "John Doe",
			Age:  30,
		},
		Active: true,
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			User struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			} `json:"user"`
			Active bool `json:"active"`
		}
		err := Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlib_Unmarshal_NestedStruct(b *testing.B) {
	data := struct {
		User struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		} `json:"user"`
		Active bool `json:"active"`
	}{
		User: struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{
			Name: "John Doe",
			Age:  30,
		},
		Active: true,
	}
	jsonData, _ := stdjson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result struct {
			User struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			} `json:"user"`
			Active bool `json:"active"`
		}
		err := stdjson.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Zero-allocation marshal benchmarks
func BenchmarkGravel_MarshalAppend_Simple(b *testing.B) {
	data := SimplePerson{
		Name: "John Doe",
		Age:  30,
	}
	buf := make([]byte, 0, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalAppend(buf[:0], data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_MarshalTo_Simple(b *testing.B) {
	data := SimplePerson{
		Name: "John Doe",
		Age:  30,
	}
	buf := make([]byte, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalTo(data, buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_MarshalAppend_Array(b *testing.B) {
	data := []SimplePerson{
		{Name: "John", Age: 30},
		{Name: "Jane", Age: 25},
		{Name: "Bob", Age: 35},
	}
	buf := make([]byte, 0, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalAppend(buf[:0], data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGravel_MarshalTo_Array(b *testing.B) {
	data := []SimplePerson{
		{Name: "John", Age: 30},
		{Name: "Jane", Age: 25},
		{Name: "Bob", Age: 35},
	}
	buf := make([]byte, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalTo(data, buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}
