package json

import (
	"errors"
	"strconv"
	"strings"
)

// FastValue represents a zero-copy JSON value
type FastValue struct {
	parser    *TapeParser
	tapePos   int
	valueType uint64
	payload   uint64
}

// ZeroCopySearch implements simdjson-go style zero-copy value extraction
func ZeroCopySearch(data []byte, path string) (*FastValue, error) {
	parser, err := ParseFast(data)
	if err != nil {
		return nil, err
	}

	// Navigate to the requested path
	pathParts := strings.Split(strings.Trim(path, "."), ".")
	value, err := parser.Navigate(pathParts)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// Navigate traverses the tape to find a value at the given path
func (p *TapeParser) Navigate(pathParts []string) (*FastValue, error) {
	currentPos := 0

	for _, part := range pathParts {
		if currentPos >= len(p.tape) {
			return nil, errors.New("path not found")
		}

		instruction := p.tape[currentPos]
		valueType, payload := decodeInstruction(instruction)

		switch valueType {
		case TapeTypeObject:
			// Navigate into object
			found, newPos, err := p.findObjectKey(part, currentPos+1)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, errors.New("key not found: " + part)
			}
			currentPos = newPos

		case TapeTypeArray:
			// Navigate into array by index
			index, err := strconv.Atoi(part)
			if err != nil {
				return nil, errors.New("invalid array index: " + part)
			}
			found, newPos, err := p.findArrayIndex(index, currentPos+1)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, errors.New("array index not found")
			}
			currentPos = newPos

		default:
			return &FastValue{
				parser:    p,
				tapePos:   currentPos,
				valueType: valueType,
				payload:   payload,
			}, nil
		}
	}

	// Return the final value
	if currentPos >= len(p.tape) {
		return nil, errors.New("invalid tape position")
	}

	instruction := p.tape[currentPos]
	valueType, payload := decodeInstruction(instruction)

	return &FastValue{
		parser:    p,
		tapePos:   currentPos,
		valueType: valueType,
		payload:   payload,
	}, nil
}

// findObjectKey locates a specific key in an object and returns the position of its value
func (p *TapeParser) findObjectKey(key string, startPos int) (bool, int, error) {
	pos := startPos

	for pos < len(p.tape) {
		instruction := p.tape[pos]
		valueType, payload := decodeInstruction(instruction)

		if valueType == TapeTypeObjectEnd {
			return false, pos, nil
		}

		if valueType == TapeTypeString {
			// This should be a key
			stringValue := p.GetString(int(payload))
			if stringValue == key {
				// Return position of the value (next instruction)
				if pos+1 >= len(p.tape) {
					return false, pos, errors.New("key without value")
				}
				return true, pos + 1, nil
			}
			// Skip key and value pair
			pos += 2
			// Skip nested structures if the value is an object or array
			if pos-1 < len(p.tape) {
				valueInstruction := p.tape[pos-1]
				valueType, _ := decodeInstruction(valueInstruction)
				if valueType == TapeTypeObject || valueType == TapeTypeArray {
					pos = p.skipNestedStructure(pos-1) + 1
				}
			}
		} else {
			pos++
		}
	}

	return false, pos, errors.New("malformed object")
}

// findArrayIndex locates a specific index in an array
func (p *TapeParser) findArrayIndex(index int, startPos int) (bool, int, error) {
	pos := startPos
	currentIndex := 0

	for pos < len(p.tape) {
		instruction := p.tape[pos]
		valueType, _ := decodeInstruction(instruction)

		if valueType == TapeTypeArrayEnd {
			return false, pos, nil
		}

		if currentIndex == index {
			return true, pos, nil
		}

		// Skip to next value
		if valueType == TapeTypeObject || valueType == TapeTypeArray {
			pos = p.skipNestedStructure(pos) + 1
		} else {
			pos++
		}
		currentIndex++
	}

	return false, pos, errors.New("malformed array")
}

// skipNestedStructure skips over a complete object or array structure
func (p *TapeParser) skipNestedStructure(startPos int) int {
	pos := startPos
	if pos >= len(p.tape) {
		return pos
	}

	instruction := p.tape[pos]
	valueType, _ := decodeInstruction(instruction)

	if valueType == TapeTypeObject {
		pos++
		level := 1
		for pos < len(p.tape) && level > 0 {
			instruction = p.tape[pos]
			valueType, _ = decodeInstruction(instruction)
			if valueType == TapeTypeObject {
				level++
			} else if valueType == TapeTypeObjectEnd {
				level--
			}
			pos++
		}
		return pos - 1
	} else if valueType == TapeTypeArray {
		pos++
		level := 1
		for pos < len(p.tape) && level > 0 {
			instruction = p.tape[pos]
			valueType, _ = decodeInstruction(instruction)
			if valueType == TapeTypeArray {
				level++
			} else if valueType == TapeTypeArrayEnd {
				level--
			}
			pos++
		}
		return pos - 1
	}

	return pos
}

// FastValue methods for zero-copy access
func (v *FastValue) GetString() (string, error) {
	if v.valueType != TapeTypeString {
		return "", errors.New("not a string value")
	}
	return v.parser.GetString(int(v.payload)), nil
}

func (v *FastValue) GetInt() (int64, error) {
	if v.valueType != TapeTypeNumber {
		return 0, errors.New("not a number value")
	}

	// Get number string from strings pool
	numberStr := v.parser.GetString(int(v.payload))
	return strconv.ParseInt(numberStr, 10, 64)
}

func (v *FastValue) GetFloat() (float64, error) {
	if v.valueType != TapeTypeNumber {
		return 0, errors.New("not a number value")
	}

	// Get number string from strings pool
	numberStr := v.parser.GetString(int(v.payload))
	return strconv.ParseFloat(numberStr, 64)
}

func (v *FastValue) GetBool() (bool, error) {
	if v.valueType != TapeTypeBool {
		return false, errors.New("not a boolean value")
	}
	return v.payload == 1, nil
}

func (v *FastValue) IsNull() bool {
	return v.valueType == TapeTypeNull
}

// High-level convenience functions that use the fast parser
func SearchString(data []byte, path string) (string, error) {
	value, err := ZeroCopySearch(data, path)
	if err != nil {
		return "", err
	}
	return value.GetString()
}

func SearchInt(data []byte, path string) (int64, error) {
	value, err := ZeroCopySearch(data, path)
	if err != nil {
		return 0, err
	}
	return value.GetInt()
}

func SearchFloat(data []byte, path string) (float64, error) {
	value, err := ZeroCopySearch(data, path)
	if err != nil {
		return 0, err
	}
	return value.GetFloat()
}

func SearchBool(data []byte, path string) (bool, error) {
	value, err := ZeroCopySearch(data, path)
	if err != nil {
		return false, err
	}
	return value.GetBool()
}
