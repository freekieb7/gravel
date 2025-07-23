package http

import (
	"errors"
	"mime"
	"path/filepath"
	"strings"
)

// MIME type detection helpers
var commonMimeTypes = map[string]string{
	".html": "text/html; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".js":   "application/javascript; charset=utf-8",
	".json": "application/json",
	".xml":  "application/xml; charset=utf-8",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".pdf":  "application/pdf",
	".txt":  "text/plain; charset=utf-8",
}

// GetMimeType returns the MIME type for a file extension
func GetMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if mimeType, exists := commonMimeTypes[ext]; exists {
		return mimeType
	}

	// Fallback to Go's mime package
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		return mimeType
	}

	return "application/octet-stream"
}

// IsSecureScheme checks if the request is over HTTPS
func IsSecureScheme(req *Request) bool {
	// Check X-Forwarded-Proto header (common in reverse proxies)
	if proto, found := req.Header([]byte("x-forwarded-proto")); found {
		return string(proto) == "https"
	}

	// Check X-Forwarded-SSL header
	if ssl, found := req.Header([]byte("x-forwarded-ssl")); found {
		return string(ssl) == "on"
	}

	// For now, assume HTTP unless explicitly set
	// In a real implementation, you'd check the connection details
	return false
}

// GetClientIP extracts the real client IP from headers
func GetClientIP(req *Request) string {
	// Check X-Forwarded-For header first (most common)
	if xff, found := req.Header([]byte("x-forwarded-for")); found {
		ips := strings.Split(string(xff), ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if realIP, found := req.Header([]byte("x-real-ip")); found {
		return string(realIP)
	}

	// Check CF-Connecting-IP (Cloudflare)
	if cfIP, found := req.Header([]byte("cf-connecting-ip")); found {
		return string(cfIP)
	}

	// Fallback - would need to get from connection in real implementation
	return "unknown"
}

// ValidateMethod checks if HTTP method is valid
func ValidateMethod(method []byte) bool {
	validMethods := [][]byte{
		[]byte("GET"), []byte("POST"), []byte("PUT"), []byte("DELETE"),
		[]byte("PATCH"), []byte("HEAD"), []byte("OPTIONS"), []byte("TRACE"),
		[]byte("CONNECT"),
	}

	for _, valid := range validMethods {
		if len(method) == len(valid) {
			match := true
			for i := 0; i < len(method); i++ {
				if method[i] != valid[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

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
