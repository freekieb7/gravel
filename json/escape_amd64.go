//go:build amd64
// +build amd64

package json

// Assembly function for optimized string escaping
//
//go:noescape
func escapeStringASM(src []byte, dst []byte) (needsEscape bool, pos int)

// Fast assembly-based string escaping that scans for escape characters
func escapeStringFast(s string, w writer) error {
	if len(s) == 0 {
		return nil
	}

	// Convert string to byte slice using unsafe
	src := unsafeStringToBytes(s)

	// Create a temporary buffer for copying unescaped portions
	dst := make([]byte, len(s))

	start := 0
	for start < len(s) {
		// Use assembly to find next character that needs escaping
		needsEscape, pos := escapeStringASM(src[start:], dst[start:])

		if !needsEscape {
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
