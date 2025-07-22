package http

import "errors"

func atoi(b []byte) (int, error) {
	var n int
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// Helper function to write integer to buffer without allocation
func writeIntToBuffer(n int, buf []byte) int {
	if n == 0 {
		buf[0] = '0'
		return 1
	}

	// Calculate digits needed
	temp := n
	digits := 0
	for temp > 0 {
		digits++
		temp /= 10
	}

	// Write digits backwards
	for i := digits - 1; i >= 0; i-- {
		buf[i] = '0' + byte(n%10)
		n /= 10
	}

	return digits
}

// Convert integer to hex without allocation
func writeHexToBuffer(n int, buf []byte) int {
	if n == 0 {
		buf[0] = '0'
		return 1
	}

	const hexDigits = "0123456789abcdef"
	digits := 0
	temp := n

	// Calculate number of hex digits needed
	for temp > 0 {
		digits++
		temp >>= 4
	}

	// Write hex digits backwards
	for i := digits - 1; i >= 0; i-- {
		buf[i] = hexDigits[n&0xF]
		n >>= 4
	}

	return digits
}

func hexToByte(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 255 // Invalid hex
}
