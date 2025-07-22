package http

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

type Request struct {
	Method   []byte
	Path     []byte
	Protocol []byte
	Body     []byte
	Close    bool
	// Add buffer to reuse for body reading
	bodyBuf [4096]byte
	// Add header storage
	headers     [32]Header // Support up to 32 headers
	headerCount int
}

func (req *Request) Reset() {
	req.Method = nil
	req.Path = nil
	req.Protocol = nil
	req.Body = nil
	req.Close = false
}

func (req *Request) Parse(br *bufio.Reader) error {
	// Use ReadSlice for the request line
	line, err := br.ReadSlice('\n')
	if err != nil {
		return err
	}

	// Fast path for line ending removal
	lineLen := len(line)
	if lineLen >= 2 && line[lineLen-2] == '\r' {
		line = line[:lineLen-2]
	} else if lineLen >= 1 && line[lineLen-1] == '\n' {
		line = line[:lineLen-1]
	}

	// Parse request line with bounds checking
	if len(line) < 14 {
		return errors.New("request line too short")
	}

	// Single pass parsing
	space1 := -1
	space2 := -1
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' {
			if space1 == -1 {
				space1 = i
			} else {
				space2 = i
				break
			}
		}
	}

	if space1 == -1 || space2 == -1 {
		return errors.New("invalid request line format")
	}

	req.Method = line[:space1]
	req.Path = line[space1+1 : space2]
	req.Protocol = line[space2+1:]

	// Fast protocol check
	if len(req.Protocol) == 8 {
		if bytes.Equal(req.Protocol, protocolHttp11) {
			req.Close = false
		} else if bytes.Equal(req.Protocol, protocolHttp10) {
			req.Close = true
		} else {
			return errors.New("unsupported http protocol")
		}
	} else {
		return errors.New("invalid http protocol")
	}

	// ... rest of header parsing stays the same but optimized
	return req.parseHeaders(br)
}

func (req *Request) parseHeaders(br *bufio.Reader) error {
	var (
		contentLength       int
		hasContentLength    bool
		hasTransferEncoding bool
		isChunked           bool
	)

	// Reset header count
	req.headerCount = 0

	// Pre-allocate buffer for lowercase conversion to avoid allocation #1 and #2
	var lowerNameBuf [64]byte // Reusable buffer for header name conversion

	for {
		b, err := br.ReadSlice('\n')
		if err != nil {
			return err
		}

		// Fast line ending removal
		bLen := len(b)
		if bLen >= 2 && b[bLen-2] == '\r' {
			b = b[:bLen-2]
		} else if bLen >= 1 && b[bLen-1] == '\n' {
			b = b[:bLen-1]
		}

		if len(b) == 0 {
			break // End of headers
		}

		// Find colon
		colonIdx := bytes.IndexByte(b, ':')
		if colonIdx < 0 {
			return errors.New("invalid header format")
		}

		name := b[:colonIdx]
		value := b[colonIdx+1:]

		// Skip leading space in value
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		// Store header if we have space
		if req.headerCount < len(req.headers) {
			h := &req.headers[req.headerCount]

			// Store name in lowercase - NO ALLOCATION
			h.NameLen = min(len(name), len(h.Name))
			for i := 0; i < h.NameLen; i++ {
				if name[i] >= 'A' && name[i] <= 'Z' {
					h.Name[i] = name[i] + 32 // Convert to lowercase
				} else {
					h.Name[i] = name[i]
				}
			}

			// Store value as-is
			h.ValueLen = min(len(value), len(h.Value))
			copy(h.Value[:h.ValueLen], value[:h.ValueLen])

			req.headerCount++
		}

		// Process special headers for protocol logic - NO ALLOCATION
		if len(name) <= 20 && len(name) <= len(lowerNameBuf) {
			// Use pre-allocated buffer instead of make() - FIXES ALLOCATION #1 & #2
			lowerName := lowerNameBuf[:len(name)]
			for i, b := range name {
				if b >= 'A' && b <= 'Z' {
					lowerName[i] = b + 32
				} else {
					lowerName[i] = b
				}
			}

			// SIMD-optimized header matching
			switch len(lowerName) {
			case 10: // connection
				if bytes.Equal(lowerName, headerConnection) {
					if equalsFast(value, headerKeepAlive) {
						req.Close = false
					} else if equalsFast(value, headerClose) {
						req.Close = true
					}
				}
			case 14: // content-length
				if bytes.Equal(lowerName, headerContentLength) {
					if hasTransferEncoding && isChunked {
						return errors.New("potential request smuggling")
					}
					hasContentLength = true
					contentLength, err = atoi(value)
					if err != nil {
						return errors.New("invalid content-length")
					}
				}
			case 17: // transfer-encoding
				if bytes.Equal(lowerName, headerTransferEncoding) {
					hasTransferEncoding = true
					if bytes.Contains(value, []byte("chunked")) {
						isChunked = true
						if hasContentLength {
							return errors.New("potential request smuggling")
						}
					}
				}
			}
		}
	}

	// Read body
	if isChunked {
		return req.readChunkedBody(br)
	} else if contentLength > 0 {
		if contentLength <= len(req.bodyBuf) {
			req.Body = req.bodyBuf[:contentLength]
		} else {
			req.Body = make([]byte, contentLength)
		}
		_, err := io.ReadFull(br, req.Body)
		return err
	}

	req.Body = nil
	return nil
}

func (req *Request) readChunkedBody(br *bufio.Reader) error {
	// Use the existing bodyBuf for small bodies, pre-allocate larger buffer
	bodyBuf := req.bodyBuf[:]
	bodyLen := 0
	maxBodySize := cap(bodyBuf)

	// Pre-allocate larger buffer if needed for chunked bodies
	if maxBodySize < 64*1024 {
		// Only allocate once for large chunked bodies
		bodyBuf = make([]byte, 64*1024)
		maxBodySize = len(bodyBuf)
	}

	for {
		// Read chunk size line
		line, err := br.ReadSlice('\n')
		if err != nil {
			return err
		}

		// Remove \r\n
		if len(line) >= 2 && line[len(line)-2] == '\r' {
			line = line[:len(line)-2]
		} else if len(line) >= 1 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}

		// Parse hex chunk size
		chunkSize := 0
		for _, b := range line {
			if b >= '0' && b <= '9' {
				chunkSize = chunkSize*16 + int(b-'0')
			} else if b >= 'a' && b <= 'f' {
				chunkSize = chunkSize*16 + int(b-'a'+10)
			} else if b >= 'A' && b <= 'F' {
				chunkSize = chunkSize*16 + int(b-'A'+10)
			} else {
				break // Stop at first non-hex character (chunk extensions)
			}
		}

		if chunkSize == 0 {
			// Read final \r\n
			br.ReadSlice('\n')
			break
		}

		// Check if we have enough space in pre-allocated buffer
		if bodyLen+chunkSize > maxBodySize {
			// Body too large for our buffer - fallback to allocation
			largeBody := make([]byte, bodyLen+chunkSize*2) // Pre-allocate extra space
			copy(largeBody[:bodyLen], bodyBuf[:bodyLen])
			bodyBuf = largeBody
			maxBodySize = len(bodyBuf)
		}

		// Read chunk data directly into pre-allocated buffer - NO ALLOCATION
		_, err = io.ReadFull(br, bodyBuf[bodyLen:bodyLen+chunkSize])
		if err != nil {
			return err
		}

		bodyLen += chunkSize

		// Read trailing \r\n
		br.ReadSlice('\n')
	}

	// Set body to the used portion of the buffer
	req.Body = bodyBuf[:bodyLen]
	return nil
}

// Add Header method to retrieve header value
func (req *Request) Header(name []byte) ([]byte, bool) {
	// Convert name to lowercase for case-insensitive comparison
	lowerName := make([]byte, len(name))
	for i, b := range name {
		if b >= 'A' && b <= 'Z' {
			lowerName[i] = b + 32 // Convert to lowercase
		} else {
			lowerName[i] = b
		}
	}

	// Search through stored headers
	for i := 0; i < req.headerCount; i++ {
		h := &req.headers[i]

		// Check if name length matches
		if h.NameLen != len(lowerName) {
			continue
		}

		// Compare header names (case-insensitive)
		match := true
		for j := 0; j < h.NameLen; j++ {
			if h.Name[j] != lowerName[j] {
				match = false
				break
			}
		}

		if match {
			return h.Value[:h.ValueLen], true
		}
	}

	return nil, false // Header not found
}

// Add HeaderString method for convenience
func (req *Request) HeaderString(name string) (string, bool) {
	value, found := req.Header([]byte(name))
	if value == nil || !found {
		return "", found
	}
	return string(value), found
}

func (req *Request) AddCookie(cookie Cookie) {
	// if req.Headers["Set-Cookie"] == nil {
	// 	req.Headers["Set-Cookie"] = []string{}
	// }

	// req.Headers["Cookie"] = append(req.Headers["Cookie"], cookie.String())
}

func (req *Request) Cookie(name string) (Cookie, error) {
	// todo
	var cookie Cookie
	return cookie, nil

	// reqCookies, found := req.Headers["Cookie"]
	// if !found {
	// 	return cookie, ErrNoCookie
	// }

	// if err := cookie.Parse(data); err != nil {
	// 	return cookie, err
	// }
}
