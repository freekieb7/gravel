package json

import (
	"errors"
	"fmt"
)

var (
	ErrUnexpectedEndOfInput = errors.New("unexpected end of input")
	ErrSyntaxError          = errors.New("syntax error")
	ErrBufferUnderrun       = errors.New("buffer underrun")
)

type TokenType int

const (
	TOKEN_TYPE_OBJECT_BEGIN TokenType = iota
	TOKEN_TYPE_OBJECT_END
	TOKEN_TYPE_ARRAY_BEGIN
	TOKEN_TYPE_ARRAY_END
	TOKEN_TYPE_TRUE
	TOKEN_TYPE_FALSE
	TOKEN_TYPE_NULL
	TOKEN_TYPE_NUMBER
	TOKEN_TYPE_PARTIAL_NUMBER
	TOKEN_TYPE_STRING
	TOKEN_TYPE_PARTIAL_STRING
	TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1
	TOKEN_TYPE_PARTIAL_STRING_ESCAPED_2
	TOKEN_TYPE_PARTIAL_STRING_ESCAPED_3
	TOKEN_TYPE_PARTIAL_STRING_ESCAPED_4
	TOKEN_TYPE_END_OF_DOCUMENT
	TOKEN_TYPE_INVALID
)

type Token struct {
	Type  TokenType
	Value []byte
}

var (
	NullToken = Token{Type: TOKEN_TYPE_INVALID, Value: nil}
)

type State int

const (
	STATE_VALUE State = iota
	STATE_POST_VALUE

	STATE_OBJECT_START
	STATE_OBJECT_POST_COMMA

	STATE_ARRAY_START

	STATE_NUMBER_MINUS
	STATE_NUMBER_LEADING_ZERO
	STATE_NUMBER_INT
	STATE_NUMBER_POST_DOT
	STATE_NUMBER_FRAC
	STATE_NUMBER_POST_E
	STATE_NUMBER_POST_E_SIGN
	STATE_NUMBER_EXP

	STATE_STRING
	STATE_STRING_BACKSLASH
	STATE_STRING_BACKSLASH_U
	STATE_STRING_BACKSLASH_U_1
	STATE_STRING_BACKSLASH_U_2
	STATE_STRING_BACKSLASH_U_3
	STATE_STRING_SURROGATE_HALF
	STATE_STRING_SURROGATE_HALF_BACKSLASH
	STATE_STRING_SURROGATE_HALF_BACKSLASH_U
	STATE_STRING_SURROGATE_HALF_BACKSLASH_U_1
	STATE_STRING_SURROGATE_HALF_BACKSLASH_U_2
	STATE_STRING_SURROGATE_HALF_BACKSLASH_U_3

	// From http://unicode.org/mail-arch/unicode-ml/y2003-m02/att-0467/01-The_Algorithm_to_Valide_an_UTF-8_String

	STATE_STRING_UTF8_LAST_BYTE                                        // State A
	STATE_STRING_UTF8_SECOND_TO_LAST_BYTE                              // State B
	STATE_STRING_UTF8_SECOND_TO_LAST_BYTE_GUARF_AGAINST_OVERLONG       // State C
	STATE_STRING_UTF8_SECOND_TO_LAST_BYTE_GUARF_AGAINST_SURROGATE_HALF // State D
	STATE_STRING_UTF8_THIRD_TO_LAST_BYTE                               // State E
	STATE_STRING_UTF8_THIRD_TO_LAST_BYTE_GUARD_AGAINST_OVERLONG        // State F
	STATE_STRING_UTF8_THIRD_TO_LAST_BYTE_GUARD_AGAINST_TOO_LARGE       // State G

	STATE_LITERAL_T
	STATE_LITERAL_TR
	STATE_LITERAL_TRU
	STATE_LITERAL_F
	STATE_LITERAL_FA
	STATE_LITERAL_FAL
	STATE_LITERAL_FALS
	STATE_LITERAL_N
	STATE_LITERAL_NU
	STATE_LITERAL_NUL
)

const (
	OBJECT_MODE byte = iota
	ARRAY_MODE
)

type Stack struct {
	Elements [100]byte
	Cursor   int
}

func (s *Stack) Len() int {
	return s.Cursor
}

func (s *Stack) Push(v byte) {
	s.Elements[s.Cursor] = v
	s.Cursor++
}

func (s *Stack) Pop() byte {
	s.Cursor--
	return s.Elements[s.Cursor]
}

func (s *Stack) Peek() byte {
	return s.Elements[s.Cursor-1]
}

type Scanner struct {
	Input        []byte
	Cursor       int
	IsEndOfInput bool

	InputLength       int
	State             State
	Stack             Stack
	ValueStart        int
	StringIsObjectKey bool
	Utf16CodeUnits    [2]uint16
}

func NewScanner(input []byte) Scanner {
	return Scanner{
		Input:        input,
		Cursor:       0,
		IsEndOfInput: true,

		InputLength: len(input),
		State:       STATE_VALUE,
		Stack: Stack{
			Elements: [100]byte{},
			Cursor:   0,
		},
		ValueStart:        0,
		StringIsObjectKey: false,
	}
}

func (scanner *Scanner) Peek() (Token, error) {
	cursor := scanner.Cursor
	isEndOfInput := scanner.IsEndOfInput
	state := scanner.State
	stackCursor := scanner.Stack.Cursor
	valueStart := scanner.ValueStart
	stringIsObjectKey := scanner.StringIsObjectKey

	t, err := scanner.Next()

	// restore
	scanner.Cursor = cursor
	scanner.IsEndOfInput = isEndOfInput
	scanner.State = state
	scanner.Stack.Cursor = stackCursor
	scanner.ValueStart = valueStart
	scanner.StringIsObjectKey = stringIsObjectKey

	return t, err
}

func (scanner *Scanner) Next() (Token, error) {
stateLoop:
	for range 100 {
		switch scanner.State {
		case STATE_VALUE:
			{
				b, err := scanner.NextByteSkipWhiteSpace()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case '{':
					{
						scanner.Stack.Push(OBJECT_MODE)
						scanner.Cursor++
						scanner.State = STATE_OBJECT_START
						return Token{Type: TOKEN_TYPE_OBJECT_BEGIN, Value: []byte{'{'}}, nil
					}
				case '[':
					{
						scanner.Stack.Push(ARRAY_MODE)
						scanner.Cursor++
						scanner.State = STATE_ARRAY_START
						return Token{Type: TOKEN_TYPE_ARRAY_BEGIN, Value: []byte{'['}}, nil
					}

					// string
				case '"':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING
						continue stateLoop
					}

					// number
				case '1', '2', '3', '4', '5', '6', '7', '8', '9':
					{
						scanner.ValueStart = scanner.Cursor
						scanner.Cursor++
						scanner.State = STATE_NUMBER_INT
						continue stateLoop
					}
				case '0':
					{
						scanner.ValueStart = scanner.Cursor
						scanner.Cursor++
						scanner.State = STATE_NUMBER_LEADING_ZERO
						continue stateLoop
					}
				case '-':
					{
						scanner.ValueStart = scanner.Cursor
						scanner.Cursor++
						scanner.State = STATE_NUMBER_MINUS
						continue stateLoop
					}

					// literals
				case 't':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_T
						continue stateLoop
					}
				case 'f':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_F
						continue stateLoop
					}
				case 'n':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_N
						continue stateLoop
					}
				default:
					return NullToken, ErrSyntaxError
				}
			}

		case STATE_POST_VALUE:
			{
				reached, err := scanner.SkipWhitespaceCheckEnd()
				if err != nil {
					return NullToken, err
				}
				if reached {
					return Token{Type: TOKEN_TYPE_END_OF_DOCUMENT, Value: []byte{'e', 'o', 'f'}}, nil
				}

				b := scanner.Input[scanner.Cursor]
				if scanner.StringIsObjectKey {
					scanner.StringIsObjectKey = false
					switch b {
					case ':':
						{
							scanner.Cursor++
							scanner.State = STATE_VALUE
							continue stateLoop
						}
					}
				}

				switch b {
				case '}':
					{
						if scanner.Stack.Pop() != OBJECT_MODE {
							return NullToken, ErrSyntaxError
						}
						scanner.Cursor++
						// stay in .post_value state.
						return Token{Type: TOKEN_TYPE_OBJECT_END, Value: []byte{'}'}}, nil
					}
				case ']':
					{
						if scanner.Stack.Pop() != ARRAY_MODE {
							return NullToken, ErrSyntaxError
						}
						scanner.Cursor++
						// stay in .post_value state.
						return Token{Type: TOKEN_TYPE_ARRAY_END, Value: []byte{']'}}, nil
					}
				case ',':
					{
						switch scanner.Stack.Peek() {
						case OBJECT_MODE:
							{
								scanner.State = STATE_OBJECT_POST_COMMA
							}
						case ARRAY_MODE:
							{
								scanner.State = STATE_VALUE
							}
						}
						scanner.Cursor++
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_OBJECT_START:
			{
				b, err := scanner.NextByteSkipWhiteSpace()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case '"':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING
						scanner.StringIsObjectKey = true
						continue stateLoop
					}
				case '}':
					{
						scanner.Cursor++
						scanner.Stack.Pop()
						scanner.State = STATE_POST_VALUE
						return Token{Type: TOKEN_TYPE_OBJECT_END, Value: []byte{'}'}}, nil
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_OBJECT_POST_COMMA:
			{
				b, err := scanner.NextByteSkipWhiteSpace()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case '"':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING
						scanner.StringIsObjectKey = true
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_ARRAY_START:
			{
				b, err := scanner.NextByteSkipWhiteSpace()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case ']':
					{
						scanner.Cursor++
						scanner.Stack.Pop()
						scanner.State = STATE_POST_VALUE
						return Token{Type: TOKEN_TYPE_ARRAY_END, Value: []byte{']'}}, nil
					}
				default:
					{
						scanner.State = STATE_VALUE
						continue stateLoop
					}
				}
			}
		case STATE_NUMBER_MINUS:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInNumber(false)
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b == '0':
					{
						scanner.Cursor++
						scanner.State = STATE_NUMBER_LEADING_ZERO
						continue stateLoop
					}
				case b >= '1' && b <= '9':
					{
						scanner.Cursor++
						scanner.State = STATE_NUMBER_INT
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_NUMBER_LEADING_ZERO:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInNumber(true)
				}

				switch scanner.Input[scanner.Cursor] {
				case '.':
					{
						scanner.Cursor++
						scanner.State = STATE_NUMBER_POST_DOT
						continue stateLoop
					}
				case 'e', 'E':
					{
						scanner.Cursor++
						scanner.State = STATE_NUMBER_POST_E
						continue stateLoop
					}
				default:
					{
						v := scanner.TakeValueSlice()
						scanner.State = STATE_POST_VALUE
						return Token{Type: TOKEN_TYPE_NUMBER, Value: v}, nil
					}
				}
			}
		case STATE_NUMBER_INT:
			{
				for {
					if scanner.Cursor >= scanner.InputLength {
						break
					}

					b := scanner.Input[scanner.Cursor]
					switch {
					case b >= '0' && b <= '9':
						{
							scanner.Cursor++
							continue
						}
					case b == '.':
						{
							scanner.Cursor++
							scanner.State = STATE_NUMBER_POST_E
							continue stateLoop
						}
					case b == 'e' || b == 'E':
						{
							scanner.Cursor++
							scanner.State = STATE_NUMBER_POST_E
							continue stateLoop
						}
					default:
						{
							v := scanner.TakeValueSlice()
							scanner.State = STATE_POST_VALUE
							return Token{Type: TOKEN_TYPE_NUMBER, Value: v}, nil
						}
					}
				}

				return scanner.EndOfBufferInNumber(true)
			}
		case STATE_NUMBER_POST_DOT:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInNumber(false)
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= '0' && b <= '9':
					{
						scanner.Cursor++
						scanner.State = STATE_NUMBER_FRAC
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_NUMBER_FRAC:
			{
				for {
					if scanner.Cursor >= scanner.InputLength {
						break
					}

					switch scanner.Input[scanner.Cursor] {
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						{
							scanner.Cursor++
							continue
						}
					case 'e', 'E':
						{
							scanner.Cursor++
							scanner.State = STATE_NUMBER_POST_E
							continue stateLoop
						}
					default:
						{
							v := scanner.TakeValueSlice()
							scanner.State = STATE_POST_VALUE
							return Token{Type: TOKEN_TYPE_NUMBER, Value: v}, nil
						}
					}
				}

				return scanner.EndOfBufferInNumber(true)
			}
		case STATE_NUMBER_POST_E:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInNumber(false)
				}

				switch scanner.Input[scanner.Cursor] {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					{
						scanner.Cursor++
						scanner.State = STATE_NUMBER_EXP
						continue stateLoop
					}
				case '+', '-':
					{
						scanner.Cursor++
						scanner.State = STATE_NUMBER_POST_E_SIGN
						continue stateLoop
					}
				default:
					{
						continue stateLoop
					}
				}
			}
		case STATE_NUMBER_POST_E_SIGN:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInNumber(false)
				}

				switch scanner.Input[scanner.Cursor] {
				case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
					{
						scanner.Cursor++
						scanner.State = STATE_NUMBER_EXP
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_NUMBER_EXP:
			{
				for {
					if scanner.Cursor >= scanner.InputLength {
						break
					}

					switch scanner.Input[scanner.Cursor] {
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						{
							scanner.Cursor++
							continue
						}
					default:
						{
							v := scanner.TakeValueSlice()
							scanner.State = STATE_POST_VALUE
							return Token{Type: TOKEN_TYPE_NUMBER, Value: v}, nil
						}
					}
				}

				return scanner.EndOfBufferInNumber(true)
			}
		case STATE_STRING:
			{
				for {
					if scanner.Cursor >= scanner.InputLength {
						break
					}

					b := scanner.Input[scanner.Cursor]
					switch {
					// Base ASCII control code in string // todo zig supports only utf-8, but does this mean these checks are useless for go?
					case b <= 0x1f:
						{
							return NullToken, ErrSyntaxError
						}
						// ASCII plain text
					case (b >= 0x20 && b <= '"'-1) || (b >= '"'+1 && b <= '\\'-1) || (b >= '\\'+1 && b <= 0x7f):
						{
							scanner.Cursor++
							continue
						}
					case b == '"':
						{
							v := scanner.TakeValueSlice()
							t := Token{Type: TOKEN_TYPE_STRING, Value: v}
							scanner.Cursor++
							scanner.State = STATE_POST_VALUE
							return t, nil
						}
					case b == '\\':
						{
							v := scanner.TakeValueSlice()
							scanner.Cursor++
							scanner.State = STATE_STRING_BACKSLASH
							if len(v) > 0 {
								return Token{Type: TOKEN_TYPE_PARTIAL_STRING, Value: v}, nil
							}
							continue stateLoop
						}
						// UTF-8 validation.
						// See http://unicode.org/mail-arch/unicode-ml/y2003-m02/att-0467/01-The_Algorithm_to_Valide_an_UTF-8_String
					case b >= 0xC2 && b <= 0xDF:
						{
							scanner.Cursor++
							scanner.State = STATE_STRING_UTF8_LAST_BYTE
							continue stateLoop
						}
					case b == 0xE0:
						{
							scanner.Cursor++
							scanner.State = STATE_STRING_UTF8_SECOND_TO_LAST_BYTE_GUARF_AGAINST_OVERLONG
							continue stateLoop
						}
					case (b >= 0xE1 && b <= 0xEC) || (b >= 0xEE && b <= 0xEF):
						{
							scanner.Cursor++
							scanner.State = STATE_STRING_UTF8_SECOND_TO_LAST_BYTE
							continue stateLoop
						}
					case b == 0xED:
						{
							scanner.Cursor++
							scanner.State = STATE_STRING_UTF8_SECOND_TO_LAST_BYTE_GUARF_AGAINST_SURROGATE_HALF
						}
					case b == 0xF0:
						{
							scanner.Cursor++
							scanner.State = STATE_STRING_UTF8_THIRD_TO_LAST_BYTE_GUARD_AGAINST_OVERLONG
						}
					case b >= 0xF1 && b <= 0xF3:
						{
							scanner.Cursor++
							scanner.State = STATE_STRING_UTF8_THIRD_TO_LAST_BYTE
						}
					case b == 0xF4:
						{
							scanner.Cursor++
							scanner.State = STATE_STRING_UTF8_THIRD_TO_LAST_BYTE_GUARD_AGAINST_TOO_LARGE
							continue stateLoop
						}
						// Invalid UTF-8
					case (b >= 0x80 && b <= 0xc1) || (b >= 0xF5):
						{
							return NullToken, ErrSyntaxError
						}
					default:
						{
							scanner.Cursor++
							continue
						}
					}
				}

				if scanner.IsEndOfInput {
					return NullToken, ErrUnexpectedEndOfInput
				}

				v := scanner.TakeValueSlice()
				if len(v) > 0 {
					return Token{Type: TOKEN_TYPE_PARTIAL_STRING, Value: v}, nil
				}

				return NullToken, ErrBufferUnderrun
			}
		case STATE_STRING_BACKSLASH:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				switch scanner.Input[scanner.Cursor] {
				case '"', '\\', '/':
					{
						scanner.ValueStart = scanner.Cursor
						scanner.Cursor++
						scanner.State = STATE_STRING
						continue stateLoop
					}
				case 'b':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING
						return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1, Value: []byte{0x08}}, nil
					}
				case 'f':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING
						return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1, Value: []byte{0x0c}}, nil
					}
				case 'n':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING
						return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1, Value: []byte{'\n'}}, nil
					}
				case 'r':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING
						return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1, Value: []byte{'\r'}}, nil
					}
				case 't':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING
						return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1, Value: []byte{'t'}}, nil
					}
				case 'u':
					{
						scanner.Cursor++
						scanner.ValueStart = scanner.Cursor
						scanner.State = STATE_STRING_BACKSLASH_U
						return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1, Value: []byte{'u'}}, nil
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_STRING_BACKSLASH_U:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= '0' && b <= '9':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'0') << 12
					}
				case b >= 'A' && b <= 'F':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'A'+10) << 12
					}
				case b >= 'a' && b <= 'f':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'a'+10) << 12
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}

				scanner.Cursor++
				scanner.State = STATE_STRING_BACKSLASH_U_1
			}
		case STATE_STRING_BACKSLASH_U_1:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= '0' && b <= '9':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'0') << 8
					}
				case b >= 'A' && b <= 'F':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'A'+10) << 8
					}
				case b >= 'a' && b <= 'f':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'a'+10) << 8
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}

				scanner.Cursor++
				scanner.State = STATE_STRING_BACKSLASH_U_2
			}
		case STATE_STRING_BACKSLASH_U_2:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= '0' && b <= '9':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'0') << 4
					}
				case b >= 'A' && b <= 'F':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'A'+10) << 4
					}
				case b >= 'a' && b <= 'f':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b-'a'+10) << 4
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}

				scanner.Cursor++
				scanner.State = STATE_STRING_BACKSLASH_U_3
			}
		case STATE_STRING_BACKSLASH_U_3:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= '0' && b <= '9':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b - '0')
					}
				case b >= 'A' && b <= 'F':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b - 'A' + 10)
					}
				case b >= 'a' && b <= 'f':
					{
						scanner.Utf16CodeUnits[0] |= uint16(b - 'a' + 10)
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}

				scanner.Cursor++

				if scanner.Utf16CodeUnits[0] & ^uint16(0x03ff) == 0xd800 { // utf16IsHighSurrogate
					scanner.State = STATE_STRING_SURROGATE_HALF
					continue stateLoop
				}

				if scanner.Utf16CodeUnits[0] & ^uint16(0x03ff) == 0xdc00 { // utf16IsLowSurrogate
					return NullToken, ErrSyntaxError // Unexpected low surrogate half.
				}

				scanner.ValueStart = scanner.Cursor
				scanner.State = STATE_STRING
				return scanner.PartialStringCodepoint(scanner.Utf16CodeUnits), nil
			}
		case STATE_STRING_SURROGATE_HALF:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				switch scanner.Input[scanner.Cursor] {
				case '\\':
					{
						scanner.Cursor++
						scanner.State = STATE_STRING_SURROGATE_HALF_BACKSLASH
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Expected low surrogate half.
					}
				}
			}
		case STATE_STRING_SURROGATE_HALF_BACKSLASH:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				switch scanner.Input[scanner.Cursor] {
				case 'u':
					{
						scanner.Cursor++
						scanner.State = STATE_STRING_SURROGATE_HALF_BACKSLASH_U
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Expected low surrogate half.
					}
				}
			}
		case STATE_STRING_SURROGATE_HALF_BACKSLASH_U:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				switch scanner.Input[scanner.Cursor] {
				case 'D', 'd':
					{
						scanner.Cursor++
						scanner.Utf16CodeUnits[1] = 0xD << 12
						scanner.State = STATE_STRING_SURROGATE_HALF_BACKSLASH_U_1
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Expected low surrogate half.
					}
				}
			}
		case STATE_STRING_SURROGATE_HALF_BACKSLASH_U_1:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= 'C' && b <= 'F':
					{
						scanner.Cursor++
						scanner.Utf16CodeUnits[1] |= uint16(b-'A'+10) << 8
						scanner.State = STATE_STRING_SURROGATE_HALF_BACKSLASH_U_2
						continue stateLoop
					}
				case b >= 'c' && b <= 'f':
					{
						scanner.Cursor++
						scanner.Utf16CodeUnits[1] |= uint16(b-'a'+10) << 8
						scanner.State = STATE_STRING_SURROGATE_HALF_BACKSLASH_U_2
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Expected low surrogate half.
					}
				}
			}
		case STATE_STRING_SURROGATE_HALF_BACKSLASH_U_2:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= '0' && b <= '9':
					{
						scanner.Cursor++
						scanner.Utf16CodeUnits[1] |= uint16(b-'0') << 4
						scanner.State = STATE_STRING_SURROGATE_HALF_BACKSLASH_U_3
					}
				case b >= 'C' && b <= 'F':
					{
						scanner.Cursor++
						scanner.Utf16CodeUnits[1] |= uint16(b-'A'+10) << 4
						scanner.State = STATE_STRING_SURROGATE_HALF_BACKSLASH_U_2
						continue stateLoop
					}
				case b >= 'c' && b <= 'f':
					{
						scanner.Cursor++
						scanner.Utf16CodeUnits[1] |= uint16(b-'a'+10) << 4
						scanner.State = STATE_STRING_SURROGATE_HALF_BACKSLASH_U_2
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_STRING_SURROGATE_HALF_BACKSLASH_U_3:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= '0' && b <= '9':
					{
						scanner.Utf16CodeUnits[1] |= uint16(b - '0')
					}
				case b >= 'C' && b <= 'F':
					{
						scanner.Utf16CodeUnits[1] |= uint16(b - 'A' + 10)
					}
				case b >= 'c' && b <= 'f':
					{
						scanner.Utf16CodeUnits[1] |= uint16(b - 'a' + 10)
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
				scanner.Cursor++
				scanner.ValueStart = scanner.Cursor
				scanner.State = STATE_STRING
				return scanner.PartialStringCodepoint(scanner.Utf16CodeUnits), nil
			}
		case STATE_STRING_UTF8_LAST_BYTE:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= 0x80 && b <= 0xBF:
					{
						scanner.Cursor++
						scanner.State = STATE_STRING
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Invalid UTF-8
					}
				}
			}
		case STATE_STRING_UTF8_SECOND_TO_LAST_BYTE:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= 0x80 && b <= 0xBF:
					{
						scanner.Cursor++
						scanner.State = STATE_STRING_UTF8_LAST_BYTE
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Invalid UTF-8
					}
				}
			}
		case STATE_STRING_UTF8_SECOND_TO_LAST_BYTE_GUARF_AGAINST_OVERLONG:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= 0xA0 && b <= 0xBF:
					{
						scanner.Cursor++
						scanner.State = STATE_STRING_UTF8_LAST_BYTE
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Invalid UTF-8
					}
				}
			}
		case STATE_STRING_UTF8_SECOND_TO_LAST_BYTE_GUARF_AGAINST_SURROGATE_HALF:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= 0x80 && b <= 0x9F:
					{
						scanner.Cursor++
						scanner.State = STATE_STRING_UTF8_LAST_BYTE
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Invalid UTF-8
					}
				}
			}
		case STATE_STRING_UTF8_THIRD_TO_LAST_BYTE:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= 0x80 && b <= 0xBF:
					{
						scanner.Cursor++
						scanner.State = STATE_STRING_UTF8_SECOND_TO_LAST_BYTE
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Invalid UTF-8
					}
				}
			}
		case STATE_STRING_UTF8_THIRD_TO_LAST_BYTE_GUARD_AGAINST_OVERLONG:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= 0x90 && b <= 0xBF:
					{
						scanner.Cursor++
						scanner.State = STATE_STRING_UTF8_SECOND_TO_LAST_BYTE
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Invalid UTF-8
					}
				}
			}
		case STATE_STRING_UTF8_THIRD_TO_LAST_BYTE_GUARD_AGAINST_TOO_LARGE:
			{
				if scanner.Cursor >= scanner.InputLength {
					return scanner.EndOfBufferInString()
				}

				b := scanner.Input[scanner.Cursor]
				switch {
				case b >= 0x80 && b <= 0xBF:
					{
						scanner.Cursor++
						scanner.State = STATE_STRING_UTF8_SECOND_TO_LAST_BYTE
						continue stateLoop
					}
				default:
					{
						return NullToken, ErrSyntaxError // Invalid UTF-8
					}
				}
			}
		case STATE_LITERAL_T:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 'r':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_TR
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_LITERAL_TR:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 'u':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_TRU
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_LITERAL_TRU:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 'e':
					{
						scanner.Cursor++
						scanner.State = STATE_POST_VALUE
						return Token{Type: TOKEN_TYPE_TRUE, Value: []byte("true")}, nil
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_LITERAL_F:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 'a':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_FA
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_LITERAL_FA:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 'l':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_FAL
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_LITERAL_FAL:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 's':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_FALS
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_LITERAL_FALS:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 'e':
					{
						scanner.Cursor++
						scanner.State = STATE_POST_VALUE
						return Token{Type: TOKEN_TYPE_FALSE, Value: []byte("false")}, nil
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_LITERAL_N:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 'u':
					{
						scanner.Cursor++
						scanner.State = STATE_LITERAL_NU
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		case STATE_LITERAL_NUL:
			{
				b, err := scanner.ExpectByte()
				if err != nil {
					return NullToken, err
				}

				switch b {
				case 'l':
					{
						scanner.Cursor++
						scanner.State = STATE_POST_VALUE
						return Token{Type: TOKEN_TYPE_NULL, Value: []byte("null")}, nil
					}
				default:
					{
						return NullToken, ErrSyntaxError
					}
				}
			}
		}
	}

	return NullToken, errors.New("unreachable")
}

func (scanner *Scanner) NextByteSkipWhiteSpace() (byte, error) {
	scanner.SkipWhitespace()

	if scanner.Cursor >= scanner.InputLength {
		return 0, ErrUnexpectedEndOfInput
	}

	return scanner.Input[scanner.Cursor], nil
}

func (scanner *Scanner) ExpectByte() (byte, error) {
	if scanner.Cursor >= scanner.InputLength {
		if scanner.IsEndOfInput {
			return 0, ErrUnexpectedEndOfInput
		}
		return 0, ErrBufferUnderrun
	}

	return scanner.Input[scanner.Cursor], nil
}

func (scanner *Scanner) SkipWhitespace() {
	for {
		if scanner.Cursor >= scanner.InputLength {
			break
		}

		b := scanner.Input[scanner.Cursor]

		switch b {
		case ' ', '\t', '\r':
			{
				scanner.Cursor++
				continue
			}
		case '\n':
			{
				// todo diagnostics
				scanner.Cursor++
				continue
			}
		default:
			return
		}
	}
}

// todo check end of input for stream
func (scanner *Scanner) SkipWhitespaceCheckEnd() (bool, error) {
	scanner.SkipWhitespace()

	if scanner.Cursor >= scanner.InputLength {
		if scanner.Stack.Len() > 0 {
			return true, ErrUnexpectedEndOfInput
		}

		return true, nil
	}

	if scanner.Stack.Len() == 0 {
		return false, ErrSyntaxError
	}

	return false, nil
}

func (scanner *Scanner) EndOfBufferInNumber(allowEnd bool) (Token, error) {
	v := scanner.TakeValueSlice()
	if scanner.IsEndOfInput {
		if !allowEnd {
			return NullToken, ErrUnexpectedEndOfInput
		}

		scanner.State = STATE_POST_VALUE
		return Token{Type: TOKEN_TYPE_NUMBER, Value: v}, nil
	}

	if len(v) == 0 {
		return NullToken, ErrBufferUnderrun
	}

	return Token{Type: TOKEN_TYPE_PARTIAL_NUMBER, Value: v}, nil
}

// Takes the value from start to end from the input
func (scanner *Scanner) TakeValueSlice() []byte {
	v := scanner.Input[scanner.ValueStart:scanner.Cursor]
	scanner.ValueStart = scanner.Cursor
	return v
}

func (scanner *Scanner) EndOfBufferInString() (Token, error) {
	if scanner.IsEndOfInput {
		return NullToken, ErrUnexpectedEndOfInput
	}

	var offset int
	switch scanner.State {
	case STATE_STRING_BACKSLASH:
		offset = 1
	case STATE_STRING_BACKSLASH_U:
		offset = 2
	case STATE_STRING_BACKSLASH_U_1:
		offset = 3
	case STATE_STRING_BACKSLASH_U_2:
		offset = 4
	case STATE_STRING_BACKSLASH_U_3:
		offset = 5
	case STATE_STRING_SURROGATE_HALF:
		offset = 6
	case STATE_STRING_SURROGATE_HALF_BACKSLASH:
		offset = 7
	case STATE_STRING_SURROGATE_HALF_BACKSLASH_U:
		offset = 8
	case STATE_STRING_SURROGATE_HALF_BACKSLASH_U_1:
		offset = 9
	case STATE_STRING_SURROGATE_HALF_BACKSLASH_U_2:
		offset = 10
	case STATE_STRING_SURROGATE_HALF_BACKSLASH_U_3:
		offset = 11
	case STATE_STRING,
		STATE_STRING_UTF8_LAST_BYTE,
		STATE_STRING_UTF8_SECOND_TO_LAST_BYTE,
		STATE_STRING_UTF8_SECOND_TO_LAST_BYTE_GUARF_AGAINST_OVERLONG,
		STATE_STRING_UTF8_SECOND_TO_LAST_BYTE_GUARF_AGAINST_SURROGATE_HALF,
		STATE_STRING_UTF8_THIRD_TO_LAST_BYTE,
		STATE_STRING_UTF8_THIRD_TO_LAST_BYTE_GUARD_AGAINST_OVERLONG,
		STATE_STRING_UTF8_THIRD_TO_LAST_BYTE_GUARD_AGAINST_TOO_LARGE:
		offset = 0
	}

	v := scanner.TakeValueSliceMinusTrailingOffset(offset)

	if len(v) == 0 {
		return NullToken, ErrBufferUnderrun
	}

	return Token{Type: TOKEN_TYPE_PARTIAL_STRING, Value: v}, nil
}

func (scanner *Scanner) TakeValueSliceMinusTrailingOffset(offset int) []byte {
	if scanner.Cursor <= scanner.ValueStart+offset {
		return []byte("")
	}

	v := scanner.Input[scanner.ValueStart : scanner.Cursor-offset]
	scanner.ValueStart = scanner.Cursor
	return v
}

func (scanner *Scanner) PartialStringCodepoint(codePoint [2]uint16) Token {
	panic("not implemented")
	// buf := make([]byte, 4)
	// r := utf16.Decode(codePoint)
	// n := utf8.EncodeRune(buf, r[0])
	// return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1, Value: buf}

	// r := utf16.Decode()
	// switch
	//  {
	// case 1:
	// 	{
	// 		return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_1, Value: buf[:1]}
	// 	}
	// case 2:
	// 	{
	// 		return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_2, Value: buf[:2]}
	// 	}
	// case 3:
	// 	{
	// 		return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_3, Value: buf[:3]}
	// 	}
	// case 4:
	// 	{
	// 		return Token{Type: TOKEN_TYPE_PARTIAL_STRING_ESCAPED_4, Value: buf[:4]}
	// 	}
	// default:
	// 	{
	// 		panic("unreachable")
	// 	}
	// }
}

// Delivers the value of the next token, returning a subset of the original input
func (scanner *Scanner) ExtractValue() ([]byte, error) {
	stackLen := scanner.Stack.Len()

	token, err := scanner.Next()
	if err != nil {
		return nil, err
	}

	switch token.Type {
	case TOKEN_TYPE_OBJECT_BEGIN:
		{
			start := scanner.Cursor - 1

			for range 10000 {
				token, err := scanner.Next()
				if err != nil {
					return nil, err
				}

				if scanner.Stack.Len() == stackLen {
					if token.Type != TOKEN_TYPE_OBJECT_END {
						return nil, fmt.Errorf("expected object end, got %d", token.Type)
					}

					end := scanner.Cursor
					return scanner.Input[start:end], nil
				}
			}

			return nil, errors.New("infinite loop detected")
		}
	case TOKEN_TYPE_ARRAY_BEGIN:
		{
			start := scanner.Cursor - 1

			for range 10000 {
				token, err := scanner.Next()
				if err != nil {
					return nil, err
				}

				if scanner.Stack.Len() == stackLen {
					if token.Type != TOKEN_TYPE_ARRAY_END {
						return nil, fmt.Errorf("expected object end, got %d", token.Type)
					}

					end := scanner.Cursor
					return scanner.Input[start:end], nil
				}
			}

			return nil, errors.New("infinite loop")
		}
	case TOKEN_TYPE_STRING:
		{
			return append(append([]byte{'"'}, token.Value...), '"'), nil
		}
	case TOKEN_TYPE_NUMBER,
		TOKEN_TYPE_TRUE,
		TOKEN_TYPE_FALSE,
		TOKEN_TYPE_NULL:
		{
			return token.Value, nil
		}
	default:
		{
			return nil, fmt.Errorf("unsupported token")
		}
	}
}
