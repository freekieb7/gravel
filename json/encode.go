package json

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
	"unsafe"
)

// Fast path type encoders cache
type typeEncoder func(reflect.Value, writer) error

var (
	encoderCache sync.Map // map[reflect.Type]typeEncoder
	stringType   = reflect.TypeOf("")
	intType      = reflect.TypeOf(int(0))
	int64Type    = reflect.TypeOf(int64(0))
	float64Type  = reflect.TypeOf(float64(0))
	boolType     = reflect.TypeOf(true)

	// Buffer pool for Marshal function to reduce allocations
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 256)
		},
	}

	// Struct field metadata cache
	structCache sync.Map // map[reflect.Type]*structMeta

	// Small integer lookup table for common values (stdlib-style optimization)
	smallInts = [...]string{
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"10", "11", "12", "13", "14", "15", "16", "17", "18", "19",
		"20", "21", "22", "23", "24", "25", "26", "27", "28", "29",
		"30", "31", "32", "33", "34", "35", "36", "37", "38", "39",
		"40", "41", "42", "43", "44", "45", "46", "47", "48", "49",
		"50", "51", "52", "53", "54", "55", "56", "57", "58", "59",
		"60", "61", "62", "63", "64", "65", "66", "67", "68", "69",
		"70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
		"80", "81", "82", "83", "84", "85", "86", "87", "88", "89",
		"90", "91", "92", "93", "94", "95", "96", "97", "98", "99",
	}

	// Extended lookup for powers of 10 and common values (medium effort optimization)
	powersOf10 = [...]string{
		"1", "10", "100", "1000", "10000", "100000", "1000000",
		"10000000", "100000000", "1000000000", "10000000000",
	}

	// Two-digit lookup table for faster multi-digit processing
	twoDigits = [...]string{
		"00", "01", "02", "03", "04", "05", "06", "07", "08", "09",
		"10", "11", "12", "13", "14", "15", "16", "17", "18", "19",
		"20", "21", "22", "23", "24", "25", "26", "27", "28", "29",
		"30", "31", "32", "33", "34", "35", "36", "37", "38", "39",
		"40", "41", "42", "43", "44", "45", "46", "47", "48", "49",
		"50", "51", "52", "53", "54", "55", "56", "57", "58", "59",
		"60", "61", "62", "63", "64", "65", "66", "67", "68", "69",
		"70", "71", "72", "73", "74", "75", "76", "77", "78", "79",
		"80", "81", "82", "83", "84", "85", "86", "87", "88", "89",
		"90", "91", "92", "93", "94", "95", "96", "97", "98", "99",
	}
)

// Cached struct metadata
type structMeta struct {
	fields []fieldMeta
}

type fieldMeta struct {
	index     int
	name      string
	omitEmpty bool
	skip      bool
}

// getStructMeta gets or builds cached struct metadata
func getStructMeta(t reflect.Type) *structMeta {
	if cached, ok := structCache.Load(t); ok {
		return cached.(*structMeta)
	}

	meta := buildStructMeta(t)
	structCache.Store(t, meta)
	return meta
}

func buildStructMeta(t reflect.Type) *structMeta {
	numFields := t.NumField()
	fields := make([]fieldMeta, 0, numFields)

	for i := 0; i < numFields; i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "-" {
			continue // Skip fields marked with json:"-"
		}

		fieldName, omitEmpty := parseJSONTag(tag, field.Name)

		fields = append(fields, fieldMeta{
			index:     i,
			name:      fieldName,
			omitEmpty: omitEmpty,
			skip:      false,
		})
	}

	return &structMeta{fields: fields}
}

// Fast integer conversion using lookup tables (medium effort optimization)
func fastFormatInt(i int64, w writer) error {
	// Handle negative numbers
	if i < 0 {
		if err := w.WriteByte('-'); err != nil {
			return err
		}
		i = -i
	}

	return fastFormatUint(uint64(i), w)
}

func fastFormatUint(u uint64, w writer) error {
	// Fast path for single digits and small numbers
	if u < uint64(len(smallInts)) {
		return w.WriteString(smallInts[u])
	}

	// Fast path for two-digit numbers (100-999)
	if u < 1000 {
		if u < 100 {
			return w.WriteString(twoDigits[u])
		}
		// Three digits: write first digit + two-digit lookup
		if err := w.WriteByte(byte('0' + u/100)); err != nil {
			return err
		}
		return w.WriteString(twoDigits[u%100])
	}

	// For larger numbers, use optimized conversion
	if u < 10000 {
		return fastFormatUint4(u, w)
	}

	// Fall back to stdlib for very large numbers
	return w.WriteString(strconv.FormatUint(u, 10))
}

// Optimized 4-digit conversion
func fastFormatUint4(u uint64, w writer) error {
	thousands := u / 1000
	remainder := u % 1000

	if err := w.WriteByte(byte('0' + thousands)); err != nil {
		return err
	}

	if remainder < 100 {
		if err := w.WriteByte('0'); err != nil {
			return err
		}
		return w.WriteString(twoDigits[remainder])
	}

	hundreds := remainder / 100
	if err := w.WriteByte(byte('0' + hundreds)); err != nil {
		return err
	}
	return w.WriteString(twoDigits[remainder%100])
}

type Marshaler interface {
	Marshal() ([]byte, error)
}

func Marshal(v any) ([]byte, error) {
	// Try fast path for large structs/objects
	if shouldUseFastPath(v) {
		return MarshalFast(v)
	}

	// Use pooled buffer to reduce allocations
	buf := bufferPool.Get().([]byte)
	defer func() {
		// Clear buffer and return to pool
		if cap(buf) < 4096 { // Don't pool extremely large buffers
			bufferPool.Put(buf[:0])
		}
	}()

	w := &sliceWriter{buf: buf}
	err := marshalValue(reflect.ValueOf(v), w)
	if err != nil {
		return nil, err
	}

	// Return a copy since we're returning the buffer to pool
	result := make([]byte, len(w.buf))
	copy(result, w.buf)
	return result, nil
}

// MarshalFast uses the two-stage parser approach for better performance
func MarshalFast(v any) ([]byte, error) {
	// Check if we have a compile-time encoder first
	rv := reflect.ValueOf(v)
	if encoder, exists := GetFastEncoder(rv.Type()); exists {
		buf := bufferPool.Get().([]byte)
		defer func() {
			if cap(buf) < 4096 {
				bufferPool.Put(buf[:0])
			}
		}()

		w := &sliceWriter{buf: buf}
		err := encoder(rv, w)
		if err != nil {
			return nil, err
		}

		result := make([]byte, len(w.buf))
		copy(result, w.buf)
		return result, nil
	}

	// Fall back to regular marshal
	buf := bufferPool.Get().([]byte)
	defer func() {
		if cap(buf) < 4096 {
			bufferPool.Put(buf[:0])
		}
	}()

	w := &sliceWriter{buf: buf}
	err := marshalValue(reflect.ValueOf(v), w)
	if err != nil {
		return nil, err
	}

	result := make([]byte, len(w.buf))
	copy(result, w.buf)
	return result, nil
}

// shouldUseFastPath determines if we should use the fast parser
func shouldUseFastPath(v any) bool {
	rv := reflect.ValueOf(v)

	// Use fast path for structs that have compile-time encoders
	if _, exists := GetFastEncoder(rv.Type()); exists {
		return true
	}

	// Use fast path for large structs (>10 fields)
	if rv.Kind() == reflect.Struct && rv.NumField() > 10 {
		return true
	}

	return false
}

// MarshalAppend appends the JSON representation of v to buf and returns the extended buffer.
// This is a zero-allocation alternative to Marshal when you can reuse buffers.
func MarshalAppend(buf []byte, v any) ([]byte, error) {
	w := &sliceWriter{buf: buf}
	err := marshalValue(reflect.ValueOf(v), w)
	return w.buf, err
}

// MarshalTo writes the JSON representation of v to the provided buffer.
// Returns the number of bytes written and any error.
// This is truly zero-allocation but requires pre-sized buffer.
func MarshalTo(v any, buf []byte) (int, error) {
	w := &fixedWriter{buf: buf, pos: 0}
	err := marshalValue(reflect.ValueOf(v), w)
	return w.pos, err
}

// sliceWriter grows the slice as needed (may allocate for growth)
type sliceWriter struct {
	buf []byte
}

func (w *sliceWriter) WriteByte(b byte) error {
	w.buf = append(w.buf, b)
	return nil
}

func (w *sliceWriter) WriteString(s string) error {
	w.buf = append(w.buf, s...)
	return nil
}

func (w *sliceWriter) WriteBytes(data []byte) error {
	w.buf = append(w.buf, data...)
	return nil
}

// fixedWriter writes to a fixed buffer (true zero allocation)
type fixedWriter struct {
	buf []byte
	pos int
}

func (w *fixedWriter) WriteByte(b byte) error {
	if w.pos >= len(w.buf) {
		return errors.New("buffer overflow")
	}
	w.buf[w.pos] = b
	w.pos++
	return nil
}

func (w *fixedWriter) WriteString(s string) error {
	if w.pos+len(s) > len(w.buf) {
		return errors.New("buffer overflow")
	}
	copy(w.buf[w.pos:], s)
	w.pos += len(s)
	return nil
}

func (w *fixedWriter) WriteBytes(data []byte) error {
	if w.pos+len(data) > len(w.buf) {
		return errors.New("buffer overflow")
	}
	copy(w.buf[w.pos:], data)
	w.pos += len(data)
	return nil
}

// Writer interface for zero-allocation marshaling (exported for codegen)
type Writer interface {
	WriteByte(byte) error
	WriteString(string) error
	WriteBytes([]byte) error
}

// writer interface for zero-allocation marshaling (internal)
type writer interface {
	WriteByte(byte) error
	WriteString(string) error
	WriteBytes([]byte) error
}

// marshalValue is the zero-allocation version of marshal
func marshalValue(rv reflect.Value, w writer) error {
	// Check for compile-time generated fast encoders first
	if encoder, exists := GetFastEncoder(rv.Type()); exists {
		return encoder(rv, w)
	}

	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			return w.WriteString("true")
		}
		return w.WriteString("false")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := rv.Int()
		// Inline fast path for negative numbers
		if i < 0 {
			if err := w.WriteByte('-'); err != nil {
				return err
			}
			i = -i
			// Fast path for small negative integers
			if i < int64(len(smallInts)) {
				return w.WriteString(smallInts[i])
			}
			// Fast path for two-digit negative numbers
			if i < 100 {
				return w.WriteString(twoDigits[i])
			}
			// Use stdlib for larger negative values
			return w.WriteString(strconv.FormatInt(i, 10))
		}
		// Fast path for small positive integers using lookup table
		if i < int64(len(smallInts)) {
			return w.WriteString(smallInts[i])
		}
		// Fast path for two-digit positive numbers
		if i < 100 {
			return w.WriteString(twoDigits[i])
		}
		// Use stdlib for larger positive values
		return w.WriteString(strconv.FormatInt(i, 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u := rv.Uint()
		// Fast path for small integers using lookup table
		if u < uint64(len(smallInts)) {
			return w.WriteString(smallInts[u])
		}
		// Fast path for two-digit numbers
		if u < 100 {
			return w.WriteString(twoDigits[u])
		}
		// Use stdlib for larger values
		return w.WriteString(strconv.FormatUint(u, 10))
	case reflect.Float32, reflect.Float64:
		f := rv.Float()

		// Fast path for common float values
		if f == 0 {
			return w.WriteByte('0')
		}
		if f == 1 {
			return w.WriteByte('1')
		}
		if f == -1 {
			return w.WriteString("-1")
		}

		// For other values, use strconv (still optimal for zero-alloc)
		bitSize := 64
		if rv.Kind() == reflect.Float32 {
			bitSize = 32
		}
		return w.WriteString(strconv.FormatFloat(f, 'g', -1, bitSize))
	case reflect.String:
		if err := w.WriteByte('"'); err != nil {
			return err
		}
		s := rv.String()
		// Fast path for simple strings without special characters using unsafe operations
		if isSimpleString(s) {
			if err := writeKnownSafeString(s, w); err != nil {
				return err
			}
		} else {
			if err := escapeStringWriter(s, w); err != nil {
				return err
			}
		}
		return w.WriteByte('"')
	case reflect.Pointer:
		if rv.IsNil() {
			return w.WriteString("null")
		}

		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				return w.WriteBytes(bytes)
			}
		}

		return marshalValue(rv.Elem(), w)
	case reflect.Array:
		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				return w.WriteBytes(bytes)
			}
		}

		if err := w.WriteByte('['); err != nil {
			return err
		}
		n := rv.Len()
		for i := 0; i < n; i++ {
			if i > 0 {
				if err := w.WriteByte(','); err != nil {
					return err
				}
			}
			if err := marshalValue(rv.Index(i), w); err != nil {
				return err
			}
		}
		return w.WriteByte(']')
	case reflect.Slice:
		if rv.IsNil() {
			return w.WriteString("null")
		}

		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				return w.WriteBytes(bytes)
			}
		}

		if err := w.WriteByte('['); err != nil {
			return err
		}
		n := rv.Len()
		for i := 0; i < n; i++ {
			if i > 0 {
				if err := w.WriteByte(','); err != nil {
					return err
				}
			}
			if err := marshalValue(rv.Index(i), w); err != nil {
				return err
			}
		}
		return w.WriteByte(']')
	case reflect.Map:
		if rv.IsNil() {
			return w.WriteString("null")
		}

		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				return w.WriteBytes(bytes)
			}
		}

		// Only support string keys
		if rv.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("json: unsupported type: map with non-string key")
		}

		if err := w.WriteByte('{'); err != nil {
			return err
		}

		keys := rv.MapKeys()
		first := true
		for _, key := range keys {
			if !first {
				if err := w.WriteByte(','); err != nil {
					return err
				}
			}
			first = false

			// Marshal key
			if err := w.WriteByte('"'); err != nil {
				return err
			}
			if err := escapeStringWriter(key.String(), w); err != nil {
				return err
			}
			if err := w.WriteByte('"'); err != nil {
				return err
			}

			if err := w.WriteByte(':'); err != nil {
				return err
			}

			// Marshal value
			if err := marshalValue(rv.MapIndex(key), w); err != nil {
				return err
			}
		}

		return w.WriteByte('}')
	case reflect.Struct:
		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				return w.WriteBytes(bytes)
			}
		}

		if err := w.WriteByte('{'); err != nil {
			return err
		}

		// Use cached struct metadata
		meta := getStructMeta(rv.Type())

		addComma := false
		for _, fieldMeta := range meta.fields {
			fv := rv.Field(fieldMeta.index)

			// Handle omitempty
			if fieldMeta.omitEmpty && isEmptyValue(fv) {
				continue
			}

			if addComma {
				if err := w.WriteByte(','); err != nil {
					return err
				}
			}
			addComma = true

			if err := w.WriteByte('"'); err != nil {
				return err
			}
			// For simple field names, use unsafe operations to avoid escaping overhead
			if isSimpleFieldName(fieldMeta.name) {
				if err := writeKnownSafeString(fieldMeta.name, w); err != nil {
					return err
				}
			} else {
				if err := escapeStringWriter(fieldMeta.name, w); err != nil {
					return err
				}
			}
			if err := w.WriteByte('"'); err != nil {
				return err
			}
			if err := w.WriteByte(':'); err != nil {
				return err
			}

			if err := marshalValue(fv, w); err != nil {
				return err
			}
		}
		return w.WriteByte('}')
	case reflect.Interface:
		if rv.IsNil() {
			return w.WriteString("null")
		}
		// Marshal the concrete value
		return marshalValue(rv.Elem(), w)
	default:
		return fmt.Errorf("gravel: Marshal with unsupported type %s", rv.Kind().String())
	}
}

// Zero-allocation integer writing
func writeIntValue(i int64, w writer) error {
	if i == 0 {
		return w.WriteByte('0')
	}

	if i < 0 {
		if err := w.WriteByte('-'); err != nil {
			return err
		}
		i = -i
	}

	return writeUintValue(uint64(i), w)
}

func writeUintValue(u uint64, w writer) error {
	if u == 0 {
		return w.WriteByte('0')
	}

	// Calculate digits in reverse order
	var digits [20]byte // max uint64 has 20 digits
	n := 0
	for u > 0 {
		digits[n] = byte('0' + u%10)
		u /= 10
		n++
	}

	// Write digits in correct order
	for i := n - 1; i >= 0; i-- {
		if err := w.WriteByte(digits[i]); err != nil {
			return err
		}
	}
	return nil
}

// Zero-allocation string escaping
func escapeStringWriter(s string, w writer) error {
	// Use SIMD-optimized scanning for better performance
	return escapeStringOptimized(s, w)
}

// SIMD-optimized string escaping
func escapeStringOptimized(s string, w writer) error {
	if len(s) == 0 {
		return nil
	}

	// Convert string to byte slice for SIMD processing
	data := unsafeStringToBytes(s)
	start := 0

	for start < len(s) {
		// Use SIMD to find next character that needs escaping
		remaining := data[start:]
		pos := scanForEscapeCharsSIMD(remaining)

		if pos == len(remaining) {
			// No more escaping needed, write the rest
			return w.WriteString(s[start:])
		}

		// Write unescaped portion
		if pos > 0 {
			if err := w.WriteString(s[start : start+pos]); err != nil {
				return err
			}
		}

		// Handle the character that needs escaping
		escapePos := start + pos
		b := s[escapePos]

		switch b {
		case '"':
			if err := w.WriteString(`\"`); err != nil {
				return err
			}
		case '\\':
			if err := w.WriteString(`\\`); err != nil {
				return err
			}
		case '\b':
			if err := w.WriteString(`\b`); err != nil {
				return err
			}
		case '\f':
			if err := w.WriteString(`\f`); err != nil {
				return err
			}
		case '\n':
			if err := w.WriteString(`\n`); err != nil {
				return err
			}
		case '\r':
			if err := w.WriteString(`\r`); err != nil {
				return err
			}
		case '\t':
			if err := w.WriteString(`\t`); err != nil {
				return err
			}
		default:
			if b < 0x20 {
				// Write \u0000 format for control characters
				if err := w.WriteString(`\u`); err != nil {
					return err
				}
				if err := writeHex4Writer(w, uint16(b)); err != nil {
					return err
				}
			} else {
				// Multi-byte UTF-8 character - write as-is
				if err := w.WriteByte(b); err != nil {
					return err
				}
			}
		}

		start = escapePos + 1
	}

	return nil
}

// Original escapeStringWriter for fallback
func escapeStringWriter_original(s string, w writer) error {
	// Fast path: check if string needs any escaping at all
	start := 0
	for i := 0; i < len(s); i++ {
		b := s[i]

		// Check for characters that need escaping or are non-ASCII
		if b < 0x20 || b >= 0x80 || b == '"' || b == '\\' {
			// Write any unescaped portion before this character
			if i > start {
				if err := w.WriteString(s[start:i]); err != nil {
					return err
				}
			}

			// Handle the special character
			switch b {
			case '"':
				if err := w.WriteString(`\"`); err != nil {
					return err
				}
			case '\\':
				if err := w.WriteString(`\\`); err != nil {
					return err
				}
			case '\b':
				if err := w.WriteString(`\b`); err != nil {
					return err
				}
			case '\f':
				if err := w.WriteString(`\f`); err != nil {
					return err
				}
			case '\n':
				if err := w.WriteString(`\n`); err != nil {
					return err
				}
			case '\r':
				if err := w.WriteString(`\r`); err != nil {
					return err
				}
			case '\t':
				if err := w.WriteString(`\t`); err != nil {
					return err
				}
			default:
				if b < 0x20 {
					// Write \u0000 format for control characters
					if err := w.WriteString(`\u`); err != nil {
						return err
					}
					if err := writeHex4Writer(w, uint16(b)); err != nil {
						return err
					}
				} else if b < 0x80 {
					// ASCII character, write directly (shouldn't happen due to check above)
					if err := w.WriteByte(b); err != nil {
						return err
					}
				} else {
					// Multi-byte UTF-8 character - handle more carefully
					// Decode rune starting at position i
					if i+1 < len(s) && (b&0xE0) == 0xC0 {
						// 2-byte sequence
						if err := w.WriteByte(b); err != nil {
							return err
						}
						i++
						if err := w.WriteByte(s[i]); err != nil {
							return err
						}
					} else if i+2 < len(s) && (b&0xF0) == 0xE0 {
						// 3-byte sequence
						if err := w.WriteByte(b); err != nil {
							return err
						}
						i++
						if err := w.WriteByte(s[i]); err != nil {
							return err
						}
						i++
						if err := w.WriteByte(s[i]); err != nil {
							return err
						}
					} else if i+3 < len(s) && (b&0xF8) == 0xF0 {
						// 4-byte sequence
						if err := w.WriteByte(b); err != nil {
							return err
						}
						i++
						if err := w.WriteByte(s[i]); err != nil {
							return err
						}
						i++
						if err := w.WriteByte(s[i]); err != nil {
							return err
						}
						i++
						if err := w.WriteByte(s[i]); err != nil {
							return err
						}
					} else {
						// Invalid UTF-8, write replacement character
						if err := w.WriteBytes([]byte{0xEF, 0xBF, 0xBD}); err != nil {
							return err
						}
					}
				}
			}
			start = i + 1
		}
	}

	// Write any remaining unescaped portion
	if start < len(s) {
		return w.WriteString(s[start:])
	}

	return nil
}

func writeHex4Writer(w writer, val uint16) error {
	const hexDigits = "0123456789abcdef"
	if err := w.WriteByte(hexDigits[(val>>12)&0xF]); err != nil {
		return err
	}
	if err := w.WriteByte(hexDigits[(val>>8)&0xF]); err != nil {
		return err
	}
	if err := w.WriteByte(hexDigits[(val>>4)&0xF]); err != nil {
		return err
	}
	return w.WriteByte(hexDigits[val&0xF])
}

func marshal(rv reflect.Value, buf *bytes.Buffer) error {
	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		writeInt(rv.Int(), buf)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		writeUint(rv.Uint(), buf)
		return nil
	case reflect.Float32, reflect.Float64:
		f := rv.Float()

		// Fast path for common float values
		if f == 0 {
			buf.WriteByte('0')
			return nil
		}
		if f == 1 {
			buf.WriteByte('1')
			return nil
		}
		if f == -1 {
			buf.WriteString("-1")
			return nil
		}

		// For other values, use strconv (still needs optimization)
		bitSize := 64
		if rv.Kind() == reflect.Float32 {
			bitSize = 32
		}
		buf.WriteString(strconv.FormatFloat(f, 'g', -1, bitSize))
		return nil
	case reflect.String:
		buf.WriteByte('"')
		escapeString(rv.String(), buf)
		buf.WriteByte('"')
		return nil
	case reflect.Pointer:
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}

		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				buf.Write(bytes)
				return nil
			}
		}

		return marshal(rv.Elem(), buf)
	case reflect.Array:
		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				buf.Write(bytes)
				return nil
			}
		}

		buf.WriteByte('[')
		n := rv.Len()
		for i := 0; i < n; i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := marshal(rv.Index(i), buf); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	case reflect.Slice:
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}

		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				buf.Write(bytes)
				return nil
			}
		}

		buf.WriteByte('[')
		n := rv.Len()
		for i := 0; i < n; i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := marshal(rv.Index(i), buf); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	case reflect.Map:
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}

		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				buf.Write(bytes)
				return nil
			}
		}

		// Only support string keys for JSON
		if rv.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("gravel: Marshal map with non-string key type %s", rv.Type().Key().String())
		}

		buf.WriteByte('{')
		keys := rv.MapKeys()
		addComma := false

		for _, key := range keys {
			value := rv.MapIndex(key)

			if addComma {
				buf.WriteByte(',')
			}
			addComma = true

			buf.WriteByte('"')
			escapeString(key.String(), buf)
			buf.WriteByte('"')
			buf.WriteByte(':')

			if err := marshal(value, buf); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	case reflect.Struct:
		if rv.Type().NumMethod() > 0 && rv.CanInterface() {
			if u, ok := rv.Interface().(Marshaler); ok {
				bytes, err := u.Marshal()
				if err != nil {
					return err
				}

				if len(bytes) == 0 {
					return nil
				}

				buf.Write(bytes)
				return nil
			}
		}

		buf.WriteByte('{')
		rt := rv.Type()
		numFields := rt.NumField()

		addComma := false
		for i := 0; i < numFields; i++ {
			field := rt.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			tag := field.Tag.Get("json")
			if tag == "-" {
				continue // Skip fields marked with json:"-"
			}

			fieldName, omitEmpty := parseJSONTag(tag, field.Name)

			fv := rv.Field(i)

			// Handle omitempty
			if omitEmpty && isEmptyValue(fv) {
				continue
			}

			if addComma {
				buf.WriteByte(',')
			}
			addComma = true

			buf.WriteByte('"')
			// For simple field names, avoid escaping overhead
			if isSimpleFieldName(fieldName) {
				buf.WriteString(fieldName)
			} else {
				escapeString(fieldName, buf)
			}
			buf.WriteByte('"')
			buf.WriteByte(':')

			if err := marshal(fv, buf); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	case reflect.Interface:
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}
		// Marshal the concrete value
		return marshal(rv.Elem(), buf)
	default:
		{
			return fmt.Errorf("gravel: Marshal with unsupported type %s", rv.Type().String())
		}
	}
}

// escapeString writes a JSON-escaped string to the buffer
func escapeString(s string, buf *bytes.Buffer) {
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				// Write \u0000 format without allocating
				buf.WriteString(`\u`)
				writeHex4(buf, uint16(r))
			} else if utf8.ValidRune(r) {
				buf.WriteRune(r)
			} else {
				buf.WriteString(`\ufffd`) // replacement character
			}
		}
	}
}

// writeHex4 writes a 4-digit hex number without allocating
func writeHex4(buf *bytes.Buffer, v uint16) {
	const hex = "0123456789abcdef"
	buf.WriteByte(hex[v>>12])
	buf.WriteByte(hex[(v>>8)&0xF])
	buf.WriteByte(hex[(v>>4)&0xF])
	buf.WriteByte(hex[v&0xF])
}

// parseJSONTag parses a struct field's json tag and returns the field name and omitempty flag
func parseJSONTag(tag, fieldName string) (string, bool) {
	if tag == "" {
		return fieldName, false
	}

	// Find first comma without allocating
	commaIndex := strings.Index(tag, ",")
	if commaIndex == -1 {
		return tag, false
	}

	name := tag[:commaIndex]
	if name == "" {
		name = fieldName
	}

	// Check for omitempty without allocating a slice
	omitEmpty := strings.Contains(tag[commaIndex:], "omitempty")

	return name, omitEmpty
}

// isSimpleFieldName checks if a field name needs escaping
func isSimpleFieldName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		// Only allow letters, digits, underscore - common JSON field characters
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// isEmptyValue reports whether v is an empty value according to JSON omitempty semantics
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}

// Fast integer writing functions to avoid string allocations

func writeInt(i int64, buf *bytes.Buffer) {
	if i < 0 {
		buf.WriteByte('-')
		i = -i
	}
	writeUint(uint64(i), buf)
}

func writeUint(u uint64, buf *bytes.Buffer) {
	// Fast path for small integers using lookup table
	if u < uint64(len(smallInts)) {
		buf.WriteString(smallInts[u])
		return
	}

	// Fast path for two-digit numbers
	if u < 100 {
		buf.WriteString(twoDigits[u])
		return
	}

	// For larger values, fall back to optimized conversion
	if u == 0 {
		buf.WriteByte('0')
		return
	}

	// Calculate number of digits without allocations
	var digits [20]byte // max uint64 has 20 digits
	n := 0
	temp := u
	for temp > 0 {
		digits[n] = byte('0' + temp%10)
		temp /= 10
		n++
	}

	// Write digits in reverse order
	for i := n - 1; i >= 0; i-- {
		buf.WriteByte(digits[i])
	}
}

// Unsafe string operations for known-safe strings
func unsafeStringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		cap int
	}{s, len(s)}))
}

// writeKnownSafeString writes a string that's known to be JSON-safe
// without any escaping checks - used for field names that are pre-validated
func writeKnownSafeString(s string, w writer) error {
	return w.WriteBytes(unsafeStringToBytes(s))
}

// isSimpleString checks if a string contains only safe ASCII characters
// that don't need escaping in JSON
func isSimpleString(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		// Check for characters that need escaping
		if b < 0x20 || b >= 0x80 || b == '"' || b == '\\' {
			return false
		}
	}
	return true
}
