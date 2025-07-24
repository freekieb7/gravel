# Gravel JSON Package

## Overview

The Gravel JSON package provides a complete JSON marshaling implementation with reflection-based encoding. This package demonstrates modern Go JSON handling with comprehensive type support, proper string escaping, and flexible struct tag parsing.

## 🚀 **Features**

### **Complete Type Support**
- **Basic types**: `bool`, `int*`, `uint*`, `float*`, `string`
- **Complex types**: `arrays`, `slices`, `maps`, `structs`, `pointers`, `interfaces`
- **Nil handling**: Proper `null` encoding for nil pointers, slices, and maps
- **Custom marshaling**: Support for types implementing `Marshaler` interface

### **Advanced String Handling**
- **Proper JSON escaping**: Quotes, backslashes, control characters
- **Unicode support**: UTF-8 validation and replacement characters
- **Control character encoding**: `\u` notation for control chars < 0x20

### **Flexible Struct Tags**
- **Standard JSON tags**: `json:"fieldName"`
- **omitempty support**: `json:"field,omitempty"` skips zero values
- **Field exclusion**: `json:"-"` completely skips fields
- **Unexported field handling**: Automatically skips unexported fields

### **Performance Optimizations**
- **Efficient float formatting**: Uses `'g'` format with proper bit size
- **Memory-efficient**: Uses `bytes.Buffer` for optimal string building
- **Minimal allocations**: Direct reflection value handling

## 📋 **API Reference**

### **Core Functions**

```go
// Marshal encodes a Go value as JSON
func Marshal(v any) ([]byte, error)

// Marshaler interface for custom encoding
type Marshaler interface {
    Marshal() ([]byte, error)
}
```

### **Usage Examples**

#### **Basic Types**
```go
// Primitives
data, _ := json.Marshal(42)          // "42"
data, _ := json.Marshal(true)        // "true"
data, _ := json.Marshal("hello")     // "\"hello\""
data, _ := json.Marshal(3.14)        // "3.14"

// Nil values
var ptr *int
data, _ := json.Marshal(ptr)         // "null"
```

#### **Collections**
```go
// Arrays and slices
data, _ := json.Marshal([3]int{1, 2, 3})    // "[1,2,3]"
data, _ := json.Marshal([]string{"a", "b"}) // "[\"a\",\"b\"]"

// Maps
m := map[string]int{"x": 1, "y": 2}
data, _ := json.Marshal(m)           // "{\"x\":1,\"y\":2}"
```

#### **Structs with Tags**
```go
type Person struct {
    Name    string `json:"name"`
    Age     int    `json:"age"`
    Email   string `json:"email,omitempty"`
    private string // skipped (unexported)
    Skip    string `json:"-"`        // skipped (explicit)
}

person := Person{
    Name: "John",
    Age:  30,
    // Email is empty, will be omitted
}

data, _ := json.Marshal(person)
// {"name":"John","age":30}
```

#### **Custom Marshaling**
```go
type CustomType struct {
    Value string
}

func (c CustomType) Marshal() ([]byte, error) {
    return []byte(fmt.Sprintf(`"custom:%s"`, c.Value)), nil
}

data, _ := json.Marshal(CustomType{Value: "test"})
// "custom:test"
```

#### **String Escaping**
```go
text := "Line 1\nLine 2\t\"quoted\""
data, _ := json.Marshal(text)
// "Line 1\nLine 2\t\"quoted\""
```

## 🧪 **Testing**

The package includes comprehensive tests covering:

- **Basic type marshaling** with all Go primitives
- **String escaping** including Unicode and control characters  
- **Collection handling** for arrays, slices, and maps
- **Struct tag parsing** with omitempty and exclusion
- **Custom marshaler** interface implementation
- **Error handling** for unsupported types
- **Edge cases** like nil pointers and empty values

Run tests:
```bash
go test ./json/ -v
```

## 🚀 **Benchmarks**

Performance comparison with standard library:

```bash
go test ./json/ -bench=. -benchmem
```

### **Results Summary**
- **Arrays**: Competitive with stdlib (sometimes faster)
- **Simple structs**: ~3x slower but with more allocations
- **Complex structures**: ~3-4x slower due to reflection overhead
- **Maps**: Similar performance profile

The performance difference is expected as this is a reflection-based implementation optimized for clarity and functionality rather than maximum speed.

## 🔧 **Implementation Details**

### **Architecture**
- **Reflection-based**: Uses Go's `reflect` package for type inspection
- **Recursive marshaling**: Handles nested structures naturally
- **Buffer-based**: Uses `bytes.Buffer` for efficient string building
- **Interface support**: Handles `interface{}` by marshaling concrete values

### **Tag Parsing**
```go
// parseJSONTag handles various tag formats:
"fieldName"           -> fieldName, no omitempty
"fieldName,omitempty" -> fieldName, with omitempty  
",omitempty"          -> original field name, with omitempty
"-"                   -> field excluded
```

### **Empty Value Detection**
```go
// isEmptyValue checks for JSON omitempty semantics:
// - Empty strings, arrays, slices, maps
// - Zero numbers and false booleans  
// - Nil pointers and interfaces
```

## 🔄 **Future Improvements**

Potential enhancements for the JSON package:

1. **Decode support**: Add `Unmarshal` functionality
2. **Stream processing**: Support for large JSON documents
3. **Performance optimization**: Pool buffers and optimize reflection
4. **Additional tags**: Support for `string` tag and other options
5. **Validation**: JSON schema validation capabilities

## 📝 **Error Handling**

The package provides detailed error messages:

```go
// Unsupported types
gravel: Marshal with unsupported type chan int

// Non-string map keys  
gravel: Marshal map with non-string key type int
```

All errors are prefixed with "gravel:" for easy identification.

## 🎯 **Best Practices**

1. **Use struct tags**: Always provide JSON tags for public fields
2. **Handle custom types**: Implement `Marshaler` for complex custom types
3. **Check errors**: Always handle marshaling errors in production code
4. **Performance considerations**: For high-performance needs, consider stdlib
5. **Testing**: Test edge cases like nil pointers and empty slices

---

This JSON package demonstrates modern Go programming practices with comprehensive type handling, proper error handling, and extensive test coverage. It serves as both a functional JSON library and an example of reflection-based Go programming.
