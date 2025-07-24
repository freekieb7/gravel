//go:build !amd64

package json

import (
	"errors"
	"unsafe"
)

// TapeParser fallback implementation for non-AMD64
type TapeParser struct {
	tape       []uint64
	strings    []byte
	stringEnds []int
	input      []byte
	position   int
}

const (
	TapeTypeObject    = 0
	TapeTypeArray     = 1
	TapeTypeString    = 2
	TapeTypeNumber    = 3
	TapeTypeBool      = 4
	TapeTypeNull      = 5
	TapeTypeObjectEnd = 6
	TapeTypeArrayEnd  = 7
)

type StructuralChars struct {
	indices []int
	length  int
}

// Fallback implementations
func findStructuralCharsAVX2(data []byte, indices []int) int {
	// Not used in new implementation
	return 0
}

func validateStringsAVX2(data []byte, start, end int) bool {
	// Not used in new implementation
	return true
}

func ParseFast(data []byte) (*TapeParser, error) {
	parser := &TapeParser{
		input:      data,
		tape:       make([]uint64, 0, len(data)/8),
		strings:    make([]byte, 0, len(data)/4),
		stringEnds: make([]int, 0, len(data)/32),
	}

	// Parse JSON directly without separating structural detection
	_, err := parser.parseJSON(data, 0)
	if err != nil {
		return nil, err
	}

	return parser, nil
}

// parseJSON recursively parses JSON and builds the tape, returns new position
func (p *TapeParser) parseJSON(data []byte, pos int) (int, error) {
	// Skip whitespace
	pos = p.skipWhitespace(data, pos)
	if pos >= len(data) {
		return pos, errors.New("unexpected end of input")
	}

	switch data[pos] {
	case '{':
		return p.parseObject(data, pos)
	case '[':
		return p.parseArray(data, pos)
	case '"':
		return p.parseString(data, pos)
	case 't', 'f':
		return p.parseBool(data, pos)
	case 'n':
		return p.parseNull(data, pos)
	default:
		if (data[pos] >= '0' && data[pos] <= '9') || data[pos] == '-' {
			return p.parseNumber(data, pos)
		}
		return pos, errors.New("unexpected character")
	}
}

func (p *TapeParser) parseObject(data []byte, pos int) (int, error) {
	p.tape = append(p.tape, encodeInstruction(TapeTypeObject, 0))
	pos++ // Skip '{'

	pos = p.skipWhitespace(data, pos)
	if pos >= len(data) {
		return pos, errors.New("unterminated object")
	}

	// Empty object
	if data[pos] == '}' {
		p.tape = append(p.tape, encodeInstruction(TapeTypeObjectEnd, 0))
		return pos + 1, nil
	}

	for {
		// Parse key
		pos = p.skipWhitespace(data, pos)
		if pos >= len(data) || data[pos] != '"' {
			return pos, errors.New("expected string key")
		}

		var err error
		pos, err = p.parseString(data, pos)
		if err != nil {
			return pos, err
		}

		// Skip colon
		pos = p.skipWhitespace(data, pos)
		if pos >= len(data) || data[pos] != ':' {
			return pos, errors.New("expected colon")
		}
		pos++

		// Parse value
		pos = p.skipWhitespace(data, pos)
		pos, err = p.parseJSON(data, pos)
		if err != nil {
			return pos, err
		}

		// Check for continuation
		pos = p.skipWhitespace(data, pos)
		if pos >= len(data) {
			return pos, errors.New("unterminated object")
		}

		if data[pos] == '}' {
			break
		} else if data[pos] == ',' {
			pos++
		} else {
			return pos, errors.New("expected comma or closing brace")
		}
	}

	p.tape = append(p.tape, encodeInstruction(TapeTypeObjectEnd, 0))
	return pos + 1, nil
}

func (p *TapeParser) parseArray(data []byte, pos int) (int, error) {
	p.tape = append(p.tape, encodeInstruction(TapeTypeArray, 0))
	pos++ // Skip '['

	pos = p.skipWhitespace(data, pos)
	if pos >= len(data) {
		return pos, errors.New("unterminated array")
	}

	// Empty array
	if data[pos] == ']' {
		p.tape = append(p.tape, encodeInstruction(TapeTypeArrayEnd, 0))
		return pos + 1, nil
	}

	for {
		// Parse value
		pos = p.skipWhitespace(data, pos)
		var err error
		pos, err = p.parseJSON(data, pos)
		if err != nil {
			return pos, err
		}

		// Check for continuation
		pos = p.skipWhitespace(data, pos)
		if pos >= len(data) {
			return pos, errors.New("unterminated array")
		}

		if data[pos] == ']' {
			break
		} else if data[pos] == ',' {
			pos++
		} else {
			return pos, errors.New("expected comma or closing bracket")
		}
	}

	p.tape = append(p.tape, encodeInstruction(TapeTypeArrayEnd, 0))
	return pos + 1, nil
}

func (p *TapeParser) parseString(data []byte, pos int) (int, error) {
	start := pos + 1 // Skip opening quote
	end := p.findStringEnd(data, pos)

	strOffset := len(p.strings)
	p.strings = append(p.strings, data[start:end]...)
	p.stringEnds = append(p.stringEnds, len(p.strings))
	p.tape = append(p.tape, encodeInstruction(TapeTypeString, uint64(strOffset)))
	return end + 1, nil // Skip closing quote
}

func (p *TapeParser) parseBool(data []byte, pos int) (int, error) {
	if pos+4 <= len(data) && string(data[pos:pos+4]) == "true" {
		p.tape = append(p.tape, encodeInstruction(TapeTypeBool, 1))
		return pos + 4, nil
	}
	if pos+5 <= len(data) && string(data[pos:pos+5]) == "false" {
		p.tape = append(p.tape, encodeInstruction(TapeTypeBool, 0))
		return pos + 5, nil
	}
	return pos, errors.New("invalid boolean")
}

func (p *TapeParser) parseNull(data []byte, pos int) (int, error) {
	if pos+4 <= len(data) && string(data[pos:pos+4]) == "null" {
		p.tape = append(p.tape, encodeInstruction(TapeTypeNull, 0))
		return pos + 4, nil
	}
	return pos, errors.New("invalid null")
}

func (p *TapeParser) parseNumber(data []byte, pos int) (int, error) {
	start := pos
	if data[pos] == '-' {
		pos++
	}

	// Skip digits
	for pos < len(data) && data[pos] >= '0' && data[pos] <= '9' {
		pos++
	}

	// Handle decimal point
	if pos < len(data) && data[pos] == '.' {
		pos++
		for pos < len(data) && data[pos] >= '0' && data[pos] <= '9' {
			pos++
		}
	}

	// Store number as string offset for now
	numStr := data[start:pos]
	strOffset := len(p.strings)
	p.strings = append(p.strings, numStr...)
	p.stringEnds = append(p.stringEnds, len(p.strings))
	p.tape = append(p.tape, encodeInstruction(TapeTypeNumber, uint64(strOffset)))
	return pos, nil
}

func (p *TapeParser) skipWhitespace(data []byte, pos int) int {
	for pos < len(data) && (data[pos] == ' ' || data[pos] == '\t' || data[pos] == '\n' || data[pos] == '\r') {
		pos++
	}
	return pos
}

func (p *TapeParser) findStringEnd(data []byte, pos int) int {
	pos++ // Skip opening quote
	for pos < len(data) {
		if data[pos] == '"' {
			// Check for escape
			backslashes := 0
			for i := pos - 1; i >= 0 && data[i] == '\\'; i-- {
				backslashes++
			}
			if backslashes%2 == 0 {
				return pos
			}
		}
		pos++
	}
	return pos
}

func encodeInstruction(typ uint64, payload uint64) uint64 {
	return (typ << 56) | (payload & 0x00FFFFFFFFFFFFFF)
}

func decodeInstruction(instruction uint64) (uint64, uint64) {
	return instruction >> 56, instruction & 0x00FFFFFFFFFFFFFF
}

func (p *TapeParser) GetString(offset int) string {
	if offset >= len(p.strings) {
		return ""
	}

	// Find string end using the stringEnds tracking
	end := len(p.strings)
	for _, strEnd := range p.stringEnds {
		if strEnd > offset {
			end = strEnd
			break
		}
	}

	return *(*string)(unsafe.Pointer(&struct {
		data uintptr
		len  int
	}{
		data: uintptr(unsafe.Pointer(&p.strings[offset])),
		len:  end - offset,
	}))
}

// GetTape returns a copy of the tape for inspection
func (p *TapeParser) GetTape() []uint64 {
	result := make([]uint64, len(p.tape))
	copy(result, p.tape)
	return result
}

// GetStringPool returns a copy of the string pool for inspection
func (p *TapeParser) GetStringPool() []byte {
	result := make([]byte, len(p.strings))
	copy(result, p.strings)
	return result
}

func hasAVX2() bool {
	return false // No AVX2 support on non-AMD64
}
